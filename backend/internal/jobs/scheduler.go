// Package jobs runs background work on a timer. Jobs only CALL services; all
// billing logic lives in service/ so "Bill now" and the cron share one code path.
package jobs

import (
	"context"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/brightloop/subscriptions/backend/pkg/clock"
)

// BillingFunc runs one billing pass (SalesService.RunBilling).
type BillingFunc func(ctx context.Context, asOf time.Time)

type Scheduler struct {
	cron    *cron.Cron
	clock   clock.Clock
	billing BillingFunc
}

func NewScheduler(clk clock.Clock, billing BillingFunc) *Scheduler {
	return &Scheduler{cron: cron.New(cron.WithLocation(clk.Now().Location())), clock: clk, billing: billing}
}

// Start registers jobs. billingSpec comes from BILLING_CRON (e.g. "@every 1h").
func (s *Scheduler) Start(billingSpec string) error {
	if _, err := s.cron.AddFunc(billingSpec, s.runBilling); err != nil {
		return err
	}
	// Daily lifecycle job: activate confirmed subs, expire quotations, close expired subs.
	if _, err := s.cron.AddFunc("15 0 * * *", s.runLifecycle); err != nil {
		return err
	}
	s.cron.Start()
	go s.runBilling() // also run once at startup to catch up after downtime
	return nil
}

func (s *Scheduler) Stop(ctx context.Context) {
	select {
	case <-s.cron.Stop().Done():
	case <-ctx.Done():
	}
}

func (s *Scheduler) runBilling() {
	t := s.clock.Today()
	asOf := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if s.billing == nil {
		slog.Info("billing job tick (no billing service wired)")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	s.billing(ctx, asOf) // activates due subscriptions, closes expired quotations, invoices every due period
}

// runLifecycle: the billing pass already activates and closes; run it daily too.
func (s *Scheduler) runLifecycle() { s.runBilling() }
