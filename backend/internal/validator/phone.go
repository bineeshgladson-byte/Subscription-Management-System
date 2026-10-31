package validator

import "strings"

// Phone accepts digits with optional +, spaces, dashes and brackets, e.g.
// "9876543210", "+91 98765 43210", "(080) 4123-4567". It returns the trimmed
// value, or "" if it has fewer than 7 or more than 15 digits or other characters.
func Phone(s string) string {
	s = strings.TrimSpace(s)
	digits := 0
	for i, r := range s {
		switch {
		case r >= '0' && r <= '9':
			digits++
		case r == '+' && i == 0, r == ' ', r == '-', r == '(', r == ')':
		default:
			return ""
		}
	}
	if digits < 7 || digits > 15 {
		return ""
	}
	return s
}
