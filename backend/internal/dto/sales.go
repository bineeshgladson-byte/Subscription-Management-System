package dto

import "time"

// ---------- quotations / subscriptions (A7/A8) ----------

type QuotationLineInput struct {
	ProductID int64  `json:"product_id" binding:"required,min=1"`
	VariantID *int64 `json:"variant_id" binding:"omitempty,min=1"`
	Quantity  int    `json:"quantity" binding:"omitempty,min=1,max=100000"`
	UnitPrice string `json:"unit_price" binding:"omitempty,max=16"` // override catalogue price
}

// CreateQuotationRequest: items optional — defaults to the template's lines, else the plan's products.
type CreateQuotationRequest struct {
	CustomerID          int64                `json:"customer_id" binding:"required,min=1"`
	PlanID              int64                `json:"plan_id" binding:"required,min=1"`
	TemplateID          *int64               `json:"template_id" binding:"omitempty,min=1"`
	StartDate           string               `json:"start_date"`            // default today
	ExpiryDate          string               `json:"expiry_date"`           // optional end of contract
	QuotationValidUntil string               `json:"quotation_valid_until"` // default today + template validity (15)
	PaymentTermsDays    *int                 `json:"payment_terms_days" binding:"omitempty,min=0,max=365"`
	Items               []QuotationLineInput `json:"items" binding:"omitempty,max=50,dive"`
}

// Customer portal request (POST /me/quotations). Customer id comes from JWT.
type CreateMyQuotationRequest struct {
	PlanID     int64  `json:"plan_id" binding:"required,min=1"`
	StartDate  string `json:"start_date"`
	ExpiryDate string `json:"expiry_date"`
}

type SubItemResponse struct {
	ID           int64  `json:"id"`
	ProductID    int64  `json:"product_id"`
	VariantID    *int64 `json:"variant_id"`
	Description  string `json:"description"`
	Quantity     int    `json:"quantity"`
	UnitPrice    string `json:"unit_price"`
	TaxRate      string `json:"tax_rate"`
	LineSubtotal string `json:"line_subtotal"`
}

type InvoiceBrief struct {
	ID            int64   `json:"id"`
	InvoiceNumber string  `json:"invoice_number"`
	PeriodStart   *string `json:"period_start"`
	PeriodEnd     *string `json:"period_end"`
	InvoiceDate   string  `json:"invoice_date"`
	DueDate       string  `json:"due_date"`
	Status        string  `json:"status"`
	PaymentStatus string  `json:"payment_status"`
	IsOverdue     bool    `json:"is_overdue"`
	Total         string  `json:"total"`
	AmountDue     string  `json:"amount_due"`
}

type SubscriptionResponse struct {
	ID                  int64             `json:"id"`
	QuotationNumber     string            `json:"quotation_number"`
	SubscriptionNumber  *string           `json:"subscription_number"`
	CustomerID          int64             `json:"customer_id"`
	CustomerName        string            `json:"customer_name"`
	PlanID              int64             `json:"plan_id"`
	PlanName            string            `json:"plan_name"`
	BillingLabel        string            `json:"billing_label"`
	IsPausable          bool              `json:"is_pausable"`
	IsClosable          bool              `json:"is_closable"`
	IsRenewable         bool              `json:"is_renewable"`
	Status              string            `json:"status"`
	StartDate           string            `json:"start_date"`
	ExpiryDate          *string           `json:"expiry_date"`
	QuotationValidUntil *string           `json:"quotation_valid_until"`
	NextBillingDate     *string           `json:"next_billing_date"`
	PaymentTermsDays    int               `json:"payment_terms_days"`
	CyclesBilled        int               `json:"cycles_billed"`
	CloseReason         *string           `json:"close_reason"`
	ParentID            *int64            `json:"parent_subscription_id"`
	Subtotal            string            `json:"subtotal"` // per billing period (preview)
	DiscountTotal       string            `json:"discount_total"`
	TaxTotal            string            `json:"tax_total"`
	Total               string            `json:"total"`
	Items               []SubItemResponse `json:"items"`
	Invoices            []InvoiceBrief    `json:"invoices,omitempty"`
	CreatedAt           time.Time         `json:"created_at"`
}

// ---------- invoices (A9) ----------

type InvoiceItemResponse struct {
	Description    string `json:"description"`
	Quantity       int    `json:"quantity"`
	UnitPrice      string `json:"unit_price"`
	LineSubtotal   string `json:"line_subtotal"`
	DiscountAmount string `json:"discount_amount"`
	TaxRate        string `json:"tax_rate"`
	TaxAmount      string `json:"tax_amount"`
	LineTotal      string `json:"line_total"`
}

type PaymentBrief struct {
	ID            int64   `json:"id"`
	PaymentNumber string  `json:"payment_number"`
	Method        string  `json:"method"`
	Amount        string  `json:"amount"`
	PaymentDate   string  `json:"payment_date"`
	Status        string  `json:"status"`
	Reference     *string `json:"reference"`
	Notes         *string `json:"notes"`
}

type InvoiceResponse struct {
	InvoiceBrief
	CustomerID         int64                 `json:"customer_id"`
	CustomerName       string                `json:"customer_name"`
	CustomerEmail      *string               `json:"customer_email"`
	BillingAddress     *string               `json:"billing_address"`
	CustomerGSTIN      *string               `json:"customer_gstin"`
	SubscriptionID     *int64                `json:"subscription_id"`
	SubscriptionNumber *string               `json:"subscription_number"`
	DaysOverdue        int                   `json:"days_overdue"`
	Subtotal           string                `json:"subtotal"`
	DiscountTotal      string                `json:"discount_total"`
	TaxTotal           string                `json:"tax_total"`
	AmountPaid         string                `json:"amount_paid"`
	SentAt             *time.Time            `json:"sent_at"`
	Items              []InvoiceItemResponse `json:"items"`
	Payments           []PaymentBrief        `json:"payments"`
}

type CancelInvoiceRequest struct {
	Reason string `json:"reason" binding:"omitempty,max=255"`
}

// ---------- payments list (A10) ----------

type PaymentResponse struct {
	PaymentBrief
	InvoiceID     int64     `json:"invoice_id"`
	InvoiceNumber string    `json:"invoice_number"`
	CustomerID    int64     `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	CreatedAt     time.Time `json:"created_at"`
}

// ---------- billing run + dashboard (A11/A12) ----------

type BillingRunRequest struct {
	AsOf string `json:"as_of"` // YYYY-MM-DD; honoured only when APP_ENV=development
}

type BillingRunResponse struct {
	AsOf            string   `json:"as_of"`
	Activated       int      `json:"activated"`
	InvoicesCreated int      `json:"invoices_created"`
	Closed          int      `json:"closed"`
	Errors          []string `json:"errors"`
}

type DashboardSummary struct {
	From                  string           `json:"from"`
	To                    string           `json:"to"`
	TotalCustomers        int64            `json:"total_customers"`
	ActiveSubscriptions   int64            `json:"active_subscriptions"`
	TotalSubscriptions    int64            `json:"total_subscriptions"`
	SubscriptionsByStatus map[string]int64 `json:"subscriptions_by_status"`
	OpenInvoices          int64            `json:"open_invoices"`
	OverdueInvoices       int64            `json:"overdue_invoices"`
	// Money: admin only (null for staff).
	MRR             *string           `json:"mrr"`
	ARR             *string           `json:"arr"`
	RevenueReceived *string           `json:"revenue_received"` // completed payments in [from, to]
	Outstanding     *string           `json:"outstanding"`
	OverdueAmount   *string           `json:"overdue_amount"`
	RecentPayments  []PaymentResponse `json:"recent_payments"`
}
