package router

import "github.com/gin-gonic/gin"

// Owner: Backend 2 — Module 6
func registerInvoiceRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/invoices", adminOrStaff)
	g.GET("", todo("list invoices"))
	g.GET("/:id", todo("get invoice"))
	g.POST("/:id/confirm", todo("confirm invoice"))
	g.POST("/:id/cancel", todo("cancel invoice"))
	g.POST("/:id/send", todo("send invoice"))
	g.GET("/:id/pdf", todo("invoice pdf"))
	// Payments are recorded against an invoice (Module 8).
	g.POST("/:id/payments", todo("record payment"))
}
