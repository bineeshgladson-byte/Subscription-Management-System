package service

import (
	"fmt"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
)

// lineSpec is one requested catalogue line (plan line, template line, price preview line).
type lineSpec struct {
	ProductID     int64
	VariantID     *int64
	Quantity      int
	PriceOverride string // "" = use catalogue price
}

// resolvedLine is a validated line with its product/variant loaded.
type resolvedLine struct {
	ProductID     int64
	VariantID     *int64
	Quantity      int
	PriceOverride *decimal.Decimal
	Product       *models.Product
	Variant       *models.ProductVariant
}

// UnitPrice = override, or sales_price + variant extra_price. The same formula
// as repository.recalcSQL.
func (l resolvedLine) UnitPrice() decimal.Decimal {
	if l.PriceOverride != nil {
		return *l.PriceOverride
	}
	p := l.Product.SalesPrice
	if l.Variant != nil {
		p = p.Add(l.Variant.ExtraPrice)
	}
	return p
}

// TaxRate is the product's default tax rate (0 if it has none or it's inactive).
func (l resolvedLine) TaxRate() decimal.Decimal {
	if l.Product.Tax != nil && l.Product.Tax.IsActive {
		return l.Product.Tax.Rate
	}
	return decimal.Zero
}

// resolveLines checks lines against the catalogue: active product, variant that
// belongs to it and is active, no duplicate product/variant pairs, valid
// override. field is the JSON field name used in error details ("products", "lines").
func resolveLines(db *gorm.DB, products *repository.ProductRepo, field string, specs []lineSpec, allowOverride bool) ([]resolvedLine, error) {
	ids := make([]int64, 0, len(specs))
	for _, l := range specs {
		ids = append(ids, l.ProductID)
	}
	found, err := products.FindByIDs(db, ids)
	if err != nil {
		return nil, err
	}
	errs := map[string]string{}
	seen := map[string]bool{}
	out := make([]resolvedLine, 0, len(specs))
	for i, l := range specs {
		f := fmt.Sprintf("%s[%d]", field, i)
		p := found[l.ProductID]
		if p == nil || p.Status != models.StatusActive {
			errs[f+".product_id"] = "must be an active product"
			continue
		}
		line := resolvedLine{ProductID: l.ProductID, VariantID: l.VariantID, Quantity: defaultInt(l.Quantity, 1), Product: p}
		if l.VariantID != nil {
			for j := range p.Variants {
				if p.Variants[j].ID == *l.VariantID {
					line.Variant = &p.Variants[j]
				}
			}
			if line.Variant == nil || !line.Variant.IsActive {
				errs[f+".variant_id"] = "must be an active variant of this product"
				continue
			}
		}
		key := fmt.Sprintf("%d|%d", l.ProductID, derefOr(l.VariantID, 0))
		if seen[key] {
			errs[f] = "this product/variant is listed twice"
			continue
		}
		seen[key] = true
		if l.PriceOverride != "" {
			if !allowOverride {
				errs[f+".price_override"] = "is not allowed here"
				continue
			}
			v, reason := parseMoney(l.PriceOverride, false)
			if reason != "" {
				errs[f+".price_override"] = reason
				continue
			}
			line.PriceOverride = &v
		}
		out = append(out, line)
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some lines are invalid", errs)
	}
	return out, nil
}
