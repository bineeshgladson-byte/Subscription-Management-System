package router

import "github.com/gin-gonic/gin"

// Owner: Backend 1 — Module 1 (auth)
func registerPublicAuthRoutes(rg *gin.RouterGroup, d Deps) {
	rg.POST("/auth/register", d.Auth.Register)
	rg.POST("/auth/login", d.Auth.Login)
	rg.POST("/auth/forgot-password", d.Auth.ForgotPassword)
	rg.POST("/auth/reset-password", d.Auth.ResetPassword)
}

func registerAuthRoutes(rg *gin.RouterGroup, d Deps) {
	rg.GET("/auth/me", anyLoggedInUser, d.Auth.Me)
	rg.PUT("/auth/me", anyLoggedInUser, d.Auth.UpdateMe)
	rg.POST("/auth/change-password", anyLoggedInUser, d.Auth.ChangePassword)
}
