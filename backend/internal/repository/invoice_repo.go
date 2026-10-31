package repository

import (
	"errors"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type InvoiceRepo struct{}

func NewInvoiceRepo() *InvoiceRepo {
	return &InvoiceRepo{}
}

func (r *InvoiceRepo) FindByID(db *gorm.DB, id int64) (*models.Invoice, error) {
	var invoice models.Invoice

	err := db.Preload("Items").First(&invoice, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &invoice, err
}

func (r *InvoiceRepo) FindByIDForUpdate(db *gorm.DB, id int64) (*models.Invoice, error) {
	var invoice models.Invoice

	err := db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Preload("Items").
		First(&invoice, id).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &invoice, err
}

func (r *InvoiceRepo) UpdatePaymentTotals(
	db *gorm.DB,
	id int64,
	amountPaid decimal.Decimal,
	status models.InvoiceStatus,
) error {
	return db.Model(&models.Invoice{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"amount_paid": amountPaid,
			"status":      status,
		}).Error
}
