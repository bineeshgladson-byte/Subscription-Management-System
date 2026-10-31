package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// CreateCustomerRequest: staff/admin create a billable customer. The customer
// does NOT need a login; if they sign up later with the same email, the login is
// linked to this record automatically.
type CreateCustomerRequest struct {
	Name           string `json:"name" binding:"required,min=2,max=150"`
	Email          string `json:"email" binding:"required,max=255"`
	Phone          string `json:"phone" binding:"omitempty,max=20"`
	CompanyName    string `json:"company_name" binding:"omitempty,max=150"`
	BillingAddress string `json:"billing_address" binding:"omitempty,max=500"`
	GSTIN          string `json:"gstin" binding:"omitempty,max=15"`
}

// UpdateCustomerRequest: only fields sent are changed; "" clears an optional field.
type UpdateCustomerRequest struct {
	Name           *string `json:"name" binding:"omitempty,min=2,max=150"`
	Email          *string `json:"email" binding:"omitempty,max=255"`
	Phone          *string `json:"phone" binding:"omitempty,max=20"`
	CompanyName    *string `json:"company_name" binding:"omitempty,max=150"`
	BillingAddress *string `json:"billing_address" binding:"omitempty,max=500"`
	GSTIN          *string `json:"gstin" binding:"omitempty,max=15"`
}

type CustomerResponse struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	Email          string    `json:"email"`
	Phone          *string   `json:"phone"`
	CompanyName    *string   `json:"company_name"`
	BillingAddress *string   `json:"billing_address"`
	GSTIN          *string   `json:"gstin"`
	HasLogin       bool      `json:"has_login"` // true once the customer has signed up
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CustomerStats is shown on the customer detail screen.
type CustomerStats struct {
	ActiveSubscriptions int64  `json:"active_subscriptions"` // active + paused
	TotalSubscriptions  int64  `json:"total_subscriptions"`
	OpenInvoices        int64  `json:"open_invoices"` // confirmed, not fully paid
	OverdueInvoices     int64  `json:"overdue_invoices"`
	AmountDue           string `json:"amount_due"` // money as string, e.g. "1178.82"
}

type CustomerDetailResponse struct {
	CustomerResponse
	Stats CustomerStats `json:"stats"`
}

func NewCustomerResponse(c *models.Customer) CustomerResponse {
	return CustomerResponse{ID: c.ID, Name: c.Name, Email: c.Email, Phone: c.Phone, CompanyName: c.CompanyName,
		BillingAddress: c.BillingAddress, GSTIN: c.GSTIN, HasLogin: c.UserID != nil,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt}
}
