package router

import "github.com/gin-gonic/gin"

// Owner: Backend 1 — Module 3 (plans) + quotation templates
func registerPlanRoutes(rg *gin.RouterGroup, d Deps) {
	p := rg.Group("/plans")
	// Customers may list plans too; the service must filter to active + available ones for them.
	p.GET("", anyLoggedInUser, todo("list plans"))
	p.GET("/:id", anyLoggedInUser, todo("get plan"))
	p.POST("", adminOnly, todo("create plan"))
	p.PUT("/:id", adminOnly, todo("update plan"))
	p.PATCH("/:id/status", adminOnly, todo("archive/reactivate plan"))

	t := rg.Group("/quotation-templates")
	t.GET("", adminOrStaff, todo("list quotation templates"))
	t.GET("/:id", adminOrStaff, todo("get quotation template"))
	t.POST("", adminOnly, todo("create quotation template"))
	t.PUT("/:id", adminOnly, todo("update quotation template"))
	t.DELETE("/:id", adminOnly, todo("delete quotation template"))
}
