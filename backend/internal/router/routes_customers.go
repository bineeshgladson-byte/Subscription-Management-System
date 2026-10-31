package router

import "github.com/gin-gonic/gin"

// Owner: Ashlin — A1 customers ("a small internal team handles customers day to day")
func registerCustomerRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/customers")
	g.GET("", adminOrStaff, d.Customers.List)
	g.POST("", adminOrStaff, d.Customers.Create)
	g.GET("/:id", adminOrStaff, d.Customers.Get)
	g.PUT("/:id", adminOrStaff, d.Customers.Update)
	g.DELETE("/:id", adminOnly, d.Customers.Archive)
}
