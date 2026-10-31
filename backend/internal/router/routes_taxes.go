package router

import "github.com/gin-gonic/gin"

// Owner: Ashlin — A2 taxes ("Taxes are calculated automatically from rules the admin has set")
func registerTaxRoutes(rg *gin.RouterGroup, d Deps) {
	t := rg.Group("/taxes")
	t.GET("", adminOrStaff, d.Taxes.List)
	t.GET("/:id", adminOrStaff, d.Taxes.Get)
	t.POST("", adminOnly, d.Taxes.Create)
	t.PUT("/:id", adminOnly, d.Taxes.Update)
	t.DELETE("/:id", adminOnly, d.Taxes.Deactivate) // deactivates, never deletes
}
