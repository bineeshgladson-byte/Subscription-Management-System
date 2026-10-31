package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/brand"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// SalesService covers A7 quotations → A8 subscriptions → A9 automatic invoices,
// plus invoice actions, the payments list and the dashboard (A11).
//
// Lifecycle: draft → quotation (sent) → confirmed → active ⇄ paused → closed / renewed.
// Billing is "in advance": each period is invoiced on its first day.
// Duplicate protection: row lock (FOR UPDATE SKIP LOCKED) + re-check + invoice and
// next_billing_date in ONE transaction + unique (subscription_id, period_start) index.
type SalesService struct {
	db          *gorm.DB
	products    *repository.ProductRepo
	discounts   *repository.DiscountRepo
	clock       clock.Clock
	mailer      mailer.Mailer
	autoConfirm bool
}

func NewSalesService(db *gorm.DB, products *repository.ProductRepo, discounts *repository.DiscountRepo,
	clk clock.Clock, m mailer.Mailer, autoConfirm bool) *SalesService {
	return &SalesService{db: db, products: products, discounts: discounts, clock: clk, mailer: m, autoConfirm: autoConfirm}
}

func (s *SalesService) today() time.Time {
	t := s.clock.Today()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func number(tx *gorm.DB, seq, prefix string, year int) (string, error) {
	var n int64
	if err := tx.Raw("SELECT nextval(?::regclass)", seq).Scan(&n).Error; err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%d/%05d", prefix, year, n), nil
}

// ======================= quotations =======================

func (s *SalesService) CreateQuotation(ctx context.Context, actorID int64, in dto.CreateQuotationRequest) (*dto.SubscriptionResponse, error) {
	db := s.db.WithContext(ctx)
	errs := map[string]string{}
	today := s.today()
	start := today
	if in.StartDate != "" {
		d, ok := parseDate(in.StartDate)
		if !ok {
			errs["start_date"] = "must be a date like 2026-11-01"
		}
		start = d
	}
	var expiry *time.Time
	if in.ExpiryDate != "" {
		if d, ok := parseDate(in.ExpiryDate); !ok {
			errs["expiry_date"] = "must be a date like 2027-10-31"
		} else if d.Before(start) {
			errs["expiry_date"] = "must be on or after the start date"
		} else {
			expiry = &d
		}
	}
	if len(errs) > 0 {
		return nil, apperr.Validation("Some fields are invalid", errs)
	}

	var cust models.Customer
	if err := db.First(&cust, in.CustomerID).Error; err != nil {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"customer_id": "must be an existing customer"})
	}
	var plan models.SubscriptionPlan
	if err := db.Preload("Products").First(&plan, in.PlanID).Error; err != nil || plan.Status != models.StatusActive {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"plan_id": "must be an active plan"})
	}
	if dateString(plan.AvailableFrom) > dateString(start) || (plan.AvailableUntil != nil && dateString(*plan.AvailableUntil) < dateString(start)) {
		return nil, apperr.Rule("PLAN_NOT_AVAILABLE", "this plan isn't available on the start date")
	}

	validity, terms := 15, 15
	var specs []lineSpec
	if in.TemplateID != nil {
		var tpl models.QuotationTemplate
		if err := db.Preload("Lines").First(&tpl, *in.TemplateID).Error; err != nil || !tpl.IsActive {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"template_id": "must be an active template"})
		}
		validity, terms = tpl.ValidityDays, tpl.PaymentTermsDays
		for _, l := range tpl.Lines {
			specs = append(specs, lineSpec{ProductID: l.ProductID, VariantID: l.VariantID, Quantity: l.Quantity})
		}
	}
	if in.PaymentTermsDays != nil {
		terms = *in.PaymentTermsDays
	}
	if len(in.Items) > 0 {
		specs = specs[:0]
		for _, it := range in.Items {
			specs = append(specs, lineSpec{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity, PriceOverride: it.UnitPrice})
		}
	}
	if len(specs) == 0 {
		for _, pp := range plan.Products {
			o := ""
			if pp.PriceOverride != nil {
				o = pp.PriceOverride.StringFixed(2)
			}
			specs = append(specs, lineSpec{ProductID: pp.ProductID, VariantID: pp.VariantID, Quantity: pp.Quantity, PriceOverride: o})
		}
	}
	lines, err := resolveLines(db, s.products, "items", specs, true)
	if err != nil {
		return nil, err
	}
	qty := 0
	for _, l := range lines {
		qty += l.Quantity
	}
	if qty < plan.MinQuantity {
		return nil, apperr.Rule("MIN_QUANTITY", fmt.Sprintf("this plan needs at least %d units in total (you have %d)", plan.MinQuantity, qty))
	}
	validUntil := today.AddDate(0, 0, validity)
	if in.QuotationValidUntil != "" {
		d, ok := parseDate(in.QuotationValidUntil)
		if !ok {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"quotation_valid_until": "must be a date"})
		}
		validUntil = d
	}

	price, err := s.price(db, lines, start)
	if err != nil {
		return nil, err
	}
	sub := &models.Subscription{CustomerID: cust.ID, PlanID: plan.ID, TemplateID: in.TemplateID, Status: models.SubDraft,
		StartDate: start, ExpiryDate: expiry, QuotationValidUntil: &validUntil, PaymentTermsDays: terms,
		BillingAnchorDay: start.Day(), Subtotal: price.Subtotal, DiscountTotal: price.DiscountTotal,
		TaxTotal: price.TaxTotal, Total: price.Total, CreatedBy: &actorID}
	err = db.Transaction(func(tx *gorm.DB) error {
		n, err := number(tx, "quotation_no_seq", "QT", today.Year())
		if err != nil {
			return err
		}
		sub.QuotationNumber = n
		if err := tx.Omit(clause.Associations).Create(sub).Error; err != nil {
			return err
		}
		for _, l := range lines {
			item := models.SubscriptionItem{SubscriptionID: sub.ID, ProductID: l.ProductID, VariantID: l.VariantID,
				Description: describe(l), Quantity: l.Quantity, UnitPrice: l.UnitPrice(), TaxID: l.Product.TaxID,
				TaxRate: l.TaxRate(), LineSubtotal: round2(l.UnitPrice().Mul(decimal.NewFromInt(int64(l.Quantity))))}
			if err := tx.Omit(clause.Associations).Create(&item).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.GetSubscription(ctx, sub.ID)
}

func describe(l resolvedLine) string {
	if l.Variant != nil {
		return l.Product.Name + " (" + l.Variant.Name + ")"
	}
	return l.Product.Name
}

func (s *SalesService) price(db *gorm.DB, lines []resolvedLine, day time.Time) (PriceResult, error) {
	eligible, err := s.discounts.Eligible(db, day)
	if err != nil {
		return PriceResult{}, err
	}
	pl := make([]PriceLine, len(lines))
	for i, l := range lines {
		pl[i] = PriceLine{ProductID: l.ProductID, Description: describe(l), Quantity: l.Quantity, UnitPrice: l.UnitPrice(), TaxRate: l.TaxRate()}
	}
	return Price(pl, ToPriceDiscounts(eligible)), nil
}

func (s *SalesService) load(db *gorm.DB, id int64) (*models.Subscription, error) {
	var sub models.Subscription
	err := db.Preload("Customer").Preload("Plan").Preload("Items", func(q *gorm.DB) *gorm.DB { return q.Order("id") }).First(&sub, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, apperr.NotFound("subscription")
	}
	return &sub, err
}

func (s *SalesService) setStatus(ctx context.Context, id int64, from []models.SubscriptionStatus, fields map[string]any, what string) (*dto.SubscriptionResponse, error) {
	sub, err := s.load(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	ok := false
	for _, f := range from {
		ok = ok || sub.Status == f
	}
	if !ok {
		return nil, apperr.InvalidState(fmt.Sprintf("can't %s a subscription that is %s", what, sub.Status))
	}
	if err := s.db.WithContext(ctx).Model(&models.Subscription{}).Where("id = ?", id).Updates(fields).Error; err != nil {
		return nil, err
	}
	return s.GetSubscription(ctx, id)
}

// SendQuotation: draft → quotation (and emails the customer).
func (s *SalesService) SendQuotation(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	out, err := s.setStatus(ctx, id, []models.SubscriptionStatus{models.SubDraft, models.SubQuotation},
		map[string]any{"status": models.SubQuotation}, "send")
	if err == nil {
		s.notify(out.CustomerID, "Your "+brand.Name+" quotation "+out.QuotationNumber,
			fmt.Sprintf("Here is your quotation %s for %s: %s per billing period (%s), valid until %s.",
				out.QuotationNumber, out.PlanName, out.Total, out.BillingLabel, deref(out.QuotationValidUntil)))
	}
	return out, err
}

func (s *SalesService) CancelQuotation(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	return s.setStatus(ctx, id, []models.SubscriptionStatus{models.SubDraft, models.SubQuotation},
		map[string]any{"status": models.SubClosed, "close_reason": "cancelled", "closed_at": s.clock.Now()}, "cancel")
}

// ConfirmQuotation: → confirmed with a subscription number; activates and bills
// the first period straight away if the start date has arrived.
func (s *SalesService) ConfirmQuotation(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	db := s.db.WithContext(ctx)
	sub, err := s.load(db, id)
	if err != nil {
		return nil, err
	}
	if sub.Status != models.SubDraft && sub.Status != models.SubQuotation {
		return nil, apperr.InvalidState("only draft or sent quotations can be confirmed")
	}
	if sub.QuotationValidUntil != nil && dateString(*sub.QuotationValidUntil) < dateString(s.today()) {
		return nil, apperr.Rule("QUOTATION_EXPIRED", "this quotation expired on "+dateString(*sub.QuotationValidUntil))
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		n, err := number(tx, "subscription_no_seq", "SUB", s.today().Year())
		if err != nil {
			return err
		}
		return tx.Model(&models.Subscription{}).Where("id = ?", id).Updates(map[string]any{
			"status": models.SubConfirmed, "subscription_number": n, "confirmed_at": s.clock.Now()}).Error
	})
	if err != nil {
		return nil, err
	}
	if dateString(sub.StartDate) <= dateString(s.today()) {
		if _, err := s.activate(ctx, id); err != nil {
			return nil, err
		}
		if _, err := s.billDue(ctx, id, s.today()); err != nil {
			return nil, err
		}
	}
	return s.GetSubscription(ctx, id)
}

func (s *SalesService) activate(ctx context.Context, id int64) (bool, error) {
	res := s.db.WithContext(ctx).Exec(`UPDATE subscriptions SET status = 'active', next_billing_date = start_date,
		billing_anchor_day = EXTRACT(DAY FROM start_date)::int, activated_at = now(), updated_at = now()
		WHERE id = ? AND status = 'confirmed'`, id)
	return res.RowsAffected == 1, res.Error
}

// ======================= subscription lifecycle =======================

func (s *SalesService) Pause(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	sub, err := s.load(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if sub.Plan != nil && !sub.Plan.IsPausable {
		return nil, apperr.Rule("PLAN_NOT_PAUSABLE", "this plan doesn't allow pausing")
	}
	return s.setStatus(ctx, id, []models.SubscriptionStatus{models.SubActive},
		map[string]any{"status": models.SubPaused, "paused_at": s.clock.Now()}, "pause")
}

// Resume starts a fresh billing cycle today (no charge for the paused time).
func (s *SalesService) Resume(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	t := s.today()
	out, err := s.setStatus(ctx, id, []models.SubscriptionStatus{models.SubPaused},
		map[string]any{"status": models.SubActive, "paused_at": nil, "next_billing_date": t, "billing_anchor_day": t.Day()}, "resume")
	if err == nil {
		_, _ = s.billDue(ctx, id, t)
		return s.GetSubscription(ctx, id)
	}
	return out, err
}

func (s *SalesService) Close(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	sub, err := s.load(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if sub.Plan != nil && !sub.Plan.IsClosable {
		return nil, apperr.Rule("PLAN_NOT_CLOSABLE", "this plan doesn't allow closing early")
	}
	return s.setStatus(ctx, id, []models.SubscriptionStatus{models.SubConfirmed, models.SubActive, models.SubPaused},
		map[string]any{"status": models.SubClosed, "close_reason": "closed_early", "closed_at": s.clock.Now()}, "close")
}

// Renew creates a new draft quotation that starts the day after this one
// ends (or today), with the same lines; this one becomes "renewed".
func (s *SalesService) Renew(ctx context.Context, actorID int64, id int64) (*dto.SubscriptionResponse, error) {
	sub, err := s.load(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if sub.Plan != nil && !sub.Plan.IsRenewable {
		return nil, apperr.Rule("PLAN_NOT_RENEWABLE", "this plan can't be renewed")
	}
	if sub.Status != models.SubActive && sub.Status != models.SubPaused && sub.Status != models.SubClosed {
		return nil, apperr.InvalidState("only active, paused or closed subscriptions can be renewed")
	}
	start := s.today()
	if sub.ExpiryDate != nil && sub.ExpiryDate.After(start) {
		start = sub.ExpiryDate.AddDate(0, 0, 1)
	}
	items := make([]dto.QuotationLineInput, len(sub.Items))
	for i, it := range sub.Items {
		items[i] = dto.QuotationLineInput{ProductID: it.ProductID, VariantID: it.VariantID, Quantity: it.Quantity}
	}
	terms := sub.PaymentTermsDays
	out, err := s.CreateQuotation(ctx, actorID, dto.CreateQuotationRequest{CustomerID: sub.CustomerID, PlanID: sub.PlanID,
		StartDate: dateString(start), PaymentTermsDays: &terms, Items: items})
	if err != nil {
		return nil, err
	}
	s.db.WithContext(ctx).Model(&models.Subscription{}).Where("id = ?", out.ID).Update("parent_subscription_id", id)
	s.db.WithContext(ctx).Model(&models.Subscription{}).Where("id = ?", id).Updates(map[string]any{"status": models.SubRenewed, "closed_at": s.clock.Now()})
	return s.GetSubscription(ctx, out.ID)
}

// BillNow invoices whatever is due for one subscription (same code as the scheduler).
func (s *SalesService) BillNow(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	sub, err := s.load(s.db.WithContext(ctx), id)
	if err != nil {
		return nil, err
	}
	if sub.Status == models.SubConfirmed && dateString(sub.StartDate) <= dateString(s.today()) {
		if _, err := s.activate(ctx, id); err != nil {
			return nil, err
		}
		sub, _ = s.load(s.db.WithContext(ctx), id)
	}
	if sub.Status != models.SubActive {
		return nil, apperr.InvalidState("only active subscriptions are billed")
	}
	n, err := s.billDue(ctx, id, s.today())
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, apperr.Rule("NOTHING_DUE", "nothing is due yet; the next invoice is due on "+deref(dateStringPtr(sub.NextBillingDate)))
	}
	return s.GetSubscription(ctx, id)
}

// ======================= billing engine (A9) =======================

// billDue bills every period that is due up to asOf (catches up after downtime).
func (s *SalesService) billDue(ctx context.Context, id int64, asOf time.Time) (int, error) {
	count := 0
	for i := 0; i < 400; i++ { // safety cap
		billed, err := s.billNextPeriod(ctx, id, asOf)
		if err != nil {
			return count, err
		}
		if !billed {
			break
		}
		count++
	}
	return count, nil
}

func (s *SalesService) billNextPeriod(ctx context.Context, id int64, asOf time.Time) (bool, error) {
	billed := false
	var notifyInvoice int64
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var sub models.Subscription
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("id = ? AND status = 'active' AND next_billing_date <= ?", id, dateString(asOf)).First(&sub).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil // not due, or another worker is billing it right now
		}
		if err != nil {
			return err
		}
		var plan models.SubscriptionPlan
		if err := tx.First(&plan, sub.PlanID).Error; err != nil {
			return err
		}
		start := *sub.NextBillingDate
		// Contract ended: close instead of billing past the expiry date.
		if sub.ExpiryDate != nil && start.After(*sub.ExpiryDate) {
			return tx.Model(&sub).Updates(map[string]any{"status": models.SubClosed, "close_reason": "expired", "closed_at": s.clock.Now()}).Error
		}
		next := NextBillingDate(start, plan.BillingPeriod, plan.BillingInterval, sub.BillingAnchorDay)
		end := next.AddDate(0, 0, -1)

		var items []models.SubscriptionItem
		if err := tx.Where("subscription_id = ?", sub.ID).Order("id").Find(&items).Error; err != nil {
			return err
		}
		var cust models.Customer
		if err := tx.Unscoped().First(&cust, sub.CustomerID).Error; err != nil {
			return err
		}
		lines := make([]PriceLine, len(items))
		for i, it := range items {
			lines[i] = PriceLine{ProductID: it.ProductID, Description: it.Description, Quantity: it.Quantity, UnitPrice: it.UnitPrice, TaxRate: it.TaxRate}
		}
		eligible, err := s.discounts.Eligible(tx, start)
		if err != nil {
			return err
		}
		discs := ToPriceDiscounts(eligible)
		res := Price(lines, discs)
		// Take one use of each applied discount; if one ran out meanwhile, price again without it.
		for attempt := 0; attempt < 3; attempt++ {
			lost := map[int64]bool{}
			for _, a := range uniqueApplied(res.Applied) {
				ok, err := s.discounts.ReserveUse(tx, a)
				if err != nil {
					return err
				}
				if !ok {
					lost[a] = true
				}
			}
			if len(lost) == 0 {
				break
			}
			for _, a := range uniqueApplied(res.Applied) { // give back the uses we did take
				if !lost[a] {
					if err := s.discounts.ReleaseUse(tx, a); err != nil {
						return err
					}
				}
			}
			kept := discs[:0]
			for _, d := range discs {
				if !lost[d.ID] {
					kept = append(kept, d)
				}
			}
			discs = kept
			res = Price(lines, discs)
		}

		status := models.InvoiceDraft
		var confirmedAt *time.Time
		if s.autoConfirm {
			status = models.InvoiceConfirmed
			now := s.clock.Now()
			confirmedAt = &now
		}
		num, err := number(tx, "invoice_no_seq", "INV", start.Year())
		if err != nil {
			return err
		}
		invDate := start
		ps, pe := start, end
		inv := &models.Invoice{InvoiceNumber: num, CustomerID: sub.CustomerID, SubscriptionID: &sub.ID, PeriodStart: &ps, PeriodEnd: &pe,
			InvoiceDate: invDate, DueDate: invDate.AddDate(0, 0, sub.PaymentTermsDays), Status: status,
			Subtotal: res.Subtotal, DiscountTotal: res.DiscountTotal, TaxTotal: res.TaxTotal, Total: res.Total,
			CustomerName: cust.Name, BillingAddress: cust.BillingAddress, CustomerGSTIN: cust.GSTIN, ConfirmedAt: confirmedAt}
		if err := tx.Omit(clause.Associations, "AmountDue").Create(inv).Error; err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return apperr.Conflict("this period was already invoiced")
			}
			return err
		}
		for i, l := range res.Lines {
			it := items[i]
			ii := models.InvoiceItem{InvoiceID: inv.ID, ProductID: &it.ProductID, VariantID: it.VariantID, Description: l.Description,
				Quantity: l.Quantity, UnitPrice: l.UnitPrice, LineSubtotal: l.Subtotal,
				DiscountAmount: l.ProductDiscount.Add(l.SubscriptionDiscount), TaxRate: l.TaxRate, TaxAmount: l.TaxAmount, LineTotal: l.Total}
			if err := tx.Omit(clause.Associations).Create(&ii).Error; err != nil {
				return err
			}
		}
		for _, a := range res.Applied {
			u := models.DiscountUsage{DiscountID: a.DiscountID, InvoiceID: inv.ID, CustomerID: sub.CustomerID, Amount: a.Amount}
			if err := tx.Exec(`INSERT INTO discount_usages (discount_id, invoice_id, customer_id, amount) VALUES (?, ?, ?, ?)
				ON CONFLICT DO NOTHING`, u.DiscountID, u.InvoiceID, u.CustomerID, u.Amount).Error; err != nil {
				return err
			}
		}
		fields := map[string]any{"next_billing_date": next, "cycles_billed": sub.CyclesBilled + 1}
		if plan.AutoCloseEnabled && plan.AutoCloseAfterCycles != nil && sub.CyclesBilled+1 >= *plan.AutoCloseAfterCycles {
			fields["status"], fields["close_reason"], fields["closed_at"] = models.SubClosed, "auto_closed", s.clock.Now()
		}
		if err := tx.Model(&models.Subscription{}).Where("id = ?", sub.ID).Updates(fields).Error; err != nil {
			return err
		}
		billed, notifyInvoice = true, inv.ID
		return nil
	})
	if err == nil && notifyInvoice != 0 && s.autoConfirm {
		go func() { _ = s.emailInvoice(context.Background(), notifyInvoice) }()
	}
	return billed, err
}

func uniqueApplied(a []AppliedDiscount) []int64 {
	seen := map[int64]bool{}
	var out []int64
	for _, x := range a {
		if !seen[x.DiscountID] {
			seen[x.DiscountID] = true
			out = append(out, x.DiscountID)
		}
	}
	return out
}

// RunBilling is the scheduler / admin "Run billing" entry point.
func (s *SalesService) RunBilling(ctx context.Context, asOf time.Time) dto.BillingRunResponse {
	out := dto.BillingRunResponse{AsOf: dateString(asOf), Errors: []string{}}
	db := s.db.WithContext(ctx)
	var confirmed []int64
	db.Model(&models.Subscription{}).Where("status = 'confirmed' AND start_date <= ?", dateString(asOf)).Pluck("id", &confirmed)
	for _, id := range confirmed {
		if ok, err := s.activate(ctx, id); err == nil && ok {
			out.Activated++
		}
	}
	// Quotations that were never accepted in time.
	res := db.Exec(`UPDATE subscriptions SET status = 'closed', close_reason = 'quotation_expired', closed_at = now()
		WHERE status IN ('draft','quotation') AND quotation_valid_until < ?`, dateString(asOf))
	out.Closed += int(res.RowsAffected)

	var due []int64
	db.Model(&models.Subscription{}).Where("status = 'active' AND next_billing_date <= ?", dateString(asOf)).Order("next_billing_date").Pluck("id", &due)
	for _, id := range due {
		n, err := s.billDue(ctx, id, asOf)
		out.InvoicesCreated += n
		if err != nil {
			out.Errors = append(out.Errors, fmt.Sprintf("subscription %d: %v", id, err))
			slog.Error("billing failed", "subscription_id", id, "err", err)
		}
	}
	return out
}

// ======================= reads =======================

func (s *SalesService) GetSubscription(ctx context.Context, id int64) (*dto.SubscriptionResponse, error) {
	db := s.db.WithContext(ctx)
	sub, err := s.load(db, id)
	if err != nil {
		return nil, err
	}
	out := subscriptionResponse(sub)
	var invs []models.Invoice
	db.Where("subscription_id = ?", id).Order("invoice_date DESC, id DESC").Find(&invs)
	for i := range invs {
		out.Invoices = append(out.Invoices, s.invoiceBrief(&invs[i]))
	}
	return &out, nil
}

type SubscriptionFilter struct {
	Search     string
	Statuses   []string
	CustomerID int64
}

func (s *SalesService) ListSubscriptions(ctx context.Context, f SubscriptionFilter, p pagination.Params) ([]dto.SubscriptionResponse, int64, error) {
	db := s.db.WithContext(ctx)
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Joins("JOIN customers c ON c.id = subscriptions.customer_id")
		if len(f.Statuses) > 0 {
			q = q.Where("subscriptions.status IN ?", f.Statuses)
		}
		if f.CustomerID > 0 {
			q = q.Where("subscriptions.customer_id = ?", f.CustomerID)
		}
		if t := strings.TrimSpace(f.Search); t != "" {
			like := "%" + strings.ToLower(t) + "%"
			q = q.Where("(lower(c.name) LIKE ? OR lower(subscriptions.quotation_number) LIKE ? OR lower(coalesce(subscriptions.subscription_number,'')) LIKE ?)", like, like, like)
		}
		return q
	}
	var total int64
	if err := db.Model(&models.Subscription{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Subscription
	err := db.Scopes(scope).Preload("Customer", func(q *gorm.DB) *gorm.DB { return q.Unscoped() }).Preload("Plan").
		Preload("Items").Order("subscriptions." + p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&list).Error
	out := make([]dto.SubscriptionResponse, len(list))
	for i := range list {
		out[i] = subscriptionResponse(&list[i])
	}
	return out, total, err
}

func subscriptionResponse(s *models.Subscription) dto.SubscriptionResponse {
	r := dto.SubscriptionResponse{ID: s.ID, QuotationNumber: s.QuotationNumber, SubscriptionNumber: s.SubscriptionNumber,
		CustomerID: s.CustomerID, PlanID: s.PlanID, Status: string(s.Status), StartDate: dateString(s.StartDate),
		ExpiryDate: dateStringPtr(s.ExpiryDate), QuotationValidUntil: dateStringPtr(s.QuotationValidUntil),
		NextBillingDate: dateStringPtr(s.NextBillingDate), PaymentTermsDays: s.PaymentTermsDays, CyclesBilled: s.CyclesBilled,
		CloseReason: s.CloseReason, ParentID: s.ParentSubscriptionID, Subtotal: s.Subtotal.StringFixed(2),
		DiscountTotal: s.DiscountTotal.StringFixed(2), TaxTotal: s.TaxTotal.StringFixed(2), Total: s.Total.StringFixed(2),
		Items: make([]dto.SubItemResponse, 0, len(s.Items)), CreatedAt: s.CreatedAt}
	if s.Customer != nil {
		r.CustomerName = s.Customer.Name
	}
	if s.Plan != nil {
		r.PlanName, r.BillingLabel = s.Plan.Name, billingLabel(s.Plan.BillingPeriod, s.Plan.BillingInterval)
		r.IsPausable, r.IsClosable, r.IsRenewable = s.Plan.IsPausable, s.Plan.IsClosable, s.Plan.IsRenewable
	}
	for _, it := range s.Items {
		r.Items = append(r.Items, dto.SubItemResponse{ID: it.ID, ProductID: it.ProductID, VariantID: it.VariantID,
			Description: it.Description, Quantity: it.Quantity, UnitPrice: it.UnitPrice.StringFixed(2),
			TaxRate: it.TaxRate.StringFixed(2), LineSubtotal: it.LineSubtotal.StringFixed(2)})
	}
	return r
}

// ======================= invoices =======================

func (s *SalesService) invoiceBrief(inv *models.Invoice) dto.InvoiceBrief {
	due := inv.Total.Sub(inv.AmountPaid)
	ps := "unpaid"
	if inv.AmountPaid.IsPositive() {
		ps = "partial"
	}
	if !due.IsPositive() {
		ps = "paid"
	}
	return dto.InvoiceBrief{ID: inv.ID, InvoiceNumber: inv.InvoiceNumber, PeriodStart: dateStringPtr(inv.PeriodStart),
		PeriodEnd: dateStringPtr(inv.PeriodEnd), InvoiceDate: dateString(inv.InvoiceDate), DueDate: dateString(inv.DueDate),
		Status: string(inv.Status), PaymentStatus: ps, Total: inv.Total.StringFixed(2), AmountDue: due.StringFixed(2),
		IsOverdue: inv.Status == models.InvoiceConfirmed && due.IsPositive() && dateString(inv.DueDate) < dateString(s.today())}
}

type InvoiceFilter struct {
	Search     string
	Status     string
	Overdue    bool
	CustomerID int64
}

func (s *SalesService) ListInvoices(ctx context.Context, f InvoiceFilter, p pagination.Params) ([]dto.InvoiceResponse, int64, error) {
	db := s.db.WithContext(ctx)
	scope := func(q *gorm.DB) *gorm.DB {
		if f.Status != "" {
			q = q.Where("status = ?", f.Status)
		}
		if f.Overdue {
			q = q.Where("status = 'confirmed' AND amount_due > 0 AND due_date < ?", dateString(s.today()))
		}
		if f.CustomerID > 0 {
			q = q.Where("customer_id = ?", f.CustomerID)
		}
		if t := strings.TrimSpace(f.Search); t != "" {
			like := "%" + strings.ToLower(t) + "%"
			q = q.Where("(lower(invoice_number) LIKE ? OR lower(customer_name) LIKE ?)", like, like)
		}
		return q
	}
	var total int64
	if err := db.Model(&models.Invoice{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Invoice
	if err := db.Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dto.InvoiceResponse, len(list))
	for i := range list {
		out[i] = s.invoiceResponse(db, &list[i], false)
	}
	return out, total, nil
}

func (s *SalesService) invoiceResponse(db *gorm.DB, inv *models.Invoice, full bool) dto.InvoiceResponse {
	r := dto.InvoiceResponse{InvoiceBrief: s.invoiceBrief(inv), CustomerID: inv.CustomerID, CustomerName: inv.CustomerName,
		BillingAddress: inv.BillingAddress, CustomerGSTIN: inv.CustomerGSTIN, SubscriptionID: inv.SubscriptionID,
		Subtotal: inv.Subtotal.StringFixed(2), DiscountTotal: inv.DiscountTotal.StringFixed(2), TaxTotal: inv.TaxTotal.StringFixed(2),
		AmountPaid: inv.AmountPaid.StringFixed(2), SentAt: inv.SentAt, Items: []dto.InvoiceItemResponse{}, Payments: []dto.PaymentBrief{}}
	if r.IsOverdue {
		r.DaysOverdue = int(s.today().Sub(inv.DueDate).Hours() / 24)
	}
	if !full {
		return r
	}
	if inv.SubscriptionID != nil {
		var sn *string
		db.Model(&models.Subscription{}).Where("id = ?", *inv.SubscriptionID).Select("subscription_number").Scan(&sn)
		r.SubscriptionNumber = sn
	}
	var email *string
	db.Unscoped().Model(&models.Customer{}).Where("id = ?", inv.CustomerID).Select("email").Scan(&email)
	r.CustomerEmail = email
	var items []models.InvoiceItem
	db.Where("invoice_id = ?", inv.ID).Order("id").Find(&items)
	for _, it := range items {
		r.Items = append(r.Items, dto.InvoiceItemResponse{Description: it.Description, Quantity: it.Quantity,
			UnitPrice: it.UnitPrice.StringFixed(2), LineSubtotal: it.LineSubtotal.StringFixed(2),
			DiscountAmount: it.DiscountAmount.StringFixed(2), TaxRate: it.TaxRate.StringFixed(2),
			TaxAmount: it.TaxAmount.StringFixed(2), LineTotal: it.LineTotal.StringFixed(2)})
	}
	var pays []models.Payment
	db.Where("invoice_id = ?", inv.ID).Order("payment_date, id").Find(&pays)
	for _, p := range pays {
		r.Payments = append(r.Payments, paymentBrief(&p))
	}
	return r
}

func paymentBrief(p *models.Payment) dto.PaymentBrief {
	return dto.PaymentBrief{ID: p.ID, PaymentNumber: p.PaymentNumber, Method: string(p.Method), Amount: p.Amount.StringFixed(2),
		PaymentDate: dateString(p.PaymentDate), Status: string(p.Status), Reference: p.Reference, Notes: p.Notes}
}

func (s *SalesService) GetInvoice(ctx context.Context, id int64) (*dto.InvoiceResponse, error) {
	db := s.db.WithContext(ctx)
	var inv models.Invoice
	if err := db.First(&inv, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("invoice")
		}
		return nil, err
	}
	out := s.invoiceResponse(db, &inv, true)
	return &out, nil
}

func (s *SalesService) ConfirmInvoice(ctx context.Context, id int64) (*dto.InvoiceResponse, error) {
	res := s.db.WithContext(ctx).Exec("UPDATE invoices SET status = 'confirmed', confirmed_at = now(), updated_at = now() WHERE id = ? AND status = 'draft'", id)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		if _, err := s.GetInvoice(ctx, id); err != nil {
			return nil, err
		}
		return nil, apperr.InvalidState("only draft invoices can be confirmed")
	}
	return s.GetInvoice(ctx, id)
}

// CancelInvoice: only while nothing has been paid; gives discount uses back.
func (s *SalesService) CancelInvoice(ctx context.Context, id int64, reason string) (*dto.InvoiceResponse, error) {
	if _, err := s.GetInvoice(ctx, id); err != nil {
		return nil, err
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(`UPDATE invoices SET status = 'cancelled', cancelled_at = now(), cancel_reason = ?, updated_at = now()
			WHERE id = ? AND status IN ('draft','confirmed') AND amount_paid = 0`, nullIfEmpty(strings.TrimSpace(reason)), id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return apperr.InvalidState("only unpaid draft or confirmed invoices can be cancelled")
		}
		var ids []int64
		tx.Table("discount_usages").Where("invoice_id = ?", id).Pluck("discount_id", &ids)
		for _, d := range ids {
			if err := s.discounts.ReleaseUse(tx, d); err != nil {
				return err
			}
		}
		return tx.Exec("DELETE FROM discount_usages WHERE invoice_id = ?", id).Error
	})
	if err != nil {
		return nil, err
	}
	return s.GetInvoice(ctx, id)
}

func (s *SalesService) SendInvoice(ctx context.Context, id int64) (*dto.InvoiceResponse, error) {
	inv, err := s.GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	if inv.Status != string(models.InvoiceConfirmed) && inv.Status != string(models.InvoicePaid) {
		return nil, apperr.InvalidState("confirm the invoice before sending it")
	}
	if err := s.emailInvoice(ctx, id); err != nil {
		return nil, apperr.Rule("EMAIL_FAILED", "the email could not be sent: "+err.Error())
	}
	return s.GetInvoice(ctx, id)
}

func (s *SalesService) emailInvoice(ctx context.Context, id int64) error {
	inv, err := s.GetInvoice(ctx, id)
	if err != nil || inv.CustomerEmail == nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Hi %s,\n\nInvoice %s from %s\nInvoice date: %s · Due: %s\n", inv.CustomerName, inv.InvoiceNumber, brand.Company, inv.InvoiceDate, inv.DueDate)
	if inv.PeriodStart != nil {
		fmt.Fprintf(&b, "Billing period: %s to %s\n", *inv.PeriodStart, *inv.PeriodEnd)
	}
	b.WriteString("\n")
	for _, it := range inv.Items {
		fmt.Fprintf(&b, "%d × %s @ ₹%s = ₹%s (discount ₹%s, GST %s%% ₹%s)\n", it.Quantity, it.Description, it.UnitPrice, it.LineSubtotal, it.DiscountAmount, it.TaxRate, it.TaxAmount)
	}
	fmt.Fprintf(&b, "\nSubtotal ₹%s\nDiscounts −₹%s\nTax ₹%s\nTotal ₹%s\nAmount due ₹%s\n", inv.Subtotal, inv.DiscountTotal, inv.TaxTotal, inv.Total, inv.AmountDue)
	if err := s.mailer.Send(ctx, mailer.Message{To: *inv.CustomerEmail, Subject: "Invoice " + inv.InvoiceNumber + " from " + brand.Company,
		Body: b.String() + brand.EmailSignature}); err != nil {
		return err
	}
	return s.db.WithContext(ctx).Exec("UPDATE invoices SET sent_at = now() WHERE id = ?", id).Error
}

func (s *SalesService) notify(customerID int64, subject, body string) {
	go func() {
		var email string
		s.db.Unscoped().Model(&models.Customer{}).Where("id = ?", customerID).Select("email").Scan(&email)
		if email != "" {
			_ = s.mailer.Send(context.Background(), mailer.Message{To: email, Subject: subject, Body: body + "\n" + brand.EmailSignature})
		}
	}()
}

// ======================= payments list =======================

type PaymentFilter struct {
	Search     string
	Method     string
	CustomerID int64
	InvoiceID  int64
}

func (s *SalesService) ListPayments(ctx context.Context, f PaymentFilter, p pagination.Params) ([]dto.PaymentResponse, int64, error) {
	db := s.db.WithContext(ctx)
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Table("payments p").Joins("JOIN invoices i ON i.id = p.invoice_id")
		if f.Method != "" {
			q = q.Where("p.method = ?", f.Method)
		}
		if f.CustomerID > 0 {
			q = q.Where("p.customer_id = ?", f.CustomerID)
		}
		if f.InvoiceID > 0 {
			q = q.Where("p.invoice_id = ?", f.InvoiceID)
		}
		if t := strings.TrimSpace(f.Search); t != "" {
			like := "%" + strings.ToLower(t) + "%"
			q = q.Where("(lower(p.payment_number) LIKE ? OR lower(i.invoice_number) LIKE ? OR lower(i.customer_name) LIKE ?)", like, like, like)
		}
		return q
	}
	var total int64
	if err := db.Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	out, err := s.paymentRows(db.Scopes(scope).Order("p.payment_date DESC, p.id DESC").Limit(p.PageSize).Offset(p.Offset()))
	return out, total, err
}

func (s *SalesService) paymentRows(q *gorm.DB) ([]dto.PaymentResponse, error) {
	var rows []struct {
		models.Payment
		InvoiceNumber string
		CustomerName  string
	}
	if err := q.Select("p.*, i.invoice_number, i.customer_name").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]dto.PaymentResponse, len(rows))
	for i := range rows {
		out[i] = dto.PaymentResponse{PaymentBrief: paymentBrief(&rows[i].Payment), InvoiceID: rows[i].InvoiceID,
			InvoiceNumber: rows[i].InvoiceNumber, CustomerID: rows[i].CustomerID, CustomerName: rows[i].CustomerName,
			CreatedAt: rows[i].CreatedAt}
	}
	return out, nil
}

// ======================= dashboard (A11) =======================

func (s *SalesService) Dashboard(ctx context.Context, role models.Role, from, to time.Time) (*dto.DashboardSummary, error) {
	db := s.db.WithContext(ctx)
	today := dateString(s.today())
	out := &dto.DashboardSummary{From: dateString(from), To: dateString(to), SubscriptionsByStatus: map[string]int64{}, RecentPayments: []dto.PaymentResponse{}}
	db.Model(&models.Customer{}).Count(&out.TotalCustomers)
	var rows []struct {
		Status string
		N      int64
	}
	db.Model(&models.Subscription{}).Select("status, count(*) AS n").Group("status").Scan(&rows)
	for _, r := range rows {
		out.SubscriptionsByStatus[r.Status] = r.N
		out.TotalSubscriptions += r.N
		if r.Status == "active" {
			out.ActiveSubscriptions = r.N
		}
	}
	db.Model(&models.Invoice{}).Where("status = 'confirmed' AND amount_due > 0").Count(&out.OpenInvoices)
	db.Model(&models.Invoice{}).Where("status = 'confirmed' AND amount_due > 0 AND due_date < ?", today).Count(&out.OverdueInvoices)
	if role != models.RoleAdmin {
		return out, nil // staff: operational numbers only
	}
	var money struct {
		MRR, Received, Outstanding, Overdue decimal.Decimal
	}
	db.Raw(`SELECT
	  (SELECT coalesce(sum((s.subtotal - s.discount_total) * CASE p.billing_period
	        WHEN 'daily' THEN 30 WHEN 'weekly' THEN 52.0/12 WHEN 'monthly' THEN 1 ELSE 1.0/12 END / p.billing_interval), 0)
	     FROM subscriptions s JOIN subscription_plans p ON p.id = s.plan_id WHERE s.status = 'active') AS mrr,
	  (SELECT coalesce(sum(amount), 0) FROM payments WHERE status = 'completed' AND payment_date BETWEEN ? AND ?) AS received,
	  (SELECT coalesce(sum(amount_due), 0) FROM invoices WHERE status = 'confirmed') AS outstanding,
	  (SELECT coalesce(sum(amount_due), 0) FROM invoices WHERE status = 'confirmed' AND due_date < ?) AS overdue`,
		dateString(from), dateString(to), today).Scan(&money)
	str := func(d decimal.Decimal) *string { v := d.StringFixed(2); return &v }
	out.MRR, out.ARR = str(money.MRR), str(money.MRR.Mul(decimal.NewFromInt(12)))
	out.RevenueReceived, out.Outstanding, out.OverdueAmount = str(money.Received), str(money.Outstanding), str(money.Overdue)
	recent, err := s.paymentRows(db.Table("payments p").Joins("JOIN invoices i ON i.id = p.invoice_id").Order("p.created_at DESC").Limit(5))
	if err == nil {
		out.RecentPayments = recent
	}
	return out, nil
}

func dateStringPtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	v := dateString(*t)
	return &v
}

func deref(s *string) string {
	if s == nil {
		return "—"
	}
	return *s
}
