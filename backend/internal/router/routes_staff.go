package router

import "github.com/gin-gonic/gin"

// Owner: Backend 1 — staff accounts (admin only: "Only the admin can create staff accounts")
func registerStaffRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/staff", adminOnly)
	g.GET("", d.Staff.List)
	g.POST("", d.Staff.Create)
	g.GET("/:id", d.Staff.Get)
	g.PUT("/:id", d.Staff.Update)
	g.PATCH("/:id/status", d.Staff.SetStatus)
}
