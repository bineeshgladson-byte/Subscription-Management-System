package router

import "github.com/gin-gonic/gin"

func registerSubscriptionRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/subscriptions", adminOrStaff)
	g.GET("", d.Subscriptions.List)
	g.GET("/:id", d.Subscriptions.Get)
	g.POST("/:id/activate", d.Subscriptions.Activate)
	g.POST("/:id/pause", d.Subscriptions.Pause)
	g.POST("/:id/resume", d.Subscriptions.Resume)
	g.POST("/:id/close", d.Subscriptions.Close)
	g.POST("/:id/renew", d.Subscriptions.Renew)
	g.POST("/:id/bill-now", d.Subscriptions.BillNow)
}
