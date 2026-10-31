package router

import "github.com/gin-gonic/gin"

// Owner: Backend 2 — Module 5
func registerSubscriptionRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/subscriptions", adminOrStaff)

	g.GET("", d.Subscription.List)
	g.GET("/:id", d.Subscription.Get)

	g.POST("/:id/activate", d.Subscription.Activate)
	g.POST("/:id/pause", d.Subscription.Pause)
	g.POST("/:id/resume", d.Subscription.Resume)
	g.POST("/:id/close", d.Subscription.Close)
}
