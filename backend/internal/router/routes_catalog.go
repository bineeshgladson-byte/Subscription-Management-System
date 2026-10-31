package router

import "github.com/gin-gonic/gin"

// Owner: Ashlin — A3 products and variants (taxes live in routes_taxes.go)
func registerCatalogRoutes(rg *gin.RouterGroup, d Deps) {
	p := rg.Group("/products")
	p.GET("", adminOrStaff, d.Products.List)
	p.GET("/:id", adminOrStaff, d.Products.Get)
	p.POST("", adminOnly, d.Products.Create)
	p.PUT("/:id", adminOnly, d.Products.Update)
	p.DELETE("/:id", adminOnly, d.Products.Archive) // archives, never deletes
	p.POST("/:id/variants", adminOnly, d.Products.AddVariant)
	p.PUT("/:id/variants/:vid", adminOnly, d.Products.UpdateVariant)
	p.DELETE("/:id/variants/:vid", adminOnly, d.Products.DeactivateVariant) // deactivates
}
