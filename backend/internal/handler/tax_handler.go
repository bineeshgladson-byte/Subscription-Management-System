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

// TaxHandler serves /taxes (read: admin + staff; write: admin).
type TaxHandler struct{ svc *service.TaxService }

func NewTaxHandler(svc *service.TaxService) *TaxHandler { return &TaxHandler{svc: svc} }

// GET /taxes?q=&is_active=&page=&page_size=&sort=
func (h *TaxHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "name ASC", "name", "rate", "created_at")
	f := repository.TaxFilter{Search: c.Query("q")}
	if v := c.Query("is_active"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			response.Error(c, apperr.Validation("Invalid filter", map[string]string{"is_active": "must be true or false"}))
			return
		}
		f.Active = &b
	}
	list, total, err := h.svc.List(c.Request.Context(), f, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

// GET /taxes/:id
func (h *TaxHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// POST /taxes
func (h *TaxHandler) Create(c *gin.Context) {
	var req dto.CreateTaxRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Create(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

// PUT /taxes/:id
func (h *TaxHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateTaxRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// DELETE /taxes/:id  (deactivates; taxes are never hard-deleted)
func (h *TaxHandler) Deactivate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Deactivate(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}
