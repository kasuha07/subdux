package outbound

import (
	"context"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

var ErrInvalidSSRFTestTarget = serviceerr.New(serviceerr.KindInvalid, "ssrf test target must be a valid hostname or ip address")

type SSRFTestInput struct {
	Target string
}

type SSRFTestResult struct {
	Target                  string
	Host                    string
	Allowed                 bool
	Reason                  string
	ResolvedIPs             []string
	ProtectionEnabled       bool
	AllowPrivateIP          bool
	DomainFilterMode        string
	IPFilterMode            string
	FilterResolvedIPs       bool
	ProxyMediated           bool
	ResolvedIPFilterApplied bool
}

func TestSSRF(ctx context.Context, db *gorm.DB, input SSRFTestInput) (*SSRFTestResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	host, err := normalizeSSRFTestTarget(input.Target)
	if err != nil {
		return nil, err
	}

	cfg := PolicyForDB(db)
	proxyMediated := TestUsesOutboundProxy(db)
	result := &SSRFTestResult{
		Target:                  strings.TrimSpace(input.Target),
		Host:                    host,
		ProtectionEnabled:       cfg.Enabled,
		AllowPrivateIP:          cfg.AllowPrivateIP,
		DomainFilterMode:        cfg.DomainFilterMode,
		IPFilterMode:            cfg.IPFilterMode,
		FilterResolvedIPs:       cfg.FilterResolvedIP,
		ProxyMediated:           proxyMediated,
		ResolvedIPFilterApplied: cfg.Enabled && cfg.FilterResolvedIP && !proxyMediated,
		ResolvedIPs:             []string{},
	}

	if proxyMediated {
		if err := ValidateHostForPurpose(host, "ssrf test target", db, PurposeAdminTest); err != nil {
			result.Allowed = false
			result.Reason = err.Error()
			return result, nil
		}
		result.Allowed = true
		result.Reason = TestAllowedReason(cfg)
		return result, nil
	}

	resolveCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ips, err := ResolveSafeHostIPs(resolveCtx, "ip", host, "ssrf test target", db)
	if err != nil {
		result.Allowed = false
		result.Reason = err.Error()
		return result, nil
	}
	result.ResolvedIPs = StringifyIPs(ips)
	result.Allowed = true
	result.Reason = TestAllowedReason(cfg)
	return result, nil
}

func normalizeSSRFTestTarget(raw string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" || len(target) > 253 {
		return "", ErrInvalidSSRFTestTarget
	}

	host := target
	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)
		if err != nil || parsed.Hostname() == "" {
			return "", ErrInvalidSSRFTestTarget
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return "", ErrInvalidSSRFTestTarget
		}
		host = parsed.Hostname()
	} else if splitHost, _, err := net.SplitHostPort(target); err == nil {
		host = splitHost
	} else if strings.HasPrefix(target, "[") && strings.HasSuffix(target, "]") {
		host = strings.TrimPrefix(strings.TrimSuffix(target, "]"), "[")
	}

	host, err := normalizeOutboundHostname(host)
	if err != nil {
		return "", ErrInvalidSSRFTestTarget
	}
	if strings.ContainsAny(host, "/\\@?# \t\r\n") {
		return "", ErrInvalidSSRFTestTarget
	}
	if ip := net.ParseIP(host); ip != nil {
		return host, nil
	}
	if !isValidHostnamePattern(host) {
		return "", ErrInvalidSSRFTestTarget
	}
	return host, nil
}

func TestUsesOutboundProxy(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	cfg, err := LoadSystemProxyConfig(db)
	if err != nil || !cfg.Enabled {
		return false
	}
	_, err = NormalizeSystemProxyURL(cfg.Type, cfg.URL)
	return err == nil
}

func TestAllowedReason(cfg Policy) string {
	if !cfg.Enabled {
		return "ssrf protection is disabled"
	}
	return "target is allowed by ssrf settings"
}

func stringifyIPs(ips []net.IP) []string {
	values := make([]string, 0, len(ips))
	for _, ip := range ips {
		if ip == nil {
			continue
		}
		values = append(values, ip.String())
	}
	return values
}

func NormalizeTestTarget(raw string) (string, error) {
	return normalizeSSRFTestTarget(raw)
}

func StringifyIPs(ips []net.IP) []string {
	return stringifyIPs(ips)
}
