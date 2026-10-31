package router

import "github.com/gin-gonic/gin"

// Owner: Backend 1 — Module 2 (products, variants, taxes)
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

	t := rg.Group("/taxes")
	t.GET("", adminOrStaff, todo("list taxes"))
	t.POST("", adminOnly, todo("create tax"))
	t.PUT("/:id", adminOnly, todo("update tax"))
	t.DELETE("/:id", adminOnly, todo("deactivate tax"))
}
