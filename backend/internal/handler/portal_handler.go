package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// customerID reads the customer from the login token (never from the URL).
func customerID(c *gin.Context) (int64, bool) {
	id, ok := middleware.CurrentCustomerID(c)
	if !ok {
		response.Error(c, apperr.Forbidden("this account has no customer profile"))
	}
	return id, ok
}

// GET /me/subscriptions
func (h *SalesHandler) MySubscriptions(c *gin.Context) {
	cid, ok := customerID(c)
	if !ok {
		return
	}
	p := pagination.Parse(c, "created_at DESC", "created_at", "start_date", "next_billing_date")
	list, total, err := h.svc.MySubscriptions(c.Request.Context(), cid, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

// GET /me/subscriptions/:id
func (h *SalesHandler) MySubscription(c *gin.Context) {
	cid, ok := customerID(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.MySubscription(c.Request.Context(), cid, id)
	reply(c, out, err)
}

// POST /me/subscriptions/:id/{pause|resume|close|renew}
func (h *SalesHandler) MyAction(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, ok := customerID(c)
		if !ok {
			return
		}
		id, ok := pathID(c)
		if !ok {
			return
		}
		out, err := h.svc.MyAction(c.Request.Context(), cid, middleware.CurrentUserID(c), id, action)
		reply(c, out, err)
	}
}

// GET /me/invoices
func (h *SalesHandler) MyInvoices(c *gin.Context) {
	cid, ok := customerID(c)
	if !ok {
		return
	}
	p := pagination.Parse(c, "invoice_date DESC", "invoice_date", "due_date")
	list, total, err := h.svc.MyInvoices(c.Request.Context(), cid, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

// GET /me/invoices/:id
func (h *SalesHandler) MyInvoice(c *gin.Context) {
	cid, ok := customerID(c)
	if !ok {
		return
	}
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := h.svc.MyInvoice(c.Request.Context(), cid, id)
	reply(c, out, err)
}

// GET /me/payments
func (h *SalesHandler) MyPayments(c *gin.Context) {
	cid, ok := customerID(c)
	if !ok {
		return
	}
	p := pagination.Parse(c, "payment_date DESC")
	list, total, err := h.svc.MyPayments(c.Request.Context(), cid, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

// POST /me/quotations
func (h *SalesHandler) MyRequestQuotation(c *gin.Context) {
	cid, ok := customerID(c)
	if !ok {
		return
	}
	var req dto.CreateMyQuotationRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.MyRequestQuotation(c.Request.Context(), cid, middleware.CurrentUserID(c), req)
	reply(c, out, err)
}
