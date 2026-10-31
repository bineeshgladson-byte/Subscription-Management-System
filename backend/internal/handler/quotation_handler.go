package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

type QuotationHandler struct {
	svc *service.SubscriptionService
}

func NewQuotationHandler(svc *service.SubscriptionService) *QuotationHandler {
	return &QuotationHandler{svc: svc}
}

func (h *QuotationHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "id DESC", "id", "created_at", "total", "quotation_number")
	isQuote := true
	f := repository.SubscriptionFilter{
		IsQuotation: &isQuote,
		Search:      c.Query("q"),
	}
	if st := c.Query("status"); st != "" && st != "all" {
		f.Status = &st
	}
	if custIDStr := c.Query("customer_id"); custIDStr != "" {
		if cid, err := strconv.ParseInt(custIDStr, 10, 64); err == nil && cid > 0 {
			f.CustomerID = &cid
		}
	}

	subs, total, err := h.svc.List(f, p)
	if err != nil {
		response.Error(c, err)
		return
	}

	out := make([]dto.SubscriptionResponse, len(subs))
	for i := range subs {
		out[i] = *dto.NewSubscriptionResponse(&subs[i])
	}
	response.List(c, out, p.Meta(total))
}

func (h *QuotationHandler) Create(c *gin.Context) {
	var req dto.CreateQuotationRequest
	if !bind(c, &req) {
		return
	}

	userID := middleware.CurrentUserID(c)
	sub, err := h.svc.CreateQuotation(c.Request.Context(), userID, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, dto.NewSubscriptionResponse(sub))
}

func (h *QuotationHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	sub, err := h.svc.FindByID(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *QuotationHandler) Send(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	sub, err := h.svc.SendQuotation(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *QuotationHandler) Confirm(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	sub, err := h.svc.ConfirmQuotation(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *QuotationHandler) Cancel(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	sub, err := h.svc.CancelQuotation(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}
