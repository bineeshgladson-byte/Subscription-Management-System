package router

import "github.com/gin-gonic/gin"

// A10 payments list (recording lives in routes_invoices.go).
func registerPaymentRoutes(rg *gin.RouterGroup, d Deps) {
	rg.GET("/payments", adminOrStaff, d.Sales.ListPayments)
}
