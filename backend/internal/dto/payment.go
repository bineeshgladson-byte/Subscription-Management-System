package dto

import "github.com/brightloop/subscriptions/backend/internal/models"

// RecordPaymentRequest contains the details required to record a payment.
type RecordPaymentRequest struct {
	Method      models.PaymentMethod `json:"method" binding:"required,oneof=upi bank_transfer card cash"`
	Amount      string               `json:"amount" binding:"required"`
	PaymentDate string               `json:"payment_date" binding:"required"`
	Reference   string               `json:"reference"`
	Notes       string               `json:"notes"`
}
