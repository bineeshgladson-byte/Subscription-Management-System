package service

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// NextBillingDate returns the start of the period after the one starting at cur.
//
// Do NOT use t.AddDate(0, 1, 0) for months: Go turns 31 Jan + 1 month into 3 Mar.
// We keep the subscription's anchor day (e.g. 31) and clamp to the month's last day,
// so 31 Jan → 28 Feb → 31 Mar → 30 Apr.
func NextBillingDate(cur time.Time, period models.BillingPeriod, interval, anchorDay int) time.Time {
	if interval < 1 {
		interval = 1
	}
	switch period {
	case models.PeriodDaily:
		return cur.AddDate(0, 0, interval)
	case models.PeriodWeekly:
		return cur.AddDate(0, 0, 7*interval)
	case models.PeriodMonthly:
		return addMonthsClamped(cur, interval, anchorDay)
	default: // yearly
		return addMonthsClamped(cur, 12*interval, anchorDay)
	}
}

func addMonthsClamped(t time.Time, months, anchorDay int) time.Time {
	first := time.Date(t.Year(), t.Month()+time.Month(months), 1, 0, 0, 0, 0, t.Location())
	last := first.AddDate(0, 1, -1).Day()
	return time.Date(first.Year(), first.Month(), min(anchorDay, last), 0, 0, 0, 0, t.Location())
}
