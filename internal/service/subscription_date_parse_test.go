package service

import "testing"

func TestParseOptionalDateString_DateOnly(t *testing.T) {
	parsed, err := parseOptionalDateString("2026-02-24")
	if err != nil {
		t.Fatalf("parseOptionalDateString() error = %v", err)
	}
	if parsed == nil {
		t.Fatal("parseOptionalDateString() returned nil date")
	}
	if got, want := parsed.Format("2006-01-02"), "2026-02-24"; got != want {
		t.Fatalf("parseOptionalDateString() date = %s, want %s", got, want)
	}
}

func TestParseOptionalDateString_Invalid(t *testing.T) {
	_, err := parseOptionalDateString("02/24/2026")
	if err == nil {
		t.Fatal("parseOptionalDateString() expected error")
	}
	if got, want := err.Error(), "invalid date format, expected YYYY-MM-DD"; got != want {
		t.Fatalf("parseOptionalDateString() error = %q, want %q", got, want)
	}
}
