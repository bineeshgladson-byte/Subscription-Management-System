package router

import "github.com/gin-gonic/gin"

// A7 quotations (rows in `subscriptions` with status draft/quotation).
func registerQuotationRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/quotations", adminOrStaff)
	g.GET("", d.Sales.ListQuotations)
	g.POST("", d.Sales.CreateQuotation)
	g.GET("/:id", d.Sales.GetSubscription)
	g.POST("/:id/send", d.Sales.SendQuotation)
	g.POST("/:id/confirm", d.Sales.ConfirmQuotation)
	g.POST("/:id/cancel", d.Sales.CancelQuotation)
}
