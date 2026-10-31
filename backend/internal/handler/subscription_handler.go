package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

type SubscriptionHandler struct {
	svc *service.SubscriptionService
}

func NewSubscriptionHandler(svc *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{svc: svc}
}

func (h *SubscriptionHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "id DESC", "id", "created_at", "total", "start_date")
	isQuote := false
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

func (h *SubscriptionHandler) Get(c *gin.Context) {
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

func (h *SubscriptionHandler) Activate(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	sub, err := h.svc.ActivateSubscription(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *SubscriptionHandler) Pause(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	sub, err := h.svc.PauseSubscription(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *SubscriptionHandler) Resume(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	sub, err := h.svc.ResumeSubscription(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *SubscriptionHandler) Close(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.CloseSubscriptionRequest
	_ = c.ShouldBindJSON(&req)

	sub, err := h.svc.CloseSubscription(c.Request.Context(), id, req.Reason)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *SubscriptionHandler) Renew(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.RenewSubscriptionRequest
	_ = c.ShouldBindJSON(&req)

	sub, err := h.svc.RenewSubscription(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewSubscriptionResponse(sub))
}

func (h *SubscriptionHandler) BillNow(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	inv, err := h.svc.BillNow(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, dto.NewInvoiceResponse(inv))
}
