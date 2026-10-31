package service

import (
	"sort"

	"github.com/shopspring/decimal"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// ---------------------------------------------------------------------------
// Pricing engine (A6). PURE functions: no database, no clock. Quotations,
// subscriptions and invoices (A7–A9) all price their lines through Price(), so
// the numbers on a quotation, a subscription and its invoices always agree.
//
// Order of operations (matches the architecture plan):
//  1. line subtotal  = quantity × unit price                     (rounded to 2 dp)
//  2. product discount: the single best eligible product-scope discount per line
//  3. subscription discount: the single best eligible subscription-wide discount,
//     spread across lines in proportion to their value
//  4. tax per line   = (subtotal − discounts) × rate / 100        (rounded half-up)
//  5. total          = Σ (subtotal − discounts + tax)
//
// Eligibility on dates / active / usage left is decided by the caller (the DB
// query); this function checks the purchase-size minimums.
// ---------------------------------------------------------------------------

var hundred = decimal.NewFromInt(100)

type PriceLine struct {
	ProductID   int64
	Description string
	Quantity    int
	UnitPrice   decimal.Decimal
	TaxRate     decimal.Decimal // percent, e.g. 18
}

type PriceDiscount struct {
	ID          int64
	Name        string
	Type        models.DiscountType
	Value       decimal.Decimal // amount, or percent
	Scope       models.DiscountScope
	ProductIDs  map[int64]bool // product scope only
	MinPurchase *decimal.Decimal
	MinQuantity *int
}

type PricedLine struct {
	PriceLine
	Subtotal             decimal.Decimal
	ProductDiscount      decimal.Decimal
	ProductDiscountID    int64 // 0 = none
	SubscriptionDiscount decimal.Decimal
	Taxable              decimal.Decimal
	TaxAmount            decimal.Decimal
	Total                decimal.Decimal
}

type AppliedDiscount struct {
	DiscountID int64
	Name       string
	Scope      models.DiscountScope
	Amount     decimal.Decimal
	LineIndex  int // product discounts: which line; -1 for subscription-wide
}

type PriceResult struct {
	Lines         []PricedLine
	Subtotal      decimal.Decimal
	DiscountTotal decimal.Decimal
	TaxTotal      decimal.Decimal
	Total         decimal.Decimal
	Applied       []AppliedDiscount
}

func round2(d decimal.Decimal) decimal.Decimal { return d.Round(2) } // half away from zero

// discountAmount is what d takes off base (never more than base).
func discountAmount(d PriceDiscount, base decimal.Decimal) decimal.Decimal {
	var amt decimal.Decimal
	if d.Type == models.DiscountPercentage {
		amt = round2(base.Mul(d.Value).Div(hundred))
	} else {
		amt = d.Value
	}
	if amt.GreaterThan(base) {
		amt = base
	}
	if amt.IsNegative() {
		amt = decimal.Zero
	}
	return amt
}

func meetsMinimums(d PriceDiscount, amount decimal.Decimal, qty int) bool {
	if d.MinPurchase != nil && amount.LessThan(*d.MinPurchase) {
		return false
	}
	if d.MinQuantity != nil && qty < *d.MinQuantity {
		return false
	}
	return true
}

// best picks the largest discount; ties go to the lower id so results are stable.
func best(cands []PriceDiscount, base decimal.Decimal) (PriceDiscount, decimal.Decimal, bool) {
	var pick PriceDiscount
	amt, found := decimal.Zero, false
	for _, d := range cands {
		a := discountAmount(d, base)
		if a.IsZero() {
			continue
		}
		if !found || a.GreaterThan(amt) || (a.Equal(amt) && d.ID < pick.ID) {
			pick, amt, found = d, a, true
		}
	}
	return pick, amt, found
}

func Price(lines []PriceLine, discounts []PriceDiscount) PriceResult {
	res := PriceResult{Lines: make([]PricedLine, len(lines))}
	totalQty := 0
	for i, l := range lines {
		sub := round2(l.UnitPrice.Mul(decimal.NewFromInt(int64(l.Quantity))))
		res.Lines[i] = PricedLine{PriceLine: l, Subtotal: sub}
		res.Subtotal = res.Subtotal.Add(sub)
		totalQty += l.Quantity
	}

	// 2. product discounts
	for i := range res.Lines {
		ln := &res.Lines[i]
		var cands []PriceDiscount
		for _, d := range discounts {
			if d.Scope == models.ScopeProduct && d.ProductIDs[ln.ProductID] && meetsMinimums(d, ln.Subtotal, ln.Quantity) {
				cands = append(cands, d)
			}
		}
		if d, amt, ok := best(cands, ln.Subtotal); ok {
			ln.ProductDiscount, ln.ProductDiscountID = amt, d.ID
			res.Applied = append(res.Applied, AppliedDiscount{DiscountID: d.ID, Name: d.Name, Scope: d.Scope, Amount: amt, LineIndex: i})
		}
	}

	// 3. subscription-wide discount on what's left
	base := decimal.Zero
	for _, ln := range res.Lines {
		base = base.Add(ln.Subtotal.Sub(ln.ProductDiscount))
	}
	var cands []PriceDiscount
	for _, d := range discounts {
		if d.Scope == models.ScopeSubscription && meetsMinimums(d, res.Subtotal, totalQty) {
			cands = append(cands, d)
		}
	}
	if d, amt, ok := best(cands, base); ok && base.IsPositive() {
		spread(res.Lines, amt, base)
		res.Applied = append(res.Applied, AppliedDiscount{DiscountID: d.ID, Name: d.Name, Scope: d.Scope, Amount: amt, LineIndex: -1})
	}

	// 4–5. tax and totals
	for i := range res.Lines {
		ln := &res.Lines[i]
		ln.Taxable = ln.Subtotal.Sub(ln.ProductDiscount).Sub(ln.SubscriptionDiscount)
		ln.TaxAmount = round2(ln.Taxable.Mul(ln.TaxRate).Div(hundred))
		ln.Total = ln.Taxable.Add(ln.TaxAmount)
		res.DiscountTotal = res.DiscountTotal.Add(ln.ProductDiscount).Add(ln.SubscriptionDiscount)
		res.TaxTotal = res.TaxTotal.Add(ln.TaxAmount)
		res.Total = res.Total.Add(ln.Total)
	}
	return res
}

// spread divides a subscription discount across lines in proportion to each
// line's remaining value. Rounding leftovers go to the largest lines first, so
// the parts always add up to exactly amt and no line goes below zero.
func spread(lines []PricedLine, amt, base decimal.Decimal) {
	given := decimal.Zero
	for i := range lines {
		remaining := lines[i].Subtotal.Sub(lines[i].ProductDiscount)
		share := amt.Mul(remaining).Div(base).RoundDown(2)
		lines[i].SubscriptionDiscount = share
		given = given.Add(share)
	}
	left := amt.Sub(given)
	if !left.IsPositive() {
		return
	}
	order := make([]int, len(lines))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		return lines[order[a]].Subtotal.GreaterThan(lines[order[b]].Subtotal)
	})
	cent := decimal.New(1, -2)
	for left.IsPositive() {
		moved := false
		for _, i := range order {
			room := lines[i].Subtotal.Sub(lines[i].ProductDiscount).Sub(lines[i].SubscriptionDiscount)
			if left.IsPositive() && room.GreaterThanOrEqual(cent) {
				lines[i].SubscriptionDiscount = lines[i].SubscriptionDiscount.Add(cent)
				left = left.Sub(cent)
				moved = true
			}
		}
		if !moved {
			return
		}
	}
}
