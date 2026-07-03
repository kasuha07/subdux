package service

import (
	"net"
	"net/url"
	"strings"

	"gorm.io/gorm"
)

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

func ssrfTestUsesOutboundProxy(db *gorm.DB) bool {
	if db == nil {
		return false
	}
	cfg, err := loadSystemProxyConfig(db)
	if err != nil || !cfg.Enabled {
		return false
	}
	_, err = normalizeSystemProxyURL(cfg.Type, cfg.URL)
	return err == nil
}

func ssrfTestAllowedReason(cfg outboundPolicy) string {
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
