package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// DiscountHandler serves /discounts (admin only) and /pricing/preview (admin + staff).
type DiscountHandler struct {
	svc     *service.DiscountService
	pricing *service.PricingService
}

func NewDiscountHandler(svc *service.DiscountService, pricing *service.PricingService) *DiscountHandler {
	return &DiscountHandler{svc: svc, pricing: pricing}
}

func (h *DiscountHandler) respond(c *gin.Context, out *dto.DiscountResponse, err error, created bool) {
	switch {
	case err != nil:
		response.Error(c, err)
	case created:
		response.Created(c, out)
	default:
		response.OK(c, out)
	}
}

// GET /discounts?q=&scope=&state=active|scheduled|expired|used_up|inactive&page=&page_size=&sort=
func (h *DiscountHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "created_at DESC", "created_at", "name", "starts_on", "ends_on", "usage_count")
	f := repository.DiscountFilter{Search: c.Query("q"), Scope: c.Query("scope"), State: c.Query("state")}
	errs := map[string]string{}
	switch f.Scope {
	case "", "subscription", "product":
	default:
		errs["scope"] = "must be subscription or product"
	}
	switch f.State {
	case "", "active", "scheduled", "expired", "used_up", "inactive":
	default:
		errs["state"] = "must be active, scheduled, expired, used_up or inactive"
	}
	if len(errs) > 0 {
		response.Error(c, apperr.Validation("Invalid filter", errs))
		return
	}
	list, total, err := h.svc.List(c.Request.Context(), f, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

func (h *DiscountHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), id)
	h.respond(c, out, err, false)
}

func (h *DiscountHandler) Create(c *gin.Context) {
	var req dto.CreateDiscountRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	h.respond(c, out, err, true)
}

func (h *DiscountHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateDiscountRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Update(c.Request.Context(), id, req)
	h.respond(c, out, err, false)
}

// DELETE /discounts/:id (deactivates)
func (h *DiscountHandler) Deactivate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Deactivate(c.Request.Context(), id)
	h.respond(c, out, err, false)
}

// GET /discounts/:id/usages
func (h *DiscountHandler) Usages(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	p := pagination.Parse(c, "created_at DESC")
	list, total, err := h.svc.Usages(c.Request.Context(), id, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

// POST /pricing/preview
func (h *DiscountHandler) Preview(c *gin.Context) {
	var req dto.PricePreviewRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.pricing.Preview(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
