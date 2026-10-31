// Package jwtutil issues and verifies HS256 access tokens.
package jwtutil

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func init() {
	// Millisecond "iat" so a token issued just before a password change can be
	// told apart from one issued just after it (the default is whole seconds).
	jwt.TimePrecision = time.Millisecond
}

type Claims struct {
	Role       string `json:"role"`
	CustomerID *int64 `json:"customer_id,omitempty"` // set only for role=customer
	jwt.RegisteredClaims
}

// UserID returns the numeric user id stored in the standard "sub" claim.
func (c Claims) UserID() int64 {
	id, _ := strconv.ParseInt(c.Subject, 10, 64)
	return id
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func New(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Issue(userID int64, role string, customerID *int64, now time.Time) (string, time.Time, error) {
	exp := now.Add(m.ttl)
	claims := Claims{
		Role:       role,
		CustomerID: customerID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			Issuer:    "brightloop",
		},
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return signed, exp, err
}

func (m *Manager) Parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		return m.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithIssuer("brightloop"))
	if err != nil {
		return nil, err
	}
	if claims.UserID() == 0 {
		return nil, errors.New("token has no subject")
	}
	return claims, nil
}
