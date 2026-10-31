package router

import "github.com/gin-gonic/gin"

// Owner: Backend 2 — Module 9. Staff get a reduced dashboard (no MRR/ARR/revenue);
// the service decides what to include based on the caller's role.
func registerReportRoutes(rg *gin.RouterGroup, d Deps) {
	rg.GET("/dashboard/summary", adminOrStaff, todo("dashboard summary"))
	rg.GET("/reports/revenue", adminOnly, todo("revenue report"))
	rg.GET("/reports/subscriptions", adminOrStaff, todo("subscription report"))
	rg.GET("/reports/overdue", adminOrStaff, todo("overdue report"))
}
