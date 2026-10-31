package router

import "github.com/gin-gonic/gin"

func registerQuotationRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/quotations", adminOrStaff)
	g.GET("", d.Quotations.List)
	g.POST("", d.Quotations.Create)
	g.GET("/:id", d.Quotations.Get)
	g.POST("/:id/send", d.Quotations.Send)
	g.POST("/:id/confirm", d.Quotations.Confirm)
	g.POST("/:id/cancel", d.Quotations.Cancel)
}
