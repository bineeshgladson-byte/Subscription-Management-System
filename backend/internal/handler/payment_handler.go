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

type PaymentHandler struct {
	svc    *service.PaymentService
	invSvc *service.InvoiceService
}

func NewPaymentHandler(svc *service.PaymentService, invSvc *service.InvoiceService) *PaymentHandler {
	return &PaymentHandler{svc: svc, invSvc: invSvc}
}

func (h *PaymentHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "id DESC", "id", "created_at", "amount", "payment_date")
	f := repository.PaymentFilter{
		Search: c.Query("q"),
	}
	if invIDStr := c.Query("invoice_id"); invIDStr != "" {
		if iid, err := strconv.ParseInt(invIDStr, 10, 64); err == nil && iid > 0 {
			f.InvoiceID = &iid
		}
	}
	if custIDStr := c.Query("customer_id"); custIDStr != "" {
		if cid, err := strconv.ParseInt(custIDStr, 10, 64); err == nil && cid > 0 {
			f.CustomerID = &cid
		}
	}

	payments, total, err := h.svc.List(f, p)
	if err != nil {
		response.Error(c, err)
		return
	}

	out := make([]dto.PaymentResponse, len(payments))
	for i := range payments {
		invNum := ""
		custName := ""
		if h.invSvc != nil {
			if inv, _ := h.invSvc.FindByID(payments[i].InvoiceID); inv != nil {
				invNum = inv.InvoiceNumber
				custName = inv.CustomerName
			}
		}
		out[i] = *dto.NewPaymentResponse(&payments[i], invNum, custName)
	}
	response.List(c, out, p.Meta(total))
}

func (h *PaymentHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	p, err := h.svc.FindByID(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	invNum := ""
	custName := ""
	if h.invSvc != nil {
		if inv, _ := h.invSvc.FindByID(p.InvoiceID); inv != nil {
			invNum = inv.InvoiceNumber
			custName = inv.CustomerName
		}
	}
	response.OK(c, dto.NewPaymentResponse(p, invNum, custName))
}

func (h *PaymentHandler) Refund(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	p, err := h.svc.RefundPayment(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	invNum := ""
	custName := ""
	if h.invSvc != nil {
		if inv, _ := h.invSvc.FindByID(p.InvoiceID); inv != nil {
			invNum = inv.InvoiceNumber
			custName = inv.CustomerName
		}
	}
	response.OK(c, dto.NewPaymentResponse(p, invNum, custName))
}
