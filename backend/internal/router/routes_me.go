package router

import "github.com/gin-gonic/gin"

// Customer portal. The customer id comes from the login token only; other
// people's records return 404. Brief: customers "see only their own
// subscriptions and invoices" and "manage their own subscriptions".
func registerMeRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/me", customerOnly)
	g.POST("/quotations", d.Sales.MyRequestQuotation)
	g.GET("/subscriptions", d.Sales.MySubscriptions)
	g.GET("/subscriptions/:id", d.Sales.MySubscription)
	g.POST("/subscriptions/:id/pause", d.Sales.MyAction("pause"))
	g.POST("/subscriptions/:id/resume", d.Sales.MyAction("resume"))
	g.POST("/subscriptions/:id/close", d.Sales.MyAction("close"))
	g.POST("/subscriptions/:id/renew", d.Sales.MyAction("renew"))
	g.GET("/invoices", d.Sales.MyInvoices)
	g.GET("/invoices/:id", d.Sales.MyInvoice)
	g.GET("/payments", d.Sales.MyPayments)
}
