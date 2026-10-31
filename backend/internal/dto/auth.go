// Package dto defines the JSON shapes of the API (the contract in docs/api/openapi.yaml).
// Keep these in sync with the OpenAPI file.
package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// ---------- Requests ----------

type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=120"`
	Email    string `json:"email" binding:"required,max=255"`
	Password string `json:"password" binding:"required,max=72"` // bcrypt ignores bytes after 72
	Phone    string `json:"phone" binding:"omitempty,max=20"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,max=255"`
	Password string `json:"password" binding:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,max=255"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,max=72"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,max=72"`
}

// UpdateProfileRequest: every field is optional; only fields sent are changed.
// Company fields are for customers only. Email can't be changed here.
type UpdateProfileRequest struct {
	FullName       *string `json:"full_name" binding:"omitempty,min=2,max=120"`
	Phone          *string `json:"phone" binding:"omitempty,max=20"`
	CompanyName    *string `json:"company_name" binding:"omitempty,max=150"`
	BillingAddress *string `json:"billing_address" binding:"omitempty,max=500"`
	GSTIN          *string `json:"gstin" binding:"omitempty,max=15"`
}

// ---------- Responses ----------

type CustomerProfile struct {
	Phone          *string `json:"phone"`
	CompanyName    *string `json:"company_name"`
	BillingAddress *string `json:"billing_address"`
	GSTIN          *string `json:"gstin"`
}

type UserResponse struct {
	ID         int64            `json:"id"`
	Email      string           `json:"email"`
	FullName   string           `json:"full_name"`
	Role       models.Role      `json:"role"`
	CustomerID *int64           `json:"customer_id,omitempty"`
	Profile    *CustomerProfile `json:"profile,omitempty"` // customers only
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresAt   time.Time    `json:"expires_at"`
	User        UserResponse `json:"user"`
}

type MessageResponse struct {
	Message string `json:"message"`
}

func NewUserResponse(u *models.User) UserResponse {
	r := UserResponse{ID: u.ID, Email: u.Email, FullName: u.FullName, Role: u.Role}
	if c := u.Customer; c != nil {
		id := c.ID
		r.CustomerID = &id
		r.Profile = &CustomerProfile{Phone: c.Phone, CompanyName: c.CompanyName,
			BillingAddress: c.BillingAddress, GSTIN: c.GSTIN}
	}
	return r
}
