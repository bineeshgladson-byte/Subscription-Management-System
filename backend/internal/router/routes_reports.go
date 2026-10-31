package router

import "github.com/gin-gonic/gin"

// A11 dashboard. Staff get counts only; money fields are admin-only (null for staff).
func registerReportRoutes(rg *gin.RouterGroup, d Deps) {
	rg.GET("/dashboard/summary", adminOrStaff, d.Sales.Dashboard)
}
