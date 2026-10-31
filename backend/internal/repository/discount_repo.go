package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type DiscountRepo struct{}

func NewDiscountRepo() *DiscountRepo { return &DiscountRepo{} }

func withDiscountProducts(db *gorm.DB) *gorm.DB {
	return db.Preload("Products", func(q *gorm.DB) *gorm.DB { return q.Unscoped().Order("name") })
}

func (r *DiscountRepo) Create(db *gorm.DB, d *models.Discount) error {
	return db.Omit("Products").Create(d).Error
}

func (r *DiscountRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Discount{}).Where("id = ?", id).Updates(fields).Error
}

func (r *DiscountRepo) FindByID(db *gorm.DB, id int64) (*models.Discount, error) {
	var d models.Discount
	err := withDiscountProducts(db).First(&d, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &d, err
}

func (r *DiscountRepo) CodeTaken(db *gorm.DB, code string, exceptID int64) (bool, error) {
	var n int64
	err := db.Model(&models.Discount{}).Where("upper(code) = upper(?) AND id <> ?", code, exceptID).Count(&n).Error
	return n > 0, err
}

// ReplaceProducts rewrites the discount_products rows (inside the caller's tx).
func (r *DiscountRepo) ReplaceProducts(tx *gorm.DB, discountID int64, productIDs []int64) error {
	if err := tx.Exec("DELETE FROM discount_products WHERE discount_id = ?", discountID).Error; err != nil {
		return err
	}
	for _, pid := range productIDs {
		if err := tx.Exec("INSERT INTO discount_products (discount_id, product_id) VALUES (?, ?)", discountID, pid).Error; err != nil {
			return err
		}
	}
	return nil
}

type DiscountFilter struct {
	Search string
	Scope  string
	State  string // active | scheduled | expired | used_up | inactive
	Today  time.Time
}

func (r *DiscountRepo) List(db *gorm.DB, f DiscountFilter, p pagination.Params) ([]models.Discount, int64, error) {
	today := f.Today.Format("2006-01-02")
	scope := func(q *gorm.DB) *gorm.DB {
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Where("(lower(name) LIKE ? OR lower(coalesce(code,'')) LIKE ?)", like, like)
		}
		if f.Scope != "" {
			q = q.Where("scope = ?", f.Scope)
		}
		switch f.State {
		case "inactive":
			q = q.Where("NOT is_active")
		case "scheduled":
			q = q.Where("is_active AND starts_on > ?", today)
		case "expired":
			q = q.Where("is_active AND ends_on < ?", today)
		case "used_up":
			q = q.Where("is_active AND max_usage IS NOT NULL AND usage_count >= max_usage")
		case "active":
			q = eligibleScope(q, today)
		}
		return q
	}
	var total int64
	if err := db.Model(&models.Discount{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var out []models.Discount
	err := withDiscountProducts(db).Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&out).Error
	return out, total, err
}

// eligibleScope: switched on, inside its dates, with uses left.
func eligibleScope(q *gorm.DB, day string) *gorm.DB {
	return q.Where("is_active AND starts_on <= ? AND (ends_on IS NULL OR ends_on >= ?) AND (max_usage IS NULL OR usage_count < max_usage)", day, day)
}

// Eligible returns every discount that can apply on day (used by the pricing engine).
func (r *DiscountRepo) Eligible(db *gorm.DB, day time.Time) ([]models.Discount, error) {
	var out []models.Discount
	err := withDiscountProducts(db).Scopes(func(q *gorm.DB) *gorm.DB { return eligibleScope(q, day.Format("2006-01-02")) }).
		Order("id").Find(&out).Error
	return out, err
}

// ReserveUse atomically takes one use of a discount. Returns false if it was
// used up in the meantime (the caller must then price without it).
// For the invoice generator (A9): call inside the invoice transaction, then
// insert a discount_usages row.
func (r *DiscountRepo) ReserveUse(tx *gorm.DB, discountID int64) (bool, error) {
	res := tx.Exec("UPDATE discounts SET usage_count = usage_count + 1, updated_at = now() "+
		"WHERE id = ? AND is_active AND (max_usage IS NULL OR usage_count < max_usage)", discountID)
	return res.RowsAffected == 1, res.Error
}

// ReleaseUse gives a use back (e.g. when an invoice is cancelled).
func (r *DiscountRepo) ReleaseUse(tx *gorm.DB, discountID int64) error {
	return tx.Exec("UPDATE discounts SET usage_count = greatest(usage_count - 1, 0), updated_at = now() WHERE id = ?", discountID).Error
}

type DiscountUsageRow struct {
	ID            int64
	InvoiceID     int64
	InvoiceNumber *string
	CustomerID    int64
	CustomerName  *string
	Amount        string
	CreatedAt     time.Time
}

func (r *DiscountRepo) Usages(db *gorm.DB, discountID int64, p pagination.Params) ([]DiscountUsageRow, int64, error) {
	var total int64
	if err := db.Table("discount_usages").Where("discount_id = ?", discountID).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []DiscountUsageRow
	err := db.Table("discount_usages du").
		Select("du.id, du.invoice_id, i.invoice_number, du.customer_id, c.name AS customer_name, du.amount::text AS amount, du.created_at").
		Joins("LEFT JOIN invoices i ON i.id = du.invoice_id").
		Joins("LEFT JOIN customers c ON c.id = du.customer_id").
		Where("du.discount_id = ?", discountID).Order("du.created_at DESC").
		Limit(p.PageSize).Offset(p.Offset()).Scan(&rows).Error
	return rows, total, err
}
