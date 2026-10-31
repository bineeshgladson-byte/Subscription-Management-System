package router

import "github.com/gin-gonic/gin"

// Owner: Backend 2 — Module 7 (admin only)
func registerDiscountRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/discounts", adminOnly)
	g.GET("", todo("list discounts"))
	g.POST("", todo("create discount"))
	g.GET("/:id", todo("get discount"))
	g.PUT("/:id", todo("update discount"))
	g.DELETE("/:id", todo("deactivate discount"))
	g.GET("/:id/usages", todo("discount usages"))
}
