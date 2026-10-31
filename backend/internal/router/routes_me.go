package router

import "github.com/gin-gonic/gin"

// Customer self-service. Handlers MUST take the customer id from
// middleware.CurrentCustomerID(c), never from the URL or body, and return
// 404 for records that belong to someone else.
func registerMeRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/me", customerOnly)
	g.GET("/dashboard", todo("my dashboard"))

	g.GET("/subscriptions", todo("my subscriptions"))
	g.POST("/subscriptions", todo("request a plan"))
	g.GET("/subscriptions/:id", todo("my subscription"))
	g.POST("/subscriptions/:id/accept", todo("accept quotation"))
	g.POST("/subscriptions/:id/pause", todo("pause my subscription"))
	g.POST("/subscriptions/:id/resume", todo("resume my subscription"))
	g.POST("/subscriptions/:id/close", todo("close my subscription"))
	g.POST("/subscriptions/:id/renew", todo("renew my subscription"))

	g.GET("/invoices", todo("my invoices"))
	g.GET("/invoices/:id", todo("my invoice"))
	g.GET("/invoices/:id/pdf", todo("my invoice pdf"))
	g.POST("/invoices/:id/pay", todo("simulated payment"))

	g.GET("/payments", todo("my payments"))
}
