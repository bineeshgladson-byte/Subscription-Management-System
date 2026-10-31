package service

import (
	"context"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// Customer portal (/me/*). Brief: customers "see only their own subscriptions and
// invoices through a portal" and "manage their own subscriptions without having
// to email anyone". The customer id ALWAYS comes from the login token; someone
// else's record answers 404, so it doesn't reveal that it exists.

var portalSubStatuses = []string{"quotation", "confirmed", "active", "paused", "closed", "renewed"}

func (s *SalesService) MySubscriptions(ctx context.Context, customerID int64, p pagination.Params) ([]dto.SubscriptionResponse, int64, error) {
	return s.ListSubscriptions(ctx, SubscriptionFilter{Statuses: portalSubStatuses, CustomerID: customerID}, p)
}

// mine checks ownership and that it's a real subscription (not an internal draft).
func (s *SalesService) mine(ctx context.Context, customerID, id int64) error {
	var sub models.Subscription
	err := s.db.WithContext(ctx).Select("id", "customer_id", "status").First(&sub, id).Error
	if err != nil || sub.CustomerID != customerID || sub.Status == models.SubDraft {
		return apperr.NotFound("subscription")
	}
	return nil
}

// MyRequestQuotation lets a customer request a quotation from an active plan.
// It creates the quotation under their own customer id and moves it to
// "quotation" so the team can process it immediately.
func (s *SalesService) MyRequestQuotation(ctx context.Context, customerID, userID int64, in dto.CreateMyQuotationRequest) (*dto.SubscriptionResponse, error) {
	out, err := s.createQuotation(ctx, userID, dto.CreateQuotationRequest{
		CustomerID: customerID,
		PlanID:     in.PlanID,
		StartDate:  in.StartDate,
		ExpiryDate: in.ExpiryDate,
	}, false)
	if err != nil {
		return nil, err
	}
	if out.Status == string(models.SubDraft) {
		out, err = s.SendQuotation(ctx, out.ID)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *SalesService) MySubscription(ctx context.Context, customerID, id int64) (*dto.SubscriptionResponse, error) {
	if err := s.mine(ctx, customerID, id); err != nil {
		return nil, err
	}
	out, err := s.GetSubscription(ctx, id)
	if err != nil {
		return nil, err
	}
	// Customers see their issued invoices only.
	kept := out.Invoices[:0]
	for _, i := range out.Invoices {
		if i.Status == string(models.InvoiceConfirmed) || i.Status == string(models.InvoicePaid) {
			kept = append(kept, i)
		}
	}
	out.Invoices = kept
	return out, nil
}

// MyAction runs pause / resume / close / renew on the customer's own subscription.
// The plan's permissions are checked by the same methods staff use.
func (s *SalesService) MyAction(ctx context.Context, customerID, userID, id int64, action string) (*dto.SubscriptionResponse, error) {
	if err := s.mine(ctx, customerID, id); err != nil {
		return nil, err
	}
	var err error
	switch action {
	case "pause":
		_, err = s.Pause(ctx, id)
	case "resume":
		_, err = s.Resume(ctx, id)
	case "close":
		_, err = s.Close(ctx, id)
	case "renew":
		// Creates a renewal quotation for staff to confirm.
		_, err = s.Renew(ctx, userID, id)
	default:
		return nil, apperr.NotFound("action")
	}
	if err != nil {
		return nil, err
	}
	return s.MySubscription(ctx, customerID, id)
}

func (s *SalesService) MyInvoices(ctx context.Context, customerID int64, p pagination.Params) ([]dto.InvoiceResponse, int64, error) {
	db := s.db.WithContext(ctx)
	q := db.Model(&models.Invoice{}).Where("customer_id = ? AND status IN ('confirmed','paid')", customerID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var list []models.Invoice
	if err := db.Where("customer_id = ? AND status IN ('confirmed','paid')", customerID).
		Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	out := make([]dto.InvoiceResponse, len(list))
	for i := range list {
		out[i] = s.invoiceResponse(db, &list[i], false)
	}
	return out, total, nil
}

func (s *SalesService) MyInvoice(ctx context.Context, customerID, id int64) (*dto.InvoiceResponse, error) {
	inv, err := s.GetInvoice(ctx, id)
	if err != nil || inv.CustomerID != customerID || (inv.Status != string(models.InvoiceConfirmed) && inv.Status != string(models.InvoicePaid)) {
		return nil, apperr.NotFound("invoice")
	}
	return inv, nil
}

func (s *SalesService) MyPayments(ctx context.Context, customerID int64, p pagination.Params) ([]dto.PaymentResponse, int64, error) {
	return s.ListPayments(ctx, PaymentFilter{CustomerID: customerID}, p)
}
