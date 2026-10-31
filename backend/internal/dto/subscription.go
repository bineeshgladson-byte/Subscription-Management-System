package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

type SubscriptionLineInput struct {
	ProductID     int64   `json:"product_id" binding:"required,min=1"`
	VariantID     *int64  `json:"variant_id" binding:"omitempty,min=1"`
	Quantity      int     `json:"quantity" binding:"omitempty,min=1,max=100000"`
	PriceOverride *string `json:"price_override" binding:"omitempty"`
}

type CreateQuotationRequest struct {
	CustomerID       int64                   `json:"customer_id" binding:"required,min=1"`
	PlanID           int64                   `json:"plan_id" binding:"required,min=1"`
	TemplateID       *int64                  `json:"template_id" binding:"omitempty,min=1"`
	StartDate        string                  `json:"start_date" binding:"required"` // YYYY-MM-DD
	ExpiryDate       *string                 `json:"expiry_date" binding:"omitempty"`
	PaymentTermsDays *int                    `json:"payment_terms_days" binding:"omitempty,min=0,max=365"`
	Lines            []SubscriptionLineInput `json:"lines" binding:"omitempty,max=100,dive"`
}

type UpdateQuotationRequest struct {
	PlanID           *int64                  `json:"plan_id" binding:"omitempty,min=1"`
	StartDate        *string                 `json:"start_date" binding:"omitempty"`
	ExpiryDate       *string                 `json:"expiry_date" binding:"omitempty"`
	PaymentTermsDays *int                    `json:"payment_terms_days" binding:"omitempty,min=0,max=365"`
	Lines            []SubscriptionLineInput `json:"lines" binding:"omitempty,min=1,max=100,dive"`
}

type CloseSubscriptionRequest struct {
	Reason string `json:"reason" binding:"omitempty,max=255"`
}

type RenewSubscriptionRequest struct {
	NewExpiryDate *string `json:"new_expiry_date" binding:"omitempty"`
}

type SubscriptionItemResponse struct {
	ID           int64   `json:"id"`
	ProductID    int64   `json:"product_id"`
	ProductName  string  `json:"product_name"`
	VariantID    *int64  `json:"variant_id"`
	VariantName  *string `json:"variant_name"`
	Description  string  `json:"description"`
	Quantity     int     `json:"quantity"`
	UnitPrice    string  `json:"unit_price"`
	TaxID        *int64  `json:"tax_id"`
	TaxRate      string  `json:"tax_rate"`
	LineSubtotal string  `json:"line_subtotal"`
}

type SubscriptionResponse struct {
	ID                  int64                      `json:"id"`
	QuotationNumber     string                     `json:"quotation_number"`
	SubscriptionNumber  *string                    `json:"subscription_number"`
	CustomerID          int64                      `json:"customer_id"`
	CustomerName        string                     `json:"customer_name"`
	CustomerEmail       string                     `json:"customer_email"`
	PlanID              int64                      `json:"plan_id"`
	PlanName            string                     `json:"plan_name"`
	BillingPeriod       string                     `json:"billing_period"`
	TemplateID          *int64                     `json:"template_id"`
	Status              string                     `json:"status"`
	StartDate           string                     `json:"start_date"`
	ExpiryDate          *string                    `json:"expiry_date"`
	QuotationValidUntil *string                    `json:"quotation_valid_until"`
	PaymentTermsDays    int                        `json:"payment_terms_days"`
	NextBillingDate     *string                    `json:"next_billing_date"`
	BillingAnchorDay    int                        `json:"billing_anchor_day"`
	CyclesBilled        int                        `json:"cycles_billed"`
	CloseReason         *string                    `json:"close_reason"`
	Subtotal            string                     `json:"subtotal"`
	DiscountTotal       string                     `json:"discount_total"`
	TaxTotal            string                     `json:"tax_total"`
	Total               string                     `json:"total"`
	ConfirmedAt         *time.Time                 `json:"confirmed_at"`
	ActivatedAt         *time.Time                 `json:"activated_at"`
	PausedAt            *time.Time                 `json:"paused_at"`
	ClosedAt            *time.Time                 `json:"closed_at"`
	Items               []SubscriptionItemResponse `json:"items"`
	CreatedAt           time.Time                  `json:"created_at"`
	UpdatedAt           time.Time                  `json:"updated_at"`
}

func NewSubscriptionResponse(s *models.Subscription) *SubscriptionResponse {
	if s == nil {
		return nil
	}
	custName := ""
	custEmail := ""
	if s.Customer != nil {
		custName = s.Customer.Name
		custEmail = s.Customer.Email
	}
	planName := ""
	billingPeriod := "monthly"
	if s.Plan != nil {
		planName = s.Plan.Name
		billingPeriod = string(s.Plan.BillingPeriod)
	}

	items := make([]SubscriptionItemResponse, len(s.Items))
	for i, it := range s.Items {
		items[i] = SubscriptionItemResponse{
			ID:           it.ID,
			ProductID:    it.ProductID,
			Description:  it.Description,
			Quantity:     it.Quantity,
			UnitPrice:    it.UnitPrice.StringFixed(2),
			TaxID:        it.TaxID,
			TaxRate:      it.TaxRate.StringFixed(2),
			LineSubtotal: it.LineSubtotal.StringFixed(2),
		}
	}

	var expDateStr *string
	if s.ExpiryDate != nil {
		str := s.ExpiryDate.Format("2006-01-02")
		expDateStr = &str
	}
	var validUntilStr *string
	if s.QuotationValidUntil != nil {
		str := s.QuotationValidUntil.Format("2006-01-02")
		validUntilStr = &str
	}
	var nextBillStr *string
	if s.NextBillingDate != nil {
		str := s.NextBillingDate.Format("2006-01-02")
		nextBillStr = &str
	}

	return &SubscriptionResponse{
		ID:                  s.ID,
		QuotationNumber:     s.QuotationNumber,
		SubscriptionNumber:  s.SubscriptionNumber,
		CustomerID:          s.CustomerID,
		CustomerName:        custName,
		CustomerEmail:       custEmail,
		PlanID:              s.PlanID,
		PlanName:            planName,
		BillingPeriod:       billingPeriod,
		TemplateID:          s.TemplateID,
		Status:              string(s.Status),
		StartDate:           s.StartDate.Format("2006-01-02"),
		ExpiryDate:          expDateStr,
		QuotationValidUntil: validUntilStr,
		PaymentTermsDays:    s.PaymentTermsDays,
		NextBillingDate:     nextBillStr,
		BillingAnchorDay:    s.BillingAnchorDay,
		CyclesBilled:        s.CyclesBilled,
		CloseReason:         s.CloseReason,
		Subtotal:            s.Subtotal.StringFixed(2),
		DiscountTotal:       s.DiscountTotal.StringFixed(2),
		TaxTotal:            s.TaxTotal.StringFixed(2),
		Total:               s.Total.StringFixed(2),
		ConfirmedAt:         s.ConfirmedAt,
		ActivatedAt:         s.ActivatedAt,
		PausedAt:            s.PausedAt,
		ClosedAt:            s.ClosedAt,
		Items:               items,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}
