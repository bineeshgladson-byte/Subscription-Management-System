package router

import "github.com/gin-gonic/gin"

// Owner: Backend 2 — Module 4. Quotations are rows in `subscriptions`
// with status draft/quotation.
func registerQuotationRoutes(rg *gin.RouterGroup, d Deps) {
	g := rg.Group("/quotations", adminOrStaff)
	g.GET("", todo("list quotations"))
	g.POST("", todo("create quotation"))
	g.GET("/:id", todo("get quotation"))
	g.PUT("/:id", todo("update quotation"))
	g.POST("/:id/send", todo("send quotation"))
	g.POST("/:id/confirm", todo("confirm quotation"))
	g.POST("/:id/cancel", todo("cancel quotation"))
}
