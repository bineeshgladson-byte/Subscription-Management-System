package service

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
)

type SubscriptionService struct {
	db   *gorm.DB
	repo *repository.SubscriptionRepo
}

func NewSubscriptionService(
	db *gorm.DB,
	repo *repository.SubscriptionRepo,
) *SubscriptionService {
	return &SubscriptionService{
		db:   db,
		repo: repo,
	}
}

// ListSubscriptions retrieves subscriptions with pagination.
func (s *SubscriptionService) ListSubscriptions(
	limit, offset int,
) ([]models.Subscription, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}

	return s.repo.List(s.db, limit, offset)
}

// GetSubscription retrieves a subscription by ID.
func (s *SubscriptionService) GetSubscription(
	id int64,
) (*models.Subscription, error) {
	if id <= 0 {
		return nil, errors.New("subscription ID must be positive")
	}

	subscription, err := s.repo.FindByID(s.db, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("subscription not found")
		}
		return nil, err
	}

	return subscription, nil
}

// ActivateSubscription activates a confirmed subscription.
func (s *SubscriptionService) ActivateSubscription(id int64) (*models.Subscription, error) {
	subscription, err := s.GetSubscription(id)
	if err != nil {
		return nil, err
	}

	if subscription.Status != models.SubConfirmed {
		return nil, errors.New("only confirmed subscriptions can be activated")
	}

	subscription.Status = models.SubActive
	now := time.Now()
	subscription.ActivatedAt = &now

	if err := s.repo.Update(s.db, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

// PauseSubscription pauses an active subscription.
func (s *SubscriptionService) PauseSubscription(id int64) (*models.Subscription, error) {
	subscription, err := s.GetSubscription(id)
	if err != nil {
		return nil, err
	}

	if subscription.Status != models.SubActive {
		return nil, errors.New("only active subscriptions can be paused")
	}

	subscription.Status = models.SubPaused
	now := time.Now()
	subscription.PausedAt = &now

	if err := s.repo.Update(s.db, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

// ResumeSubscription resumes a paused subscription.
func (s *SubscriptionService) ResumeSubscription(id int64) (*models.Subscription, error) {
	subscription, err := s.GetSubscription(id)
	if err != nil {
		return nil, err
	}

	if subscription.Status != models.SubPaused {
		return nil, errors.New("only paused subscriptions can be resumed")
	}

	subscription.Status = models.SubActive
	subscription.PausedAt = nil

	if err := s.repo.Update(s.db, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}

// CloseSubscription closes an active or paused subscription.
func (s *SubscriptionService) CloseSubscription(id int64, reason string) (*models.Subscription, error) {
	subscription, err := s.GetSubscription(id)
	if err != nil {
		return nil, err
	}

	if subscription.Status != models.SubActive && subscription.Status != models.SubPaused {
		return nil, errors.New("only active or paused subscriptions can be closed")
	}

	subscription.Status = models.SubClosed
	subscription.CloseReason = &reason
	now := time.Now()
	subscription.ClosedAt = &now

	if err := s.repo.Update(s.db, subscription); err != nil {
		return nil, err
	}

	return subscription, nil
}
