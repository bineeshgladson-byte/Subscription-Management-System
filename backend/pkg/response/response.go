// Package response writes the standard JSON envelope used by every endpoint:
//
//	{ "success": true,  "data": ..., "meta": {...} }
//	{ "success": false, "error": { "code": "...", "message": "...", "details": {...} } }
package response

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
)

type Envelope struct {
	Success bool          `json:"success"`
	Data    any           `json:"data,omitempty"`
	Meta    any           `json:"meta,omitempty"`
	Error   *apperr.Error `json:"error,omitempty"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data})
}

func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Envelope{Success: true, Data: data})
}

func List(c *gin.Context, data any, meta any) {
	c.JSON(http.StatusOK, Envelope{Success: true, Data: data, Meta: meta})
}

// Error maps any error to the right status. Unknown errors become a 500 and are
// logged, but their text is never sent to the client.
func Error(c *gin.Context, err error) {
	if e, ok := apperr.As(err); ok {
		c.AbortWithStatusJSON(e.Status, Envelope{Success: false, Error: e})
		return
	}
	slog.Error("unhandled error", "err", err, "path", c.FullPath(), "request_id", c.GetString("request_id"))
	c.AbortWithStatusJSON(http.StatusInternalServerError, Envelope{Success: false,
		Error: &apperr.Error{Code: "INTERNAL_ERROR", Message: "something went wrong"}})
}
