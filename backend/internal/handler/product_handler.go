package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// ProductHandler serves /products (read: admin + staff; write: admin).
type ProductHandler struct{ svc *service.ProductService }

func NewProductHandler(svc *service.ProductService) *ProductHandler { return &ProductHandler{svc: svc} }

// Staff see products but not cost price / margin.
func showCost(c *gin.Context) bool { return middleware.CurrentRole(c) == models.RoleAdmin }

func (h *ProductHandler) respond(c *gin.Context, p *models.Product, err error, created bool) {
	if err != nil {
		response.Error(c, err)
		return
	}
	out := dto.NewProductResponse(p, showCost(c))
	if created {
		response.Created(c, out)
	} else {
		response.OK(c, out)
	}
}

// GET /products?q=&status=active|archived|all&product_type=&tax_id=&page=&page_size=&sort=
func (h *ProductHandler) List(c *gin.Context) {
	p := pagination.Parse(c, "name ASC", "name", "sales_price", "created_at")
	f := repository.ProductFilter{Search: c.Query("q"), Status: c.Query("status"), ProductType: c.Query("product_type")}
	errs := map[string]string{}
	if f.Status != "" && f.Status != "active" && f.Status != "archived" && f.Status != "all" {
		errs["status"] = "must be active, archived or all"
	}
	if f.ProductType != "" && f.ProductType != "software" && f.ProductType != "service" && f.ProductType != "addon" {
		errs["product_type"] = "must be software, service or addon"
	}
	if v := c.Query("tax_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil || id <= 0 {
			errs["tax_id"] = "must be a positive number"
		}
		f.TaxID = id
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
	out := make([]dto.ProductResponse, len(list))
	for i := range list {
		out[i] = dto.NewProductResponse(&list[i], showCost(c))
	}
	response.List(c, out, p.Meta(total))
}

// GET /products/:id
func (h *ProductHandler) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	p, err := h.svc.Get(c.Request.Context(), id)
	h.respond(c, p, err, false)
}

// POST /products (optionally with variants)
func (h *ProductHandler) Create(c *gin.Context) {
	var req dto.CreateProductRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.svc.Create(c.Request.Context(), req)
	h.respond(c, p, err, true)
}

// PUT /products/:id
func (h *ProductHandler) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.UpdateProductRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.svc.Update(c.Request.Context(), id, req)
	h.respond(c, p, err, false)
}

// DELETE /products/:id  (archives)
func (h *ProductHandler) Archive(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	p, err := h.svc.Archive(c.Request.Context(), id)
	h.respond(c, p, err, false)
}

// POST /products/:id/variants
func (h *ProductHandler) AddVariant(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req dto.VariantInput
	if !bind(c, &req) {
		return
	}
	p, err := h.svc.AddVariant(c.Request.Context(), id, req)
	h.respond(c, p, err, true)
}

// PUT /products/:id/variants/:vid
func (h *ProductHandler) UpdateVariant(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	vid, ok := pathParamID(c, "vid")
	if !ok {
		return
	}
	var req dto.UpdateVariantRequest
	if !bind(c, &req) {
		return
	}
	p, err := h.svc.UpdateVariant(c.Request.Context(), id, vid, req)
	h.respond(c, p, err, false)
}

// DELETE /products/:id/variants/:vid  (deactivates)
func (h *ProductHandler) DeactivateVariant(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	vid, ok := pathParamID(c, "vid")
	if !ok {
		return
	}
	p, err := h.svc.DeactivateVariant(c.Request.Context(), id, vid)
	h.respond(c, p, err, false)
}
