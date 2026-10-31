package router

import "github.com/gin-gonic/gin"

// Owner: Backend 1 — Module 1 (auth)
func registerPublicAuthRoutes(rg *gin.RouterGroup, d Deps) {
	rg.POST("/auth/register", d.Auth.Register)
	rg.POST("/auth/login", d.Auth.Login)
	rg.POST("/auth/forgot-password", todo("forgot password"))
	rg.POST("/auth/reset-password", todo("reset password"))
}

func registerAuthRoutes(rg *gin.RouterGroup, d Deps) {
	rg.GET("/auth/me", anyLoggedInUser, d.Auth.Me)
	rg.PUT("/auth/me", anyLoggedInUser, todo("update profile"))
	rg.POST("/auth/change-password", anyLoggedInUser, todo("change password"))
}
