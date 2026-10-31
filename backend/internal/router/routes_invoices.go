package router

import "github.com/gin-gonic/gin"

func registerInvoiceRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/invoices", adminOrStaff)
	g.GET("", d.Invoices.List)
	g.GET("/:id", d.Invoices.Get)
	g.POST("/:id/confirm", d.Invoices.Confirm)
	g.POST("/:id/cancel", d.Invoices.Cancel)
	g.POST("/:id/send", d.Invoices.Send)
	g.GET("/:id/pdf", d.Invoices.PDF)
	g.POST("/:id/payments", d.Invoices.RecordPayment)

	// Razorpay online checkout endpoints for invoices
	g.POST("/:id/razorpay/order", d.Invoices.RazorpayOrder)
	g.POST("/:id/razorpay/verify", d.Invoices.RazorpayVerify)
}
