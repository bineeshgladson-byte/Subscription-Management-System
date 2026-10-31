package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/middleware"
	"github.com/brightloop/subscriptions/backend/internal/service"
	"github.com/brightloop/subscriptions/backend/pkg/response"
)

type MeHandler struct {
	svc *service.MeService
}

func NewMeHandler(svc *service.MeService) *MeHandler {
	return &MeHandler{svc: svc}
}

// GET /me/subscriptions
func (h *MeHandler) ListSubscriptions(c *gin.Context) {
	customerID, ok := middleware.CurrentCustomerID(c)

	if !ok || customerID <= 0 {
		response.Error(c, apperr.Forbidden("Customer access required."))
		return
	}

	subscriptions, err := h.svc.ListSubscriptions(
		c.Request.Context(),
		customerID,
	)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.OK(c, gin.H{
		"subscriptions": subscriptions,
	})
}
