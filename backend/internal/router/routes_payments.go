package router

import "github.com/gin-gonic/gin"

func registerPaymentRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/payments")
	g.GET("", adminOrStaff, d.Payments.List)
	g.GET("/:id", adminOrStaff, d.Payments.Get)
	g.POST("/:id/refund", adminOnly, d.Payments.Refund)
}
