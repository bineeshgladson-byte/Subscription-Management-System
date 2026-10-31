package service

import (
	"context"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
)

type MeService struct {
	db            *gorm.DB
	subscriptions *repository.SubscriptionRepo
}

func NewMeService(
	db *gorm.DB,
	subscriptions *repository.SubscriptionRepo,
) *MeService {
	return &MeService{
		db:            db,
		subscriptions: subscriptions,
	}
}

func (s *MeService) ListSubscriptions(
	ctx context.Context,
	customerID int64,
) ([]models.Subscription, error) {
	return s.subscriptions.FindByCustomerID(
		ctx,
		s.db.WithContext(ctx),
		customerID,
	)
}
