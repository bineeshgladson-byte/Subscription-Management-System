package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type SubscriptionService struct {
	db          *gorm.DB
	subs        *repository.SubscriptionRepo
	customers   *repository.CustomerRepo
	plans       *repository.PlanRepo
	templates   *repository.TemplateRepo
	products    *repository.ProductRepo
	discounts   *repository.DiscountRepo
	mailer      mailer.Mailer
	invoiceGen  InvoiceGenerator
}

type InvoiceGenerator interface {
	GenerateForSubscription(ctx context.Context, sub *models.Subscription) (*models.Invoice, error)
}

func NewSubscriptionService(
	db *gorm.DB,
	subs *repository.SubscriptionRepo,
	customers *repository.CustomerRepo,
	plans *repository.PlanRepo,
	templates *repository.TemplateRepo,
	products *repository.ProductRepo,
	discounts *repository.DiscountRepo,
	m mailer.Mailer,
) *SubscriptionService {
	return &SubscriptionService{
		db:        db,
		subs:      subs,
		customers: customers,
		plans:     plans,
		templates: templates,
		products:  products,
		discounts: discounts,
		mailer:    m,
	}
}

func (s *SubscriptionService) SetInvoiceGenerator(gen InvoiceGenerator) {
	s.invoiceGen = gen
}

func (s *SubscriptionService) CreateQuotation(ctx context.Context, userID int64, req dto.CreateQuotationRequest) (*models.Subscription, error) {
	cust, err := s.customers.FindByID(s.db, req.CustomerID)
	if err != nil {
		return nil, err
	}
	if cust == nil {
		return nil, apperr.NotFound("customer not found")
	}

	plan, err := s.plans.FindByID(s.db, req.PlanID)
	if err != nil {
		return nil, err
	}
	if plan == nil || plan.Status != models.StatusActive {
		return nil, apperr.Validation("plan must be active", map[string]string{"plan_id": "must be an active plan"})
	}

	startDate, ok := parseDate(req.StartDate)
	if !ok {
		return nil, apperr.Validation("must be a date like 2026-11-01", map[string]string{"start_date": "must be a date like 2026-11-01"})
	}

	var expiryDate *time.Time
	if req.ExpiryDate != nil && *req.ExpiryDate != "" {
		ed, ok := parseDate(*req.ExpiryDate)
		if !ok {
			return nil, apperr.Validation("must be a date like 2026-11-01", map[string]string{"expiry_date": "must be a date like 2026-11-01"})
		}
		if ed.Before(startDate) {
			return nil, apperr.Validation("expiry date cannot be before start date", map[string]string{"expiry_date": "cannot be before start date"})
		}
		expiryDate = &ed
	}

	validityDays := 15
	paymentTerms := 15
	if req.PaymentTermsDays != nil {
		paymentTerms = *req.PaymentTermsDays
	}

	if req.TemplateID != nil && *req.TemplateID > 0 {
		tpl, err := s.templates.FindByID(s.db, *req.TemplateID)
		if err == nil && tpl != nil {
			validityDays = tpl.ValidityDays
			if req.PaymentTermsDays == nil {
				paymentTerms = tpl.PaymentTermsDays
			}
		}
	}

	// Prepare specs
	var specs []lineSpec
	if len(req.Lines) > 0 {
		for _, l := range req.Lines {
			override := ""
			if l.PriceOverride != nil {
				override = *l.PriceOverride
			}
			specs = append(specs, lineSpec{
				ProductID:     l.ProductID,
				VariantID:     l.VariantID,
				Quantity:      defaultInt(l.Quantity, 1),
				PriceOverride: override,
			})
		}
	} else if req.TemplateID != nil && *req.TemplateID > 0 {
		tpl, _ := s.templates.FindByID(s.db, *req.TemplateID)
		if tpl != nil && len(tpl.Lines) > 0 {
			for _, tl := range tpl.Lines {
				specs = append(specs, lineSpec{
					ProductID: tl.ProductID,
					VariantID: tl.VariantID,
					Quantity:  tl.Quantity,
				})
			}
		}
	}

	if len(specs) == 0 {
		// Use plan products
		for _, pp := range plan.Products {
			override := ""
			if pp.PriceOverride != nil {
				override = pp.PriceOverride.StringFixed(2)
			}
			specs = append(specs, lineSpec{
				ProductID:     pp.ProductID,
				VariantID:     pp.VariantID,
				Quantity:      pp.Quantity,
				PriceOverride: override,
			})
		}
	}

	resolved, err := resolveLines(s.db, s.products, "lines", specs, true)
	if err != nil {
		return nil, err
	}
	if len(resolved) == 0 {
		return nil, apperr.Validation("quotation must have at least one line", map[string]string{"lines": "must have at least one line"})
	}

	// Build price lines
	priceLines := make([]PriceLine, len(resolved))
	items := make([]models.SubscriptionItem, len(resolved))
	for i, rl := range resolved {
		uPrice := rl.UnitPrice()
		tRate := rl.TaxRate()
		priceLines[i] = PriceLine{
			ProductID: rl.ProductID,
			Quantity:  rl.Quantity,
			UnitPrice: uPrice,
			TaxRate:   tRate,
		}
		var taxID *int64
		if rl.Product.TaxID != nil {
			taxID = rl.Product.TaxID
		}
		desc := rl.Product.Name
		if rl.Variant != nil {
			desc += " (" + rl.Variant.Name + ")"
		}
		items[i] = models.SubscriptionItem{
			ProductID:    rl.ProductID,
			VariantID:    rl.VariantID,
			Description:  desc,
			Quantity:     rl.Quantity,
			UnitPrice:    uPrice,
			TaxID:        taxID,
			TaxRate:      tRate,
			LineSubtotal: uPrice.Mul(decimal.NewFromInt(int64(rl.Quantity))).Round(2),
		}
	}

	eligibleDiscounts, _ := s.discounts.Eligible(s.db, startDate)
	pDiscounts := make([]PriceDiscount, len(eligibleDiscounts))
	for i, d := range eligibleDiscounts {
		pMap := make(map[int64]bool, len(d.Products))
		for _, dp := range d.Products {
			pMap[dp.ID] = true
		}
		pDiscounts[i] = PriceDiscount{
			ID:          d.ID,
			Name:        d.Name,
			Type:        d.DiscountType,
			Value:       d.Value,
			Scope:       d.Scope,
			ProductIDs:  pMap,
			MinPurchase: d.MinPurchaseAmount,
			MinQuantity: d.MinQuantity,
		}
	}

	priceRes := Price(priceLines, pDiscounts)
	qNum, err := s.subs.NextQuotationNumber(s.db, startDate.Year())
	if err != nil {
		return nil, err
	}

	validUntil := startDate.AddDate(0, 0, validityDays)

	sub := &models.Subscription{
		QuotationNumber:     qNum,
		CustomerID:          req.CustomerID,
		PlanID:              req.PlanID,
		TemplateID:          req.TemplateID,
		Status:              models.SubQuotation,
		StartDate:           startDate,
		ExpiryDate:          expiryDate,
		QuotationValidUntil: &validUntil,
		PaymentTermsDays:    paymentTerms,
		BillingAnchorDay:    startDate.Day(),
		Subtotal:            priceRes.Subtotal,
		DiscountTotal:       priceRes.DiscountTotal,
		TaxTotal:            priceRes.TaxTotal,
		Total:               priceRes.Total,
		CreatedBy:           &userID,
		Items:               items,
	}

	if err := s.subs.Create(s.db, sub); err != nil {
		return nil, err
	}

	return s.subs.FindByID(s.db, sub.ID)
}

func (s *SubscriptionService) List(f repository.SubscriptionFilter, p pagination.Params) ([]models.Subscription, int64, error) {
	return s.subs.List(s.db, f, p)
}

func (s *SubscriptionService) FindByID(id int64) (*models.Subscription, error) {
	sub, err := s.subs.FindByID(s.db, id)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, apperr.NotFound("subscription/quotation not found")
	}
	return sub, nil
}

func (s *SubscriptionService) SendQuotation(ctx context.Context, id int64) (*models.Subscription, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub.Status != models.SubDraft && sub.Status != models.SubQuotation {
		return nil, apperr.Conflict("only draft or quotation can be sent")
	}

	_ = s.subs.UpdateFields(s.db, id, map[string]any{
		"status": models.SubQuotation,
	})

	if s.mailer != nil && sub.Customer != nil && sub.Customer.Email != "" {
		_ = s.mailer.Send(ctx, mailer.Message{
			To:      sub.Customer.Email,
			Subject: fmt.Sprintf("Your Quotation %s from Subly", sub.QuotationNumber),
			Body: fmt.Sprintf("Hello %s,\n\nHere is your quotation %s for %s totaling INR %s.\n\nThank you for choosing Subly!",
				sub.Customer.Name, sub.QuotationNumber, sub.Plan.Name, sub.Total.StringFixed(2)),
		})
	}

	return s.subs.FindByID(s.db, id)
}

func (s *SubscriptionService) ConfirmQuotation(ctx context.Context, id int64) (*models.Subscription, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub.Status != models.SubDraft && sub.Status != models.SubQuotation {
		return nil, apperr.Conflict("only draft or quotation can be confirmed")
	}

	year := sub.StartDate.Year()
	subNum, err := s.subs.NextSubscriptionNumber(s.db, year)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	nextBilling := sub.StartDate
	updates := map[string]any{
		"status":              models.SubConfirmed,
		"subscription_number": subNum,
		"confirmed_at":        now,
		"next_billing_date":   nextBilling,
	}

	if err := s.subs.UpdateFields(s.db, id, updates); err != nil {
		return nil, err
	}

	return s.subs.FindByID(s.db, id)
}

func (s *SubscriptionService) CancelQuotation(ctx context.Context, id int64) (*models.Subscription, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub.Status != models.SubDraft && sub.Status != models.SubQuotation {
		return nil, apperr.Conflict("only draft or quotation can be cancelled")
	}

	now := time.Now()
	reason := "Cancelled by user"
	if err := s.subs.UpdateFields(s.db, id, map[string]any{
		"status":       models.SubClosed,
		"closed_at":    now,
		"close_reason": reason,
	}); err != nil {
		return nil, err
	}

	return s.subs.FindByID(s.db, id)
}

func (s *SubscriptionService) ActivateSubscription(ctx context.Context, id int64) (*models.Subscription, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub.Status != models.SubConfirmed && sub.Status != models.SubPaused {
		return nil, apperr.Conflict("only confirmed or paused subscriptions can be activated")
	}

	now := time.Now()
	updates := map[string]any{
		"status":       models.SubActive,
		"activated_at": now,
	}
	if sub.NextBillingDate == nil {
		updates["next_billing_date"] = sub.StartDate
	}

	if err := s.subs.UpdateFields(s.db, id, updates); err != nil {
		return nil, err
	}
	return s.subs.FindByID(s.db, id)
}

func (s *SubscriptionService) PauseSubscription(ctx context.Context, id int64) (*models.Subscription, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub.Status != models.SubActive {
		return nil, apperr.Conflict("only active subscriptions can be paused")
	}
	if sub.Plan != nil && !sub.Plan.IsPausable {
		return nil, apperr.Rule("PLAN_NOT_PAUSABLE", "this subscription plan does not allow pausing")
	}

	now := time.Now()
	if err := s.subs.UpdateFields(s.db, id, map[string]any{
		"status":    models.SubPaused,
		"paused_at": now,
	}); err != nil {
		return nil, err
	}
	return s.subs.FindByID(s.db, id)
}

func (s *SubscriptionService) ResumeSubscription(ctx context.Context, id int64) (*models.Subscription, error) {
	return s.ActivateSubscription(ctx, id)
}

func (s *SubscriptionService) CloseSubscription(ctx context.Context, id int64, reason string) (*models.Subscription, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub.Status == models.SubClosed {
		return nil, apperr.Conflict("subscription is already closed")
	}
	if sub.Plan != nil && !sub.Plan.IsClosable {
		return nil, apperr.Rule("PLAN_NOT_CLOSABLE", "this subscription plan cannot be closed early")
	}

	now := time.Now()
	if reason == "" {
		reason = "Closed by administrator"
	}
	if err := s.subs.UpdateFields(s.db, id, map[string]any{
		"status":       models.SubClosed,
		"closed_at":    now,
		"close_reason": reason,
	}); err != nil {
		return nil, err
	}
	return s.subs.FindByID(s.db, id)
}

func (s *SubscriptionService) RenewSubscription(ctx context.Context, id int64, req dto.RenewSubscriptionRequest) (*models.Subscription, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if sub.Plan != nil && !sub.Plan.IsRenewable {
		return nil, apperr.Rule("PLAN_NOT_RENEWABLE", "this subscription plan cannot be renewed")
	}

	updates := map[string]any{
		"status": models.SubActive,
	}
	if req.NewExpiryDate != nil && *req.NewExpiryDate != "" {
		ed, ok := parseDate(*req.NewExpiryDate)
		if !ok {
			return nil, apperr.Validation("must be a date like 2026-11-01", map[string]string{"new_expiry_date": "must be a date like 2026-11-01"})
		}
		updates["expiry_date"] = ed
	} else if sub.Plan != nil {
		// extend by one period
		cur := time.Now()
		if sub.ExpiryDate != nil && sub.ExpiryDate.After(cur) {
			cur = *sub.ExpiryDate
		}
		newExp := NextBillingDate(cur, sub.Plan.BillingPeriod, sub.Plan.BillingInterval, sub.BillingAnchorDay)
		updates["expiry_date"] = newExp
	}

	if err := s.subs.UpdateFields(s.db, id, updates); err != nil {
		return nil, err
	}
	return s.subs.FindByID(s.db, id)
}

func (s *SubscriptionService) BillNow(ctx context.Context, id int64) (*models.Invoice, error) {
	sub, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if s.invoiceGen == nil {
		return nil, fmt.Errorf("invoice generator not wired")
	}
	return s.invoiceGen.GenerateForSubscription(ctx, sub)
}
