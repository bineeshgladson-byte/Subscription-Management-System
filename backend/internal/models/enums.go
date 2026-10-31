package models

// These string types mirror the PostgreSQL ENUM types created in
// database/migrations/*_init_schema.up.sql. Keep both in sync.

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleStaff    Role = "staff"
	RoleCustomer Role = "customer"
)

type ProductType string

const (
	ProductSoftware ProductType = "software"
	ProductService  ProductType = "service"
	ProductAddon    ProductType = "addon"
)

type RecordStatus string

const (
	StatusActive   RecordStatus = "active"
	StatusArchived RecordStatus = "archived"
)

type BillingPeriod string

const (
	PeriodDaily   BillingPeriod = "daily"
	PeriodWeekly  BillingPeriod = "weekly"
	PeriodMonthly BillingPeriod = "monthly"
	PeriodYearly  BillingPeriod = "yearly"
)

type SubscriptionStatus string

const (
	SubDraft     SubscriptionStatus = "draft"
	SubQuotation SubscriptionStatus = "quotation"
	SubConfirmed SubscriptionStatus = "confirmed"
	SubActive    SubscriptionStatus = "active"
	SubPaused    SubscriptionStatus = "paused"
	SubClosed    SubscriptionStatus = "closed"
	SubRenewed   SubscriptionStatus = "renewed"
)

type InvoiceStatus string

const (
	InvoiceDraft     InvoiceStatus = "draft"
	InvoiceConfirmed InvoiceStatus = "confirmed"
	InvoicePaid      InvoiceStatus = "paid"
	InvoiceCancelled InvoiceStatus = "cancelled"
)

type DiscountType string

const (
	DiscountFixed      DiscountType = "fixed"
	DiscountPercentage DiscountType = "percentage"
)

type DiscountScope string

const (
	ScopeSubscription DiscountScope = "subscription"
	ScopeProduct      DiscountScope = "product"
)

type PaymentMethod string

const (
	MethodUPI          PaymentMethod = "upi"
	MethodBankTransfer PaymentMethod = "bank_transfer"
	MethodCard         PaymentMethod = "card"
	MethodCash         PaymentMethod = "cash"
)

type PaymentStatus string

const (
	PaymentPending   PaymentStatus = "pending"
	PaymentCompleted PaymentStatus = "completed"
	PaymentFailed    PaymentStatus = "failed"
	PaymentRefunded  PaymentStatus = "refunded"
)

type EmailStatus string

const (
	EmailPending EmailStatus = "pending"
	EmailSent    EmailStatus = "sent"
	EmailFailed  EmailStatus = "failed"
)
