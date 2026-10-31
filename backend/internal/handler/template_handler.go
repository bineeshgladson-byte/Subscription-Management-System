package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// TemplateHandler serves /quotation-templates (read: admin + staff; write: admin).
type TemplateHandler struct{ svc *service.TemplateService }

func NewTemplateHandler(svc *service.TemplateService) *TemplateHandler {
	return &TemplateHandler{svc: svc}
}

func (h *TemplateHandler) respond(c *gin.Context, out *dto.TemplateResponse, err error, created bool) {
	switch {
	case err != nil:
		response.Error(c, err)
	case created:
		response.Created(c, out)
	default:
		response.OK(c, out)
	}
}

// GET /quotation-templates?q=&is_active=&plan_id=&page=&page_size=&sort=
func (h *TemplateHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "name ASC", "name", "created_at", "validity_days")
	f := repository.TemplateFilter{Search: c.Query("q")}
	errs := map[string]string{}
	if v := c.Query("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs["is_active"] = "must be true or false"
		}
		f.Active = &b
	}
	if v := c.Query("plan_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			errs["plan_id"] = "must be a positive number"
		}
		f.PlanID = id
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

func (h *TemplateHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), id)
	h.respond(c, out, err, false)
}

func (h *TemplateHandler) Create(c *gin.Context) {
	var req dto.CreateTemplateRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Create(c.Request.Context(), req)
	h.respond(c, out, err, true)
}

func (h *TemplateHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateTemplateRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Update(c.Request.Context(), id, req)
	h.respond(c, out, err, false)
}

// DELETE /quotation-templates/:id (deactivates)
func (h *TemplateHandler) Deactivate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Deactivate(c.Request.Context(), id)
	h.respond(c, out, err, false)
}
