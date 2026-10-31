// Package validator holds business-level validation rules shared by services.
package validator

import (
	"net/mail"
	"strings"
	"unicode"
)

const (
	MinPasswordLength = 9  // brief: "more than 8 characters"
	MaxPasswordBytes  = 72 // bcrypt's limit
)

// Password returns "" if the password is acceptable, otherwise a human-readable reason.
// Go's regexp package has no look-ahead, so we check characters in a loop.
func Password(p string) string {
	if len([]rune(p)) < MinPasswordLength {
		return "must be at least 9 characters"
	}
	if len(p) > MaxPasswordBytes {
		return "must be at most 72 bytes"
	}
	var upper, lower, special bool
	for _, r := range p {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r), unicode.IsSpace(r):
			// allowed, not required
		default:
			special = true
		}
	}
	switch {
	case !upper:
		return "must include an uppercase letter"
	case !lower:
		return "must include a lowercase letter"
	case !special:
		return "must include a special character"
	}
	return ""
}

// NormalizeEmail lower-cases and trims; returns "" if the address is invalid.
func NormalizeEmail(e string) string {
	e = strings.ToLower(strings.TrimSpace(e))
	addr, err := mail.ParseAddress(e)
	if err != nil || addr.Address != e {
		return ""
	}
	return e
}
