package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// ProductService — A3. Brief: "The admin adds it with its name, type, sales
// price, cost price and notes, and offers it in variants, for example by brand,
// where choosing the Odoo brand adds 560 to the price."
//
// Admin writes; admin + staff read (staff don't see cost price / margin).
// Products and variants are never hard-deleted: archive / deactivate instead,
// so old subscriptions and invoices keep pointing at real rows.
type ProductService struct {
	db       *gorm.DB
	products *repository.ProductRepo
	taxes    *repository.TaxRepo
	plans    *repository.PlanRepo
}

func NewProductService(db *gorm.DB, products *repository.ProductRepo, taxes *repository.TaxRepo, plans *repository.PlanRepo) *ProductService {
	return &ProductService{db: db, products: products, taxes: taxes, plans: plans}
}

func (s *ProductService) List(ctx context.Context, f repository.ProductFilter, p pagination.Params) ([]models.Product, int64, error) {
	return s.products.List(s.db.WithContext(ctx), f, p)
}

func (s *ProductService) Get(ctx context.Context, id int64) (*models.Product, error) {
	p, err := s.products.FindByID(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperr.NotFound("product")
	}
	return p, nil
}

func (s *ProductService) Create(ctx context.Context, in dto.CreateProductRequest) (*models.Product, error) {
	errs := map[string]string{}
	name := strings.TrimSpace(in.Name)
	sales, reason := parseMoney(in.SalesPrice, false)
	if reason != "" {
		errs["sales_price"] = reason
	}
	cost := decimal.Zero
	if in.CostPrice != "" {
		if cost, reason = parseMoney(in.CostPrice, false); reason != "" {
			errs["cost_price"] = reason
		}
	}

	// Validate variants up front so nothing is written if any is wrong.
	variants := make([]models.ProductVariant, 0, len(in.Variants))
	seen := map[string]bool{}
	for i, v := range in.Variants {
		key := strings.ToLower(strings.TrimSpace(v.Attribute)) + "|" + strings.ToLower(strings.TrimSpace(v.Name))
		field := fmt.Sprintf("variants[%d]", i)
		if seen[key] {
			errs[field] = "duplicate attribute + name"
			continue
		}
		seen[key] = true
		extra := decimal.Zero
		if v.ExtraPrice != "" {
			if extra, reason = parseMoney(v.ExtraPrice, true); reason != "" {
				errs[field+".extra_price"] = reason
				continue
			}
		}
		if len(errs) == 0 && sales.Add(extra).IsNegative() {
			errs[field+".extra_price"] = "would make the variant price negative"
			continue
		}
		mv := models.ProductVariant{Attribute: strings.TrimSpace(v.Attribute), Name: strings.TrimSpace(v.Name),
			ExtraPrice: extra, IsActive: true}
		if sku := strings.TrimSpace(v.SKU); sku != "" {
			mv.SKU = &sku
		}
		variants = append(variants, mv)
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}

	db := s.db.WithContext(ctx)
	if err := s.checkTax(db, in.TaxID); err != nil {
		return nil, err
	}
	if taken, err := s.products.ActiveNameTaken(db, name, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, productNameTaken()
	}
	for i := range variants {
		if variants[i].SKU != nil {
			if taken, err := s.products.SKUTaken(db, *variants[i].SKU, 0); err != nil {
				return nil, err
			} else if taken {
				return nil, skuTaken(fmt.Sprintf("variants[%d].sku", i))
			}
		}
	}

	p := &models.Product{Name: name, ProductType: models.ProductType(in.ProductType), SalesPrice: sales,
		CostPrice: cost, Description: optional(strings.TrimSpace(in.Description)), TaxID: in.TaxID,
		Status: models.StatusActive}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := s.products.Create(tx, p); err != nil {
			return mapProductDupErr(err)
		}
		for i := range variants {
			variants[i].ProductID = p.ID
			if err := s.products.CreateVariant(tx, &variants[i]); err != nil {
				return mapProductDupErr(err)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, p.ID)
}

func (s *ProductService) Update(ctx context.Context, id int64, in dto.UpdateProductRequest) (*models.Product, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	fields := map[string]any{}
	errs := map[string]string{}
	newSales := p.SalesPrice

	if in.Name != nil {
		fields["name"] = strings.TrimSpace(*in.Name)
	}
	if in.ProductType != nil {
		fields["product_type"] = *in.ProductType
	}
	if in.SalesPrice != nil {
		v, reason := parseMoney(*in.SalesPrice, false)
		if reason != "" {
			errs["sales_price"] = reason
		} else {
			newSales = v
			fields["sales_price"] = v
			// No active variant may end up with a negative price.
			for _, vr := range p.Variants {
				if vr.IsActive && v.Add(vr.ExtraPrice).IsNegative() {
					errs["sales_price"] = fmt.Sprintf("would make variant %q negative", vr.Name)
				}
			}
		}
	}
	if in.CostPrice != nil {
		if v, reason := parseMoney(*in.CostPrice, false); reason != "" {
			errs["cost_price"] = reason
		} else {
			fields["cost_price"] = v
		}
	}
	if in.Description != nil {
		fields["description"] = nullIfEmpty(strings.TrimSpace(*in.Description))
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}
	if in.TaxID != nil {
		if *in.TaxID == 0 {
			fields["tax_id"] = nil
		} else {
			if err := s.checkTax(db, in.TaxID); err != nil {
				return nil, err
			}
			fields["tax_id"] = *in.TaxID
		}
	}

	targetStatus := p.Status
	if in.Status != nil {
		targetStatus = models.RecordStatus(*in.Status)
		if targetStatus == models.StatusArchived && p.Status == models.StatusActive {
			if err := s.checkNotInActivePlans(db, p.ID, 0, "PRODUCT_IN_USE", "product"); err != nil {
				return nil, err
			}
		}
		fields["status"] = targetStatus
	}
	// Name must be unique among ACTIVE products: check whenever the result is active.
	if targetStatus == models.StatusActive && (in.Name != nil || (in.Status != nil && p.Status != models.StatusActive)) {
		name := p.Name
		if in.Name != nil {
			name = strings.TrimSpace(*in.Name)
		}
		if taken, err := s.products.ActiveNameTaken(db, name, p.ID); err != nil {
			return nil, err
		} else if taken {
			return nil, productNameTaken()
		}
	}
	if len(fields) == 0 {
		return p, nil
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := s.products.Update(tx, p.ID, fields); err != nil {
			return mapProductDupErr(err)
		}
		if !newSales.Equal(p.SalesPrice) {
			return s.plans.RecalculateForProduct(tx, p.ID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Archive is DELETE /products/:id. Refused while an active plan includes the
// product (archive or edit the plan first).
func (s *ProductService) Archive(ctx context.Context, id int64) (*models.Product, error) {
	archived := string(models.StatusArchived)
	return s.Update(ctx, id, dto.UpdateProductRequest{Status: &archived})
}

// ---------- variants ----------

func (s *ProductService) AddVariant(ctx context.Context, productID int64, in dto.VariantInput) (*models.Product, error) {
	p, err := s.Get(ctx, productID)
	if err != nil {
		return nil, err
	}
	extra := decimal.Zero
	if in.ExtraPrice != "" {
		var reason string
		if extra, reason = parseMoney(in.ExtraPrice, true); reason != "" {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"extra_price": reason})
		}
	}
	if p.SalesPrice.Add(extra).IsNegative() {
		return nil, apperr.Validation("Some fields are invalid",
			map[string]string{"extra_price": "would make the variant price negative"})
	}
	db := s.db.WithContext(ctx)
	attr, name := strings.TrimSpace(in.Attribute), strings.TrimSpace(in.Name)
	if taken, err := s.products.VariantNameTaken(db, p.ID, attr, name, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, variantNameTaken()
	}
	v := &models.ProductVariant{ProductID: p.ID, Attribute: attr, Name: name, ExtraPrice: extra, IsActive: true}
	if sku := strings.TrimSpace(in.SKU); sku != "" {
		if taken, err := s.products.SKUTaken(db, sku, 0); err != nil {
			return nil, err
		} else if taken {
			return nil, skuTaken("sku")
		}
		v.SKU = &sku
	}
	if err := s.products.CreateVariant(db, v); err != nil {
		return nil, mapProductDupErr(err)
	}
	return s.Get(ctx, productID)
}

func (s *ProductService) UpdateVariant(ctx context.Context, productID, variantID int64, in dto.UpdateVariantRequest) (*models.Product, error) {
	p, err := s.Get(ctx, productID)
	if err != nil {
		return nil, err
	}
	db := s.db.WithContext(ctx)
	v, err := s.products.FindVariant(db, productID, variantID)
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, apperr.NotFound("variant")
	}

	fields := map[string]any{}
	attr, name := v.Attribute, v.Name
	if in.Attribute != nil {
		attr = strings.TrimSpace(*in.Attribute)
		fields["attribute"] = attr
	}
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
		fields["name"] = name
	}
	if in.Attribute != nil || in.Name != nil {
		if taken, err := s.products.VariantNameTaken(db, productID, attr, name, v.ID); err != nil {
			return nil, err
		} else if taken {
			return nil, variantNameTaken()
		}
	}
	priceChanged := false
	if in.ExtraPrice != nil {
		extra, reason := parseMoney(*in.ExtraPrice, true)
		if reason == "" && p.SalesPrice.Add(extra).IsNegative() {
			reason = "would make the variant price negative"
		}
		if reason != "" {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"extra_price": reason})
		}
		priceChanged = !extra.Equal(v.ExtraPrice)
		fields["extra_price"] = extra
	}
	if in.SKU != nil {
		sku := strings.TrimSpace(*in.SKU)
		if sku == "" {
			fields["sku"] = nil
		} else {
			if taken, err := s.products.SKUTaken(db, sku, v.ID); err != nil {
				return nil, err
			} else if taken {
				return nil, skuTaken("sku")
			}
			fields["sku"] = sku
		}
	}
	if in.IsActive != nil {
		if !*in.IsActive && v.IsActive {
			if err := s.checkNotInActivePlans(db, productID, v.ID, "VARIANT_IN_USE", "variant"); err != nil {
				return nil, err
			}
		}
		fields["is_active"] = *in.IsActive
	}
	if len(fields) == 0 {
		return p, nil
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := s.products.UpdateVariant(tx, v.ID, fields); err != nil {
			return mapProductDupErr(err)
		}
		if priceChanged {
			return s.plans.RecalculateForProduct(tx, productID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, productID)
}

// DeactivateVariant is DELETE /products/:id/variants/:vid (never hard-deletes).
func (s *ProductService) DeactivateVariant(ctx context.Context, productID, variantID int64) (*models.Product, error) {
	off := false
	return s.UpdateVariant(ctx, productID, variantID, dto.UpdateVariantRequest{IsActive: &off})
}

// ---------- helpers ----------

// checkTax: a product's default tax must exist and be active.
func (s *ProductService) checkTax(db *gorm.DB, taxID *int64) error {
	if taxID == nil {
		return nil
	}
	t, err := s.taxes.FindByID(db, *taxID)
	if err != nil {
		return err
	}
	if t == nil || !t.IsActive {
		return apperr.Validation("Some fields are invalid", map[string]string{"tax_id": "must be an active tax"})
	}
	return nil
}

func (s *ProductService) checkNotInActivePlans(db *gorm.DB, productID, variantID int64, rule, what string) error {
	n, err := s.products.ActivePlansUsing(db, productID, variantID)
	if err != nil {
		return err
	}
	if n > 0 {
		e := apperr.Rule(rule, fmt.Sprintf("%d active plan(s) include this %s; archive or edit those plans first", n, what))
		e.Details["active_plans"] = fmt.Sprint(n)
		return e
	}
	return nil
}

func productNameTaken() error {
	e := apperr.Conflict("an active product with this name already exists")
	e.Details = map[string]string{"name": "is already used by another active product"}
	return e
}

func variantNameTaken() error {
	e := apperr.Conflict("this product already has a variant with that attribute and name")
	e.Details = map[string]string{"name": "already exists for this attribute"}
	return e
}

func skuTaken(field string) error {
	e := apperr.Conflict("this SKU is already used")
	e.Details = map[string]string{field: "is already used by another variant"}
	return e
}

// mapProductDupErr turns a unique-index race into a 409 instead of a 500.
func mapProductDupErr(err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return apperr.Conflict("a product, variant or SKU with these details already exists")
	}
	return err
}
