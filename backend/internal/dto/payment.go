package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

type RecordPaymentRequest struct {
	Method      string `json:"method" binding:"required,oneof=upi bank_transfer card cash"`
	Amount      string `json:"amount" binding:"required"`
	PaymentDate string `json:"payment_date" binding:"required"` // YYYY-MM-DD
	Reference   string `json:"reference" binding:"omitempty,max=100"`
	Notes       string `json:"notes" binding:"omitempty,max=1000"`
}

type RazorpayOrderResponse struct {
	OrderID       string `json:"order_id"`
	Amount        int64  `json:"amount"` // in paise
	Currency      string `json:"currency"`
	KeyID         string `json:"key_id"`
	InvoiceNumber string `json:"invoice_number"`
	CustomerName  string `json:"customer_name"`
	CustomerEmail string `json:"customer_email"`
}

type VerifyRazorpayRequest struct {
	RazorpayOrderID   string `json:"razorpay_order_id" binding:"required"`
	RazorpayPaymentID string `json:"razorpay_payment_id" binding:"required"`
	RazorpaySignature string `json:"razorpay_signature" binding:"required"`
}

type PaymentResponse struct {
	ID            int64     `json:"id"`
	PaymentNumber string    `json:"payment_number"`
	InvoiceID     int64     `json:"invoice_id"`
	InvoiceNumber string    `json:"invoice_number"`
	CustomerID    int64     `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	Method        string    `json:"method"`
	Amount        string    `json:"amount"`
	PaymentDate   string    `json:"payment_date"`
	Reference     *string   `json:"reference"`
	Notes         *string   `json:"notes"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func NewPaymentResponse(p *models.Payment, invNum, custName string) *PaymentResponse {
	if p == nil {
		return nil
	}
	return &PaymentResponse{
		ID:            p.ID,
		PaymentNumber: p.PaymentNumber,
		InvoiceID:     p.InvoiceID,
		InvoiceNumber: invNum,
		CustomerID:    p.CustomerID,
		CustomerName:  custName,
		Method:        string(p.Method),
		Amount:        p.Amount.StringFixed(2),
		PaymentDate:   p.PaymentDate.Format("2006-01-02"),
		Reference:     p.Reference,
		Notes:         p.Notes,
		Status:        string(p.Status),
		CreatedAt:     p.CreatedAt,
	}
}
