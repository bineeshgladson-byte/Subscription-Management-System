package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

type AuthHandler struct{ svc *service.AuthService }

func NewAuthHandler(svc *service.AuthService) *AuthHandler { return &AuthHandler{svc: svc} }

// POST /auth/register  (public)
func (h *AuthHandler) Register(c *gin.Context) {
	var req dto.RegisterRequest
	if !bind(c, &req) {
		return
	}
	user, err := h.svc.Register(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, gin.H{"user": dto.NewUserResponse(user)})
}

// POST /auth/login  (public)
func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Login(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// POST /auth/forgot-password  (public, always 200)
func (h *AuthHandler) ForgotPassword(c *gin.Context) {
	var req dto.ForgotPasswordRequest
	if !bind(c, &req) {
		return
	}
	response.OK(c, h.svc.ForgotPassword(req.Email))
}

// POST /auth/reset-password  (public)
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var req dto.ResetPasswordRequest
	if !bind(c, &req) {
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.MessageResponse{Message: "Password updated. You can now log in."})
}

// GET /auth/me  (any logged-in user)
func (h *AuthHandler) Me(c *gin.Context) {
	user, err := h.svc.Me(c.Request.Context(), middleware.CurrentUserID(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewUserResponse(user))
}

// PUT /auth/me  (any logged-in user)
func (h *AuthHandler) UpdateMe(c *gin.Context) {
	var req dto.UpdateProfileRequest
	if !bind(c, &req) {
		return
	}
	user, err := h.svc.UpdateProfile(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewUserResponse(user))
}

// POST /auth/change-password  (any logged-in user) → returns a fresh token
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.ChangePassword(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
