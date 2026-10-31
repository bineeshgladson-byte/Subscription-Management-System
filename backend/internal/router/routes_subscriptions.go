package router

import "github.com/gin-gonic/gin"

// A8 subscription lifecycle.
func registerSubscriptionRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/subscriptions", adminOrStaff)
	g.GET("", d.Sales.ListSubscriptions)
	g.GET("/:id", d.Sales.GetSubscription)
	g.POST("/:id/pause", d.Sales.Pause)
	g.POST("/:id/resume", d.Sales.Resume)
	g.POST("/:id/close", d.Sales.Close)
	g.POST("/:id/renew", d.Sales.Renew)
	g.POST("/:id/bill-now", d.Sales.BillNow)
}
