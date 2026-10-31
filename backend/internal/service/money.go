package service

import (
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// maxMoney is the largest value NUMERIC(12,2) can hold.
var maxMoney = decimal.RequireFromString("9999999999.99")

// parseMoney accepts "999", "999.5", "999.50": at most 2 decimals and within
// NUMERIC(12,2). Negative values only when allowNegative (e.g. a cheaper variant).
// Returns the value rounded to 2 dp, or a reason for the field error.
func parseMoney(raw string, allowNegative bool) (decimal.Decimal, string) {
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	switch {
	case err != nil:
		return d, `must be an amount, e.g. "999.00"`
	case d.IsNegative() && !allowNegative:
		return d, "can't be negative"
	case d.Abs().GreaterThan(maxMoney):
		return d, "is too large"
	case !d.Equal(d.Round(2)):
		return d, "can have at most 2 decimal places"
	}
	return d.Round(2), ""
}

// parseDate reads "YYYY-MM-DD". Dates are stored as UTC midnight so the
// DATE column never shifts a day because of time zones.
func parseDate(raw string) (time.Time, bool) {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	return t, err == nil
}

func dateString(t time.Time) string { return t.Format("2006-01-02") }
