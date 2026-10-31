package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/repository"
	"github.com/brightloop/subscriptions/backend/internal/validator"
	"github.com/brightloop/subscriptions/backend/pkg/hash"
	"github.com/brightloop/subscriptions/backend/pkg/pagination"
)

// StaffService manages staff accounts. Every route that reaches it is
// admin-only (see router/routes_staff.go); the brief: "Only the admin can
// create staff accounts."
type StaffService struct {
	db        *gorm.DB
	users     *repository.UserRepo
	hasher    *hash.Hasher
	auth      *AuthService
	inviteTTL time.Duration
}

func NewStaffService(db *gorm.DB, users *repository.UserRepo, hasher *hash.Hasher, auth *AuthService, inviteTTL time.Duration) *StaffService {
	return &StaffService{db: db, users: users, hasher: hasher, auth: auth, inviteTTL: inviteTTL}
}

func (s *StaffService) List(ctx context.Context, search string, active *bool, p pagination.Params) ([]models.User, int64, error) {
	return s.users.List(s.db.WithContext(ctx), repository.UserFilter{Role: models.RoleStaff, Search: search, Active: active}, p)
}

func (s *StaffService) Get(ctx context.Context, id int64) (*models.User, error) {
	u, err := s.users.FindByIDAndRole(s.db.WithContext(ctx), id, models.RoleStaff)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, apperr.NotFound("staff member")
	}
	return u, nil
}

// Create makes a staff account. With a password, the admin shares it with the
// staff member. Without one, the account gets an unusable random password and
// the staff member is emailed a link to set their own (preferred).
func (s *StaffService) Create(ctx context.Context, in dto.CreateStaffRequest) (*models.User, bool, error) {
	email := validator.NormalizeEmail(in.Email)
	if email == "" {
		return nil, false, apperr.Validation("Some fields are invalid", map[string]string{"email": "is not a valid email"})
	}

	invite := in.Password == ""
	password := in.Password
	if invite {
		b := make([]byte, 32)
		if _, err := rand.Read(b); err != nil {
			return nil, false, err
		}
		password = hex.EncodeToString(b) // never shown to anyone
	} else if reason := validator.Password(password); reason != "" {
		return nil, false, apperr.Validation("Some fields are invalid", map[string]string{"password": reason})
	}

	hashed, err := s.hasher.Hash(password)
	if err != nil {
		return nil, false, err
	}

	db := s.db.WithContext(ctx)
	taken, err := s.users.EmailTaken(db, email, 0)
	if err != nil {
		return nil, false, err
	}
	if taken {
		return nil, false, emailTaken()
	}

	user := &models.User{Email: email, PasswordHash: hashed, FullName: in.FullName,
		Role: models.RoleStaff, IsActive: true}
	if err := s.users.Create(db, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, false, emailTaken()
		}
		return nil, false, err
	}

	sent := false
	if invite {
		// The account exists even if email fails; the admin can use
		// "forgot password" for them later. Don't fail the request.
		if err := s.auth.SendInvite(ctx, user, s.inviteTTL); err != nil {
			slog.Error("staff invite email failed", "user_id", user.ID, "err", err)
		} else {
			sent = true
		}
	}
	return user, sent, nil
}

func (s *StaffService) Update(ctx context.Context, id int64, in dto.UpdateStaffRequest) (*models.User, error) {
	user, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	fields := map[string]any{}
	if in.FullName != nil {
		fields["full_name"] = *in.FullName
	}
	if in.Email != nil {
		email := validator.NormalizeEmail(*in.Email)
		if email == "" {
			return nil, apperr.Validation("Some fields are invalid", map[string]string{"email": "is not a valid email"})
		}
		taken, err := s.users.EmailTaken(s.db.WithContext(ctx), email, user.ID)
		if err != nil {
			return nil, err
		}
		if taken {
			return nil, emailTaken()
		}
		fields["email"] = email
	}
	if len(fields) > 0 {
		if err := s.users.Update(s.db.WithContext(ctx), user.ID, fields); err != nil {
			if errors.Is(err, gorm.ErrDuplicatedKey) {
				return nil, emailTaken()
			}
			return nil, err
		}
	}
	return s.Get(ctx, id)
}

// SetActive activates or deactivates a staff account. Deactivation takes
// effect immediately: the auth middleware rejects their existing tokens.
func (s *StaffService) SetActive(ctx context.Context, id int64, active bool) (*models.User, error) {
	if _, err := s.Get(ctx, id); err != nil {
		return nil, err
	}
	if err := s.users.Update(s.db.WithContext(ctx), id, map[string]any{"is_active": active}); err != nil {
		return nil, err
	}
	return s.Get(ctx, id)
}
