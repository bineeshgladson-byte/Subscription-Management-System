package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// ---------- requests ----------

// PlanLineInput is one product (optionally a variant) included in a plan.
// unit price = price_override if given, otherwise sales_price + variant extra_price.
type PlanLineInput struct {
	ProductID     int64  `json:"product_id" binding:"required,min=1"`
	VariantID     *int64 `json:"variant_id" binding:"omitempty,min=1"`
	Quantity      int    `json:"quantity" binding:"omitempty,min=1,max=100000"` // default 1
	PriceOverride string `json:"price_override" binding:"omitempty,max=16"`
}

type CreatePlanRequest struct {
	Name                 string          `json:"name" binding:"required,min=2,max=120"`
	Description          string          `json:"description" binding:"omitempty,max=2000"`
	BillingPeriod        string          `json:"billing_period" binding:"required,oneof=daily weekly monthly yearly"`
	BillingInterval      int             `json:"billing_interval" binding:"omitempty,min=1,max=99"` // default 1
	MinQuantity          int             `json:"min_quantity" binding:"omitempty,min=1,max=100000"` // default 1
	AvailableFrom        string          `json:"available_from" binding:"omitempty"`                // YYYY-MM-DD, default today
	AvailableUntil       string          `json:"available_until" binding:"omitempty"`               // YYYY-MM-DD, optional
	AutoCloseEnabled     bool            `json:"auto_close_enabled"`
	AutoCloseAfterCycles *int            `json:"auto_close_after_cycles" binding:"omitempty,min=1,max=1000"`
	IsPausable           *bool           `json:"is_pausable"`  // default false
	IsClosable           *bool           `json:"is_closable"`  // early closure, default true
	IsRenewable          *bool           `json:"is_renewable"` // default true
	Products             []PlanLineInput `json:"products" binding:"required,min=1,max=50,dive"`
}

// UpdatePlanRequest: only fields sent are changed. "products" REPLACES all lines.
// available_until: "" removes the end date. auto_close_after_cycles: 0 removes it.
type UpdatePlanRequest struct {
	Name                 *string         `json:"name" binding:"omitempty,min=2,max=120"`
	Description          *string         `json:"description" binding:"omitempty,max=2000"`
	BillingPeriod        *string         `json:"billing_period" binding:"omitempty,oneof=daily weekly monthly yearly"`
	BillingInterval      *int            `json:"billing_interval" binding:"omitempty,min=1,max=99"`
	MinQuantity          *int            `json:"min_quantity" binding:"omitempty,min=1,max=100000"`
	AvailableFrom        *string         `json:"available_from"`
	AvailableUntil       *string         `json:"available_until"`
	AutoCloseEnabled     *bool           `json:"auto_close_enabled"`
	AutoCloseAfterCycles *int            `json:"auto_close_after_cycles" binding:"omitempty,min=0,max=1000"`
	IsPausable           *bool           `json:"is_pausable"`
	IsClosable           *bool           `json:"is_closable"`
	IsRenewable          *bool           `json:"is_renewable"`
	Products             []PlanLineInput `json:"products" binding:"omitempty,min=1,max=50,dive"`
}

type UpdatePlanStatusRequest struct {
	Status string `json:"status" binding:"required,oneof=active archived"`
}

// ---------- responses ----------

type PlanLineResponse struct {
	ID            int64   `json:"id"`
	ProductID     int64   `json:"product_id"`
	ProductName   string  `json:"product_name"`
	VariantID     *int64  `json:"variant_id"`
	VariantName   *string `json:"variant_name"`
	Quantity      int     `json:"quantity"`
	PriceOverride *string `json:"price_override"`
	UnitPrice     string  `json:"unit_price"`
	LineTotal     string  `json:"line_total"`
}

type PlanResponse struct {
	ID                   int64                `json:"id"`
	Name                 string               `json:"name"`
	Description          *string              `json:"description"`
	Price                string               `json:"price"` // per billing period, = sum of line totals
	BillingPeriod        models.BillingPeriod `json:"billing_period"`
	BillingInterval      int                  `json:"billing_interval"`
	BillingLabel         string               `json:"billing_label"` // "Monthly", "Every 3 months"
	MinQuantity          int                  `json:"min_quantity"`
	AvailableFrom        string               `json:"available_from"`
	AvailableUntil       *string              `json:"available_until"`
	IsAvailableNow       bool                 `json:"is_available_now"`
	AutoCloseEnabled     bool                 `json:"auto_close_enabled"`
	AutoCloseAfterCycles *int                 `json:"auto_close_after_cycles"`
	IsPausable           bool                 `json:"is_pausable"`
	IsClosable           bool                 `json:"is_closable"`
	IsRenewable          bool                 `json:"is_renewable"`
	Status               models.RecordStatus  `json:"status"`
	Products             []PlanLineResponse   `json:"products"`
	// Admin/staff only (null for customers).
	ActiveSubscriptions *int64    `json:"active_subscriptions,omitempty"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}
