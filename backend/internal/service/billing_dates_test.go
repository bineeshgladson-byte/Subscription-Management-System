package service

import (
	"testing"
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

func d(s string) time.Time {
	t, _ := time.Parse("2006-01-02", s)
	return t
}

func TestNextBillingDate(t *testing.T) {
	cases := []struct {
		name     string
		cur      string
		period   models.BillingPeriod
		interval int
		anchor   int
		want     string
	}{
		{"monthly simple", "2026-11-01", models.PeriodMonthly, 1, 1, "2026-12-01"},
		{"monthly year rollover", "2026-12-01", models.PeriodMonthly, 1, 1, "2027-01-01"},
		{"31 Jan clamps to Feb", "2027-01-31", models.PeriodMonthly, 1, 31, "2027-02-28"},
		{"Feb returns to 31 Mar", "2027-02-28", models.PeriodMonthly, 1, 31, "2027-03-31"},
		{"leap year", "2028-01-31", models.PeriodMonthly, 1, 31, "2028-02-29"},
		{"quarterly", "2026-11-30", models.PeriodMonthly, 3, 30, "2027-02-28"},
		{"yearly leap day", "2028-02-29", models.PeriodYearly, 1, 29, "2029-02-28"},
		{"weekly", "2026-11-01", models.PeriodWeekly, 1, 1, "2026-11-08"},
		{"daily x2", "2026-11-30", models.PeriodDaily, 2, 30, "2026-12-02"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := NextBillingDate(d(c.cur), c.period, c.interval, c.anchor).Format("2006-01-02")
			if got != c.want {
				t.Errorf("got %s, want %s", got, c.want)
			}
		})
	}
}
