package handler

import (
	"github.com/gin-gonic/gin"
	"strconv"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// PlanHandler serves /plans. Everyone logged in can read (customers only see
// active plans available today); only the admin writes.
type PlanHandler struct{ svc *service.PlanService }

func NewPlanHandler(svc *service.PlanService) *PlanHandler { return &PlanHandler{svc: svc} }

func (h *PlanHandler) respond(c *gin.Context, out *dto.PlanResponse, err error, created bool) {
	if err != nil {
		response.Error(c, err)
		return
	}
	if created {
		response.Created(c, out)
	} else {
		response.OK(c, out)
	}
}

// GET /plans?q=&status=active|archived|all&billing_period=&available_on=YYYY-MM-DD&page=&page_size=&sort=
func (h *PlanHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "name ASC", "name", "price", "created_at", "available_from")
	f := repository.PlanFilter{Search: c.Query("q"), Status: c.Query("status"), BillingPeriod: c.Query("billing_period")}
	errs := map[string]string{}
	if value := c.Query("product_id"); value != "" {
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil || n < 1 {
			errs["product_id"] = "must be a positive product id"
		} else {
			f.ProductID = n
		}
	}
	if f.Status != "" && f.Status != "active" && f.Status != "archived" && f.Status != "all" {
		errs["status"] = "must be active, archived or all"
	}
	switch f.BillingPeriod {
	case "", "daily", "weekly", "monthly", "yearly":
	default:
		errs["billing_period"] = "must be daily, weekly, monthly or yearly"
	}
	if v := c.Query("available_on"); v != "" {
		d, ok := service.ParseDate(v)
		if !ok {
			errs["available_on"] = "must be a date like 2026-11-01"
		}
		f.AvailableOn = &d
	}
	if len(errs) > 0 {
		response.Error(c, apperr.Validation("Invalid filter", errs))
		return
	}
	list, total, err := h.svc.List(c.Request.Context(), middleware.CurrentRole(c), f, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

// GET /plans/:id
func (h *PlanHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), middleware.CurrentRole(c), id)
	h.respond(c, out, err, false)
}

// POST /plans
func (h *PlanHandler) Create(c *gin.Context) {
	var req dto.CreatePlanRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Create(c.Request.Context(), req)
	h.respond(c, out, err, true)
}

// PUT /plans/:id
func (h *PlanHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdatePlanRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Update(c.Request.Context(), id, req)
	h.respond(c, out, err, false)
}

// PATCH /plans/:id/status  {"status": "archived" | "active"}
func (h *PlanHandler) SetStatus(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdatePlanStatusRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.SetStatus(c.Request.Context(), id, models.RecordStatus(req.Status))
	h.respond(c, out, err, false)
}
