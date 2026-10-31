package repository

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type SubscriptionRepo struct{}

func NewSubscriptionRepo() *SubscriptionRepo { return &SubscriptionRepo{} }

func (r *SubscriptionRepo) NextQuotationNumber(db *gorm.DB, year int) (string, error) {
	var seq int64
	if err := db.Raw("SELECT nextval('quotation_no_seq')").Scan(&seq).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("QT/%d/%05d", year, seq), nil
}

func (r *SubscriptionRepo) NextSubscriptionNumber(db *gorm.DB, year int) (string, error) {
	var seq int64
	if err := db.Raw("SELECT nextval('subscription_no_seq')").Scan(&seq).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("SUB/%d/%05d", year, seq), nil
}

func (r *SubscriptionRepo) Create(db *gorm.DB, s *models.Subscription) error {
	return db.Create(s).Error
}

func (r *SubscriptionRepo) Update(db *gorm.DB, s *models.Subscription) error {
	return db.Save(s).Error
}

func (r *SubscriptionRepo) ReplaceItems(db *gorm.DB, subID int64, items []models.SubscriptionItem) error {
	if err := db.Where("subscription_id = ?", subID).Delete(&models.SubscriptionItem{}).Error; err != nil {
		return err
	}
	for i := range items {
		items[i].SubscriptionID = subID
	}
	if len(items) > 0 {
		return db.Create(&items).Error
	}
	return nil
}

func (r *SubscriptionRepo) FindByID(db *gorm.DB, id int64) (*models.Subscription, error) {
	var s models.Subscription
	err := db.Preload("Customer").
		Preload("Plan").
		Preload("Items").
		First(&s, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &s, err
}

type SubscriptionFilter struct {
	IsQuotation *bool // true: draft/quotation; false: confirmed/active/paused/closed/renewed
	Status      *string
	CustomerID  *int64
	Search      string
}

func (r *SubscriptionRepo) List(db *gorm.DB, f SubscriptionFilter, p pagination.Params) ([]models.Subscription, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if f.IsQuotation != nil {
			if *f.IsQuotation {
				q = q.Where("subscriptions.status IN ('draft', 'quotation')")
			} else {
				q = q.Where("subscriptions.status NOT IN ('draft', 'quotation')")
			}
		}
		if f.Status != nil && *f.Status != "" {
			q = q.Where("subscriptions.status = ?", *f.Status)
		}
		if f.CustomerID != nil && *f.CustomerID > 0 {
			q = q.Where("subscriptions.customer_id = ?", *f.CustomerID)
		}
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Joins("LEFT JOIN customers ON customers.id = subscriptions.customer_id").
				Where("(lower(subscriptions.quotation_number) LIKE ? OR lower(coalesce(subscriptions.subscription_number, '')) LIKE ? OR lower(customers.name) LIKE ? OR lower(customers.email) LIKE ?)",
					like, like, like, like)
		}
		return q
	}

	var total int64
	if err := db.Model(&models.Subscription{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Subscription
	err := db.Scopes(scope).
		Preload("Customer").
		Preload("Plan").
		Preload("Items").
		Order("subscriptions.id DESC").
		Limit(p.PageSize).
		Offset(p.Offset()).
		Find(&out).Error

	return out, total, err
}

func (r *SubscriptionRepo) UpdateFields(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Subscription{}).Where("id = ?", id).Updates(fields).Error
}
