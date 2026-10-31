package router

import "github.com/gin-gonic/gin"

// Owner: Backend 2 — Module 8 (recording lives in routes_invoices.go)
func registerPaymentRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/payments")
	g.GET("", adminOrStaff, todo("list payments"))
	g.GET("/:id", adminOrStaff, todo("get payment"))
	g.POST("/:id/refund", adminOnly, todo("refund payment"))
}
