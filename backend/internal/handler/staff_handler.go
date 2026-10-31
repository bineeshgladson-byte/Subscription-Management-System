package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// StaffHandler serves /staff. All routes are admin-only (router/routes_staff.go).
type StaffHandler struct{ svc *service.StaffService }

func NewStaffHandler(svc *service.StaffService) *StaffHandler { return &StaffHandler{svc: svc} }

// GET /staff?q=&is_active=&page=&page_size=&sort=
func (h *StaffHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "created_at DESC", "created_at", "full_name", "email", "last_login_at")

	var active *bool
	if v := c.Query("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			response.Error(c, apperr.Validation("Invalid filter", map[string]string{"is_active": "must be true or false"}))
			return
		}
		active = &b
	}

	users, total, err := h.svc.List(c.Request.Context(), c.Query("q"), active, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	out := make([]dto.StaffResponse, len(users))
	for i := range users {
		out[i] = dto.NewStaffResponse(&users[i])
	}
	response.List(c, out, p.Meta(total))
}

// POST /staff
func (h *StaffHandler) Create(c *gin.Context) {
	var req dto.CreateStaffRequest
	if !bind(c, &req) {
		return
	}
	user, invited, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, dto.CreateStaffResponse{Staff: dto.NewStaffResponse(user), InviteSent: invited})
}

// GET /staff/:id
func (h *StaffHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	user, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewStaffResponse(user))
}

// PUT /staff/:id
func (h *StaffHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateStaffRequest
	if !bind(c, &req) {
		return
	}
	user, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewStaffResponse(user))
}

// PATCH /staff/:id/status  {"is_active": false}
func (h *StaffHandler) SetStatus(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateStaffStatusRequest
	if !bind(c, &req) {
		return
	}
	user, err := h.svc.SetActive(c.Request.Context(), id, *req.IsActive)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewStaffResponse(user))
}
