package repository

import (
	"context"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"gorm.io/gorm"
)

type SubscriptionRepo struct{}

func NewSubscriptionRepo() *SubscriptionRepo {
	return &SubscriptionRepo{}
}

func (r *SubscriptionRepo) FindByCustomerID(
	ctx context.Context,
	db *gorm.DB,
	customerID int64,
) ([]models.Subscription, error) {
	var subscriptions []models.Subscription

	err := db.WithContext(ctx).
		Preload("Plan").
		Preload("Items").
		Where("customer_id = ?", customerID).
		Order("created_at DESC").
		Find(&subscriptions).Error

	return subscriptions, err
}
