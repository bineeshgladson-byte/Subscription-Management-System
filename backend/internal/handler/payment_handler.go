package handler

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

type PaymentHandler struct {
	svc *service.PaymentService
}

func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

// POST /invoices/:id/payments
func (h *PaymentHandler) RecordPayment(c *gin.Context) {
	var req dto.RecordPaymentRequest

	if !bind(c, &req) {
		return
	}

	var invoiceID int64
	if _, err := fmt.Sscan(c.Param("id"), &invoiceID); err != nil || invoiceID <= 0 {
		response.Error(c, apperr.Validation(
			"Invalid invoice ID.",
			map[string]string{"id": "Invalid invoice ID"},
		))
		return
	}

	payment, invoice, err := h.svc.RecordPayment(
		c.Request.Context(),
		invoiceID,
		req,
		c.GetHeader("Idempotency-Key"),
		middleware.CurrentUserID(c),
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	amountDue := invoice.Total.Sub(invoice.AmountPaid)

	paymentState := paymentStatus(invoice.AmountPaid, amountDue)

	response.Created(c, gin.H{
		"payment": gin.H{
			"id":             payment.ID,
			"payment_number": payment.PaymentNumber,
			"invoice_id":     payment.InvoiceID,
			"method":         payment.Method,
			"amount":         payment.Amount.StringFixed(2),
			"payment_date":   payment.PaymentDate.Format("2006-01-02"),
			"status":         payment.Status,
		},
		"invoice": gin.H{
			"id":             invoice.ID,
			"invoice_number": invoice.InvoiceNumber,
			"customer": gin.H{
				"id":   invoice.CustomerID,
				"name": invoice.CustomerName,
			},
			"invoice_date":   invoice.InvoiceDate.Format("2006-01-02"),
			"due_date":       invoice.DueDate.Format("2006-01-02"),
			"total":          invoice.Total.StringFixed(2),
			"amount_paid":    invoice.AmountPaid.StringFixed(2),
			"amount_due":     amountDue.StringFixed(2),
			"status":         invoice.Status,
			"payment_status": paymentState,
			"is_overdue": amountDue.GreaterThan(decimal.Zero) &&
				invoice.DueDate.Before(time.Now()),
		},
	})
}

func paymentStatus(paid, due decimal.Decimal) string {
	if paid.IsZero() {
		return "unpaid"
	}
	if due.IsZero() {
		return "paid"
	}
	return "partial"
}
