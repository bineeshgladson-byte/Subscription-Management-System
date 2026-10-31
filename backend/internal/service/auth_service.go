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
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
)

// AuthDeps groups everything AuthService needs, so adding a dependency
// doesn't change every call site.
type AuthDeps struct {
	DB          *gorm.DB
	Users       *repository.UserRepo
	Customers   *repository.CustomerRepo
	ResetTokens *repository.ResetTokenRepo
	Hasher      *hash.Hasher
	JWT         *jwtutil.Manager
	Clock       clock.Clock
	Mailer      mailer.Mailer
	FrontendURL string
	ResetTTL    time.Duration
	// ShowCodesInEmail adds the raw reset/invite code to emails for API testing.
	ShowCodesInEmail bool
}

type AuthService struct{ AuthDeps }

func NewAuthService(d AuthDeps) *AuthService { return &AuthService{d} }

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
	if in.Phone != "" && validator.Phone(in.Phone) == "" {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"phone": "is not a valid phone number"})
	}

	hashed, err := s.Hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}

	var user *models.User
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		taken, err := s.Users.EmailTaken(tx, email, 0)
		if err != nil {
			return err
		}
		if taken {
			return emailTaken()
		}

		user = &models.User{Email: email, PasswordHash: hashed, FullName: in.FullName,
			Role: models.RoleCustomer, IsActive: true}
		if err := s.Users.Create(tx, user); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) { // lost a race with another sign-up
				return emailTaken()
			}
			return err
		}

		// Staff may already have created a customer record with this email
		// (e.g. from a sales call). Link the new login to it instead of creating
		// a duplicate, so the customer sees their existing subscriptions.
		existing, err := s.Customers.FindLiveByEmail(tx, email)
		if err != nil {
			return err
		}
		if existing != nil {
			if existing.UserID != nil {
				return emailTaken()
			}
			link := map[string]any{"user_id": user.ID}
			if existing.Phone == nil && in.Phone != "" {
				link["phone"] = in.Phone
			}
			if err := s.Customers.Update(tx, existing.ID, link); err != nil {
				return err
			}
			existing.UserID = &user.ID
			user.Customer = existing
			return nil
		}

		cust := &models.Customer{UserID: &user.ID, Name: in.FullName, Email: email}
		if in.Phone != "" {
			cust.Phone = &in.Phone
		}
		if err := s.Customers.Create(tx, cust); err != nil {
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
	db := s.DB.WithContext(ctx)
	email := validator.NormalizeEmail(in.Email)
	if email == "" {
		return nil, apperr.Unauthenticated("invalid email or password")
	}
	user, err := s.Users.FindByEmail(db, email)
	if err != nil {
		return nil, err
	}
	if user == nil || !s.Hasher.Matches(user.PasswordHash, in.Password) {
		return nil, apperr.Unauthenticated("invalid email or password")
	}
	if !user.IsActive {
		return nil, apperr.Forbidden("account deactivated")
	}

	_ = s.Users.TouchLastLogin(db, user.ID, s.Clock.Now())
	return s.issueToken(user)
}

func (s *AuthService) issueToken(user *models.User) (*dto.LoginResponse, error) {
	var customerID *int64
	if user.Customer != nil {
		customerID = &user.Customer.ID
	}
	token, exp, err := s.JWT.Issue(user.ID, string(user.Role), customerID, s.Clock.Now())
	if err != nil {
		return nil, err
	}
	return &dto.LoginResponse{AccessToken: token, ExpiresAt: exp, User: dto.NewUserResponse(user)}, nil
}

// CheckSession is called by the auth middleware on every request. A valid
// signature isn't enough: the account must still be active and the token must
// have been issued after the last password change/reset.
func (s *AuthService) CheckSession(ctx context.Context, userID int64, issuedAt time.Time) error {
	u, err := s.Users.SessionState(s.DB.WithContext(ctx), userID)
	if err != nil {
		return err
	}
	if u == nil || !u.IsActive {
		return apperr.Unauthenticated("account is no longer active")
	}
	// JWT "iat" has millisecond precision (see pkg/jwtutil), so compare at that precision.
	if u.PasswordChangedAt != nil && issuedAt.Before(u.PasswordChangedAt.Truncate(time.Millisecond)) {
		return apperr.Unauthenticated("session expired, please log in again")
	}
	return nil
}

func (s *AuthService) Me(ctx context.Context, userID int64) (*models.User, error) {
	u, err := s.Users.FindByID(s.DB.WithContext(ctx), userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, apperr.NotFound("user")
	}
	return u, nil
}

// UpdateProfile changes the caller's own name and, for customers, their
// billing details. Email changes are not allowed here (they'd need verification).
func (s *AuthService) UpdateProfile(ctx context.Context, userID int64, in dto.UpdateProfileRequest) (*models.User, error) {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return nil, err
	}

	isCustomer := user.Customer != nil
	if !isCustomer && (in.Phone != nil || in.CompanyName != nil || in.BillingAddress != nil || in.GSTIN != nil) {
		return nil, apperr.Validation("Only customers have billing details",
			map[string]string{"body": "only full_name can be changed for staff and admin accounts"})
	}

	userFields := map[string]any{}
	custFields := map[string]any{}
	if in.FullName != nil {
		userFields["full_name"] = *in.FullName
		custFields["name"] = *in.FullName // keep the customer record's name in sync
	}
	if in.Phone != nil {
		if *in.Phone != "" && validator.Phone(*in.Phone) == "" {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"phone": "is not a valid phone number"})
		}
		custFields["phone"] = nullIfEmpty(*in.Phone)
	}
	if in.CompanyName != nil {
		custFields["company_name"] = nullIfEmpty(*in.CompanyName)
	}
	if in.BillingAddress != nil {
		custFields["billing_address"] = nullIfEmpty(*in.BillingAddress)
	}
	if in.GSTIN != nil {
		if *in.GSTIN == "" {
			custFields["gstin"] = nil
		} else if g := validator.GSTIN(*in.GSTIN); g == "" {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"gstin": "is not a valid GSTIN"})
		} else {
			custFields["gstin"] = g
		}
	}

	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(userFields) > 0 {
			if err := s.Users.Update(tx, userID, userFields); err != nil {
				return err
			}
		}
		if isCustomer && len(custFields) > 0 {
			return s.Customers.Update(tx, user.Customer.ID, custFields)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.Me(ctx, userID)
}

// EnsureAdmin creates the first admin from SEED_ADMIN_* env vars if no admin exists.
// Only the admin can create staff, so the system needs one admin to start with.
func (s *AuthService) EnsureAdmin(ctx context.Context, email, password string) (created bool, err error) {
	if email == "" || password == "" {
		return false, nil
	}
	db := s.DB.WithContext(ctx)
	n, err := s.Users.CountByRole(db, models.RoleAdmin)
	if err != nil || n > 0 {
		return false, err
	}
	if reason := validator.Password(password); reason != "" {
		return false, errors.New("SEED_ADMIN_PASSWORD " + reason)
	}
	hashed, err := s.Hasher.Hash(password)
	if err != nil {
		return false, err
	}
	now := s.Clock.Now()
	u := &models.User{Email: validator.NormalizeEmail(email), PasswordHash: hashed, FullName: "Administrator",
		Role: models.RoleAdmin, IsActive: true, PasswordChangedAt: &now}
	return true, s.Users.Create(db, u)
}

func emailTaken() error {
	e := apperr.Conflict("email already registered")
	e.Details = map[string]string{"email": "is already registered"}
	return e
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
