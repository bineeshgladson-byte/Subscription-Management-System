package handler

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/config"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// SalesHandler: quotations, subscriptions, invoices, payments list, dashboard, billing run.
type SalesHandler struct {
	svc   *service.SalesService
	cfg   config.Config
	clock clock.Clock
}

func NewSalesHandler(svc *service.SalesService, cfg config.Config, clk clock.Clock) *SalesHandler {
	return &SalesHandler{svc: svc, cfg: cfg, clock: clk}
}

func reply[T any](c *gin.Context, out T, err error) {
	if err != nil {
		response.Error(c, err)
		return
	}
	response.OK(c, out)
}

func queryID(c *gin.Context, name string, errs map[string]string) int64 {
	v := c.Query(name)
	if v == "" {
		return 0
	}
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		errs[name] = "must be a positive number"
	}
	return id
}

// ---------- quotations ----------

// GET /quotations?q=&status=draft|quotation&customer_id=
func (h *SalesHandler) ListQuotations(c *gin.Context) {
	h.list(c, []string{"draft", "quotation"})
}

// GET /subscriptions?q=&status=&customer_id=
func (h *SalesHandler) ListSubscriptions(c *gin.Context) {
	h.list(c, []string{"confirmed", "active", "paused", "closed", "renewed"})
}

func (h *SalesHandler) list(c *gin.Context, allowed []string) {
	p := pagination.Parse(c, "created_at DESC", "created_at", "start_date", "next_billing_date")
	errs := map[string]string{}
	f := service.SubscriptionFilter{Search: c.Query("q"), Statuses: allowed, CustomerID: queryID(c, "customer_id", errs)}
	if st := c.Query("status"); st != "" {
		ok := false
		for _, a := range allowed {
			ok = ok || a == st
		}
		if !ok {
			errs["status"] = "must be one of " + strings.Join(allowed, ", ")
		}
		f.Statuses = []string{st}
	}
	if len(errs) > 0 {
		response.Error(c, apperr.Validation("Invalid filter", errs))
		return
	}
	list, total, err := h.svc.ListSubscriptions(c.Request.Context(), f, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

func (h *SalesHandler) CreateQuotation(c *gin.Context) {
	var req dto.CreateQuotationRequest
	if !bind(c, &req) {
		return
	}
	out, err := h.svc.CreateQuotation(c.Request.Context(), middleware.CurrentUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, out)
}

func (h *SalesHandler) byID(c *gin.Context, fn func(id int64) (*dto.SubscriptionResponse, error)) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := fn(id)
	reply(c, out, err)
}

func (h *SalesHandler) GetSubscription(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) {
		return h.svc.GetSubscription(c.Request.Context(), id)
	})
}
func (h *SalesHandler) SendQuotation(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) { return h.svc.SendQuotation(c.Request.Context(), id) })
}
func (h *SalesHandler) ConfirmQuotation(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) {
		return h.svc.ConfirmQuotation(c.Request.Context(), id)
	})
}
func (h *SalesHandler) CancelQuotation(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) {
		return h.svc.CancelQuotation(c.Request.Context(), id)
	})
}
func (h *SalesHandler) Pause(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) { return h.svc.Pause(c.Request.Context(), id) })
}
func (h *SalesHandler) Resume(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) { return h.svc.Resume(c.Request.Context(), id) })
}
func (h *SalesHandler) Close(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) { return h.svc.Close(c.Request.Context(), id) })
}
func (h *SalesHandler) Renew(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) {
		return h.svc.Renew(c.Request.Context(), middleware.CurrentUserID(c), id)
	})
}
func (h *SalesHandler) BillNow(c *gin.Context) {
	h.byID(c, func(id int64) (*dto.SubscriptionResponse, error) { return h.svc.BillNow(c.Request.Context(), id) })
}

// ---------- invoices ----------

// GET /invoices?q=&status=&overdue=true&customer_id=
func (h *SalesHandler) ListInvoices(c *gin.Context) {
	p := pagination.Parse(c, "invoice_date DESC", "invoice_date", "due_date", "total", "created_at")
	errs := map[string]string{}
	f := service.InvoiceFilter{Search: c.Query("q"), Status: c.Query("status"), CustomerID: queryID(c, "customer_id", errs)}
	switch f.Status {
	case "", "draft", "confirmed", "paid", "cancelled":
	default:
		errs["status"] = "must be draft, confirmed, paid or cancelled"
	}
	if v := c.Query("overdue"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			errs["overdue"] = "must be true or false"
		}
		f.Overdue = b
	}
	if len(errs) > 0 {
		response.Error(c, apperr.Validation("Invalid filter", errs))
		return
	}
	list, total, err := h.svc.ListInvoices(c.Request.Context(), f, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

func (h *SalesHandler) invoice(c *gin.Context, fn func(id int64) (*dto.InvoiceResponse, error)) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	out, err := fn(id)
	reply(c, out, err)
}

func (h *SalesHandler) GetInvoice(c *gin.Context) {
	h.invoice(c, func(id int64) (*dto.InvoiceResponse, error) { return h.svc.GetInvoice(c.Request.Context(), id) })
}
func (h *SalesHandler) ConfirmInvoice(c *gin.Context) {
	h.invoice(c, func(id int64) (*dto.InvoiceResponse, error) { return h.svc.ConfirmInvoice(c.Request.Context(), id) })
}
func (h *SalesHandler) SendInvoice(c *gin.Context) {
	h.invoice(c, func(id int64) (*dto.InvoiceResponse, error) { return h.svc.SendInvoice(c.Request.Context(), id) })
}
func (h *SalesHandler) CancelInvoice(c *gin.Context) {
	var req dto.CancelInvoiceRequest
	_ = c.ShouldBindJSON(&req) // body optional
	h.invoice(c, func(id int64) (*dto.InvoiceResponse, error) {
		return h.svc.CancelInvoice(c.Request.Context(), id, req.Reason)
	})
}

// ---------- payments list ----------

// GET /payments?q=&method=&customer_id=&invoice_id=
func (h *SalesHandler) ListPayments(c *gin.Context) {
	p := pagination.Parse(c, "payment_date DESC")
	errs := map[string]string{}
	f := service.PaymentFilter{Search: c.Query("q"), Method: c.Query("method"),
		CustomerID: queryID(c, "customer_id", errs), InvoiceID: queryID(c, "invoice_id", errs)}
	switch f.Method {
	case "", "upi", "bank_transfer", "card", "cash":
	default:
		errs["method"] = "must be upi, bank_transfer, card or cash"
	}
	if len(errs) > 0 {
		response.Error(c, apperr.Validation("Invalid filter", errs))
		return
	}
	list, total, err := h.svc.ListPayments(c.Request.Context(), f, p)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.List(c, list, p.Meta(total))
}

// ---------- dashboard + billing run ----------

// GET /dashboard/summary?from=&to=  (default: this month)
func (h *SalesHandler) Dashboard(c *gin.Context) {
	t := h.clock.Today()
	from := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	errs := map[string]string{}
	if v := c.Query("from"); v != "" {
		d, ok := service.ParseDate(v)
		if !ok {
			errs["from"] = "must be a date"
		}
		from = d
	}
	if v := c.Query("to"); v != "" {
		d, ok := service.ParseDate(v)
		if !ok {
			errs["to"] = "must be a date"
		}
		to = d
	}
	if len(errs) > 0 {
		response.Error(c, apperr.Validation("Invalid filter", errs))
		return
	}
	out, err := h.svc.Dashboard(c.Request.Context(), middleware.CurrentRole(c), from, to)
	reply(c, out, err)
}

// POST /admin/billing/run {"as_of": "2026-12-01"} — as_of only in development.
func (h *SalesHandler) RunBilling(c *gin.Context) {
	var req dto.BillingRunRequest
	_ = c.ShouldBindJSON(&req)
	t := h.clock.Today()
	asOf := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	if req.AsOf != "" {
		if !h.cfg.IsDev() {
			response.Error(c, apperr.Forbidden("as_of is only allowed in development"))
			return
		}
		d, ok := service.ParseDate(req.AsOf)
		if !ok {
			response.Error(c, apperr.Validation("Invalid date", map[string]string{"as_of": "must be a date like 2026-12-01"}))
			return
		}
		asOf = d
	}
	response.OK(c, h.svc.RunBilling(c.Request.Context(), asOf))
}
