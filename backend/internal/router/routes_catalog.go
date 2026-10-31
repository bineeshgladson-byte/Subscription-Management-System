package router

import "github.com/gin-gonic/gin"

// Owner: Ashlin — A3 products and variants (taxes live in routes_taxes.go)
func registerCatalogRoutes(rg *gin.RouterGroup, d Deps) {
	p := rg.Group("/products")
	p.GET("", adminOrStaff, todo("list products"))
	p.GET("/:id", adminOrStaff, todo("get product"))
	p.POST("", adminOnly, todo("create product"))
	p.PUT("/:id", adminOnly, todo("update product"))
	p.DELETE("/:id", adminOnly, todo("archive product"))
	p.POST("/:id/variants", adminOnly, todo("create variant"))
	p.PUT("/:id/variants/:vid", adminOnly, todo("update variant"))
	p.DELETE("/:id/variants/:vid", adminOnly, todo("delete variant"))
}
