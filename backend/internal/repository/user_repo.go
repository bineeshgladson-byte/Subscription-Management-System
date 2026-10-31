// Package repository contains all database access. Every method takes a *gorm.DB
// so services can pass either the normal connection or an open transaction.
package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
)

type UserRepo struct{}

func NewUserRepo() *UserRepo { return &UserRepo{} }

// FindByEmail returns (nil, nil) when no user exists.
func (r *UserRepo) FindByEmail(db *gorm.DB, email string) (*models.User, error) {
	var u models.User
	err := db.Preload("Customer").Where("lower(email) = lower(?)", email).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) FindByID(db *gorm.DB, id int64) (*models.User, error) {
	var u models.User
	err := db.Preload("Customer").First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) Create(db *gorm.DB, u *models.User) error {
	return db.Omit("Customer").Create(u).Error
}

func (r *UserRepo) TouchLastLogin(db *gorm.DB, id int64, at time.Time) error {
	return db.Model(&models.User{}).Where("id = ?", id).Update("last_login_at", at).Error
}

func (r *UserRepo) CountByRole(db *gorm.DB, role models.Role) (int64, error) {
	var n int64
	err := db.Model(&models.User{}).Where("role = ?", role).Count(&n).Error
	return n, err
}
