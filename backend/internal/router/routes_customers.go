package router

import "github.com/gin-gonic/gin"

// Owner: Backend 1 — customers
func registerCustomerRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/customers")
	g.GET("", adminOrStaff, todo("list customers"))
	g.POST("", adminOrStaff, todo("create customer"))
	g.GET("/:id", adminOrStaff, todo("get customer"))
	g.PUT("/:id", adminOrStaff, todo("update customer"))
	g.DELETE("/:id", adminOnly, todo("archive customer"))
}
