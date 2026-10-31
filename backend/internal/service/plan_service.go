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

// PlanService — A4. Brief: "what it costs and how often it bills (daily,
// weekly, monthly or yearly), the minimum quantity, and the dates it is
// available. Some plans should close on their own when they end, and others can
// be closed early, paused or renewed. The admin decides which, plan by plan."
//
// The plan price is NOT typed in: it's the sum of its product lines, kept in
// sync by repository.recalcSQL whenever lines or product prices change.
type PlanService struct {
	db       *gorm.DB
	plans    *repository.PlanRepo
	products *repository.ProductRepo
	clock    clock.Clock
}

func NewPlanService(db *gorm.DB, plans *repository.PlanRepo, products *repository.ProductRepo, clk clock.Clock) *PlanService {
	return &PlanService{db: db, plans: plans, products: products, clock: clk}
}

// List: customers only ever see active plans that are available today.
func (s *PlanService) List(ctx context.Context, role models.Role, f repository.PlanFilter, p pagination.Params) ([]dto.PlanResponse, int64, error) {
	if role == models.RoleCustomer {
		today := s.clock.Today()
		f = repository.PlanFilter{ProductID: f.ProductID, Search: f.Search, BillingPeriod: f.BillingPeriod, Status: "active", AvailableOn: &today}
	}
	db := s.db.WithContext(ctx)
	list, total, err := s.plans.List(db, f, p)
	if err != nil {
		return nil, 0, err
	}
	var counts map[int64]int64
	if role != models.RoleCustomer {
		ids := make([]int64, len(list))
		for i := range list {
			ids[i] = list[i].ID
		}
		if counts, err = s.plans.ActiveSubscriptions(db, ids); err != nil {
			return nil, 0, err
		}
	}
	out := make([]dto.PlanResponse, len(list))
	for i := range list {
		out[i] = s.response(&list[i], role, counts)
	}
	return out, total, nil
}

func (s *PlanService) Get(ctx context.Context, role models.Role, id int64) (*dto.PlanResponse, error) {
	db := s.db.WithContext(ctx)
	p, err := s.plans.FindByID(db, id)
	if err != nil {
		return nil, err
	}
	// Customers get 404 for plans they can't buy (archived / outside availability).
	if p == nil || (role == models.RoleCustomer && !(p.Status == models.StatusActive && s.availableOn(p, s.clock.Today()))) {
		return nil, apperr.NotFound("plan")
	}
	var counts map[int64]int64
	if role != models.RoleCustomer {
		if counts, err = s.plans.ActiveSubscriptions(db, []int64{id}); err != nil {
			return nil, err
		}
	}
	out := s.response(p, role, counts)
	return &out, nil
}

func (s *PlanService) Create(ctx context.Context, in dto.CreatePlanRequest) (*dto.PlanResponse, error) {
	errs := map[string]string{}
	interval := defaultInt(in.BillingInterval, 1)
	minQty := defaultInt(in.MinQuantity, 1)

	from := s.clock.Today()
	from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	if in.AvailableFrom != "" {
		d, ok := parseDate(in.AvailableFrom)
		if !ok {
			errs["available_from"] = "must be a date like 2026-11-01"
		}
		from = d
	}
	var until *time.Time
	if in.AvailableUntil != "" {
		d, ok := parseDate(in.AvailableUntil)
		if !ok {
			errs["available_until"] = "must be a date like 2026-12-31"
		} else {
			until = &d
		}
	}
	if until != nil && until.Before(from) {
		errs["available_until"] = "must be on or after available_from"
	}
	cycles := in.AutoCloseAfterCycles
	if msg := checkAutoClose(in.AutoCloseEnabled, cycles); msg != "" {
		errs["auto_close_after_cycles"] = msg
	}
	if !in.AutoCloseEnabled {
		cycles = nil
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}

	db := s.db.WithContext(ctx)
	name := strings.TrimSpace(in.Name)
	if taken, err := s.plans.NameTaken(db, name, 0); err != nil {
		return nil, err
	} else if taken {
		return nil, planNameTaken()
	}
	lines, err := s.buildLines(db, in.Products)
	if err != nil {
		return nil, err
	}

	plan := &models.SubscriptionPlan{
		Name: name, Description: optional(strings.TrimSpace(in.Description)),
		BillingPeriod: models.BillingPeriod(in.BillingPeriod), BillingInterval: interval, MinQuantity: minQty,
		AvailableFrom: from, AvailableUntil: until, AutoCloseEnabled: in.AutoCloseEnabled, AutoCloseAfterCycles: cycles,
		IsPausable: boolOr(in.IsPausable, false), IsClosable: boolOr(in.IsClosable, true),
		IsRenewable: boolOr(in.IsRenewable, true), Status: models.StatusActive,
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := s.plans.Create(tx, plan); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return planNameTaken()
			}
			return err
		}
		if err := s.plans.ReplaceLines(tx, plan.ID, lines); err != nil {
			return err
		}
		return s.plans.RecalculatePrice(tx, plan.ID)
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, models.RoleAdmin, plan.ID)
}

func (s *PlanService) Update(ctx context.Context, id int64, in dto.UpdatePlanRequest) (*dto.PlanResponse, error) {
	db := s.db.WithContext(ctx)
	p, err := s.plans.FindByID(db, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperr.NotFound("plan")
	}

	fields := map[string]any{}
	errs := map[string]string{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if taken, err := s.plans.NameTaken(db, name, id); err != nil {
			return nil, err
		} else if taken {
			return nil, planNameTaken()
		}
		fields["name"] = name
	}
	if in.Description != nil {
		fields["description"] = nullIfEmpty(strings.TrimSpace(*in.Description))
	}

	// Changing how often a plan bills would break billing for people already on it.
	billingChange := (in.BillingPeriod != nil && models.BillingPeriod(*in.BillingPeriod) != p.BillingPeriod) ||
		(in.BillingInterval != nil && *in.BillingInterval != p.BillingInterval)
	if billingChange {
		counts, err := s.plans.ActiveSubscriptions(db, []int64{id})
		if err != nil {
			return nil, err
		}
		if n := counts[id]; n > 0 {
			e := apperr.Rule("PLAN_IN_USE", fmt.Sprintf(
				"%d subscription(s) are on this plan; create a new plan instead of changing how often it bills", n))
			e.Details["active_subscriptions"] = fmt.Sprint(n)
			return nil, e
		}
		if in.BillingPeriod != nil {
			fields["billing_period"] = *in.BillingPeriod
		}
		if in.BillingInterval != nil {
			fields["billing_interval"] = *in.BillingInterval
		}
	}
	if in.MinQuantity != nil {
		fields["min_quantity"] = *in.MinQuantity
	}

	from, until := p.AvailableFrom, p.AvailableUntil
	if in.AvailableFrom != nil {
		d, ok := parseDate(*in.AvailableFrom)
		if !ok {
			errs["available_from"] = "must be a date like 2026-11-01"
		} else {
			from = d
			fields["available_from"] = d
		}
	}
	if in.AvailableUntil != nil {
		if *in.AvailableUntil == "" {
			until = nil
			fields["available_until"] = nil
		} else if d, ok := parseDate(*in.AvailableUntil); !ok {
			errs["available_until"] = "must be a date like 2026-12-31"
		} else {
			until = &d
			fields["available_until"] = d
		}
	}
	if until != nil && until.Before(from) {
		errs["available_until"] = "must be on or after available_from"
	}

	enabled, cycles := p.AutoCloseEnabled, p.AutoCloseAfterCycles
	if in.AutoCloseEnabled != nil {
		enabled = *in.AutoCloseEnabled
		fields["auto_close_enabled"] = enabled
	}
	if in.AutoCloseAfterCycles != nil {
		if *in.AutoCloseAfterCycles == 0 {
			cycles = nil
		} else {
			cycles = in.AutoCloseAfterCycles
		}
	}
	if in.AutoCloseEnabled != nil || in.AutoCloseAfterCycles != nil {
		if msg := checkAutoClose(enabled, cycles); msg != "" {
			errs["auto_close_after_cycles"] = msg
		}
		if !enabled {
			cycles = nil
		}
		fields["auto_close_after_cycles"] = cycles
	}
	for key, val := range map[string]*bool{"is_pausable": in.IsPausable, "is_closable": in.IsClosable, "is_renewable": in.IsRenewable} {
		if val != nil {
			fields[key] = *val
		}
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}

	var lines []models.PlanProduct
	if in.Products != nil {
		if lines, err = s.buildLines(db, in.Products); err != nil {
			return nil, err
		}
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if len(fields) > 0 {
			if err := s.plans.Update(tx, id, fields); err != nil {
				if errors.Is(err, gorm.ErrDuplicatedKey) {
					return planNameTaken()
				}
				return err
			}
		}
		if in.Products != nil {
			if err := s.plans.ReplaceLines(tx, id, lines); err != nil {
				return err
			}
			return s.plans.RecalculatePrice(tx, id)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, models.RoleAdmin, id)
}

// SetStatus archives (hide from customers and new quotations; existing
// subscriptions keep billing) or reactivates a plan.
func (s *PlanService) SetStatus(ctx context.Context, id int64, status models.RecordStatus) (*dto.PlanResponse, error) {
	db := s.db.WithContext(ctx)
	p, err := s.plans.FindByID(db, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperr.NotFound("plan")
	}
	if status == models.StatusActive {
		if taken, err := s.plans.NameTaken(db, p.Name, id); err != nil {
			return nil, err
		} else if taken {
			return nil, planNameTaken()
		}
		for _, l := range p.Products {
			if l.Product == nil || l.Product.Status != models.StatusActive || (l.Variant != nil && !l.Variant.IsActive) {
				return nil, apperr.Rule("PLAN_HAS_INACTIVE_PRODUCTS",
					"this plan includes archived products or inactive variants; edit its products before reactivating")
			}
		}
	}
	if err := s.plans.Update(db, id, map[string]any{"status": status}); err != nil {
		return nil, err
	}
	return s.Get(ctx, models.RoleAdmin, id)
}

// buildLines validates plan lines (see resolveLines) and turns them into rows.
func (s *PlanService) buildLines(db *gorm.DB, in []dto.PlanLineInput) ([]models.PlanProduct, error) {
	specs := make([]lineSpec, len(in))
	for i, l := range in {
		specs[i] = lineSpec{ProductID: l.ProductID, VariantID: l.VariantID, Quantity: l.Quantity, PriceOverride: l.PriceOverride}
	}
	resolved, err := resolveLines(db, s.products, "products", specs, true)
	if err != nil {
		return nil, err
	}
	lines := make([]models.PlanProduct, len(resolved))
	for i, r := range resolved {
		lines[i] = models.PlanProduct{ProductID: r.ProductID, VariantID: r.VariantID, Quantity: r.Quantity, PriceOverride: r.PriceOverride}
	}
	return lines, nil
}

func (s *PlanService) availableOn(p *models.SubscriptionPlan, day time.Time) bool {
	d := dateString(day)
	return dateString(p.AvailableFrom) <= d && (p.AvailableUntil == nil || dateString(*p.AvailableUntil) >= d)
}

func (s *PlanService) response(p *models.SubscriptionPlan, role models.Role, counts map[int64]int64) dto.PlanResponse {
	r := dto.PlanResponse{
		ID: p.ID, Name: p.Name, Description: p.Description, Price: p.Price.StringFixed(2),
		BillingPeriod: p.BillingPeriod, BillingInterval: p.BillingInterval,
		BillingLabel: billingLabel(p.BillingPeriod, p.BillingInterval), MinQuantity: p.MinQuantity,
		AvailableFrom: dateString(p.AvailableFrom), IsAvailableNow: p.Status == models.StatusActive && s.availableOn(p, s.clock.Today()),
		AutoCloseEnabled: p.AutoCloseEnabled, AutoCloseAfterCycles: p.AutoCloseAfterCycles,
		IsPausable: p.IsPausable, IsClosable: p.IsClosable, IsRenewable: p.IsRenewable, Status: p.Status,
		Products: make([]dto.PlanLineResponse, 0, len(p.Products)), CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
	}
	if p.AvailableUntil != nil {
		u := dateString(*p.AvailableUntil)
		r.AvailableUntil = &u
	}
	for _, l := range p.Products {
		lr := dto.PlanLineResponse{ID: l.ID, ProductID: l.ProductID, VariantID: l.VariantID, Quantity: l.Quantity}
		unit := decimal.Zero
		if l.Product != nil {
			lr.ProductName = l.Product.Name
			unit = l.Product.SalesPrice
		}
		if l.Variant != nil {
			name := l.Variant.Name
			lr.VariantName = &name
			unit = unit.Add(l.Variant.ExtraPrice)
		}
		if l.PriceOverride != nil {
			o := l.PriceOverride.StringFixed(2)
			lr.PriceOverride = &o
			unit = *l.PriceOverride
		}
		lr.UnitPrice = unit.StringFixed(2)
		lr.LineTotal = unit.Mul(decimal.NewFromInt(int64(l.Quantity))).StringFixed(2)
		r.Products = append(r.Products, lr)
	}
	if role != models.RoleCustomer && counts != nil {
		n := counts[p.ID]
		r.ActiveSubscriptions = &n
	}
	return r
}

// billingLabel: "Monthly", "Every 3 months", "Yearly", "Every 2 weeks".
func billingLabel(period models.BillingPeriod, interval int) string {
	single := map[models.BillingPeriod]string{models.PeriodDaily: "Daily", models.PeriodWeekly: "Weekly",
		models.PeriodMonthly: "Monthly", models.PeriodYearly: "Yearly"}
	unit := map[models.BillingPeriod]string{models.PeriodDaily: "days", models.PeriodWeekly: "weeks",
		models.PeriodMonthly: "months", models.PeriodYearly: "years"}
	if interval <= 1 {
		return single[period]
	}
	return fmt.Sprintf("Every %d %s", interval, unit[period])
}

func checkAutoClose(enabled bool, cycles *int) string {
	if enabled && (cycles == nil || *cycles < 1) {
		return "is required (1 or more) when auto_close_enabled is true"
	}
	return ""
}

func planNameTaken() error {
	e := apperr.Conflict("a plan with this name already exists")
	e.Details = map[string]string{"name": "is already used by another plan"}
	return e
}

func defaultInt(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

func boolOr(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}

func derefOr(v *int64, def int64) int64 {
	if v == nil {
		return def
	}
	return *v
}
