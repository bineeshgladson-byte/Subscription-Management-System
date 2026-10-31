package router

import "github.com/gin-gonic/gin"

// Owner: Ashlin — A6 discounts ("Only the admin can create discount rules") + pricing engine
func registerDiscountRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/discounts", adminOnly)
	g.GET("", d.Discounts.List)
	g.POST("", d.Discounts.Create)
	g.GET("/:id", d.Discounts.Get)
	g.PUT("/:id", d.Discounts.Update)
	g.DELETE("/:id", d.Discounts.Deactivate) // deactivates, keeps usage history
	g.GET("/:id/usages", d.Discounts.Usages)

	// Prices catalogue lines with today's taxes and discounts (staff build quotations with it).
	rg.POST("/pricing/preview", adminOrStaff, d.Discounts.Preview)
}
