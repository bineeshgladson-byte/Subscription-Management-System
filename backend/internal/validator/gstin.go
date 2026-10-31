package validator

import (
	"regexp"
	"strings"
)

// Indian GSTIN: 2-digit state code, 10-char PAN, entity number, 'Z', checksum.
// Example: 29ABCDE1234F1Z5
var gstinRe = regexp.MustCompile(`^[0-9]{2}[A-Z]{5}[0-9]{4}[A-Z][1-9A-Z]Z[0-9A-Z]$`)

// GSTIN upper-cases the input and returns it, or "" if the format is invalid.
func GSTIN(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !gstinRe.MatchString(s) {
		return ""
	}
	return s
}
