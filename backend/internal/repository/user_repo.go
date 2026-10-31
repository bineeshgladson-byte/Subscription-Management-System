// Package repository contains all database access. Every method takes a *gorm.DB
// so services can pass either the normal connection or an open transaction.
package repository

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

type UserRepo struct{}

func NewUserRepo() *UserRepo { return &UserRepo{} }

// FindByEmail returns (nil, nil) when no user exists.
func (r *UserRepo) FindByEmail(db *gorm.DB, email string) (*models.User, error) {
	var u models.User
	err := db.Preload("Customer").Where("lower(email) = lower(?)", strings.TrimSpace(email)).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// FindByID returns (nil, nil) when no user exists (soft-deleted users are excluded).
func (r *UserRepo) FindByID(db *gorm.DB, id int64) (*models.User, error) {
	var u models.User
	err := db.Preload("Customer").First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// FindByIDAndRole is used by staff management so admins can't edit
// other admins or customers through /staff/:id.
func (r *UserRepo) FindByIDAndRole(db *gorm.DB, id int64, role models.Role) (*models.User, error) {
	var u models.User
	err := db.Where("role = ?", role).First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

// SessionState loads just the columns the auth middleware checks on every request.
func (r *UserRepo) SessionState(db *gorm.DB, id int64) (*models.User, error) {
	var u models.User
	err := db.Select("id", "is_active", "password_changed_at").First(&u, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &u, err
}

func (r *UserRepo) Create(db *gorm.DB, u *models.User) error {
	return db.Omit("Customer").Create(u).Error
}

// EmailTaken checks every user, including soft-deleted ones, because the
// unique index on lower(email) covers them too.
func (r *UserRepo) EmailTaken(db *gorm.DB, email string, exceptID int64) (bool, error) {
	var n int64
	err := db.Unscoped().Model(&models.User{}).
		Where("lower(email) = lower(?) AND id <> ?", email, exceptID).Count(&n).Error
	return n > 0, err
}

func (r *UserRepo) Update(db *gorm.DB, id int64, fields map[string]any) error {
	return db.Model(&models.User{}).Where("id = ?", id).Updates(fields).Error
}

func (r *UserRepo) UpdatePassword(db *gorm.DB, id int64, hash string, changedAt time.Time) error {
	return r.Update(db, id, map[string]any{"password_hash": hash, "password_changed_at": changedAt})
}

func (r *UserRepo) TouchLastLogin(db *gorm.DB, id int64, at time.Time) error {
	return db.Model(&models.User{}).Where("id = ?", id).Update("last_login_at", at).Error
}

func (r *UserRepo) CountByRole(db *gorm.DB, role models.Role) (int64, error) {
	var n int64
	err := db.Model(&models.User{}).Where("role = ?", role).Count(&n).Error
	return n, err
}

type UserFilter struct {
	Role   models.Role
	Search string // matches name or email
	Active *bool
}

func (r *UserRepo) List(db *gorm.DB, f UserFilter, p pagination.Params) ([]models.User, int64, error) {
	scope := func(q *gorm.DB) *gorm.DB {
		q = q.Where("role = ?", f.Role)
		if s := strings.TrimSpace(f.Search); s != "" {
			like := "%" + escapeLike(strings.ToLower(s)) + "%"
			q = q.Where("(lower(full_name) LIKE ? OR lower(email) LIKE ?)", like, like)
		}
		if f.Active != nil {
			q = q.Where("is_active = ?", *f.Active)
		}
		return q
	}

	var total int64
	if err := db.Model(&models.User{}).Scopes(scope).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []models.User
	err := db.Scopes(scope).Order(p.Sort).Limit(p.PageSize).Offset(p.Offset()).Find(&users).Error
	return users, total, err
}

// escapeLike stops user input like "%" or "_" acting as wildcards.
func escapeLike(s string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(s)
}
