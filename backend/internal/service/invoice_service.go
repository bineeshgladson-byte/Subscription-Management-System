package service

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type InvoiceService struct {
	db          *gorm.DB
	invoices    *repository.InvoiceRepo
	subs        *repository.SubscriptionRepo
	customers   *repository.CustomerRepo
	autoConfirm bool
	mailer      mailer.Mailer
}

func NewInvoiceService(
	db *gorm.DB,
	invoices *repository.InvoiceRepo,
	subs *repository.SubscriptionRepo,
	customers *repository.CustomerRepo,
	autoConfirm bool,
	m mailer.Mailer,
) *InvoiceService {
	return &InvoiceService{
		db:          db,
		invoices:    invoices,
		subs:        subs,
		customers:   customers,
		autoConfirm: autoConfirm,
		mailer:      m,
	}
}

func (s *InvoiceService) GenerateForSubscription(ctx context.Context, sub *models.Subscription) (*models.Invoice, error) {
	if sub == nil {
		return nil, apperr.Validation("subscription required", map[string]string{"subscription_id": "required"})
	}

	cust, err := s.customers.FindByID(s.db, sub.CustomerID)
	if err != nil {
		return nil, err
	}
	if cust == nil {
		return nil, apperr.NotFound("customer not found")
	}

	pStart := time.Now()
	if sub.NextBillingDate != nil {
		pStart = *sub.NextBillingDate
	} else if !sub.StartDate.IsZero() {
		pStart = sub.StartDate
	}

	period := models.PeriodMonthly
	interval := 1
	anchor := sub.BillingAnchorDay
	if anchor < 1 || anchor > 31 {
		anchor = pStart.Day()
	}
	if sub.Plan != nil {
		period = sub.Plan.BillingPeriod
		interval = sub.Plan.BillingInterval
	}

	nextDate := NextBillingDate(pStart, period, interval, anchor)
	pEnd := nextDate.AddDate(0, 0, -1)

	invDate := time.Now()
	dueDate := invDate.AddDate(0, 0, sub.PaymentTermsDays)

	year := invDate.Year()
	invNum, err := s.invoices.NextInvoiceNumber(s.db, year)
	if err != nil {
		return nil, err
	}

	status := models.InvoiceDraft
	var confirmedAt *time.Time
	if s.autoConfirm {
		status = models.InvoiceConfirmed
		now := time.Now()
		confirmedAt = &now
	}

	invItems := make([]models.InvoiceItem, 0, len(sub.Items))
	var subtotal, taxTotal, total decimal.Decimal
	for _, it := range sub.Items {
		lSub := it.LineSubtotal
		tAmt := lSub.Mul(it.TaxRate).Div(hundred).Round(2)
		lTot := lSub.Add(tAmt)

		subtotal = subtotal.Add(lSub)
		taxTotal = taxTotal.Add(tAmt)
		total = total.Add(lTot)

		pID := it.ProductID
		vID := it.VariantID
		invItems = append(invItems, models.InvoiceItem{
			ProductID:    &pID,
			VariantID:    vID,
			Description:  it.Description,
			Quantity:     it.Quantity,
			UnitPrice:    it.UnitPrice,
			LineSubtotal: lSub,
			TaxRate:      it.TaxRate,
			TaxAmount:    tAmt,
			LineTotal:    lTot,
		})
	}

	subID := sub.ID
	inv := &models.Invoice{
		InvoiceNumber:  invNum,
		CustomerID:     cust.ID,
		SubscriptionID: &subID,
		PeriodStart:    &pStart,
		PeriodEnd:      &pEnd,
		InvoiceDate:    invDate,
		DueDate:        dueDate,
		Status:         status,
		Subtotal:       subtotal,
		DiscountTotal:  sub.DiscountTotal,
		TaxTotal:       taxTotal,
		Total:          total.Sub(sub.DiscountTotal),
		AmountPaid:     decimal.Zero,
		CustomerName:   cust.Name,
		BillingAddress: cust.BillingAddress,
		CustomerGSTIN:  cust.GSTIN,
		ConfirmedAt:    confirmedAt,
		CreatedBy:      sub.CreatedBy,
		Items:          invItems,
	}

	if err := s.invoices.Create(s.db, inv); err != nil {
		return nil, err
	}

	// Update subscription: cycles_billed + 1, next_billing_date = nextDate
	newCycles := sub.CyclesBilled + 1
	subUpdates := map[string]any{
		"cycles_billed":     newCycles,
		"next_billing_date": nextDate,
	}

	if sub.Plan != nil && sub.Plan.AutoCloseEnabled && sub.Plan.AutoCloseAfterCycles != nil && newCycles >= *sub.Plan.AutoCloseAfterCycles {
		now := time.Now()
		subUpdates["status"] = models.SubClosed
		subUpdates["closed_at"] = now
		subUpdates["close_reason"] = "Plan cycle limit reached"
	}

	_ = s.subs.UpdateFields(s.db, sub.ID, subUpdates)

	return s.invoices.FindByID(s.db, inv.ID)
}

func (s *InvoiceService) List(f repository.InvoiceFilter, p pagination.Params) ([]models.Invoice, int64, error) {
	return s.invoices.List(s.db, f, p)
}

func (s *InvoiceService) FindByID(id int64) (*models.Invoice, error) {
	inv, err := s.invoices.FindByID(s.db, id)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, apperr.NotFound("invoice not found")
	}
	return inv, nil
}

func (s *InvoiceService) ConfirmInvoice(ctx context.Context, id int64) (*models.Invoice, error) {
	inv, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if inv.Status != models.InvoiceDraft {
		return nil, apperr.Conflict("only draft invoices can be confirmed")
	}
	now := time.Now()
	if err := s.invoices.UpdateFields(s.db, id, map[string]any{
		"status":       models.InvoiceConfirmed,
		"confirmed_at": now,
	}); err != nil {
		return nil, err
	}
	return s.invoices.FindByID(s.db, id)
}

func (s *InvoiceService) CancelInvoice(ctx context.Context, id int64, reason string) (*models.Invoice, error) {
	inv, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if inv.Status == models.InvoicePaid {
		return nil, apperr.Conflict("paid invoices cannot be cancelled")
	}
	if reason == "" {
		reason = "Cancelled by user"
	}
	now := time.Now()
	if err := s.invoices.UpdateFields(s.db, id, map[string]any{
		"status":        models.InvoiceCancelled,
		"cancelled_at":  now,
		"cancel_reason": reason,
	}); err != nil {
		return nil, err
	}
	return s.invoices.FindByID(s.db, id)
}

func (s *InvoiceService) SendInvoice(ctx context.Context, id int64) (*models.Invoice, error) {
	inv, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}

	cust, _ := s.customers.FindByID(s.db, inv.CustomerID)
	now := time.Now()
	_ = s.invoices.UpdateFields(s.db, id, map[string]any{
		"sent_at": now,
	})

	if s.mailer != nil && cust != nil && cust.Email != "" {
		_ = s.mailer.Send(ctx, mailer.Message{
			To:      cust.Email,
			Subject: fmt.Sprintf("Your Invoice %s from Subly", inv.InvoiceNumber),
			Body: fmt.Sprintf("Hello %s,\n\nYour invoice %s totaling INR %s is ready. Due date: %s.\n\nThank you for choosing Subly!",
				cust.Name, inv.InvoiceNumber, inv.Total.StringFixed(2), inv.DueDate.Format("2006-01-02")),
		})
	}

	return s.invoices.FindByID(s.db, id)
}

func (s *InvoiceService) GeneratePDF(ctx context.Context, id int64) ([]byte, string, error) {
	inv, err := s.FindByID(id)
	if err != nil {
		return nil, "", err
	}

	billingAddr := ""
	if inv.BillingAddress != nil {
		billingAddr = *inv.BillingAddress
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <title>Invoice %s</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; padding: 40px; color: #1e293b; max-width: 800px; margin: auto; }
    .header { display: flex; justify-content: space-between; border-bottom: 2px solid #e2e8f0; padding-bottom: 20px; }
    .brand { font-size: 24px; font-weight: bold; color: #4338ca; }
    .inv-num { font-size: 20px; font-weight: 600; text-align: right; }
    .meta { margin-top: 24px; display: flex; justify-content: space-between; }
    table { width: 100%%; border-collapse: collapse; margin-top: 32px; }
    th { text-align: left; padding: 12px; background: #f8fafc; border-bottom: 1px solid #cbd5e1; font-size: 13px; text-transform: uppercase; }
    td { padding: 12px; border-bottom: 1px solid #e2e8f0; }
    .totals { margin-top: 24px; margin-left: auto; width: 300px; }
    .totals tr td:last-child { text-align: right; font-weight: 600; }
    .status { display: inline-block; padding: 4px 12px; border-radius: 9999px; font-size: 12px; font-weight: 600; text-transform: uppercase; }
    .status-paid { background: #dcfce7; color: #15803d; }
    .status-confirmed { background: #e0e7ff; color: #4338ca; }
  </style>
</head>
<body>
  <div class="header">
    <div>
      <div class="brand">Subly</div>
      <div>Brightloop Software</div>
    </div>
    <div>
      <div class="inv-num">%s</div>
      <div style="margin-top: 6px;"><span class="status status-%s">%s</span></div>
    </div>
  </div>
  <div class="meta">
    <div>
      <strong>Billed To:</strong><br>
      %s<br>
      %s
    </div>
    <div style="text-align: right;">
      <strong>Invoice Date:</strong> %s<br>
      <strong>Due Date:</strong> %s
    </div>
  </div>
  <table>
    <thead>
      <tr>
        <th>Description</th>
        <th style="text-align: center;">Qty</th>
        <th style="text-align: right;">Unit Price</th>
        <th style="text-align: right;">Tax</th>
        <th style="text-align: right;">Total</th>
      </tr>
    </thead>
    <tbody>`,
		inv.InvoiceNumber,
		inv.Status,
		inv.Status,
		inv.CustomerName,
		billingAddr,
		inv.InvoiceDate.Format("2006-01-02"),
		inv.DueDate.Format("2006-01-02"),
	)

	for _, it := range inv.Items {
		html += fmt.Sprintf(`
      <tr>
        <td>%s</td>
        <td style="text-align: center;">%d</td>
        <td style="text-align: right;">INR %s</td>
        <td style="text-align: right;">INR %s (%s%%)</td>
        <td style="text-align: right;">INR %s</td>
      </tr>`,
			it.Description,
			it.Quantity,
			it.UnitPrice.StringFixed(2),
			it.TaxAmount.StringFixed(2),
			it.TaxRate.StringFixed(2),
			it.LineTotal.StringFixed(2),
		)
	}

	html += fmt.Sprintf(`
    </tbody>
  </table>
  <table class="totals">
    <tr><td>Subtotal:</td><td>INR %s</td></tr>
    <tr><td>Discount:</td><td>INR %s</td></tr>
    <tr><td>Tax:</td><td>INR %s</td></tr>
    <tr style="font-size: 16px; border-top: 2px solid #cbd5e1;"><td>Total:</td><td>INR %s</td></tr>
    <tr><td>Amount Paid:</td><td>INR %s</td></tr>
    <tr style="font-size: 16px; color: #b91c1c;"><td>Amount Due:</td><td>INR %s</td></tr>
  </table>
</body>
</html>`,
		inv.Subtotal.StringFixed(2),
		inv.DiscountTotal.StringFixed(2),
		inv.TaxTotal.StringFixed(2),
		inv.Total.StringFixed(2),
		inv.AmountPaid.StringFixed(2),
		inv.AmountDue.StringFixed(2),
	)

	return []byte(html), fmt.Sprintf("%s.html", inv.InvoiceNumber), nil
}
