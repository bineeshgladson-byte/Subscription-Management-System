// Package jobs runs background work on a timer. Jobs only CALL services; all
// billing logic lives in service/ so "Bill now" and the cron share one code path.
package jobs

import (
	"context"
	"log/slog"

	"github.com/robfig/cron/v3"

	"github.com/brightloop/subscriptions/backend/pkg/clock"
)

type Scheduler struct {
	cron  *cron.Cron
	clock clock.Clock
}

func NewScheduler(clk clock.Clock) *Scheduler {
	return &Scheduler{cron: cron.New(cron.WithLocation(clk.Now().Location())), clock: clk}
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
	// TODO(Phase 6): call BillingService.RunDue(ctx, s.clock.Today()).
	// See the "Automatic billing" section of the architecture plan.
	slog.Info("billing job tick (not implemented yet)", "as_of", s.clock.Today().Format("2006-01-02"))
}

func (s *Scheduler) runLifecycle() {
	// TODO(Phase 5/6): activate confirmed subscriptions whose start_date <= today,
	// close expired quotations and subscriptions past expiry_date.
	slog.Info("lifecycle job tick (not implemented yet)")
}
