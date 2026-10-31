package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

type InvoiceHandler struct {
	svc     *service.InvoiceService
	paySvc  *service.PaymentService
}

func NewInvoiceHandler(svc *service.InvoiceService, paySvc *service.PaymentService) *InvoiceHandler {
	return &InvoiceHandler{svc: svc, paySvc: paySvc}
}

func (h *InvoiceHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "id DESC", "id", "created_at", "total", "invoice_date", "due_date")
	f := repository.InvoiceFilter{
		Search: c.Query("q"),
		Today:  time.Now(),
	}
	if st := c.Query("status"); st != "" && st != "all" {
		f.Status = &st
	}
	if ov := c.Query("overdue"); ov == "true" {
		isOv := true
		f.Overdue = &isOv
	}
	if custIDStr := c.Query("customer_id"); custIDStr != "" {
		if cid, err := strconv.ParseInt(custIDStr, 10, 64); err == nil && cid > 0 {
			f.CustomerID = &cid
		}
	}
	if subIDStr := c.Query("subscription_id"); subIDStr != "" {
		if sid, err := strconv.ParseInt(subIDStr, 10, 64); err == nil && sid > 0 {
			f.SubscriptionID = &sid
		}
	}

	invs, total, err := h.svc.List(f, p)
	if err != nil {
		response.Error(c, err)
		return
	}

	out := make([]dto.InvoiceResponse, len(invs))
	for i := range invs {
		out[i] = *dto.NewInvoiceResponse(&invs[i])
	}
	response.List(c, out, p.Meta(total))
}

func (h *InvoiceHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	inv, err := h.svc.FindByID(id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewInvoiceResponse(inv))
}

func (h *InvoiceHandler) Confirm(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	inv, err := h.svc.ConfirmInvoice(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewInvoiceResponse(inv))
}

func (h *InvoiceHandler) Cancel(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.CancelInvoiceRequest
	_ = c.ShouldBindJSON(&req)

	inv, err := h.svc.CancelInvoice(c.Request.Context(), id, req.Reason)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewInvoiceResponse(inv))
}

func (h *InvoiceHandler) Send(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	inv, err := h.svc.SendInvoice(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, dto.NewInvoiceResponse(inv))
}

func (h *InvoiceHandler) PDF(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	bytes, filename, err := h.svc.GeneratePDF(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	c.Header("Content-Disposition", "inline; filename="+filename)
	c.Data(http.StatusOK, "text/html; charset=utf-8", bytes)
}

func (h *InvoiceHandler) RecordPayment(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.RecordPaymentRequest
	if !bind(c, &req) {
		return
	}

	userID := middleware.CurrentUserID(c)
	var uPtr *int64
	if userID > 0 {
		uPtr = &userID
	}

	p, err := h.paySvc.RecordPayment(c.Request.Context(), uPtr, id, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	inv, _ := h.svc.FindByID(id)
	invNum := ""
	custName := ""
	if inv != nil {
		invNum = inv.InvoiceNumber
		custName = inv.CustomerName
	}

	response.Created(c, dto.NewPaymentResponse(p, invNum, custName))
}

func (h *InvoiceHandler) RazorpayOrder(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	order, err := h.paySvc.CreateRazorpayOrder(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, order)
}

func (h *InvoiceHandler) RazorpayVerify(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.VerifyRazorpayRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.paySvc.VerifyRazorpayPayment(c.Request.Context(), id, req)
	if err != nil {
		response.Error(c, err)
		return
	}

	inv, _ := h.svc.FindByID(id)
	invNum := ""
	custName := ""
	if inv != nil {
		invNum = inv.InvoiceNumber
		custName = inv.CustomerName
	}
	response.OK(c, gin.H{
		"payment": dto.NewPaymentResponse(p, invNum, custName),
		"invoice": dto.NewInvoiceResponse(inv),
	})
}
