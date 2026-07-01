package service

import (
	"errors"
	"fmt"
	"net"
	"net/netip"
	"sort"
	"strings"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

const (
	ssrfFilterModeBlacklist = "blacklist"
	ssrfFilterModeWhitelist = "whitelist"

	ssrfProtectionEnabledKey = "ssrf_protection_enabled"
	ssrfAllowPrivateIPKey    = "ssrf_allow_private_ip"
	ssrfDomainFilterModeKey  = "ssrf_domain_filter_mode"
	ssrfDomainFilterListKey  = "ssrf_domain_filter_list"
	ssrfIPFilterModeKey      = "ssrf_ip_filter_mode"
	ssrfIPFilterListKey      = "ssrf_ip_filter_list"
	ssrfFilterResolvedIPsKey = "ssrf_filter_resolved_ips"

	maxSSRFDomainFilterListLength = 500
	maxSSRFIPFilterListLength     = 500
)

var (
	ErrInvalidSSRFFilterMode       = errors.New("ssrf filter mode must be blacklist or whitelist")
	ErrInvalidSSRFDomainFilterList = errors.New("invalid ssrf domain filter list")
	ErrSSRFDomainFilterListTooLong = errors.New("ssrf domain filter list is too long")
	ErrInvalidSSRFIPFilterList     = errors.New("invalid ssrf ip filter list")
	ErrSSRFIPFilterListTooLong     = errors.New("ssrf ip filter list is too long")
)

type ssrfProtectionConfig struct {
	Enabled          bool
	AllowPrivateIP   bool
	DomainFilterMode string
	DomainFilters    []string
	IPFilterMode     string
	IPFilters        []netip.Prefix
	FilterResolvedIP bool
}

func defaultSSRFProtectionConfig() ssrfProtectionConfig {
	return ssrfProtectionConfig{
		Enabled:          true,
		AllowPrivateIP:   false,
		DomainFilterMode: ssrfFilterModeBlacklist,
		DomainFilters:    nil,
		IPFilterMode:     ssrfFilterModeBlacklist,
		IPFilters:        nil,
		FilterResolvedIP: true,
	}
}

func loadSSRFProtectionConfig(db *gorm.DB) (ssrfProtectionConfig, error) {
	cfg := defaultSSRFProtectionConfig()
	if db == nil {
		return cfg, nil
	}

	var items []model.SystemSetting
	if err := db.Where("key IN ?", []string{
		ssrfProtectionEnabledKey,
		ssrfAllowPrivateIPKey,
		ssrfDomainFilterModeKey,
		ssrfDomainFilterListKey,
		ssrfIPFilterModeKey,
		ssrfIPFilterListKey,
		ssrfFilterResolvedIPsKey,
	}).Find(&items).Error; err != nil {
		return cfg, err
	}

	for _, item := range items {
		value := strings.TrimSpace(item.Value)
		switch item.Key {
		case ssrfProtectionEnabledKey:
			cfg.Enabled = value == "true"
		case ssrfAllowPrivateIPKey:
			cfg.AllowPrivateIP = value == "true"
		case ssrfDomainFilterModeKey:
			if mode, err := normalizeSSRFFilterMode(value); err == nil {
				cfg.DomainFilterMode = mode
			}
		case ssrfDomainFilterListKey:
			if domains, err := parseSSRFDomainFilterList(value); err == nil {
				cfg.DomainFilters = domains
			}
		case ssrfIPFilterModeKey:
			if mode, err := normalizeSSRFFilterMode(value); err == nil {
				cfg.IPFilterMode = mode
			}
		case ssrfIPFilterListKey:
			if prefixes, err := parseSSRFIPFilterList(value); err == nil {
				cfg.IPFilters = prefixes
			}
		case ssrfFilterResolvedIPsKey:
			cfg.FilterResolvedIP = value == "true"
		}
	}

	return cfg, nil
}

func ssrfProtectionConfigForDB(db *gorm.DB) ssrfProtectionConfig {
	cfg, err := loadSSRFProtectionConfig(db)
	if err != nil {
		return defaultSSRFProtectionConfig()
	}
	return cfg
}

func normalizeSSRFFilterMode(mode string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(mode))
	if normalized == "" {
		return ssrfFilterModeBlacklist, nil
	}
	switch normalized {
	case ssrfFilterModeBlacklist, ssrfFilterModeWhitelist:
		return normalized, nil
	default:
		return "", ErrInvalidSSRFFilterMode
	}
}

func normalizeSSRFDomainFilterList(raw string) (string, error) {
	domains, err := parseSSRFDomainFilterList(raw)
	if err != nil {
		return "", err
	}
	if len(domains) == 0 {
		return "", nil
	}

	normalized := strings.Join(domains, "\n")
	if len(normalized) > maxSSRFDomainFilterListLength {
		return "", ErrSSRFDomainFilterListTooLong
	}
	return normalized, nil
}

func parseSSRFDomainFilterList(raw string) ([]string, error) {
	parts := strings.FieldsFunc(raw, ssrfListSeparator)
	if len(parts) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(parts))
	domains := make([]string, 0, len(parts))
	for _, part := range parts {
		domain, err := normalizeSSRFDomainPattern(part)
		if err != nil {
			return nil, ErrInvalidSSRFDomainFilterList
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

func normalizeSSRFDomainPattern(raw string) (string, error) {
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
		strings.Contains(domain, ":") ||
		strings.ContainsAny(domain, " \t\r\n") {
		return "", ErrInvalidSSRFDomainFilterList
	}
	if ip := net.ParseIP(domain); ip != nil {
		return "", ErrInvalidSSRFDomainFilterList
	}
	if !isValidHostnamePattern(domain) {
		return "", ErrInvalidSSRFDomainFilterList
	}
	return domain, nil
}

func isValidHostnamePattern(hostname string) bool {
	if hostname == "" || len(hostname) > 253 {
		return false
	}
	if strings.Contains(hostname, "..") ||
		strings.HasPrefix(hostname, ".") ||
		strings.HasSuffix(hostname, ".") {
		return false
	}

	labels := strings.Split(hostname, ".")
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
	return true
}

func normalizeSSRFIPFilterList(raw string) (string, error) {
	prefixes, err := parseSSRFIPFilterList(raw)
	if err != nil {
		return "", err
	}
	if len(prefixes) == 0 {
		return "", nil
	}

	values := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		values = append(values, ssrfIPFilterPrefixString(prefix))
	}

	normalized := strings.Join(values, "\n")
	if len(normalized) > maxSSRFIPFilterListLength {
		return "", ErrSSRFIPFilterListTooLong
	}
	return normalized, nil
}

func parseSSRFIPFilterList(raw string) ([]netip.Prefix, error) {
	parts := strings.FieldsFunc(raw, ssrfListSeparator)
	if len(parts) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{}, len(parts))
	prefixes := make([]netip.Prefix, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}

		var prefix netip.Prefix
		var err error
		if strings.Contains(value, "/") {
			prefix, err = netip.ParsePrefix(value)
			if err == nil {
				prefix = prefix.Masked()
			}
		} else {
			var addr netip.Addr
			addr, err = netip.ParseAddr(value)
			if err == nil {
				if addr.Is4() {
					prefix = netip.PrefixFrom(addr, 32)
				} else {
					prefix = netip.PrefixFrom(addr, 128)
				}
			}
		}
		if err != nil || !prefix.IsValid() {
			return nil, ErrInvalidSSRFIPFilterList
		}

		key := prefix.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		prefixes = append(prefixes, prefix)
	}

	sort.Slice(prefixes, func(i, j int) bool {
		return prefixes[i].String() < prefixes[j].String()
	})
	return prefixes, nil
}

func ssrfIPFilterPrefixString(prefix netip.Prefix) string {
	if !prefix.IsValid() {
		return ""
	}
	addr := prefix.Addr()
	if (addr.Is4() && prefix.Bits() == 32) || (addr.Is6() && prefix.Bits() == 128) {
		return addr.String()
	}
	return prefix.String()
}

func ssrfListSeparator(r rune) bool {
	return r == '\n' || r == ',' || r == ';'
}

func validateSSRFProtectionSettings(input UpdateSettingsInput) error {
	if input.SSRFDomainFilterMode != nil {
		if _, err := normalizeSSRFFilterMode(*input.SSRFDomainFilterMode); err != nil {
			return err
		}
	}
	if input.SSRFIPFilterMode != nil {
		if _, err := normalizeSSRFFilterMode(*input.SSRFIPFilterMode); err != nil {
			return err
		}
	}
	if input.SSRFDomainFilterList != nil {
		if _, err := normalizeSSRFDomainFilterList(*input.SSRFDomainFilterList); err != nil {
			return err
		}
	}
	if input.SSRFIPFilterList != nil {
		if _, err := normalizeSSRFIPFilterList(*input.SSRFIPFilterList); err != nil {
			return err
		}
	}
	return nil
}

func domainMatchesSSRFFilter(hostname string, filters []string) bool {
	normalized, err := normalizeOutboundHostname(hostname)
	if err != nil || net.ParseIP(normalized) != nil {
		return false
	}

	for _, filter := range filters {
		if normalized == filter || strings.HasSuffix(normalized, "."+filter) {
			return true
		}
	}
	return false
}

func ipMatchesSSRFFilter(ip net.IP, filters []netip.Prefix) bool {
	addr, ok := netIPToAddr(ip)
	if !ok {
		return false
	}
	for _, filter := range filters {
		if filter.Contains(addr) {
			return true
		}
	}
	return false
}

func netIPToAddr(ip net.IP) (netip.Addr, bool) {
	if ip == nil {
		return netip.Addr{}, false
	}
	if ipv4 := ip.To4(); ipv4 != nil {
		return netip.AddrFrom4([4]byte{ipv4[0], ipv4[1], ipv4[2], ipv4[3]}), true
	}
	ipv6 := ip.To16()
	if ipv6 == nil {
		return netip.Addr{}, false
	}
	return netip.AddrFrom16([16]byte{
		ipv6[0], ipv6[1], ipv6[2], ipv6[3],
		ipv6[4], ipv6[5], ipv6[6], ipv6[7],
		ipv6[8], ipv6[9], ipv6[10], ipv6[11],
		ipv6[12], ipv6[13], ipv6[14], ipv6[15],
	}), true
}

func ssrfFilterError(fieldLabel string, mode string, targetType string) error {
	if mode == ssrfFilterModeWhitelist {
		return fmt.Errorf("%s is not allowed by ssrf %s whitelist", fieldLabel, targetType)
	}
	return fmt.Errorf("%s is blocked by ssrf %s blacklist", fieldLabel, targetType)
}
