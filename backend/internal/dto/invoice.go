package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

type CancelInvoiceRequest struct {
	Reason string `json:"reason" binding:"omitempty,max=255"`
}

type InvoiceItemResponse struct {
	ID             int64   `json:"id"`
	ProductID      *int64  `json:"product_id"`
	ProductName    string  `json:"product_name"`
	VariantID      *int64  `json:"variant_id"`
	VariantName    *string `json:"variant_name"`
	Description    string  `json:"description"`
	Quantity       int     `json:"quantity"`
	UnitPrice      string  `json:"unit_price"`
	LineSubtotal   string  `json:"line_subtotal"`
	DiscountAmount string  `json:"discount_amount"`
	TaxRate        string  `json:"tax_rate"`
	TaxAmount      string  `json:"tax_amount"`
	LineTotal      string  `json:"line_total"`
}

type InvoiceResponse struct {
	ID                 int64                 `json:"id"`
	InvoiceNumber      string                `json:"invoice_number"`
	CustomerID         int64                 `json:"customer_id"`
	CustomerName       string                `json:"customer_name"`
	BillingAddress     *string               `json:"billing_address"`
	CustomerGSTIN      *string               `json:"customer_gstin"`
	SubscriptionID     *int64                `json:"subscription_id"`
	SubscriptionNumber *string               `json:"subscription_number"`
	PeriodStart        *string               `json:"period_start"`
	PeriodEnd          *string               `json:"period_end"`
	InvoiceDate        string                `json:"invoice_date"`
	DueDate            string                `json:"due_date"`
	Status             string                `json:"status"`
	Subtotal           string                `json:"subtotal"`
	DiscountTotal      string                `json:"discount_total"`
	TaxTotal           string                `json:"tax_total"`
	Total              string                `json:"total"`
	AmountPaid         string                `json:"amount_paid"`
	AmountDue          string                `json:"amount_due"`
	IsOverdue          bool                  `json:"is_overdue"`
	SentAt             *time.Time            `json:"sent_at"`
	ConfirmedAt        *time.Time            `json:"confirmed_at"`
	CancelledAt        *time.Time            `json:"cancelled_at"`
	CancelReason       *string               `json:"cancel_reason"`
	Items              []InvoiceItemResponse `json:"items"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
}

func NewInvoiceResponse(inv *models.Invoice) *InvoiceResponse {
	if inv == nil {
		return nil
	}
	items := make([]InvoiceItemResponse, len(inv.Items))
	for i, it := range inv.Items {
		items[i] = InvoiceItemResponse{
			ID:             it.ID,
			ProductID:      it.ProductID,
			Description:    it.Description,
			Quantity:       it.Quantity,
			UnitPrice:      it.UnitPrice.StringFixed(2),
			LineSubtotal:   it.LineSubtotal.StringFixed(2),
			DiscountAmount: it.DiscountAmount.StringFixed(2),
			TaxRate:        it.TaxRate.StringFixed(2),
			TaxAmount:      it.TaxAmount.StringFixed(2),
			LineTotal:      it.LineTotal.StringFixed(2),
		}
	}

	var pStart, pEnd *string
	if inv.PeriodStart != nil {
		s := inv.PeriodStart.Format("2006-01-02")
		pStart = &s
	}
	if inv.PeriodEnd != nil {
		s := inv.PeriodEnd.Format("2006-01-02")
		pEnd = &s
	}

	isOverdue := inv.Status == models.InvoiceConfirmed &&
		inv.DueDate.Before(time.Now()) &&
		inv.AmountDue.GreaterThan(inv.AmountPaid.Sub(inv.AmountPaid)) // AmountDue > 0

	return &InvoiceResponse{
		ID:             inv.ID,
		InvoiceNumber:  inv.InvoiceNumber,
		CustomerID:     inv.CustomerID,
		CustomerName:   inv.CustomerName,
		BillingAddress: inv.BillingAddress,
		CustomerGSTIN:  inv.CustomerGSTIN,
		SubscriptionID: inv.SubscriptionID,
		PeriodStart:    pStart,
		PeriodEnd:      pEnd,
		InvoiceDate:    inv.InvoiceDate.Format("2006-01-02"),
		DueDate:        inv.DueDate.Format("2006-01-02"),
		Status:         string(inv.Status),
		Subtotal:       inv.Subtotal.StringFixed(2),
		DiscountTotal:  inv.DiscountTotal.StringFixed(2),
		TaxTotal:       inv.TaxTotal.StringFixed(2),
		Total:          inv.Total.StringFixed(2),
		AmountPaid:     inv.AmountPaid.StringFixed(2),
		AmountDue:      inv.AmountDue.StringFixed(2),
		IsOverdue:      isOverdue,
		SentAt:         inv.SentAt,
		ConfirmedAt:    inv.ConfirmedAt,
		CancelledAt:    inv.CancelledAt,
		CancelReason:   inv.CancelReason,
		Items:          items,
		CreatedAt:      inv.CreatedAt,
		UpdatedAt:      inv.UpdatedAt,
	}
}
