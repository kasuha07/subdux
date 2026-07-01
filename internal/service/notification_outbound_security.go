package service

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

var errRestrictedOutboundTarget = errors.New("restricted outbound target")

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
	return errRestrictedOutboundTarget
}

func validateHTTPURL(rawURL string, fieldLabel string, requireHTTPS bool) (*url.URL, error) {
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

func validateOutboundChannelURL(rawURL string, fieldLabel string, requireHTTPS bool, db *gorm.DB) error {
	parsed, err := validateHTTPURL(rawURL, fieldLabel, requireHTTPS)
	if err != nil {
		return err
	}
	return validateOutboundHost(parsed.Hostname(), fieldLabel, db)
}

func validateOutboundHost(hostname string, fieldLabel string, db *gorm.DB) error {
	cfg := ssrfProtectionConfigForDB(db)
	return validateOutboundHostWithConfig(hostname, fieldLabel, cfg)
}

func validateOutboundHostWithConfig(hostname string, fieldLabel string, cfg ssrfProtectionConfig) error {
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
		if cfg.DomainFilterMode == ssrfFilterModeWhitelist && cfg.IPFilterMode != ssrfFilterModeWhitelist {
			if isRestrictedOutboundIP(ip, cfg.AllowPrivateIP) {
				return restrictedOutboundTargetError{fieldLabel: fieldLabel}
			}
			return ssrfFilterError(fieldLabel, cfg.DomainFilterMode, "domain")
		}
		return validateOutboundIPWithConfig(ip, fieldLabel, cfg)
	}

	switch cfg.DomainFilterMode {
	case ssrfFilterModeWhitelist:
		if !domainMatchesSSRFFilter(normalized, cfg.DomainFilters) {
			return ssrfFilterError(fieldLabel, cfg.DomainFilterMode, "domain")
		}
	case ssrfFilterModeBlacklist:
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

func validateOutboundIPWithConfig(ip net.IP, fieldLabel string, cfg ssrfProtectionConfig) error {
	if !cfg.Enabled {
		return nil
	}
	if isRestrictedOutboundIP(ip, cfg.AllowPrivateIP) {
		return restrictedOutboundTargetError{fieldLabel: fieldLabel}
	}

	switch cfg.IPFilterMode {
	case ssrfFilterModeWhitelist:
		if !ipMatchesSSRFFilter(ip, cfg.IPFilters) {
			return ssrfFilterError(fieldLabel, cfg.IPFilterMode, "ip")
		}
	case ssrfFilterModeBlacklist:
		if ipMatchesSSRFFilter(ip, cfg.IPFilters) {
			return ssrfFilterError(fieldLabel, cfg.IPFilterMode, "ip")
		}
	}
	return nil
}

func validateResolvedOutboundHost(hostname string, db *gorm.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := resolveSafeOutboundHostIPs(ctx, "ip", hostname, "outbound request url", db)
	return err
}

func resolveSafeOutboundHostIPs(ctx context.Context, network string, hostname string, fieldLabel string, db *gorm.DB) ([]net.IP, error) {
	cfg := ssrfProtectionConfigForDB(db)
	if err := validateOutboundHostWithConfig(hostname, fieldLabel, cfg); err != nil {
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
				if errors.Is(err, errRestrictedOutboundTarget) {
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

func doNotificationRequest(client *http.Client, req *http.Request, db *gorm.DB) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("invalid outbound request")
	}

	if client == nil {
		client = NewSafeOutboundHTTPClient(db, 15*time.Second)
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
		return validateOutboundHost(hostname, "outbound request url", db)
	}
	return validateResolvedOutboundHost(hostname, db)
}

func (s *NotificationService) newNotificationHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return NewSafeOutboundHTTPClient(s.DB, timeout)
}

func (s *NotificationService) newFixedNotificationHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return NewOutboundHTTPClient(s.DB, timeout)
}
