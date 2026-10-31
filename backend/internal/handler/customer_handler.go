package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// CustomerHandler serves /customers (admin + staff; archive is admin-only).
type CustomerHandler struct{ svc *service.CustomerService }

func NewCustomerHandler(svc *service.CustomerService) *CustomerHandler {
	return &CustomerHandler{svc: svc}
}

// GET /customers?q=&has_login=&page=&page_size=&sort=
func (h *CustomerHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "created_at DESC", "created_at", "name", "email", "company_name")
	f := repository.CustomerFilter{Search: c.Query("q")}
	if v := c.Query("has_login"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			response.Error(c, apperr.Validation("Invalid filter", map[string]string{"has_login": "must be true or false"}))
			return
		}
		f.HasLogin = &b
	}
	list, total, err := h.svc.List(c.Request.Context(), f, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	out := make([]dto.CustomerResponse, len(list))
	for i := range list {
		out[i] = dto.NewCustomerResponse(&list[i])
	}
	response.List(c, out, p.Meta(total))
}

// POST /customers
func (h *CustomerHandler) Create(c *gin.Context) {
	var req dto.CreateCustomerRequest
	if !bind(c, &req) {
		return
	}
	cust, err := h.svc.Create(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, dto.NewCustomerResponse(cust))
}

// GET /customers/:id  (includes stats)
func (h *CustomerHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.Detail(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

// PUT /customers/:id
func (h *CustomerHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateCustomerRequest
	if !bind(c, &req) {
		return
	}
	cust, err := h.svc.Update(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewCustomerResponse(cust))
}

// DELETE /customers/:id  (admin only; archives)
func (h *CustomerHandler) Archive(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.Archive(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.MessageResponse{Message: "Customer archived."})
}
