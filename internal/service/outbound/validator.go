package outbound

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"gorm.io/gorm"
)

var lookupOutboundHostIPs = func(ctx context.Context, network string, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

var ErrRestrictedOutboundTarget = errors.New("restricted outbound target")

func SetLookupHostIPsForTest(fn func(context.Context, string, string) ([]net.IP, error)) func() {
	previous := lookupOutboundHostIPs
	lookupOutboundHostIPs = fn
	return func() {
		lookupOutboundHostIPs = previous
	}
}

type restrictedOutboundTargetError struct {
	fieldLabel string
	resolved   bool
}

func (e restrictedOutboundTargetError) Error() string {
	if e.resolved {
		return fmt.Sprintf("%s resolves to localhost or private network addresses", e.fieldLabel)
	}
	return fmt.Sprintf("%s must not target localhost or private network addresses", e.fieldLabel)
}

func (e restrictedOutboundTargetError) Unwrap() error {
	return ErrRestrictedOutboundTarget
}

func ValidateHTTPURL(rawURL string, fieldLabel string, requireHTTPS bool) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		if requireHTTPS {
			return nil, fmt.Errorf("%s must start with https://", fieldLabel)
		}
		return nil, fmt.Errorf("%s must start with http:// or https://", fieldLabel)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if requireHTTPS {
		if scheme != "https" {
			return nil, fmt.Errorf("%s must start with https://", fieldLabel)
		}
	} else {
		if scheme != "http" && scheme != "https" {
			return nil, fmt.Errorf("%s must start with http:// or https://", fieldLabel)
		}
	}

	return parsed, nil
}

func ValidateChannelURL(rawURL string, fieldLabel string, requireHTTPS bool, db *gorm.DB) error {
	return ValidateURLWithOptions(context.Background(), db, rawURL, fieldLabel, requireHTTPS, PurposeNotification)
}

func ValidateURL(ctx context.Context, db *gorm.DB, rawURL string, purpose Purpose) error {
	return ValidateURLWithOptions(ctx, db, rawURL, "outbound request url", false, purpose)
}

func ValidateURLWithOptions(_ context.Context, db *gorm.DB, rawURL string, fieldLabel string, requireHTTPS bool, purpose Purpose) error {
	parsed, err := ValidateHTTPURL(rawURL, fieldLabel, requireHTTPS)
	if err != nil {
		return err
	}
	return ValidateHostForPurpose(parsed.Hostname(), fieldLabel, db, purpose)
}

func ValidateHost(hostname string, fieldLabel string, db *gorm.DB) error {
	return ValidateHostForPurpose(hostname, fieldLabel, db, "")
}

func ValidateHostForPurpose(hostname string, fieldLabel string, db *gorm.DB, purpose Purpose) error {
	if !PurposeAppliesSSRFPolicy(purpose) {
		if _, err := normalizeOutboundHostname(hostname); err != nil {
			return fmt.Errorf("%s must include a host", fieldLabel)
		}
		return nil
	}

	cfg := PolicyForDB(db)
	return ValidateHostWithConfig(hostname, fieldLabel, cfg)
}

// PurposeAppliesSSRFPolicy is the trust-boundary map for hostname
// policy. User-configurable outbound targets are checked against the SSRF
// policy; administrator-configured or fixed provider endpoints are trusted as
// administrator policy and rely on the configured proxy/network ACL boundary.
func PurposeAppliesSSRFPolicy(purpose Purpose) bool {
	switch purpose {
	case PurposeOIDC,
		PurposeFixedNotification,
		PurposeIconProxy,
		PurposeExchangeRate:
		return false
	default:
		return true
	}
}

func ValidateHostWithConfig(hostname string, fieldLabel string, cfg Policy) error {
	normalized, err := normalizeOutboundHostname(hostname)
	if err != nil {
		return fmt.Errorf("%s must include a host", fieldLabel)
	}

	if !cfg.Enabled {
		return nil
	}

	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") {
		return restrictedOutboundTargetError{fieldLabel: fieldLabel}
	}

	if ip := net.ParseIP(normalized); ip != nil {
		if cfg.DomainFilterMode == FilterModeWhitelist && cfg.IPFilterMode != FilterModeWhitelist {
			if isRestrictedOutboundIP(ip, cfg.AllowPrivateIP) {
				return restrictedOutboundTargetError{fieldLabel: fieldLabel}
			}
			return ssrfFilterError(fieldLabel, cfg.DomainFilterMode, "domain")
		}
		return validateOutboundIPWithConfig(ip, fieldLabel, cfg)
	}

	switch cfg.DomainFilterMode {
	case FilterModeWhitelist:
		if !domainMatchesSSRFFilter(normalized, cfg.DomainFilters) {
			return ssrfFilterError(fieldLabel, cfg.DomainFilterMode, "domain")
		}
	case FilterModeBlacklist:
		if domainMatchesSSRFFilter(normalized, cfg.DomainFilters) {
			return ssrfFilterError(fieldLabel, cfg.DomainFilterMode, "domain")
		}
	}

	return nil
}

func normalizeOutboundHostname(hostname string) (string, error) {
	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), ".")
	if normalized == "" {
		return "", errors.New("hostname is empty")
	}
	return normalized, nil
}

func isRestrictedOutboundIP(ip net.IP, allowPrivateIP bool) bool {
	if ip == nil {
		return true
	}

	if ip.IsPrivate() {
		return !allowPrivateIP
	}

	return ip.IsLoopback() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() ||
		isCarrierGradeNATIP(ip)
}

func isCarrierGradeNATIP(ip net.IP) bool {
	ipv4 := ip.To4()
	return ipv4 != nil && ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127
}

func validateOutboundIPWithConfig(ip net.IP, fieldLabel string, cfg Policy) error {
	if !cfg.Enabled {
		return nil
	}
	if isRestrictedOutboundIP(ip, cfg.AllowPrivateIP) {
		return restrictedOutboundTargetError{fieldLabel: fieldLabel}
	}

	switch cfg.IPFilterMode {
	case FilterModeWhitelist:
		if !ipMatchesSSRFFilter(ip, cfg.IPFilters) {
			return ssrfFilterError(fieldLabel, cfg.IPFilterMode, "ip")
		}
	case FilterModeBlacklist:
		if ipMatchesSSRFFilter(ip, cfg.IPFilters) {
			return ssrfFilterError(fieldLabel, cfg.IPFilterMode, "ip")
		}
	}
	return nil
}

func ValidateResolvedHost(hostname string, db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := ResolveSafeHostIPs(ctx, "ip", hostname, "outbound request url", db)
	return err
}

func ResolveSafeHostIPs(ctx context.Context, network string, hostname string, fieldLabel string, db *gorm.DB) ([]net.IP, error) {
	cfg := PolicyForDB(db)
	if err := ValidateHostWithConfig(hostname, fieldLabel, cfg); err != nil {
		return nil, err
	}

	normalized, err := normalizeOutboundHostname(hostname)
	if err != nil {
		return nil, fmt.Errorf("%s must include a host", fieldLabel)
	}
	if ip := net.ParseIP(normalized); ip != nil {
		return []net.IP{ip}, nil
	}

	ips, err := lookupOutboundHostIPs(ctx, lookupIPNetwork(network), normalized)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s host: %w", fieldLabel, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("%s host resolves to no addresses", fieldLabel)
	}

	if cfg.Enabled && cfg.FilterResolvedIP {
		for _, resolvedIP := range ips {
			if err := validateOutboundIPWithConfig(resolvedIP, fieldLabel, cfg); err != nil {
				if errors.Is(err, ErrRestrictedOutboundTarget) {
					return nil, restrictedOutboundTargetError{fieldLabel: fieldLabel, resolved: true}
				}
				return nil, err
			}
		}
	}

	return ips, nil
}

func lookupIPNetwork(network string) string {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "tcp4", "ip4":
		return "ip4"
	case "tcp6", "ip6":
		return "ip6"
	default:
		return "ip"
	}
}

func DoNotificationRequest(client *http.Client, req *http.Request, db *gorm.DB) (*http.Response, error) {
	return DoRequest(client, req, db, PurposeNotification)
}

func DoRequest(client *http.Client, req *http.Request, db *gorm.DB, purpose Purpose) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("invalid outbound request")
	}

	if client == nil {
		var err error
		client, err = BuildHTTPClientWithTimeout(req.Context(), db, purpose, 15*time.Second)
		if err != nil {
			return nil, err
		}
	}

	proxyMediated := clientUsesOutboundProxy(client)
	if err := validateOutboundRequestHost(req.URL.Hostname(), proxyMediated, db); err != nil {
		return nil, err
	}

	checkedClient := *client
	originalCheckRedirect := client.CheckRedirect
	checkedClient.CheckRedirect = func(redirectReq *http.Request, via []*http.Request) error {
		if redirectReq == nil || redirectReq.URL == nil {
			return errors.New("invalid outbound request")
		}
		if err := validateOutboundRequestHost(redirectReq.URL.Hostname(), proxyMediated, db); err != nil {
			return err
		}
		if originalCheckRedirect != nil {
			return originalCheckRedirect(redirectReq, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}

	return checkedClient.Do(req)
}

func validateOutboundRequestHost(hostname string, proxyMediated bool, db *gorm.DB) error {
	if proxyMediated {
		return ValidateHost(hostname, "outbound request url", db)
	}
	return ValidateResolvedHost(hostname, db)
}
