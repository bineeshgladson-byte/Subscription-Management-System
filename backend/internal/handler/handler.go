// Package handler converts HTTP ⇄ service calls. A handler should only:
// bind + validate input, read the current user, call ONE service method, respond.
package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

// bind parses the JSON body into dst and turns binding errors into a
// VALIDATION_ERROR with per-field details Flutter can show under each input.
func bind(c *gin.Context, dst any) bool {
	if err := c.ShouldBindJSON(dst); err != nil {
		details := map[string]string{}
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			for _, fe := range ve {
				details[toSnake(fe.Field())] = validationMessage(fe)
			}
		} else {
			details["body"] = "malformed JSON"
		}
		response.Error(c, apperr.Validation("Some fields are invalid", details))
		return false
	}
	return true
}

func validationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "email":
		return "is not a valid email"
	case "min":
		return "must be at least " + fe.Param()
	case "max":
		return "must be at most " + fe.Param()
	case "oneof":
		return "must be one of: " + fe.Param()
	default:
		return "is invalid"
	}
}

func toSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			r += 'a' - 'A'
		}
		b.WriteRune(r)
	}
	return b.String()
}

// NotImplemented is a placeholder for routes that are registered (with the
// correct roles) but whose handler hasn't been written yet. Replace it.
func NotImplemented(what string) gin.HandlerFunc {
	return func(c *gin.Context) { response.Error(c, apperr.NotImplemented(what)) }
}

// Health reports whether the API and the database are reachable.
func Health(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(c.Request.Context())
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, response.Envelope{Success: false,
				Error: &apperr.Error{Code: "DB_UNAVAILABLE", Message: "database unreachable"}})
			return
		}
		response.OK(c, gin.H{"status": "ok", "database": "ok"})
	}
}

// pathID reads a positive integer ":id" path parameter, or writes a 400.
func pathID(c *gin.Context) (int64, bool) { return pathParamID(c, "id") }

// pathParamID reads any positive integer path parameter (e.g. ":vid"), or writes a 400.
func pathParamID(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.Error(c, apperr.Validation("Invalid id", map[string]string{name: "must be a positive number"}))
		return 0, false
	}
	return id, true
}
