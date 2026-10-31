// Package dto defines the JSON shapes of the API (the contract in docs/api/openapi.yaml).
// Keep these in sync with the OpenAPI file.
package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

type RegisterRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=120"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Phone    string `json:"phone" binding:"omitempty,max=20"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UserResponse struct {
	ID         int64       `json:"id"`
	Email      string      `json:"email"`
	FullName   string      `json:"full_name"`
	Role       models.Role `json:"role"`
	CustomerID *int64      `json:"customer_id,omitempty"`
}

type LoginResponse struct {
	AccessToken string       `json:"access_token"`
	ExpiresAt   time.Time    `json:"expires_at"`
	User        UserResponse `json:"user"`
}

func NewUserResponse(u *models.User) UserResponse {
	r := UserResponse{ID: u.ID, Email: u.Email, FullName: u.FullName, Role: u.Role}
	if u.Customer != nil {
		id := u.Customer.ID
		r.CustomerID = &id
	}
	return r
}
