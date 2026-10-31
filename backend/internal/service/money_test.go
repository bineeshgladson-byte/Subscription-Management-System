package service

import "testing"

func TestParseMoney(t *testing.T) {
	ok := map[string]string{"999": "999.00", "999.5": "999.50", " 560.00 ": "560.00", "0": "0.00", "1.230": "1.23"}
	for in, want := range ok {
		got, reason := parseMoney(in, false)
		if reason != "" || got.StringFixed(2) != want {
			t.Errorf("parseMoney(%q) = %s (%q), want %s", in, got.StringFixed(2), reason, want)
		}
	}
	for _, bad := range []string{"", "abc", "-1", "1.234", "10000000000"} {
		if _, reason := parseMoney(bad, false); reason == "" {
			t.Errorf("parseMoney(%q) should fail", bad)
		}
	}
	if got, reason := parseMoney("-100", true); reason != "" || got.StringFixed(2) != "-100.00" {
		t.Errorf("negative allowed: got %s %q", got, reason)
	}
}

func TestParseDate(t *testing.T) {
	if d, ok := parseDate("2026-11-01"); !ok || dateString(d) != "2026-11-01" {
		t.Errorf("parseDate failed: %v %v", d, ok)
	}
	for _, bad := range []string{"", "01-11-2026", "2026-13-01", "2026-02-30"} {
		if _, ok := parseDate(bad); ok {
			t.Errorf("parseDate(%q) should fail", bad)
		}
	}
}
