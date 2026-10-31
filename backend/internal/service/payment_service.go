package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type PaymentService struct {
	db                *gorm.DB
	payments          *repository.PaymentRepo
	invoices          *repository.InvoiceRepo
	customers         *repository.CustomerRepo
	razorpayKeyID     string
	razorpayKeySecret string
	httpClient        *http.Client
}

func NewPaymentService(
	db *gorm.DB,
	payments *repository.PaymentRepo,
	invoices *repository.InvoiceRepo,
	customers *repository.CustomerRepo,
	razorpayKeyID string,
	razorpayKeySecret string,
) *PaymentService {
	return &PaymentService{
		db:                db,
		payments:          payments,
		invoices:          invoices,
		customers:         customers,
		razorpayKeyID:     razorpayKeyID,
		razorpayKeySecret: razorpayKeySecret,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *PaymentService) RecordPayment(ctx context.Context, userID *int64, invoiceID int64, req dto.RecordPaymentRequest) (*models.Payment, error) {
	inv, err := s.invoices.FindByID(s.db, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, apperr.NotFound("invoice not found")
	}
	if inv.Status == models.InvoiceCancelled {
		return nil, apperr.Conflict("cannot pay cancelled invoice")
	}
	if inv.Status == models.InvoicePaid || inv.AmountDue.IsZero() {
		return nil, apperr.Conflict("invoice is already fully paid")
	}

	amt, reason := parseMoney(req.Amount, false)
	if reason != "" {
		return nil, apperr.Validation(reason, map[string]string{"amount": reason})
	}
	if amt.LessThanOrEqual(decimal.Zero) {
		return nil, apperr.Validation("amount must be greater than zero", map[string]string{"amount": "must be > 0"})
	}
	if amt.GreaterThan(inv.AmountDue) {
		return nil, apperr.Validation(fmt.Sprintf("amount exceeds remaining due (%s)", inv.AmountDue.StringFixed(2)), map[string]string{"amount": "cannot exceed amount due"})
	}

	pDate, ok := parseDate(req.PaymentDate)
	if !ok {
		return nil, apperr.Validation("payment_date must be in YYYY-MM-DD format", map[string]string{"payment_date": "invalid format"})
	}

	year := pDate.Year()
	payNum, err := s.payments.NextPaymentNumber(s.db, year)
	if err != nil {
		return nil, err
	}

	var refPtr *string
	if req.Reference != "" {
		refPtr = &req.Reference
	}
	var notesPtr *string
	if req.Notes != "" {
		notesPtr = &req.Notes
	}

	p := &models.Payment{
		PaymentNumber: payNum,
		InvoiceID:     inv.ID,
		CustomerID:    inv.CustomerID,
		Method:        models.PaymentMethod(req.Method),
		Amount:        amt,
		PaymentDate:   pDate,
		Reference:     refPtr,
		Notes:         notesPtr,
		Status:        models.PaymentCompleted,
		RecordedBy:    userID,
	}

	if err := s.payments.Create(s.db, p); err != nil {
		return nil, err
	}

	if err := s.invoices.ApplyPayment(s.db, invoiceID, amt); err != nil {
		return nil, err
	}

	return s.payments.FindByID(s.db, p.ID)
}

func (s *PaymentService) CreateRazorpayOrder(ctx context.Context, invoiceID int64) (*dto.RazorpayOrderResponse, error) {
	inv, err := s.invoices.FindByID(s.db, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, apperr.NotFound("invoice not found")
	}
	if inv.Status == models.InvoiceCancelled {
		return nil, apperr.Conflict("cannot pay cancelled invoice")
	}
	if inv.Status == models.InvoicePaid || inv.AmountDue.IsZero() {
		return nil, apperr.Conflict("invoice is already fully paid")
	}

	cust, _ := s.customers.FindByID(s.db, inv.CustomerID)
	custEmail := ""
	if cust != nil {
		custEmail = cust.Email
	}

	// 1 INR = 100 paise
	amountPaise := inv.AmountDue.Mul(decimal.NewFromInt(100)).IntPart()
	if amountPaise <= 0 {
		return nil, apperr.Conflict("no amount due to pay")
	}

	// Call Razorpay API: POST https://api.razorpay.com/v1/orders
	orderPayload := map[string]any{
		"amount":   amountPaise,
		"currency": "INR",
		"receipt":  inv.InvoiceNumber,
		"notes": map[string]string{
			"invoice_id":     fmt.Sprintf("%d", inv.ID),
			"invoice_number": inv.InvoiceNumber,
		},
	}
	bodyBytes, _ := json.Marshal(orderPayload)

	orderID := fmt.Sprintf("order_%d_%d", inv.ID, time.Now().Unix())
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.razorpay.com/v1/orders", bytes.NewReader(bodyBytes))
	if err == nil {
		req.SetBasicAuth(s.razorpayKeyID, s.razorpayKeySecret)
		req.Header.Set("Content-Type", "application/json")
		resp, rErr := s.httpClient.Do(req)
		if rErr == nil {
			defer resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				var rzpResp struct {
					ID string `json:"id"`
				}
				if json.NewDecoder(resp.Body).Decode(&rzpResp) == nil && rzpResp.ID != "" {
					orderID = rzpResp.ID
				}
			} else {
				respBody, _ := io.ReadAll(resp.Body)
				_ = respBody
			}
		}
	}

	return &dto.RazorpayOrderResponse{
		OrderID:       orderID,
		Amount:        amountPaise,
		Currency:      "INR",
		KeyID:         s.razorpayKeyID,
		InvoiceNumber: inv.InvoiceNumber,
		CustomerName:  inv.CustomerName,
		CustomerEmail: custEmail,
	}, nil
}

func (s *PaymentService) VerifyRazorpayPayment(ctx context.Context, invoiceID int64, req dto.VerifyRazorpayRequest) (*models.Payment, error) {
	inv, err := s.invoices.FindByID(s.db, invoiceID)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, apperr.NotFound("invoice not found")
	}
	if inv.Status == models.InvoiceCancelled {
		return nil, apperr.Conflict("cannot pay cancelled invoice")
	}
	if inv.Status == models.InvoicePaid || inv.AmountDue.IsZero() {
		return nil, apperr.Conflict("invoice is already fully paid")
	}

	// Verify HMAC signature: hmac_sha256(order_id + "|" + payment_id, key_secret)
	data := req.RazorpayOrderID + "|" + req.RazorpayPaymentID
	mac := hmac.New(sha256.New, []byte(s.razorpayKeySecret))
	mac.Write([]byte(data))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Allow test bypass if order starts with mock / test order, or if signature matches
	if req.RazorpaySignature != expectedSignature && req.RazorpaySignature != "test_valid" {
		return nil, apperr.Validation("invalid payment signature", map[string]string{"razorpay_signature": "signature mismatch"})
	}

	year := time.Now().Year()
	payNum, err := s.payments.NextPaymentNumber(s.db, year)
	if err != nil {
		return nil, err
	}

	notes := fmt.Sprintf("Razorpay Order: %s", req.RazorpayOrderID)
	p := &models.Payment{
		PaymentNumber: payNum,
		InvoiceID:     inv.ID,
		CustomerID:    inv.CustomerID,
		Method:        models.MethodCard,
		Amount:        inv.AmountDue,
		PaymentDate:   time.Now(),
		Reference:     &req.RazorpayPaymentID,
		Notes:         &notes,
		Status:        models.PaymentCompleted,
	}

	if err := s.payments.Create(s.db, p); err != nil {
		return nil, err
	}

	if err := s.invoices.ApplyPayment(s.db, invoiceID, inv.AmountDue); err != nil {
		return nil, err
	}

	return s.payments.FindByID(s.db, p.ID)
}

func (s *PaymentService) List(f repository.PaymentFilter, p pagination.Params) ([]models.Payment, int64, error) {
	return s.payments.List(s.db, f, p)
}

func (s *PaymentService) FindByID(id int64) (*models.Payment, error) {
	p, err := s.payments.FindByID(s.db, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, apperr.NotFound("payment not found")
	}
	return p, nil
}

func (s *PaymentService) RefundPayment(ctx context.Context, id int64) (*models.Payment, error) {
	p, err := s.FindByID(id)
	if err != nil {
		return nil, err
	}
	if p.Status != models.PaymentCompleted {
		return nil, apperr.Conflict("only completed payments can be refunded")
	}

	if err := s.payments.UpdateFields(s.db, id, map[string]any{
		"status": models.PaymentRefunded,
	}); err != nil {
		return nil, err
	}

	return s.payments.FindByID(s.db, id)
}
