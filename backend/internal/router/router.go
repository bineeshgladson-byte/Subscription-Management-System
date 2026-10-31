// Package router wires URLs to handlers. Each module has its OWN routes_*.go file
// so developers working on different modules don't edit the same file.
//
// Every endpoint from the architecture plan is already registered with the right
// roles. Routes still pointing at handler.NotImplemented return HTTP 501;
// replace them with real handlers as you build each module.
package router

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/config"
	"github.com/brightloop/subscriptions/backend/internal/handler"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/jwtutil"
)

// Deps carries everything route files need. Add new handlers here as modules land.
type Deps struct {
	Config config.Config
	DB     *gorm.DB
	JWT    *jwtutil.Manager

	Sessions middleware.SessionChecker

	Auth  *handler.AuthHandler
	Staff *handler.StaffHandler
}

// Role guards, reused by every routes_*.go file.
var (
	adminOnly       = middleware.RequireRoles(models.RoleAdmin)
	adminOrStaff    = middleware.RequireRoles(models.RoleAdmin, models.RoleStaff)
	customerOnly    = middleware.RequireRoles(models.RoleCustomer)
	anyLoggedInUser = middleware.RequireRoles(models.RoleAdmin, models.RoleStaff, models.RoleCustomer)
)

var todo = handler.NotImplemented

func New(d Deps) *gin.Engine {
	if !d.Config.IsDev() {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.New()
	r.Use(middleware.RequestID(), middleware.Logger(), gin.Recovery(), middleware.CORS(d.Config.CORSAllowedOrigins))

	r.GET("/health", handler.Health(d.DB))

	api := r.Group("/api/v1")
	api.GET("/health", handler.Health(d.DB))

	// Public routes (no token).
	registerPublicAuthRoutes(api, d)

	// Everything below requires a valid JWT.
	authed := api.Group("")
	authed.Use(middleware.Auth(d.JWT, d.Sessions))

	registerAuthRoutes(authed, d)
	registerStaffRoutes(authed, d)
	registerCustomerRoutes(authed, d)
	registerCatalogRoutes(authed, d)
	registerPlanRoutes(authed, d)
	registerQuotationRoutes(authed, d)
	registerSubscriptionRoutes(authed, d)
	registerInvoiceRoutes(authed, d)
	registerDiscountRoutes(authed, d)
	registerPaymentRoutes(authed, d)
	registerReportRoutes(authed, d)
	registerMeRoutes(authed, d)
	registerAdminRoutes(authed, d)

	return r
}
