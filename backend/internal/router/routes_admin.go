package router

import "github.com/gin-gonic/gin"

// Admin system operations.
func registerAdminRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/admin", adminOnly)
	// Body: {"as_of": "2026-12-01"} — as_of only honoured when APP_ENV=development.
	g.POST("/billing/run", todo("run billing"))
}
