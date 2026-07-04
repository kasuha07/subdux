package iconproxy

import (
	"net"
	"sort"
	"strings"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/kasuha07/subdux/internal/service/settings"
)

const DefaultDomainWhitelist = settings.DefaultIconProxyDomainWhitelist

var (
	ErrInvalidIconProxyDomainWhitelist = serviceerr.New(serviceerr.KindInvalid, "invalid icon proxy domain whitelist")
	ErrIconProxyDomainWhitelistTooLong = serviceerr.New(serviceerr.KindInvalid, "icon proxy domain whitelist is too long")
)

func NormalizeDomainWhitelist(raw string) (string, error) {
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

	if !isValidDomainName(domain) {
		return "", ErrInvalidIconProxyDomainWhitelist
	}
	return domain, nil
}

func IsDomainAllowed(hostname string, whitelist string) bool {
	allowedDomains, err := parseIconProxyDomainWhitelist(whitelist)
	if err != nil || len(allowedDomains) == 0 {
		return false
	}
	allowedDomains = expandIconProxyAllowedDomains(allowedDomains)

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

func expandIconProxyAllowedDomains(domains []string) []string {
	if len(domains) == 0 {
		return nil
	}

	expanded := make([]string, 0, len(domains)+1)
	seen := make(map[string]struct{}, len(domains)+1)
	addDomain := func(domain string) {
		if domain == "" {
			return
		}
		if _, exists := seen[domain]; exists {
			return
		}
		seen[domain] = struct{}{}
		expanded = append(expanded, domain)
	}

	for _, domain := range domains {
		addDomain(domain)
		if domain == "google.com" {
			// Google's favicon endpoint redirects to *.gstatic.com. Keep older
			// google.com-only configs working without requiring a manual setting
			// migration first.
			addDomain("gstatic.com")
		}
	}

	sort.Strings(expanded)
	return expanded
}

func isValidDomainName(domain string) bool {
	if strings.Contains(domain, "..") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return false
	}

	labels := strings.Split(domain, ".")
	if len(labels) < 2 {
		return false
	}

	for _, label := range labels {
		if label == "" || len(label) > 63 {
			return false
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for i := 0; i < len(label); i++ {
			ch := label[i]
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			return false
		}
	}

	tld := labels[len(labels)-1]
	return len(tld) >= 2
}
