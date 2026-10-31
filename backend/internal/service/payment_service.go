package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type PaymentService struct {
	db       *gorm.DB
	payments *repository.PaymentRepo
	invoices *repository.InvoiceRepo
}

func NewPaymentService(
	db *gorm.DB,
	payments *repository.PaymentRepo,
	invoices *repository.InvoiceRepo,
) *PaymentService {
	return &PaymentService{
		db:       db,
		payments: payments,
		invoices: invoices,
	}
}

func (s *PaymentService) RecordPayment(
	ctx context.Context,
	invoiceID int64,
	req dto.RecordPaymentRequest,
	idempotencyKey string,
	recordedBy int64,
) (*models.Payment, *models.Invoice, error) {

	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil || !amount.GreaterThan(decimal.Zero) {
		return nil, nil, apperr.Validation(
			"Payment amount must be greater than zero.",
			map[string]string{"amount": "Invalid amount"},
		)
	}

	paymentDate, err := time.Parse("2006-01-02", req.PaymentDate)
	if err != nil {
		return nil, nil, apperr.Validation(
			"Payment date must use YYYY-MM-DD format.",
			map[string]string{"payment_date": "Invalid date"},
		)
	}

	var payment *models.Payment
	var invoice *models.Invoice

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {

		if idempotencyKey != "" {
			existing, err := s.payments.FindByIdempotencyKey(tx, idempotencyKey)
			if err != nil {
				return err
			}
			if existing != nil {
				if existing.InvoiceID != invoiceID ||
					existing.Amount.Cmp(amount) != 0 ||
					existing.Method != req.Method ||
					existing.PaymentDate.Format("2006-01-02") != req.PaymentDate {
					return apperr.Conflict("Idempotency key was already used for a different payment.")
				}

				payment = existing
				invoice, err = s.invoices.FindByID(tx, invoiceID)
				return err
			}
		}

		invoice, err = s.invoices.FindByIDForUpdate(tx, invoiceID)
		if err != nil {
			return err
		}
		if invoice == nil {
			return apperr.NotFound("Invoice")
		}

		if invoice.Status != models.InvoiceConfirmed {
			return apperr.InvalidState("Only confirmed invoices can receive payments.")
		}

		if amount.GreaterThan(invoice.AmountDue) {
			return apperr.Rule(
				"PAYMENT_EXCEEDS_BALANCE",
				"Payment amount exceeds the invoice balance.",
			)
		}

		var sequenceNumber int64
		if err := tx.Raw("SELECT nextval('payment_no_seq')").Scan(&sequenceNumber).Error; err != nil {
			return err
		}

		paymentNumber := fmt.Sprintf("PAY-%06d", sequenceNumber)

		reference := strings.TrimSpace(req.Reference)
		notes := strings.TrimSpace(req.Notes)
		key := strings.TrimSpace(idempotencyKey)

		payment = &models.Payment{
			PaymentNumber: paymentNumber,
			InvoiceID:     invoice.ID,
			CustomerID:    invoice.CustomerID,
			Method:        req.Method,
			Amount:        amount,
			PaymentDate:   paymentDate,
			Status:        models.PaymentCompleted,
			RecordedBy:    &recordedBy,
		}

		if reference != "" {
			payment.Reference = &reference
		}
		if notes != "" {
			payment.Notes = &notes
		}
		if key != "" {
			payment.IdempotencyKey = &key
		}

		if err := s.payments.Create(tx, payment); err != nil {
			return err
		}

		newAmountPaid := invoice.AmountPaid.Add(amount)
		newStatus := invoice.Status

		if newAmountPaid.GreaterThanOrEqual(invoice.Total) {
			newStatus = models.InvoicePaid
		}

		if err := s.invoices.UpdatePaymentTotals(
			tx,
			invoice.ID,
			newAmountPaid,
			newStatus,
		); err != nil {
			return err
		}

		invoice.AmountPaid = newAmountPaid
		invoice.Status = newStatus

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	return payment, invoice, nil
}
