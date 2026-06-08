package service

import (
	"errors"
	"net/url"
	"strings"
	"unicode"
)

var ErrInvalidSubscriptionURL = errors.New("subscription url must be a valid http or https URL")

func normalizeSubscriptionURL(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	if strings.IndexFunc(trimmed, unicode.IsSpace) >= 0 || strings.ContainsFunc(trimmed, unicode.IsControl) {
		return "", ErrInvalidSubscriptionURL
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", ErrInvalidSubscriptionURL
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", ErrInvalidSubscriptionURL
	}
	if parsed.Host == "" || parsed.Hostname() == "" {
		return "", ErrInvalidSubscriptionURL
	}

	return parsed.String(), nil
}
