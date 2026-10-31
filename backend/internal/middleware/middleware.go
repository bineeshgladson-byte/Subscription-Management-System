// Package middleware holds cross-cutting HTTP concerns: request IDs, logging,
// CORS, JWT authentication and role checks (RBAC).
package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/jwtutil"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

const (
	CtxUserID     = "user_id"
	CtxRole       = "role"
	CtxCustomerID = "customer_id"
)

// RequestID tags every request so a log line can be matched to a client error.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			b := make([]byte, 8)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.Info("http",
			"method", c.Request.Method, "path", c.Request.URL.Path, "status", c.Writer.Status(),
			"ms", time.Since(start).Milliseconds(), "request_id", c.GetString("request_id"))
	}
}

func CORS(allowedOrigins []string) gin.HandlerFunc {
	return cors.New(cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type", "Idempotency-Key", "X-Request-ID"},
		ExposeHeaders:    []string{"X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	})
}

// Auth requires a valid "Authorization: Bearer <jwt>" header and stores the
// user id, role and (for customers) customer id in the Gin context.
func Auth(jwt *jwtutil.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || token == "" {
			response.Error(c, apperr.Unauthenticated("missing bearer token"))
			return
		}
		claims, err := jwt.Parse(token)
		if err != nil {
			response.Error(c, apperr.Unauthenticated("invalid or expired token"))
			return
		}
		c.Set(CtxUserID, claims.UserID())
		c.Set(CtxRole, models.Role(claims.Role))
		if claims.CustomerID != nil {
			c.Set(CtxCustomerID, *claims.CustomerID)
		}
		c.Next()
	}
}

// RequireRoles must come after Auth. This is the real security boundary:
// hiding a button in Flutter is only a convenience.
func RequireRoles(roles ...models.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := CurrentRole(c)
		for _, r := range roles {
			if r == role {
				c.Next()
				return
			}
		}
		response.Error(c, apperr.Forbidden("you do not have access to this resource"))
	}
}

func CurrentUserID(c *gin.Context) int64 { return c.GetInt64(CtxUserID) }

func CurrentRole(c *gin.Context) models.Role {
	r, _ := c.Get(CtxRole)
	role, _ := r.(models.Role)
	return role
}

// CurrentCustomerID is used by every /me/* handler. Never read a customer id
// from the URL or body for customer-facing endpoints.
func CurrentCustomerID(c *gin.Context) (int64, bool) {
	v, ok := c.Get(CtxCustomerID)
	if !ok {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok
}
