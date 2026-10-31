package service

import "testing"

func TestParseRate(t *testing.T) {
	ok := map[string]string{"18": "18.00", "18.0": "18.00", "18.00": "18.00", "0": "0.00", "100": "100.00", " 12.5 ": "12.50", "5.250": "5.25"}
	for in, want := range ok {
		got, reason := parseRate(in)
		if reason != "" || got.StringFixed(2) != want {
			t.Errorf("parseRate(%q) = %s (%q), want %s", in, got.StringFixed(2), reason, want)
		}
	}
	for _, bad := range []string{"", "abc", "-1", "100.01", "101", "18.555", "1e3"} {
		if _, reason := parseRate(bad); reason == "" {
			t.Errorf("parseRate(%q) should fail", bad)
		}
	}
}
