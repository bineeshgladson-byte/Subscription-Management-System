package dto

import "time"

// SubscriptionResponse contains subscription details visible to a customer.
type SubscriptionResponse struct {
	ID                 int64      `json:"id"`
	QuotationNumber    string     `json:"quotation_number"`
	SubscriptionNumber *string    `json:"subscription_number"`
	PlanID             int64      `json:"plan_id"`
	Status             string     `json:"status"`
	StartDate          time.Time  `json:"start_date"`
	ExpiryDate         *time.Time `json:"expiry_date"`
	NextBillingDate    *time.Time `json:"next_billing_date"`
	Subtotal           string     `json:"subtotal"`
	DiscountTotal      string     `json:"discount_total"`
	TaxTotal           string     `json:"tax_total"`
	Total              string     `json:"total"`
	CreatedAt          time.Time  `json:"created_at"`
}
