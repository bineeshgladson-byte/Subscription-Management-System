package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// ---------- requests ----------

type VariantInput struct {
	Attribute  string `json:"attribute" binding:"required,min=1,max=60"` // e.g. "Brand"
	Name       string `json:"name" binding:"required,min=1,max=100"`     // e.g. "Odoo Brand"
	ExtraPrice string `json:"extra_price" binding:"omitempty,max=16"`    // added to sales_price; default "0"
	SKU        string `json:"sku" binding:"omitempty,max=60"`
}

type CreateProductRequest struct {
	Name        string         `json:"name" binding:"required,min=2,max=150"`
	ProductType string         `json:"product_type" binding:"required,oneof=software service addon"`
	SalesPrice  string         `json:"sales_price" binding:"required,max=16"`
	CostPrice   string         `json:"cost_price" binding:"omitempty,max=16"`
	Description string         `json:"description" binding:"omitempty,max=2000"`
	TaxID       *int64         `json:"tax_id"`
	Variants    []VariantInput `json:"variants" binding:"omitempty,max=50,dive"`
}

// UpdateProductRequest: only fields sent are changed.
// tax_id: 0 removes the default tax. status: "archived" archives, "active" restores.
type UpdateProductRequest struct {
	Name        *string `json:"name" binding:"omitempty,min=2,max=150"`
	ProductType *string `json:"product_type" binding:"omitempty,oneof=software service addon"`
	SalesPrice  *string `json:"sales_price" binding:"omitempty,max=16"`
	CostPrice   *string `json:"cost_price" binding:"omitempty,max=16"`
	Description *string `json:"description" binding:"omitempty,max=2000"`
	TaxID       *int64  `json:"tax_id"`
	Status      *string `json:"status" binding:"omitempty,oneof=active archived"`
}

type UpdateVariantRequest struct {
	Attribute  *string `json:"attribute" binding:"omitempty,min=1,max=60"`
	Name       *string `json:"name" binding:"omitempty,min=1,max=100"`
	ExtraPrice *string `json:"extra_price" binding:"omitempty,max=16"`
	SKU        *string `json:"sku" binding:"omitempty,max=60"`
	IsActive   *bool   `json:"is_active"`
}

// ---------- responses ----------

type TaxSummary struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Rate string `json:"rate"`
}

type VariantResponse struct {
	ID         int64   `json:"id"`
	Attribute  string  `json:"attribute"`
	Name       string  `json:"name"`
	ExtraPrice string  `json:"extra_price"`
	Price      string  `json:"price"` // sales_price + extra_price
	SKU        *string `json:"sku"`
	IsActive   bool    `json:"is_active"`
}

type ProductResponse struct {
	ID          int64              `json:"id"`
	Name        string             `json:"name"`
	ProductType models.ProductType `json:"product_type"`
	SalesPrice  string             `json:"sales_price"`
	// Cost and margin are admin-only (staff get null): they are business-sensitive.
	CostPrice   *string             `json:"cost_price"`
	Margin      *string             `json:"margin"`
	Description *string             `json:"description"`
	Tax         *TaxSummary         `json:"tax"`
	Status      models.RecordStatus `json:"status"`
	Variants    []VariantResponse   `json:"variants"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

func NewVariantResponse(p *models.Product, v *models.ProductVariant) VariantResponse {
	return VariantResponse{ID: v.ID, Attribute: v.Attribute, Name: v.Name, ExtraPrice: v.ExtraPrice.StringFixed(2),
		Price: p.SalesPrice.Add(v.ExtraPrice).StringFixed(2), SKU: v.SKU, IsActive: v.IsActive}
}

func NewProductResponse(p *models.Product, showCost bool) ProductResponse {
	r := ProductResponse{ID: p.ID, Name: p.Name, ProductType: p.ProductType, SalesPrice: p.SalesPrice.StringFixed(2),
		Description: p.Description, Status: p.Status, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		Variants: make([]VariantResponse, 0, len(p.Variants))}
	if showCost {
		cost := p.CostPrice.StringFixed(2)
		margin := p.SalesPrice.Sub(p.CostPrice).StringFixed(2)
		r.CostPrice, r.Margin = &cost, &margin
	}
	if p.Tax != nil {
		r.Tax = &TaxSummary{ID: p.Tax.ID, Name: p.Tax.Name, Rate: p.Tax.Rate.StringFixed(2)}
	}
	for i := range p.Variants {
		r.Variants = append(r.Variants, NewVariantResponse(p, &p.Variants[i]))
	}
	return r
}
