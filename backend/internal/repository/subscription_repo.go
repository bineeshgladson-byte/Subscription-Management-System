package repository

import (
	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

type SubscriptionRepo struct{}

func NewSubscriptionRepo() *SubscriptionRepo {
	return &SubscriptionRepo{}
}

// FindByID retrieves a subscription by its ID.
func (r *SubscriptionRepo) FindByID(db *gorm.DB, id int64) (*models.Subscription, error) {
	var subscription models.Subscription

	err := db.Preload("Items").
		Preload("Customer").
		Preload("Plan").
		First(&subscription, id).Error

	if err != nil {
		return nil, err
	}

	return &subscription, nil
}

// List retrieves subscriptions with pagination.
func (r *SubscriptionRepo) List(db *gorm.DB, limit, offset int) ([]models.Subscription, error) {
	var subscriptions []models.Subscription

	err := db.Preload("Items").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&subscriptions).Error

	if err != nil {
		return nil, err
	}

	return subscriptions, nil
}

// Update saves changes to an existing subscription.
func (r *SubscriptionRepo) Update(db *gorm.DB, subscription *models.Subscription) error {
	return db.Save(subscription).Error
}
