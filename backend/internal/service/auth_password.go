package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"time"

	"gorm.io/gorm"

	"github.com/brightloop/subscriptions/backend/internal/apperr"
	"github.com/brightloop/subscriptions/backend/internal/dto"
	"github.com/brightloop/subscriptions/backend/internal/models"
	"github.com/brightloop/subscriptions/backend/internal/validator"
	"github.com/brightloop/subscriptions/backend/pkg/brand"
	"github.com/brightloop/subscriptions/backend/pkg/mailer"
)

// ---------------------------------------------------------------------------
// Forgot / reset password
//
// 1. POST /auth/forgot-password {email}
//    - ALWAYS answers 200 with the same message, whether or not the email exists
//      (otherwise anyone could test which emails are registered).
//    - The real work runs in the background so the response time is the same too.
//    - We create a random 32-byte token, store only its SHA-256 hash with an
//      expiry, and email the raw token inside a link to the Flutter app.
// 2. POST /auth/reset-password {token, password}
//    - Hash the token, find an unused + unexpired row (locked FOR UPDATE),
//      set the new bcrypt hash, set password_changed_at (kills old JWTs),
//      and mark ALL of the user's tokens used.
// ---------------------------------------------------------------------------

const forgotPasswordMessage = "If that email is registered, a password reset link has been sent."

// ForgotPassword returns immediately; the email is sent in the background.
func (s *AuthService) ForgotPassword(email string) dto.MessageResponse {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := s.sendResetLink(ctx, email); err != nil {
			slog.Error("forgot password: could not send reset link", "err", err)
		}
	}()
	return dto.MessageResponse{Message: forgotPasswordMessage}
}

func (s *AuthService) sendResetLink(ctx context.Context, email string) error {
	email = validator.NormalizeEmail(email)
	if email == "" {
		return nil
	}
	user, err := s.Users.FindByEmail(s.DB.WithContext(ctx), email)
	if err != nil {
		return err
	}
	if user == nil || !user.IsActive {
		return nil // silently do nothing; the caller already got the generic message
	}
	raw, err := s.issueResetToken(ctx, user.ID, s.ResetTTL)
	if err != nil {
		return err
	}
	link := s.passwordLink("reset-password", raw)
	body := fmt.Sprintf("Hi %s,\n\nWe received a request to reset your %s password.\n"+
		"Open this link to choose a new one (valid for %d minutes):\n\n%s\n\n"+
		"If you didn't ask for this, you can ignore this email; your password won't change.\n",
		user.FullName, brand.Name, int(s.ResetTTL.Minutes()), link)
	return s.Mailer.Send(ctx, mailer.Message{
		To:      user.Email,
		Subject: "Reset your " + brand.Name + " password",
		Body:    body + s.devCode("Reset code", raw) + brand.EmailSignature,
	})
}

// SendInvite emails a new staff member a link to set their own password.
func (s *AuthService) SendInvite(ctx context.Context, user *models.User, ttl time.Duration) error {
	raw, err := s.issueResetToken(ctx, user.ID, ttl)
	if err != nil {
		return err
	}
	link := s.passwordLink("set-password", raw)
	body := fmt.Sprintf("Hi %s,\n\n%s has created a %s staff account for you (%s).\n"+
		"Open this link to set your password (valid for %d hours):\n\n%s\n",
		user.FullName, brand.Company, brand.Name, user.Email, int(ttl.Hours()), link)
	return s.Mailer.Send(ctx, mailer.Message{
		To:      user.Email,
		Subject: "Your " + brand.Name + " staff account",
		Body:    body + s.devCode("Setup code", raw) + brand.EmailSignature,
	})
}

// devCode adds the raw code as plain text so it can be pasted into Postman.
// Only when EMAIL_SHOW_CODES is on (default: on in development); real
// customers only ever see the link.
func (s *AuthService) devCode(label, raw string) string {
	if !s.ShowCodesInEmail {
		return ""
	}
	return fmt.Sprintf("\n%s (for API testing): %s\n", label, raw)
}

// issueResetToken invalidates older links and stores the hash of a new one.
func (s *AuthService) issueResetToken(ctx context.Context, userID int64, ttl time.Duration) (string, error) {
	raw, hashed, err := newToken()
	if err != nil {
		return "", err
	}
	now := s.Clock.Now()
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.ResetTokens.InvalidateForUser(tx, userID, now); err != nil {
			return err
		}
		return s.ResetTokens.Create(tx, &models.PasswordResetToken{
			UserID: userID, TokenHash: hashed, ExpiresAt: now.Add(ttl)})
	})
	return raw, err
}

// ResetPassword sets a new password using an emailed token.
func (s *AuthService) ResetPassword(ctx context.Context, in dto.ResetPasswordRequest) error {
	if reason := validator.Password(in.Password); reason != "" {
		return apperr.Validation("Some fields are invalid", map[string]string{"password": reason})
	}
	invalid := apperr.Validation("This reset link is invalid or has expired",
		map[string]string{"token": "is invalid or has expired; request a new link"})
	if len(in.Token) != 64 {
		return invalid
	}
	hashed, err := s.Hasher.Hash(in.Password)
	if err != nil {
		return err
	}

	now := s.Clock.Now()
	return s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		tok, err := s.ResetTokens.FindUsableForUpdate(tx, hashToken(in.Token), now)
		if err != nil {
			return err
		}
		if tok == nil {
			return invalid
		}
		user, err := s.Users.FindByID(tx, tok.UserID)
		if err != nil {
			return err
		}
		if user == nil || !user.IsActive {
			return invalid
		}
		if err := s.Users.UpdatePassword(tx, user.ID, hashed, now); err != nil {
			return err
		}
		return s.ResetTokens.InvalidateForUser(tx, user.ID, now)
	})
}

// ChangePassword lets a logged-in user change their password. All other
// sessions are signed out; the caller gets a fresh token to stay logged in.
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, in dto.ChangePasswordRequest) (*dto.LoginResponse, error) {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !s.Hasher.Matches(user.PasswordHash, in.CurrentPassword) {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"current_password": "is incorrect"})
	}
	if in.NewPassword == in.CurrentPassword {
		return nil, apperr.Validation("Some fields are invalid",
			map[string]string{"new_password": "must be different from the current password"})
	}
	if reason := validator.Password(in.NewPassword); reason != "" {
		return nil, apperr.Validation("Some fields are invalid", map[string]string{"new_password": reason})
	}
	hashed, err := s.Hasher.Hash(in.NewPassword)
	if err != nil {
		return nil, err
	}

	now := s.Clock.Now()
	err = s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.Users.UpdatePassword(tx, user.ID, hashed, now); err != nil {
			return err
		}
		return s.ResetTokens.InvalidateForUser(tx, user.ID, now)
	})
	if err != nil {
		return nil, err
	}
	return s.issueToken(user)
}

func (s *AuthService) passwordLink(page, raw string) string {
	return fmt.Sprintf("%s/%s?token=%s", s.FrontendURL, page, url.QueryEscape(raw))
}

// newToken returns a random token (sent to the user) and its SHA-256 hash (stored).
// SHA-256 is fine here (unlike for passwords) because the token is 256 random bits,
// so it can't be guessed; the hash just means a database leak doesn't leak live links.
func newToken() (raw, hashed string, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, hashToken(raw), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
