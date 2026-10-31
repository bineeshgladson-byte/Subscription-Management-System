package repository

import (
	"errors"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"gorm.io/gorm"
)

type PaymentRepo struct{}

func NewPaymentRepo() *PaymentRepo {
	return &PaymentRepo{}
}

func (r *PaymentRepo) Create(db *gorm.DB, payment *models.Payment) error {
	return db.Create(payment).Error
}

func (r *PaymentRepo) FindByID(db *gorm.DB, id int64) (*models.Payment, error) {
	var payment models.Payment

	err := db.First(&payment, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &payment, err
}

func (r *PaymentRepo) FindByInvoiceID(db *gorm.DB, invoiceID int64) ([]models.Payment, error) {
	var payments []models.Payment

	err := db.Where("invoice_id = ?", invoiceID).
		Order("payment_date DESC").
		Find(&payments).Error

	return payments, err
}

func (r *PaymentRepo) FindByIdempotencyKey(db *gorm.DB, key string) (*models.Payment, error) {
	var payment models.Payment

	err := db.Where("idempotency_key = ?", key).First(&payment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &payment, err
}
