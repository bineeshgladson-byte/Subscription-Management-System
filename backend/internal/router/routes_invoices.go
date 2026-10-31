package router

import "github.com/gin-gonic/gin"

// A9 invoices + A10 recording payments (Gayatri's PaymentHandler).
func registerInvoiceRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/invoices", adminOrStaff)
	g.GET("", d.Sales.ListInvoices)
	g.GET("/:id", d.Sales.GetInvoice)
	g.POST("/:id/confirm", d.Sales.ConfirmInvoice)
	g.POST("/:id/cancel", d.Sales.CancelInvoice)
	g.POST("/:id/send", d.Sales.SendInvoice)
	g.POST("/:id/payments", d.Payment.RecordPayment)
}
