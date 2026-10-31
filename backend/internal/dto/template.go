package dto

import "time"

type TemplateLineInput struct {
	ProductID int64  `json:"product_id" binding:"required,min=1"`
	VariantID *int64 `json:"variant_id" binding:"omitempty,min=1"`
	Quantity  int    `json:"quantity" binding:"omitempty,min=1,max=100000"`
}

// CreateTemplateRequest: give lines, or a plan_id (its products become the lines), or both.
type CreateTemplateRequest struct {
	Name             string              `json:"name" binding:"required,min=2,max=120"`
	PlanID           *int64              `json:"plan_id" binding:"omitempty,min=1"`
	ValidityDays     int                 `json:"validity_days" binding:"omitempty,min=1,max=365"`      // default 15
	PaymentTermsDays *int                `json:"payment_terms_days" binding:"omitempty,min=0,max=365"` // default 15
	Terms            string              `json:"terms" binding:"omitempty,max=5000"`
	Lines            []TemplateLineInput `json:"lines" binding:"omitempty,max=50,dive"`
}

// UpdateTemplateRequest: only fields sent change. plan_id 0 removes the plan;
// lines REPLACES all lines; is_active false deactivates.
type UpdateTemplateRequest struct {
	Name             *string             `json:"name" binding:"omitempty,min=2,max=120"`
	PlanID           *int64              `json:"plan_id" binding:"omitempty,min=0"`
	ValidityDays     *int                `json:"validity_days" binding:"omitempty,min=1,max=365"`
	PaymentTermsDays *int                `json:"payment_terms_days" binding:"omitempty,min=0,max=365"`
	Terms            *string             `json:"terms" binding:"omitempty,max=5000"`
	Lines            []TemplateLineInput `json:"lines" binding:"omitempty,min=1,max=50,dive"`
	IsActive         *bool               `json:"is_active"`
}

type PlanSummary struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	BillingLabel string `json:"billing_label"`
	Price        string `json:"price"`
	Status       string `json:"status"`
}

type TemplateLineResponse struct {
	ID          int64   `json:"id"`
	ProductID   int64   `json:"product_id"`
	ProductName string  `json:"product_name"`
	VariantID   *int64  `json:"variant_id"`
	VariantName *string `json:"variant_name"`
	Quantity    int     `json:"quantity"`
	UnitPrice   string  `json:"unit_price"`
	LineTotal   string  `json:"line_total"`
}

type TemplateResponse struct {
	ID               int64                  `json:"id"`
	Name             string                 `json:"name"`
	Plan             *PlanSummary           `json:"plan"`
	ValidityDays     int                    `json:"validity_days"`
	PaymentTermsDays int                    `json:"payment_terms_days"`
	Terms            *string                `json:"terms"`
	IsActive         bool                   `json:"is_active"`
	Lines            []TemplateLineResponse `json:"lines"`
	EstimatedTotal   string                 `json:"estimated_total"` // before tax and discounts
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}
