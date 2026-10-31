// Package service holds the business rules. Services know nothing about HTTP.
//
// AuthService is the REFERENCE implementation of the layering pattern:
// handler (bind/validate) → service (rules + transaction) → repository (SQL).
// Copy this shape for every other module.
package service

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/validator"
	"github.com/brightloop/subscriptions/backend/pkg/clock"
	"github.com/brightloop/subscriptions/backend/pkg/hash"
	"github.com/brightloop/subscriptions/backend/pkg/jwtutil"
)

type AuthService struct {
	db        *gorm.DB
	users     *repository.UserRepo
	customers *repository.CustomerRepo
	hasher    *hash.Hasher
	jwt       *jwtutil.Manager
	clock     clock.Clock
}

func NewAuthService(db *gorm.DB, users *repository.UserRepo, customers *repository.CustomerRepo,
	hasher *hash.Hasher, jwt *jwtutil.Manager, clk clock.Clock) *AuthService {
	return &AuthService{db: db, users: users, customers: customers, hasher: hasher, jwt: jwt, clock: clk}
}

// Register creates a customer login and its customer record in ONE transaction:
// either both rows exist afterwards or neither does.
func (s *AuthService) Register(ctx context.Context, in dto.RegisterRequest) (*models.User, error) {
	email := validator.NormalizeEmail(in.Email)
	if email == "" {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"email": "is not a valid email"})
	}
	if reason := validator.Password(in.Password); reason != "" {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"password": reason})
	}

	hashed, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	var user *models.User
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		existing, err := s.users.FindByEmail(tx, email)
		if err != nil {
			return err
		}
		if existing != nil {
			return apperr.Conflict("email already registered")
		}

		user = &models.User{Email: email, PasswordHash: hashed, FullName: in.FullName,
			Role: models.RoleCustomer, IsActive: true}
		if err := s.users.Create(tx, user); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) { // lost a race with another sign-up
				return apperr.Conflict("email already registered")
			}
			return err
		}

		cust := &models.Customer{UserID: &user.ID, Name: in.FullName, Email: email}
		if in.Phone != "" {
			cust.Phone = &in.Phone
		}
		if err := s.customers.Create(tx, cust); err != nil {
			return err
		}
		user.Customer = cust
		return nil
	})
	return user, err
}

// Login checks credentials and issues a JWT. Wrong email and wrong password
// return the SAME error so attackers can't discover which emails exist.
func (s *AuthService) Login(ctx context.Context, in dto.LoginRequest) (*dto.LoginResponse, error) {
	db := s.db.WithContext(ctx)
	user, err := s.users.FindByEmail(db, in.Email)
	if err != nil {
		return nil, err
	}
	if user == nil || !s.hasher.Matches(user.PasswordHash, in.Password) {
		return nil, apperr.Unauthenticated("invalid email or password")
	}
	if !user.IsActive {
		return nil, apperr.Forbidden("account deactivated")
	}

	var customerID *int64
	if user.Customer != nil {
		customerID = &user.Customer.ID
	}
	now := s.clock.Now()
	token, exp, err := s.jwt.Issue(user.ID, string(user.Role), customerID, now)
	if err != nil {
		return nil, err
	}
	_ = s.users.TouchLastLogin(db, user.ID, now)

	return &dto.LoginResponse{AccessToken: token, ExpiresAt: exp, User: dto.NewUserResponse(user)}, nil
}

func (s *AuthService) Me(ctx context.Context, userID int64) (*models.User, error) {
	u, err := s.users.FindByID(s.db.WithContext(ctx), userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, apperr.NotFound("user")
	}
	return u, nil
}

// EnsureAdmin creates the first admin from SEED_ADMIN_* env vars if no admin exists.
// Only the admin can create staff, so the system needs one admin to start with.
func (s *AuthService) EnsureAdmin(ctx context.Context, email, password string) (created bool, err error) {
	if email == "" || password == "" {
		return false, nil
	}
	db := s.db.WithContext(ctx)
	n, err := s.users.CountByRole(db, models.RoleAdmin)
	if err != nil || n > 0 {
		return false, err
	}
	if reason := validator.Password(password); reason != "" {
		return false, errors.New("SEED_ADMIN_PASSWORD " + reason)
	}
	hashed, err := s.hasher.Hash(password)
	if err != nil {
		return false, err
	}
	now := time.Now()
	u := &models.User{Email: validator.NormalizeEmail(email), PasswordHash: hashed, FullName: "Administrator",
		Role: models.RoleAdmin, IsActive: true, PasswordChangedAt: &now}
	return true, s.users.Create(db, u)
}
