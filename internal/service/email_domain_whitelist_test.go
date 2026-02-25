package service

import (
	"errors"
	"testing"
)

func TestNormalizeEmailDomainWhitelist(t *testing.T) {
	t.Run("normalize_and_sort_unique", func(t *testing.T) {
		got, err := normalizeEmailDomainWhitelist("  EXAMPLE.com ; @sub.example.com. ,foo.com\nexample.com")
		if err != nil {
			t.Fatalf("normalizeEmailDomainWhitelist() error = %v", err)
		}
		want := "example.com\nfoo.com\nsub.example.com"
		if got != want {
			t.Fatalf("normalizeEmailDomainWhitelist() = %q, want %q", got, want)
		}
	})

	t.Run("empty_input_allows_all", func(t *testing.T) {
		got, err := normalizeEmailDomainWhitelist(" \n ; , ")
		if err != nil {
			t.Fatalf("normalizeEmailDomainWhitelist() error = %v", err)
		}
		if got != "" {
			t.Fatalf("normalizeEmailDomainWhitelist() = %q, want empty", got)
		}
	})

	t.Run("invalid_domain_rejected", func(t *testing.T) {
		_, err := normalizeEmailDomainWhitelist("http://example.com")
		if !errors.Is(err, ErrInvalidEmailDomainWhitelist) {
			t.Fatalf("error = %v, want %v", err, ErrInvalidEmailDomainWhitelist)
		}
	})

	t.Run("length_limit_rejected", func(t *testing.T) {
		tooLong := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa0.com," +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1.com," +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2.com," +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa3.com," +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa4.com," +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa5.com," +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa6.com," +
			"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa7.com"
		_, err := normalizeEmailDomainWhitelist(tooLong)
		if !errors.Is(err, ErrEmailDomainWhitelistTooLong) {
			t.Fatalf("error = %v, want %v", err, ErrEmailDomainWhitelistTooLong)
		}
	})
}

func TestIsEmailDomainAllowed(t *testing.T) {
	t.Run("empty_whitelist_allow_all", func(t *testing.T) {
		if !isEmailDomainAllowed("user@any-domain.com", "") {
			t.Fatal("expected empty whitelist to allow any domain")
		}
	})

	t.Run("exact_domain_match", func(t *testing.T) {
		if !isEmailDomainAllowed("user@example.com", "example.com") {
			t.Fatal("expected exact domain to match")
		}
	})

	t.Run("subdomain_match", func(t *testing.T) {
		if !isEmailDomainAllowed("user@a.b.example.com", "example.com") {
			t.Fatal("expected subdomain to match")
		}
	})

	t.Run("non_match", func(t *testing.T) {
		if isEmailDomainAllowed("user@example.net", "example.com") {
			t.Fatal("expected non-matching domain to be blocked")
		}
	})
}
