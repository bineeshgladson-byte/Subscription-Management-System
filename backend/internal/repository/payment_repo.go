package repository

import (
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type PaymentRepo struct{}

func NewPaymentRepo() *PaymentRepo { return &PaymentRepo{} }

func (r *PaymentRepo) NextPaymentNumber(db *gorm.DB, year int) (string, error) {
	var seq int64
	if err := db.Raw("SELECT nextval('payment_no_seq')").Scan(&seq).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("PAY/%d/%05d", year, seq), nil
}

func (r *PaymentRepo) Create(db *gorm.DB, p *models.Payment) error {
	return db.Create(p).Error
}

func (r *PaymentRepo) FindByID(db *gorm.DB, id int64) (*models.Payment, error) {
	var p models.Payment
	err := db.First(&p, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

type PaymentFilter struct {
	InvoiceID  *int64
	CustomerID *int64
	Search     string
}

func (r *PaymentRepo) List(db *gorm.DB, f PaymentFilter, p pagination.Params) ([]models.Payment, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if f.InvoiceID != nil && *f.InvoiceID > 0 {
			q = q.Where("payments.invoice_id = ?", *f.InvoiceID)
		}
		if f.CustomerID != nil && *f.CustomerID > 0 {
			q = q.Where("payments.customer_id = ?", *f.CustomerID)
		}
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Where("(lower(payments.payment_number) LIKE ? OR lower(coalesce(payments.reference, '')) LIKE ?)", like, like)
		}
		return q
	}

	var total int64
	if err := db.Model(&models.Payment{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Payment
	err := db.Scopes(scope).
		Order("payments.id DESC").
		Limit(p.PageSize).
		Offset(p.Offset()).
		Find(&out).Error

	return out, total, err
}

func (r *PaymentRepo) UpdateFields(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Payment{}).Where("id = ?", id).Updates(fields).Error
}
