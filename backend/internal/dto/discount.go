package dto

import "time"

// CreateDiscountRequest — only the admin creates discount rules.
// scope "product" requires product_ids; "subscription" applies to the whole order.
type CreateDiscountRequest struct {
	Name              string  `json:"name" binding:"required,min=2,max=120"`
	Code              string  `json:"code" binding:"omitempty,max=40"` // optional label, stored upper-case
	DiscountType      string  `json:"discount_type" binding:"required,oneof=fixed percentage"`
	Value             string  `json:"value" binding:"required,max=16"` // amount, or percent (≤ 100)
	Scope             string  `json:"scope" binding:"required,oneof=subscription product"`
	ProductIDs        []int64 `json:"product_ids" binding:"omitempty,max=200,dive,min=1"`
	MinPurchaseAmount string  `json:"min_purchase_amount" binding:"omitempty,max=16"`
	MinQuantity       *int    `json:"min_quantity" binding:"omitempty,min=1,max=100000"`
	StartsOn          string  `json:"starts_on"` // YYYY-MM-DD, default today
	EndsOn            string  `json:"ends_on"`   // YYYY-MM-DD, optional
	MaxUsage          *int    `json:"max_usage" binding:"omitempty,min=1,max=10000000"`
}

// UpdateDiscountRequest: only fields sent change. "" clears code, min_purchase_amount
// and ends_on; 0 clears min_quantity and max_usage; product_ids replaces the list.
type UpdateDiscountRequest struct {
	Name              *string `json:"name" binding:"omitempty,min=2,max=120"`
	Code              *string `json:"code" binding:"omitempty,max=40"`
	DiscountType      *string `json:"discount_type" binding:"omitempty,oneof=fixed percentage"`
	Value             *string `json:"value" binding:"omitempty,max=16"`
	Scope             *string `json:"scope" binding:"omitempty,oneof=subscription product"`
	ProductIDs        []int64 `json:"product_ids" binding:"omitempty,max=200,dive,min=1"`
	MinPurchaseAmount *string `json:"min_purchase_amount" binding:"omitempty,max=16"`
	MinQuantity       *int    `json:"min_quantity" binding:"omitempty,min=0,max=100000"`
	StartsOn          *string `json:"starts_on"`
	EndsOn            *string `json:"ends_on"`
	MaxUsage          *int    `json:"max_usage" binding:"omitempty,min=0,max=10000000"`
	IsActive          *bool   `json:"is_active"`
}

type ProductRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type DiscountResponse struct {
	ID                int64        `json:"id"`
	Name              string       `json:"name"`
	Code              *string      `json:"code"`
	DiscountType      string       `json:"discount_type"`
	Value             string       `json:"value"`
	Label             string       `json:"label"` // "10% off", "₹500 off"
	Scope             string       `json:"scope"`
	Products          []ProductRef `json:"products"`
	MinPurchaseAmount *string      `json:"min_purchase_amount"`
	MinQuantity       *int         `json:"min_quantity"`
	StartsOn          string       `json:"starts_on"`
	EndsOn            *string      `json:"ends_on"`
	MaxUsage          *int         `json:"max_usage"`
	UsageCount        int          `json:"usage_count"`
	RemainingUses     *int         `json:"remaining_uses"`
	IsActive          bool         `json:"is_active"`
	// State: active | scheduled (starts later) | expired | used_up | inactive (switched off)
	State     string    `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type DiscountUsageResponse struct {
	ID            int64     `json:"id"`
	InvoiceID     int64     `json:"invoice_id"`
	InvoiceNumber *string   `json:"invoice_number"`
	CustomerID    int64     `json:"customer_id"`
	CustomerName  *string   `json:"customer_name"`
	Amount        string    `json:"amount"`
	CreatedAt     time.Time `json:"created_at"`
}

// ---------- price preview (pricing engine) ----------

type PriceLineInput struct {
	ProductID int64  `json:"product_id" binding:"required,min=1"`
	VariantID *int64 `json:"variant_id" binding:"omitempty,min=1"`
	Quantity  int    `json:"quantity" binding:"omitempty,min=1,max=100000"`
	UnitPrice string `json:"unit_price" binding:"omitempty,max=16"` // override the catalogue price
}

type PricePreviewRequest struct {
	Lines []PriceLineInput `json:"lines" binding:"required,min=1,max=50,dive"`
	Date  string           `json:"date"` // which day's discounts apply; default today
}

type PricedLineResponse struct {
	ProductID            int64   `json:"product_id"`
	Description          string  `json:"description"`
	Quantity             int     `json:"quantity"`
	UnitPrice            string  `json:"unit_price"`
	Subtotal             string  `json:"subtotal"`
	ProductDiscount      string  `json:"product_discount"`
	ProductDiscountName  *string `json:"product_discount_name"`
	SubscriptionDiscount string  `json:"subscription_discount"`
	TaxRate              string  `json:"tax_rate"`
	TaxAmount            string  `json:"tax_amount"`
	Total                string  `json:"total"`
}

type AppliedDiscountResponse struct {
	DiscountID int64  `json:"discount_id"`
	Name       string `json:"name"`
	Scope      string `json:"scope"`
	Amount     string `json:"amount"`
}

type PricePreviewResponse struct {
	Date          string                    `json:"date"`
	Lines         []PricedLineResponse      `json:"lines"`
	Subtotal      string                    `json:"subtotal"`
	DiscountTotal string                    `json:"discount_total"`
	TaxTotal      string                    `json:"tax_total"`
	Total         string                    `json:"total"`
	Discounts     []AppliedDiscountResponse `json:"discounts"`
}
