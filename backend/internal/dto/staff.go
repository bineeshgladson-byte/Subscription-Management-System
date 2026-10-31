package dto

import (
	"time"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// CreateStaffRequest: if password is omitted, the new staff member is emailed a
// "set your password" link instead (recommended: the admin never knows it).
type CreateStaffRequest struct {
	FullName string `json:"full_name" binding:"required,min=2,max=120"`
	Email    string `json:"email" binding:"required,max=255"`
	Password string `json:"password" binding:"omitempty,max=72"`
}

type UpdateStaffRequest struct {
	FullName *string `json:"full_name" binding:"omitempty,min=2,max=120"`
	Email    *string `json:"email" binding:"omitempty,max=255"`
}

type UpdateStaffStatusRequest struct {
	// Pointer so a missing field is an error instead of silently meaning false.
	IsActive *bool `json:"is_active" binding:"required"`
}

type StaffResponse struct {
	ID          int64      `json:"id"`
	Email       string     `json:"email"`
	FullName    string     `json:"full_name"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

type CreateStaffResponse struct {
	Staff      StaffResponse `json:"staff"`
	InviteSent bool          `json:"invite_sent"`
}

func NewStaffResponse(u *models.User) StaffResponse {
	return StaffResponse{ID: u.ID, Email: u.Email, FullName: u.FullName, IsActive: u.IsActive,
		LastLoginAt: u.LastLoginAt, CreatedAt: u.CreatedAt}
}
