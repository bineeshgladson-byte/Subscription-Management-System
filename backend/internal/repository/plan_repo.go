package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type PlanRepo struct{}

func NewPlanRepo() *PlanRepo { return &PlanRepo{} }

func withLines(db *gorm.DB) *gorm.DB {
	return db.Preload("Products", func(q *gorm.DB) *gorm.DB { return q.Order("id") }).
		Preload("Products.Product").Preload("Products.Variant")
}

func (r *PlanRepo) Create(db *gorm.DB, p *models.SubscriptionPlan) error {
	return db.Omit("Products").Create(p).Error
}

func (r *PlanRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.SubscriptionPlan{}).Where("id = ?", id).Updates(fields).Error
}

func (r *PlanRepo) FindByID(db *gorm.DB, id int64) (*models.SubscriptionPlan, error) {
	var p models.SubscriptionPlan
	err := withLines(db).First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *PlanRepo) NameTaken(db *gorm.DB, name string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.SubscriptionPlan{}).
		Where("lower(name) = lower(?) AND id <> ?", strings.TrimSpace(name), exceptID).Count(&n).Error
	return n > 0, err
}

// ReplaceLines deletes a plan's lines and inserts the new ones (inside the caller's tx).
func (r *PlanRepo) ReplaceLines(tx *gorm.DB, planID int64, lines []models.PlanProduct) error {
	if err := tx.Where("plan_id = ?", planID).Delete(&models.PlanProduct{}).Error; err != nil {
		return err
	}
	for i := range lines {
		lines[i].ID = 0
		lines[i].PlanID = planID
	}
	if len(lines) == 0 {
		return nil
	}
	return tx.Omit("Product", "Variant").Create(&lines).Error
}

// recalcSQL is the ONE place the plan price formula lives:
// price = Σ quantity × (price_override, or sales_price + variant extra_price).
const recalcSQL = `
UPDATE subscription_plans sp SET price = coalesce((
    SELECT sum(pp.quantity * coalesce(pp.price_override, p.sales_price + coalesce(v.extra_price, 0)))
    FROM plan_products pp
    JOIN products p ON p.id = pp.product_id
    LEFT JOIN product_variants v ON v.id = pp.variant_id
    WHERE pp.plan_id = sp.id), 0),
  updated_at = now()
WHERE `

func (r *PlanRepo) RecalculatePrice(tx *gorm.DB, planID int64) error {
	return tx.Exec(recalcSQL+"sp.id = ?", planID).Error
}

// RecalculateForProduct refreshes every plan that contains the product (after
// its sales price or a variant's extra price changed).
func (r *PlanRepo) RecalculateForProduct(tx *gorm.DB, productID int64) error {
	return tx.Exec(recalcSQL+"sp.id IN (SELECT plan_id FROM plan_products WHERE product_id = ?)", productID).Error
}

type PlanFilter struct {
	Search        string
	Status        string // "active" (default), "archived", "all"
	BillingPeriod string
	AvailableOn   *time.Time // only plans available on this date
}

func (r *PlanRepo) List(db *gorm.DB, f PlanFilter, p pagination.Params) ([]models.SubscriptionPlan, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Where("(lower(name) LIKE ? OR lower(coalesce(description,'')) LIKE ?)", like, like)
		}
		switch f.Status {
		case "all":
		case "archived":
			q = q.Where("status = ?", models.StatusArchived)
		default:
			q = q.Where("status = ?", models.StatusActive)
		}
		if f.BillingPeriod != "" {
			q = q.Where("billing_period = ?", f.BillingPeriod)
		}
		if f.AvailableOn != nil {
			d := f.AvailableOn.Format("2006-01-02")
			q = q.Where("available_from <= ? AND (available_until IS NULL OR available_until >= ?)", d, d)
		}
		return q
	}
	var total int64
	if err := db.Model(&models.SubscriptionPlan{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.SubscriptionPlan
	err := withLines(db).Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

// ActiveSubscriptions counts confirmed/active/paused subscriptions per plan (one query).
func (r *PlanRepo) ActiveSubscriptions(db *gorm.DB, planIDs []int64) (map[int64]int64, error) {
	out := make(map[int64]int64, len(planIDs))
	if len(planIDs) == 0 {
		return out, nil
	}
	var rows []struct {
		PlanID int64
		N      int64
	}
	err := db.Table("subscriptions").Select("plan_id, count(*) AS n").
		Where("plan_id IN ? AND status IN ('confirmed','active','paused')", planIDs).
		Group("plan_id").Scan(&rows).Error
	for _, r := range rows {
		out[r.PlanID] = r.N
	}
	return out, err
}
