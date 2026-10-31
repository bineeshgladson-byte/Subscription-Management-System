package router

import "github.com/gin-gonic/gin"

// Owner: Backend 1 — staff accounts (admin only)
func registerStaffRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/staff", adminOnly)
	g.GET("", todo("list staff"))
	g.POST("", todo("create staff"))
	g.GET("/:id", todo("get staff"))
	g.PUT("/:id", todo("update staff"))
	g.PATCH("/:id/status", todo("activate/deactivate staff"))
}
