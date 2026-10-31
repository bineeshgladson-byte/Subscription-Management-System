package service

import (
	"testing"

	"github.com/shopspring/decimal"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

func dm(s string) decimal.Decimal { return decimal.RequireFromString(s) }
func dp(s string) *decimal.Decimal {
	d := dm(s)
	return &d
}
func ip(i int) *int { return &i }

func TestPriceAcmeExample(t *testing.T) {
	// From the architecture plan: 10 seats of the Odoo Brand variant (999 + 560),
	// "10% off orders over 10,000", GST 18% → ₹16,556.58.
	lines := []PriceLine{{ProductID: 1, Quantity: 10, UnitPrice: dm("1559"), TaxRate: dm("18")}}
	discounts := []PriceDiscount{{ID: 5, Name: "10% off big orders", Type: models.DiscountPercentage, Value: dm("10"),
		Scope: models.ScopeSubscription, MinPurchase: dp("10000")}}
	r := Price(lines, discounts)
	checks := map[string][2]decimal.Decimal{
		"subtotal": {r.Subtotal, dm("15590.00")}, "discount": {r.DiscountTotal, dm("1559.00")},
		"tax": {r.TaxTotal, dm("2525.58")}, "total": {r.Total, dm("16556.58")},
	}
	for name, c := range checks {
		if !c[0].Equal(c[1]) {
			t.Errorf("%s = %s, want %s", name, c[0], c[1])
		}
	}
	if len(r.Applied) != 1 || r.Applied[0].DiscountID != 5 {
		t.Errorf("expected discount 5 applied, got %+v", r.Applied)
	}
}

func TestPriceRahulNoDiscount(t *testing.T) {
	r := Price([]PriceLine{{ProductID: 1, Quantity: 1, UnitPrice: dm("999"), TaxRate: dm("18")}}, nil)
	if !r.Total.Equal(dm("1178.82")) || !r.TaxTotal.Equal(dm("179.82")) {
		t.Errorf("Rahul: total %s tax %s", r.Total, r.TaxTotal)
	}
}

func TestPriceMinimumsAndBestOf(t *testing.T) {
	lines := []PriceLine{
		{ProductID: 1, Quantity: 2, UnitPrice: dm("1000"), TaxRate: dm("18")},
		{ProductID: 2, Quantity: 1, UnitPrice: dm("500"), TaxRate: dm("0")},
	}
	discounts := []PriceDiscount{
		// product 1: fixed 300 vs 20% (=400) → 20% wins
		{ID: 1, Type: models.DiscountFixed, Value: dm("300"), Scope: models.ScopeProduct, ProductIDs: map[int64]bool{1: true}},
		{ID: 2, Type: models.DiscountPercentage, Value: dm("20"), Scope: models.ScopeProduct, ProductIDs: map[int64]bool{1: true}},
		// product 2: needs 5 units → not applied
		{ID: 3, Type: models.DiscountFixed, Value: dm("100"), Scope: models.ScopeProduct, ProductIDs: map[int64]bool{2: true}, MinQuantity: ip(5)},
		// subscription: needs 10,000 purchase → not applied
		{ID: 4, Type: models.DiscountPercentage, Value: dm("50"), Scope: models.ScopeSubscription, MinPurchase: dp("10000")},
	}
	r := Price(lines, discounts)
	if r.Lines[0].ProductDiscountID != 2 || !r.Lines[0].ProductDiscount.Equal(dm("400")) {
		t.Errorf("line 1 should get discount 2 (400): %+v", r.Lines[0])
	}
	if !r.Lines[1].ProductDiscount.IsZero() {
		t.Errorf("line 2 must not be discounted: %+v", r.Lines[1])
	}
	// (2000-400)*1.18 + 500 = 1888 + 500
	if !r.Total.Equal(dm("2388.00")) {
		t.Errorf("total %s", r.Total)
	}
}

func TestPriceFixedDiscountCappedAndSpread(t *testing.T) {
	// A fixed ₹40 subscription discount spread over 3 lines must add up to exactly 40.
	lines := []PriceLine{
		{ProductID: 1, Quantity: 1, UnitPrice: dm("33.33"), TaxRate: dm("18")},
		{ProductID: 2, Quantity: 1, UnitPrice: dm("33.33"), TaxRate: dm("18")},
		{ProductID: 3, Quantity: 1, UnitPrice: dm("33.34"), TaxRate: dm("18")},
	}
	r := Price(lines, []PriceDiscount{{ID: 1, Type: models.DiscountFixed, Value: dm("40"), Scope: models.ScopeSubscription}})
	sum := decimal.Zero
	for _, l := range r.Lines {
		sum = sum.Add(l.SubscriptionDiscount)
	}
	if !sum.Equal(dm("40")) || !r.DiscountTotal.Equal(dm("40")) {
		t.Errorf("spread sum %s, discount total %s", sum, r.DiscountTotal)
	}
	// A discount bigger than the order is capped at the order value.
	r = Price([]PriceLine{{ProductID: 1, Quantity: 1, UnitPrice: dm("50"), TaxRate: dm("18")}},
		[]PriceDiscount{{ID: 1, Type: models.DiscountFixed, Value: dm("80"), Scope: models.ScopeSubscription}})
	if !r.DiscountTotal.Equal(dm("50")) || !r.Total.IsZero() {
		t.Errorf("cap: discount %s total %s", r.DiscountTotal, r.Total)
	}
}
