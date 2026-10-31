package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// Rates are sent as strings like money ("18.00"), never floats.
type CreateTaxRequest struct {
	Name string `json:"name" binding:"required,min=2,max=60"`
	Rate string `json:"rate" binding:"required,max=6"`
}

// UpdateTaxRequest: only fields sent are changed. is_active=true reactivates.
type UpdateTaxRequest struct {
	Name     *string `json:"name" binding:"omitempty,min=2,max=60"`
	Rate     *string `json:"rate" binding:"omitempty,max=6"`
	IsActive *bool   `json:"is_active"`
}

type TaxResponse struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Rate          string    `json:"rate"` // percentage, e.g. "18.00"
	IsActive      bool      `json:"is_active"`
	ProductsUsing int64     `json:"products_using"` // active products with this as default tax
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func NewTaxResponse(t *models.Tax, productsUsing int64) TaxResponse {
	return TaxResponse{ID: t.ID, Name: t.Name, Rate: t.Rate.StringFixed(2), IsActive: t.IsActive,
		ProductsUsing: productsUsing, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt}
}
