package service

import (
	"errors"
	"net"
	"sort"
	"strings"
)

const defaultIconProxyDomainWhitelist = "google.com\nicon.horse"

var (
	ErrInvalidIconProxyDomainWhitelist = errors.New("invalid icon proxy domain whitelist")
	ErrIconProxyDomainWhitelistTooLong = errors.New("icon proxy domain whitelist is too long")
)

func normalizeIconProxyDomainWhitelist(raw string) (string, error) {
	domains, err := parseIconProxyDomainWhitelist(raw)
	if err != nil {
		return "", err
	}
	if len(domains) == 0 {
		return "", nil
	}

	normalized := strings.Join(domains, "\n")
	if len(normalized) > 500 {
		return "", ErrIconProxyDomainWhitelistTooLong
	}
	return normalized, nil
}

func parseIconProxyDomainWhitelist(raw string) ([]string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	if len(parts) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})
	domains := make([]string, 0, len(parts))
	for _, part := range parts {
		domain, err := normalizeIconProxyDomain(part)
		if err != nil {
			return nil, ErrInvalidIconProxyDomainWhitelist
		}
		if domain == "" {
			continue
		}
		if _, exists := seen[domain]; exists {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}

	sort.Strings(domains)
	return domains, nil
}

func normalizeIconProxyDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(raw))
	domain = strings.TrimRight(domain, ".")
	if domain == "" {
		return "", nil
	}

	if strings.Contains(domain, "://") ||
		strings.Contains(domain, "/") ||
		strings.Contains(domain, `\`) ||
		strings.Contains(domain, "@") ||
		strings.Contains(domain, "?") ||
		strings.Contains(domain, "#") ||
		strings.Contains(domain, ":") {
		return "", ErrInvalidIconProxyDomainWhitelist
	}

	if ip := net.ParseIP(domain); ip != nil {
		return "", ErrInvalidIconProxyDomainWhitelist
	}

	if err := validateOutboundHost(domain, "icon proxy domain whitelist"); err != nil {
		return "", ErrInvalidIconProxyDomainWhitelist
	}
	if !isValidEmailDomain(domain) {
		return "", ErrInvalidIconProxyDomainWhitelist
	}
	return domain, nil
}

func isIconProxyDomainAllowed(hostname string, whitelist string) bool {
	allowedDomains, err := parseIconProxyDomainWhitelist(whitelist)
	if err != nil || len(allowedDomains) == 0 {
		return false
	}

	normalized, err := normalizeIconProxyDomain(hostname)
	if err != nil || normalized == "" {
		return false
	}

	for _, allowed := range allowedDomains {
		if normalized == allowed || strings.HasSuffix(normalized, "."+allowed) {
			return true
		}
	}
	return false
}
