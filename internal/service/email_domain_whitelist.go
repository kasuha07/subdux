package service

import (
	"errors"
	"net/mail"
	"sort"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

var (
	ErrInvalidEmailDomainWhitelist = errors.New("invalid email domain whitelist")
	ErrEmailDomainWhitelistTooLong = errors.New("email domain whitelist is too long")
	ErrEmailDomainNotAllowed       = errors.New("email domain is not allowed")
)

func normalizeEmailDomainWhitelist(raw string) (string, error) {
	domains, err := parseEmailDomainWhitelist(raw)
	if err != nil {
		return "", err
	}
	if len(domains) == 0 {
		return "", nil
	}

	normalized := strings.Join(domains, "\n")
	if len(normalized) > 500 {
		return "", ErrEmailDomainWhitelistTooLong
	}
	return normalized, nil
}

func parseEmailDomainWhitelist(raw string) ([]string, error) {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '\n' || r == ',' || r == ';'
	})
	if len(parts) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})
	domains := make([]string, 0, len(parts))
	for _, part := range parts {
		domain, err := normalizeEmailDomain(part)
		if err != nil {
			return nil, ErrInvalidEmailDomainWhitelist
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

func normalizeEmailDomain(raw string) (string, error) {
	domain := strings.ToLower(strings.TrimSpace(raw))
	domain = strings.TrimLeft(domain, "@")
	domain = strings.TrimRight(domain, ".")
	if domain == "" {
		return "", nil
	}

	if !isValidEmailDomain(domain) {
		return "", ErrInvalidEmailDomainWhitelist
	}
	return domain, nil
}

func isValidEmailDomain(domain string) bool {
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
	if len(tld) < 2 {
		return false
	}
	return true
}

func extractEmailDomain(email string) (string, error) {
	normalized, err := sanitizeAndValidateEmail(email)
	if err != nil {
		return "", err
	}

	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", ErrInvalidEmail
	}

	at := strings.LastIndex(normalized, "@")
	if at <= 0 || at >= len(normalized)-1 {
		return "", ErrInvalidEmail
	}
	domain, err := normalizeEmailDomain(normalized[at+1:])
	if err != nil || domain == "" {
		return "", ErrInvalidEmail
	}
	return domain, nil
}

func isEmailDomainAllowed(email string, whitelist string) bool {
	allowedDomains, err := parseEmailDomainWhitelist(whitelist)
	if err != nil || len(allowedDomains) == 0 {
		return err == nil
	}

	domain, err := extractEmailDomain(email)
	if err != nil {
		return false
	}

	for _, allowed := range allowedDomains {
		if domain == allowed || strings.HasSuffix(domain, "."+allowed) {
			return true
		}
	}
	return false
}

func (s *AuthService) enforceEmailDomainWhitelist(email string) error {
	whitelist, err := getEmailDomainWhitelist(s.DB)
	if err != nil {
		return err
	}
	if whitelist == "" {
		return nil
	}
	if !isEmailDomainAllowed(email, whitelist) {
		return ErrEmailDomainNotAllowed
	}
	return nil
}

func getEmailDomainWhitelist(db *gorm.DB) (string, error) {
	var setting model.SystemSetting
	if err := db.Where("key = ?", "email_domain_whitelist").First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return setting.Value, nil
}
