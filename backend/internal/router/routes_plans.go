package router

import "github.com/gin-gonic/gin"

// Owner: Ashlin — A4 plans + A5 quotation templates
func registerPlanRoutes(rg *gin.RouterGroup, d Deps) {
	p := rg.Group("/plans")
	// Customers can read too; the service limits them to active plans available today.
	p.GET("", anyLoggedInUser, d.Plans.List)
	p.GET("/:id", anyLoggedInUser, d.Plans.Get)
	p.POST("", adminOnly, d.Plans.Create)
	p.PUT("/:id", adminOnly, d.Plans.Update)
	p.PATCH("/:id/status", adminOnly, d.Plans.SetStatus)

	// A5 quotation templates
	t := rg.Group("/quotation-templates")
	t.GET("", adminOrStaff, d.Templates.List)
	t.GET("/:id", adminOrStaff, d.Templates.Get)
	t.POST("", adminOnly, d.Templates.Create)
	t.PUT("/:id", adminOnly, d.Templates.Update)
	t.DELETE("/:id", adminOnly, d.Templates.Deactivate) // deactivates
}
