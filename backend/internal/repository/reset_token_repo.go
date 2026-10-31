package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

// ResetTokenRepo stores password reset / staff invite tokens.
// Only the SHA-256 hash of a token is ever stored.
type ResetTokenRepo struct{}

func NewResetTokenRepo() *ResetTokenRepo { return &ResetTokenRepo{} }

func (r *ResetTokenRepo) Create(db *gorm.DB, t *models.PasswordResetToken) error {
	return db.Create(t).Error
}

// FindUsableForUpdate locks the token row so two simultaneous reset requests
// with the same link can't both succeed. Returns (nil, nil) if the token is
// unknown, already used or expired.
func (r *ResetTokenRepo) FindUsableForUpdate(tx *gorm.DB, tokenHash string, now time.Time) (*models.PasswordResetToken, error) {
	var t models.PasswordResetToken
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, now).
		First(&t).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &t, err
}

// InvalidateForUser marks every unused token of a user as used, so only the
// newest link works and a successful reset kills all older links.
func (r *ResetTokenRepo) InvalidateForUser(db *gorm.DB, userID int64, now time.Time) error {
	return db.Model(&models.PasswordResetToken{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now).Error
}
