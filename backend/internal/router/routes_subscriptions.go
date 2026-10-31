package router

import "github.com/gin-gonic/gin"

// Owner: Backend 2 — Module 5
func registerSubscriptionRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/subscriptions", adminOrStaff)
	g.GET("", todo("list subscriptions"))
	g.GET("/:id", todo("get subscription"))
	g.POST("/:id/activate", todo("activate subscription"))
	g.POST("/:id/pause", todo("pause subscription"))
	g.POST("/:id/resume", todo("resume subscription"))
	g.POST("/:id/close", todo("close subscription"))
	g.POST("/:id/renew", todo("renew subscription"))
	g.POST("/:id/bill-now", todo("bill subscription now"))
}
