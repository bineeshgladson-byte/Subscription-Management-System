package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// DiscountService — A6. Brief: "Only the admin can create discount rules: a
// fixed amount or a percentage, with a minimum purchase or quantity, start and
// end dates, and a limit on how many times it can be used. A discount can apply
// to specific products or to whole subscriptions. Once it applies, it should
// appear on the next invoice on its own … and stop when it expires or runs out."
//
// "On its own" = the pricing engine (pricing.go) picks up every eligible
// discount automatically; nobody types it into an invoice.
type DiscountService struct {
	db        *gorm.DB
	discounts *repository.DiscountRepo
	products  *repository.ProductRepo
	clock     clock.Clock
}

func NewDiscountService(db *gorm.DB, discounts *repository.DiscountRepo, products *repository.ProductRepo, clk clock.Clock) *DiscountService {
	return &DiscountService{db: db, discounts: discounts, products: products, clock: clk}
}

func (s *DiscountService) today() time.Time {
	t := s.clock.Today()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (s *DiscountService) List(ctx context.Context, f repository.DiscountFilter, p pagination.Params) ([]dto.DiscountResponse, int64, error) {
	f.Today = s.today()
	list, total, err := s.discounts.List(s.db.WithContext(ctx), f, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.DiscountResponse, len(list))
	for i := range list {
		out[i] = discountResponse(&list[i], f.Today)
	}
	return out, total, nil
}

func (s *DiscountService) Get(ctx context.Context, id int64) (*dto.DiscountResponse, error) {
	d, err := s.discounts.FindByID(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if d == nil {
		return nil, apperr.NotFound("discount")
	}
	out := discountResponse(d, s.today())
	return &out, nil
}

// discountDraft is the full desired state, validated as a whole (so the same
// checks run for create and update).
type discountDraft struct {
	name, dtype, scope string
	code               *string
	value              decimal.Decimal
	productIDs         []int64
	minPurchase        *decimal.Decimal
	minQty, maxUsage   *int
	startsOn           time.Time
	endsOn             *time.Time
	usageCount         int
	replaceProducts    bool
}

func (s *DiscountService) validate(db *gorm.DB, d *discountDraft, exceptID int64) error {
	errs := map[string]string{}
	if d.value.LessThanOrEqual(decimal.Zero) {
		errs["value"] = "must be greater than 0"
	} else if d.dtype == string(models.DiscountPercentage) && d.value.GreaterThan(hundred) {
		errs["value"] = "a percentage can't be more than 100"
	}
	if d.endsOn != nil && d.endsOn.Before(d.startsOn) {
		errs["ends_on"] = "must be on or after starts_on"
	}
	if d.maxUsage != nil && *d.maxUsage < d.usageCount {
		errs["max_usage"] = fmt.Sprintf("can't be below the %d time(s) it has already been used", d.usageCount)
	}
	if d.scope == string(models.ScopeProduct) && len(d.productIDs) == 0 {
		errs["product_ids"] = "choose at least one product for a product discount"
	}
	if len(errs) > 0 {
		return apperr.Validation("Some fields are invalid", errs)
	}
	if d.scope == string(models.ScopeSubscription) {
		d.productIDs = nil
	}
	if d.replaceProducts && len(d.productIDs) > 0 {
		found, err := s.products.FindByIDs(db, d.productIDs)
		if err != nil {
			return err
		}
		for i, pid := range d.productIDs {
			if p := found[pid]; p == nil || p.Status != models.StatusActive {
				return apperr.Validation("Some fields are invalid",
					map[string]string{fmt.Sprintf("product_ids[%d]", i): "must be an active product"})
			}
		}
	}
	if d.code != nil {
		if taken, err := s.discounts.CodeTaken(db, *d.code, exceptID); err != nil {
			return err
		} else if taken {
			e := apperr.Conflict("another discount already uses this code")
			e.Details = map[string]string{"code": "is already used"}
			return e
		}
	}
	return nil
}

func (s *DiscountService) Create(ctx context.Context, actorID int64, in dto.CreateDiscountRequest) (*dto.DiscountResponse, error) {
	errs := map[string]string{}
	d := &discountDraft{name: strings.TrimSpace(in.Name), dtype: in.DiscountType, scope: in.Scope,
		productIDs: uniqueIDs(in.ProductIDs), minQty: in.MinQuantity, maxUsage: in.MaxUsage, startsOn: s.today(),
		replaceProducts: true}
	if v, reason := parseMoney(in.Value, false); reason != "" {
		errs["value"] = reason
	} else {
		d.value = v
	}
	if c := strings.ToUpper(strings.TrimSpace(in.Code)); c != "" {
		d.code = &c
	}
	if in.MinPurchaseAmount != "" {
		if v, reason := parseMoney(in.MinPurchaseAmount, false); reason != "" {
			errs["min_purchase_amount"] = reason
		} else {
			d.minPurchase = &v
		}
	}
	if in.StartsOn != "" {
		if t, ok := parseDate(in.StartsOn); !ok {
			errs["starts_on"] = "must be a date like 2026-11-01"
		} else {
			d.startsOn = t
		}
	}
	if in.EndsOn != "" {
		if t, ok := parseDate(in.EndsOn); !ok {
			errs["ends_on"] = "must be a date like 2026-12-31"
		} else {
			d.endsOn = &t
		}
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}
	db := s.db.WithContext(ctx)
	if err := s.validate(db, d, 0); err != nil {
		return nil, err
	}
	m := &models.Discount{Name: d.name, Code: d.code, DiscountType: models.DiscountType(d.dtype), Value: d.value,
		Scope: models.DiscountScope(d.scope), MinPurchaseAmount: d.minPurchase, MinQuantity: d.minQty,
		StartsOn: d.startsOn, EndsOn: d.endsOn, MaxUsage: d.maxUsage, IsActive: true, CreatedBy: &actorID}
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := s.discounts.Create(tx, m); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return apperr.Conflict("another discount already uses this code")
			}
			return err
		}
		return s.discounts.ReplaceProducts(tx, m.ID, d.productIDs)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, m.ID)
}

func (s *DiscountService) Update(ctx context.Context, id int64, in dto.UpdateDiscountRequest) (*dto.DiscountResponse, error) {
	db := s.db.WithContext(ctx)
	cur, err := s.discounts.FindByID(db, id)
	if err != nil {
		return nil, err
	}
	if cur == nil {
		return nil, apperr.NotFound("discount")
	}
	// Start from the current state, apply changes, then validate the result.
	d := &discountDraft{name: cur.Name, dtype: string(cur.DiscountType), scope: string(cur.Scope), code: cur.Code,
		value: cur.Value, minPurchase: cur.MinPurchaseAmount, minQty: cur.MinQuantity, maxUsage: cur.MaxUsage,
		startsOn: cur.StartsOn, endsOn: cur.EndsOn, usageCount: cur.UsageCount}
	for _, p := range cur.Products {
		d.productIDs = append(d.productIDs, p.ID)
	}
	errs := map[string]string{}
	fields := map[string]any{}
	if in.Name != nil {
		d.name = strings.TrimSpace(*in.Name)
		fields["name"] = d.name
	}
	if in.Code != nil {
		if c := strings.ToUpper(strings.TrimSpace(*in.Code)); c == "" {
			d.code = nil
		} else {
			d.code = &c
		}
		fields["code"] = d.code
	}
	if in.DiscountType != nil {
		d.dtype = *in.DiscountType
		fields["discount_type"] = d.dtype
	}
	if in.Value != nil {
		if v, reason := parseMoney(*in.Value, false); reason != "" {
			errs["value"] = reason
		} else {
			d.value = v
			fields["value"] = v
		}
	}
	if in.Scope != nil {
		d.scope = *in.Scope
		fields["scope"] = d.scope
		d.replaceProducts = true
	}
	if in.ProductIDs != nil {
		d.productIDs = uniqueIDs(in.ProductIDs)
		d.replaceProducts = true
	}
	if in.MinPurchaseAmount != nil {
		if *in.MinPurchaseAmount == "" {
			d.minPurchase = nil
		} else if v, reason := parseMoney(*in.MinPurchaseAmount, false); reason != "" {
			errs["min_purchase_amount"] = reason
		} else {
			d.minPurchase = &v
		}
		fields["min_purchase_amount"] = d.minPurchase
	}
	if in.MinQuantity != nil {
		if *in.MinQuantity == 0 {
			d.minQty = nil
		} else {
			d.minQty = in.MinQuantity
		}
		fields["min_quantity"] = d.minQty
	}
	if in.StartsOn != nil {
		if t, ok := parseDate(*in.StartsOn); !ok {
			errs["starts_on"] = "must be a date like 2026-11-01"
		} else {
			d.startsOn = t
			fields["starts_on"] = t
		}
	}
	if in.EndsOn != nil {
		if *in.EndsOn == "" {
			d.endsOn = nil
		} else if t, ok := parseDate(*in.EndsOn); !ok {
			errs["ends_on"] = "must be a date like 2026-12-31"
		} else {
			d.endsOn = &t
		}
		fields["ends_on"] = d.endsOn
	}
	if in.MaxUsage != nil {
		if *in.MaxUsage == 0 {
			d.maxUsage = nil
		} else {
			d.maxUsage = in.MaxUsage
		}
		fields["max_usage"] = d.maxUsage
	}
	if in.IsActive != nil {
		fields["is_active"] = *in.IsActive
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}
	if err := s.validate(db, d, id); err != nil {
		return nil, err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if len(fields) > 0 {
			if err := s.discounts.Update(tx, id, fields); err != nil {
				return err
			}
		}
		if d.replaceProducts {
			return s.discounts.ReplaceProducts(tx, id, d.productIDs)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}

// Deactivate is DELETE /discounts/:id (rules are kept for the usage history).
func (s *DiscountService) Deactivate(ctx context.Context, id int64) (*dto.DiscountResponse, error) {
	off := false
	return s.Update(ctx, id, dto.UpdateDiscountRequest{IsActive: &off})
}

func (s *DiscountService) Usages(ctx context.Context, id int64, p pagination.Params) ([]dto.DiscountUsageResponse, int64, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, 0, err
	}
	rows, total, err := s.discounts.Usages(s.db.WithContext(ctx), id, p)
	if err != nil {
		return nil, 0, err
	}
	out := make([]dto.DiscountUsageResponse, len(rows))
	for i, r := range rows {
		amt, _ := decimal.NewFromString(r.Amount)
		out[i] = dto.DiscountUsageResponse{ID: r.ID, InvoiceID: r.InvoiceID, InvoiceNumber: r.InvoiceNumber,
			CustomerID: r.CustomerID, CustomerName: r.CustomerName, Amount: amt.StringFixed(2), CreatedAt: r.CreatedAt}
	}
	return out, total, nil
}

// ToPriceDiscounts converts DB rows to pricing-engine inputs.
func ToPriceDiscounts(list []models.Discount) []PriceDiscount {
	out := make([]PriceDiscount, 0, len(list))
	for _, d := range list {
		pd := PriceDiscount{ID: d.ID, Name: d.Name, Type: d.DiscountType, Value: d.Value, Scope: d.Scope,
			MinPurchase: d.MinPurchaseAmount, MinQuantity: d.MinQuantity, ProductIDs: map[int64]bool{}}
		for _, p := range d.Products {
			pd.ProductIDs[p.ID] = true
		}
		out = append(out, pd)
	}
	return out
}

func discountState(d *models.Discount, today time.Time) string {
	day := dateString(today)
	switch {
	case !d.IsActive:
		return "inactive"
	case dateString(d.StartsOn) > day:
		return "scheduled"
	case d.EndsOn != nil && dateString(*d.EndsOn) < day:
		return "expired"
	case d.MaxUsage != nil && d.UsageCount >= *d.MaxUsage:
		return "used_up"
	}
	return "active"
}

func discountResponse(d *models.Discount, today time.Time) dto.DiscountResponse {
	r := dto.DiscountResponse{ID: d.ID, Name: d.Name, Code: d.Code, DiscountType: string(d.DiscountType),
		Value: d.Value.StringFixed(2), Scope: string(d.Scope), Products: make([]dto.ProductRef, 0, len(d.Products)),
		MinQuantity: d.MinQuantity, StartsOn: dateString(d.StartsOn), MaxUsage: d.MaxUsage, UsageCount: d.UsageCount,
		IsActive: d.IsActive, State: discountState(d, today), CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
	if d.DiscountType == models.DiscountPercentage {
		r.Label = d.Value.String() + "% off"
	} else {
		r.Label = "₹" + d.Value.StringFixed(2) + " off"
	}
	for _, p := range d.Products {
		r.Products = append(r.Products, dto.ProductRef{ID: p.ID, Name: p.Name})
	}
	if d.MinPurchaseAmount != nil {
		v := d.MinPurchaseAmount.StringFixed(2)
		r.MinPurchaseAmount = &v
	}
	if d.EndsOn != nil {
		v := dateString(*d.EndsOn)
		r.EndsOn = &v
	}
	if d.MaxUsage != nil {
		left := *d.MaxUsage - d.UsageCount
		if left < 0 {
			left = 0
		}
		r.RemainingUses = &left
	}
	return r
}

func uniqueIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

// ---------------------------------------------------------------------------
// PricingService: POST /pricing/preview — prices catalogue lines with today's
// (or a given day's) discounts and taxes. Staff use it while building
// quotations; A7–A9 call Price() the same way.
// ---------------------------------------------------------------------------

type PricingService struct {
	db        *gorm.DB
	products  *repository.ProductRepo
	discounts *repository.DiscountRepo
	clock     clock.Clock
}

func NewPricingService(db *gorm.DB, products *repository.ProductRepo, discounts *repository.DiscountRepo, clk clock.Clock) *PricingService {
	return &PricingService{db: db, products: products, discounts: discounts, clock: clk}
}

func (s *PricingService) Preview(ctx context.Context, in dto.PricePreviewRequest) (*dto.PricePreviewResponse, error) {
	t := s.clock.Today()
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if in.Date != "" {
		d, ok := parseDate(in.Date)
		if !ok {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"date": "must be a date like 2026-11-01"})
		}
		day = d
	}
	db := s.db.WithContext(ctx)
	specs := make([]lineSpec, len(in.Lines))
	for i, l := range in.Lines {
		specs[i] = lineSpec{ProductID: l.ProductID, VariantID: l.VariantID, Quantity: l.Quantity, PriceOverride: l.UnitPrice}
	}
	resolved, err := resolveLines(db, s.products, "lines", specs, true)
	if err != nil {
		return nil, err
	}
	eligible, err := s.discounts.Eligible(db, day)
	if err != nil {
		return nil, err
	}
	lines := make([]PriceLine, len(resolved))
	for i, r := range resolved {
		desc := r.Product.Name
		if r.Variant != nil {
			desc += " (" + r.Variant.Name + ")"
		}
		lines[i] = PriceLine{ProductID: r.ProductID, Description: desc, Quantity: r.Quantity,
			UnitPrice: r.UnitPrice(), TaxRate: r.TaxRate()}
	}
	res := Price(lines, ToPriceDiscounts(eligible))

	names := map[int64]string{}
	for _, d := range eligible {
		names[d.ID] = d.Name
	}
	out := &dto.PricePreviewResponse{Date: dateString(day), Subtotal: res.Subtotal.StringFixed(2),
		DiscountTotal: res.DiscountTotal.StringFixed(2), TaxTotal: res.TaxTotal.StringFixed(2), Total: res.Total.StringFixed(2),
		Lines: make([]dto.PricedLineResponse, len(res.Lines)), Discounts: make([]dto.AppliedDiscountResponse, 0, len(res.Applied))}
	for i, l := range res.Lines {
		pl := dto.PricedLineResponse{ProductID: l.ProductID, Description: l.Description, Quantity: l.Quantity,
			UnitPrice: l.UnitPrice.StringFixed(2), Subtotal: l.Subtotal.StringFixed(2),
			ProductDiscount: l.ProductDiscount.StringFixed(2), SubscriptionDiscount: l.SubscriptionDiscount.StringFixed(2),
			TaxRate: l.TaxRate.StringFixed(2), TaxAmount: l.TaxAmount.StringFixed(2), Total: l.Total.StringFixed(2)}
		if l.ProductDiscountID != 0 {
			n := names[l.ProductDiscountID]
			pl.ProductDiscountName = &n
		}
		out.Lines[i] = pl
	}
	for _, a := range res.Applied {
		out.Discounts = append(out.Discounts, dto.AppliedDiscountResponse{DiscountID: a.DiscountID, Name: a.Name,
			Scope: string(a.Scope), Amount: a.Amount.StringFixed(2)})
	}
	return out, nil
}
