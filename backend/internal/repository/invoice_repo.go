package repository

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type InvoiceRepo struct{}

func NewInvoiceRepo() *InvoiceRepo { return &InvoiceRepo{} }

func (r *InvoiceRepo) NextInvoiceNumber(db *gorm.DB, year int) (string, error) {
	var seq int64
	if err := db.Raw("SELECT nextval('invoice_no_seq')").Scan(&seq).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("INV/%d/%05d", year, seq), nil
}

func (r *InvoiceRepo) Create(db *gorm.DB, inv *models.Invoice) error {
	return db.Create(inv).Error
}

func (r *InvoiceRepo) FindByID(db *gorm.DB, id int64) (*models.Invoice, error) {
	var inv models.Invoice
	err := db.Preload("Items").
		First(&inv, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &inv, err
}

type InvoiceFilter struct {
	Status         *string
	CustomerID     *int64
	SubscriptionID *int64
	Overdue        *bool
	From           *time.Time
	To             *time.Time
	Search         string
	Today          time.Time
}

func (r *InvoiceRepo) List(db *gorm.DB, f InvoiceFilter, p pagination.Params) ([]models.Invoice, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		if f.Status != nil && *f.Status != "" {
			q = q.Where("invoices.status = ?", *f.Status)
		}
		if f.CustomerID != nil && *f.CustomerID > 0 {
			q = q.Where("invoices.customer_id = ?", *f.CustomerID)
		}
		if f.SubscriptionID != nil && *f.SubscriptionID > 0 {
			q = q.Where("invoices.subscription_id = ?", *f.SubscriptionID)
		}
		if f.Overdue != nil && *f.Overdue {
			today := f.Today
			if today.IsZero() {
				today = time.Now()
			}
			q = q.Where("invoices.status = 'confirmed' AND invoices.due_date < ? AND (invoices.total - invoices.amount_paid) > 0", today.Format("2006-01-02"))
		}
		if f.From != nil {
			q = q.Where("invoices.invoice_date >= ?", f.From.Format("2006-01-02"))
		}
		if f.To != nil {
			q = q.Where("invoices.invoice_date <= ?", f.To.Format("2006-01-02"))
		}
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Where("(lower(invoices.invoice_number) LIKE ? OR lower(invoices.customer_name) LIKE ?)", like, like)
		}
		return q
	}

	var total int64
	if err := db.Model(&models.Invoice{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var out []models.Invoice
	err := db.Scopes(scope).
		Preload("Items").
		Order("invoices.id DESC").
		Limit(p.PageSize).
		Offset(p.Offset()).
		Find(&out).Error

	return out, total, err
}

func (r *InvoiceRepo) UpdateFields(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.Invoice{}).Where("id = ?", id).Updates(fields).Error
}

func (r *InvoiceRepo) ApplyPayment(db *gorm.DB, invoiceID int64, amount decimal.Decimal) error {
	var inv models.Invoice
	if err := db.First(&inv, invoiceID).Error; err != nil {
		return err
	}
	newPaid := inv.AmountPaid.Add(amount)
	updates := map[string]any{
		"amount_paid": newPaid,
	}
	if newPaid.GreaterThanOrEqual(inv.Total) {
		updates["status"] = models.InvoicePaid
	}
	return db.Model(&models.Invoice{}).Where("id = ?", invoiceID).Updates(updates).Error
}
