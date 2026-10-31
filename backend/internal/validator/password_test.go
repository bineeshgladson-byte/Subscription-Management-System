package validator

import "testing"

func TestPassword(t *testing.T) {
	cases := []struct {
		in   string
		okay bool
	}{
		{"Short#1a", false}, // 8 chars: not "more than 8"
		{"Brightloop#2026", true},
		{"brightloop#2026", false}, // no uppercase
		{"BRIGHTLOOP#2026", false}, // no lowercase
		{"Brightloop2026", false},  // no special
		{"Abcdefgh!", true},        // exactly 9
	}
	for _, c := range cases {
		got := Password(c.in) == ""
		if got != c.okay {
			t.Errorf("Password(%q) ok=%v, want %v (%s)", c.in, got, c.okay, Password(c.in))
		}
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Rahul@Example.IN "); got != "rahul@example.in" {
		t.Errorf("got %q", got)
	}
	if got := NormalizeEmail("not-an-email"); got != "" {
		t.Errorf("expected invalid, got %q", got)
	}
}

func TestGSTIN(t *testing.T) {
	if got := GSTIN(" 29abcde1234f1z5 "); got != "29ABCDE1234F1Z5" {
		t.Errorf("valid GSTIN rejected: %q", got)
	}
	for _, bad := range []string{"", "29ABCDE1234F1Y5", "ABCDE1234F1Z5", "29ABCDE1234F1Z"} {
		if GSTIN(bad) != "" {
			t.Errorf("invalid GSTIN accepted: %q", bad)
		}
	}
}
