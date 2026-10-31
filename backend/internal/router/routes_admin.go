package router

import "github.com/gin-gonic/gin"

// A12 admin system operations.
func registerAdminRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/admin", adminOnly)
	g.POST("/billing/run", d.Sales.RunBilling) // {"as_of": "2026-12-01"} only in development
}
