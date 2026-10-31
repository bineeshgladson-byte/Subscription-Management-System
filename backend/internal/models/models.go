// Package models holds GORM structs that mirror the database tables.
// Models are NOT sent over the wire directly; handlers convert them to DTOs.
//
// Money fields use decimal.Decimal (never float64).
package models

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Timestamps struct {
	CreatedAt time.Time
	UpdatedAt time.Time
}

// ---------- Auth & people ----------

type User struct {
	ID                int64 `gorm:"primaryKey"`
	Email             string
	PasswordHash      string
	FullName          string
	Role              Role
	IsActive          bool
	PasswordChangedAt *time.Time
	LastLoginAt       *time.Time
	Timestamps
	DeletedAt gorm.DeletedAt

	Customer *Customer `gorm:"foreignKey:UserID"`
}

type Customer struct {
	ID             int64 `gorm:"primaryKey"`
	UserID         *int64
	Name           string
	CompanyName    *string
	Email          string
	Phone          *string
	BillingAddress *string
	GSTIN          *string `gorm:"column:gstin"`
	CreatedBy      *int64
	Timestamps
	DeletedAt gorm.DeletedAt
}

type PasswordResetToken struct {
	ID        int64 `gorm:"primaryKey"`
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	UsedAt    *time.Time
	CreatedAt time.Time
}

// ---------- Catalogue ----------

type Tax struct {
	ID       int64 `gorm:"primaryKey"`
	Name     string
	Rate     decimal.Decimal `gorm:"type:numeric(5,2)"`
	IsActive bool
	Timestamps
}

type Product struct {
	ID          int64 `gorm:"primaryKey"`
	Name        string
	ProductType ProductType
	SalesPrice  decimal.Decimal `gorm:"type:numeric(12,2)"`
	CostPrice   decimal.Decimal `gorm:"type:numeric(12,2)"`
	Description *string
	TaxID       *int64
	Status      RecordStatus
	Timestamps
	DeletedAt gorm.DeletedAt

	Tax      *Tax
	Variants []ProductVariant
}

type ProductVariant struct {
	ID         int64 `gorm:"primaryKey"`
	ProductID  int64
	Attribute  string
	Name       string
	ExtraPrice decimal.Decimal `gorm:"type:numeric(12,2)"`
	SKU        *string         `gorm:"column:sku"`
	IsActive   bool
	Timestamps
}

type SubscriptionPlan struct {
	ID                   int64 `gorm:"primaryKey"`
	Name                 string
	Description          *string
	Price                decimal.Decimal `gorm:"type:numeric(12,2)"`
	BillingPeriod        BillingPeriod
	BillingInterval      int
	MinQuantity          int
	AvailableFrom        time.Time  `gorm:"type:date"`
	AvailableUntil       *time.Time `gorm:"type:date"`
	AutoCloseEnabled     bool
	AutoCloseAfterCycles *int
	IsPausable           bool
	IsClosable           bool
	IsRenewable          bool
	Status               RecordStatus
	Timestamps

	Products []PlanProduct `gorm:"foreignKey:PlanID"`
}

type PlanProduct struct {
	ID            int64 `gorm:"primaryKey"`
	PlanID        int64
	ProductID     int64
	VariantID     *int64
	Quantity      int
	PriceOverride *decimal.Decimal `gorm:"type:numeric(12,2)"`

	Product *Product
	Variant *ProductVariant
}

type QuotationTemplate struct {
	ID               int64 `gorm:"primaryKey"`
	Name             string
	PlanID           *int64
	ValidityDays     int
	PaymentTermsDays int
	Terms            *string
	IsActive         bool
	Timestamps

	Plan  *SubscriptionPlan
	Lines []QuotationTemplateLine `gorm:"foreignKey:TemplateID"`
}

type QuotationTemplateLine struct {
	ID         int64 `gorm:"primaryKey"`
	TemplateID int64
	ProductID  int64
	VariantID  *int64
	Quantity   int

	Product *Product
	Variant *ProductVariant
}

// ---------- Subscriptions (quotations are subscriptions in draft/quotation state) ----------

type Subscription struct {
	ID                   int64 `gorm:"primaryKey"`
	QuotationNumber      string
	SubscriptionNumber   *string
	CustomerID           int64
	PlanID               int64
	TemplateID           *int64
	Status               SubscriptionStatus
	StartDate            time.Time  `gorm:"type:date"`
	ExpiryDate           *time.Time `gorm:"type:date"`
	QuotationValidUntil  *time.Time `gorm:"type:date"`
	PaymentTermsDays     int
	NextBillingDate      *time.Time `gorm:"type:date"`
	BillingAnchorDay     int
	CyclesBilled         int
	ParentSubscriptionID *int64
	CloseReason          *string
	Subtotal             decimal.Decimal `gorm:"type:numeric(12,2)"`
	DiscountTotal        decimal.Decimal `gorm:"type:numeric(12,2)"`
	TaxTotal             decimal.Decimal `gorm:"type:numeric(12,2)"`
	Total                decimal.Decimal `gorm:"type:numeric(12,2)"`
	ConfirmedAt          *time.Time
	ActivatedAt          *time.Time
	PausedAt             *time.Time
	ClosedAt             *time.Time
	CreatedBy            *int64
	Timestamps

	Customer *Customer
	Plan     *SubscriptionPlan
	Items    []SubscriptionItem
}

type SubscriptionItem struct {
	ID             int64 `gorm:"primaryKey"`
	SubscriptionID int64
	ProductID      int64
	VariantID      *int64
	Description    string
	Quantity       int
	UnitPrice      decimal.Decimal `gorm:"type:numeric(12,2)"`
	TaxID          *int64
	TaxRate        decimal.Decimal `gorm:"type:numeric(5,2)"`
	LineSubtotal   decimal.Decimal `gorm:"type:numeric(12,2)"`
	Timestamps
}

// ---------- Invoices ----------

type Invoice struct {
	ID             int64 `gorm:"primaryKey"`
	InvoiceNumber  string
	CustomerID     int64
	SubscriptionID *int64
	PeriodStart    *time.Time `gorm:"type:date"`
	PeriodEnd      *time.Time `gorm:"type:date"`
	InvoiceDate    time.Time  `gorm:"type:date"`
	DueDate        time.Time  `gorm:"type:date"`
	Status         InvoiceStatus
	Subtotal       decimal.Decimal `gorm:"type:numeric(12,2)"`
	DiscountTotal  decimal.Decimal `gorm:"type:numeric(12,2)"`
	TaxTotal       decimal.Decimal `gorm:"type:numeric(12,2)"`
	Total          decimal.Decimal `gorm:"type:numeric(12,2)"`
	AmountPaid     decimal.Decimal `gorm:"type:numeric(12,2)"`
	// amount_due is a GENERATED column in Postgres: read it, never write it.
	AmountDue      decimal.Decimal `gorm:"type:numeric(12,2);->"`
	CustomerName   string
	BillingAddress *string
	CustomerGSTIN  *string `gorm:"column:customer_gstin"`
	SentAt         *time.Time
	ConfirmedAt    *time.Time
	CancelledAt    *time.Time
	CancelReason   *string
	CreatedBy      *int64 // nil = created by the billing scheduler
	Timestamps

	Items []InvoiceItem
}

type InvoiceItem struct {
	ID             int64 `gorm:"primaryKey"`
	InvoiceID      int64
	ProductID      *int64
	VariantID      *int64
	Description    string
	Quantity       int
	UnitPrice      decimal.Decimal `gorm:"type:numeric(12,2)"`
	LineSubtotal   decimal.Decimal `gorm:"type:numeric(12,2)"`
	DiscountAmount decimal.Decimal `gorm:"type:numeric(12,2)"`
	TaxRate        decimal.Decimal `gorm:"type:numeric(5,2)"`
	TaxAmount      decimal.Decimal `gorm:"type:numeric(12,2)"`
	LineTotal      decimal.Decimal `gorm:"type:numeric(12,2)"`
	CreatedAt      time.Time
}

// ---------- Discounts ----------

type Discount struct {
	ID                int64 `gorm:"primaryKey"`
	Name              string
	Code              *string
	DiscountType      DiscountType
	Value             decimal.Decimal `gorm:"type:numeric(12,2)"`
	Scope             DiscountScope
	MinPurchaseAmount *decimal.Decimal `gorm:"type:numeric(12,2)"`
	MinQuantity       *int
	StartsOn          time.Time  `gorm:"type:date"`
	EndsOn            *time.Time `gorm:"type:date"`
	MaxUsage          *int
	UsageCount        int
	IsActive          bool
	CreatedBy         *int64
	Timestamps

	Products []Product `gorm:"many2many:discount_products;"`
}

type DiscountUsage struct {
	ID            int64 `gorm:"primaryKey"`
	DiscountID    int64
	InvoiceID     int64
	InvoiceItemID *int64
	CustomerID    int64
	Amount        decimal.Decimal `gorm:"type:numeric(12,2)"`
	CreatedAt     time.Time
}

// ---------- Payments ----------

type Payment struct {
	ID             int64 `gorm:"primaryKey"`
	PaymentNumber  string
	InvoiceID      int64
	CustomerID     int64
	Method         PaymentMethod
	Amount         decimal.Decimal `gorm:"type:numeric(12,2)"`
	PaymentDate    time.Time       `gorm:"type:date"`
	Reference      *string
	Notes          *string
	Status         PaymentStatus
	IdempotencyKey *string
	RecordedBy     *int64
	Timestamps
}

// ---------- Infrastructure ----------

type EmailOutbox struct {
	ID            int64 `gorm:"primaryKey"`
	ToEmail       string
	Subject       string
	Body          string
	AttachmentRef *string
	Status        EmailStatus
	Attempts      int
	LastError     *string
	SentAt        *time.Time
	CreatedAt     time.Time
}

func (EmailOutbox) TableName() string { return "email_outbox" }

type AuditLog struct {
	ID        int64 `gorm:"primaryKey"`
	UserID    *int64
	Action    string
	Entity    string
	EntityID  *int64
	Changes   []byte `gorm:"type:jsonb"`
	CreatedAt time.Time
}
