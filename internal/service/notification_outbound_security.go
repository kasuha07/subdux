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
)

var lookupOutboundHostIPs = func(ctx context.Context, network string, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

func validateOutboundChannelURL(rawURL string, fieldLabel string, requireHTTPS bool) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		if requireHTTPS {
			return fmt.Errorf("%s must start with https://", fieldLabel)
		}
		return fmt.Errorf("%s must start with http:// or https://", fieldLabel)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if requireHTTPS {
		if scheme != "https" {
			return fmt.Errorf("%s must start with https://", fieldLabel)
		}
	} else {
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("%s must start with http:// or https://", fieldLabel)
		}
	}

	return validateOutboundHost(parsed.Hostname(), fieldLabel)
}

func validateOutboundHost(hostname string, fieldLabel string) error {
	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), ".")
	if normalized == "" {
		return fmt.Errorf("%s must include a host", fieldLabel)
	}
	if normalized == "localhost" || strings.HasSuffix(normalized, ".localhost") {
		return fmt.Errorf("%s must not target localhost or private network addresses", fieldLabel)
	}

	if ip := net.ParseIP(normalized); ip != nil && isLocalOrPrivateIP(ip) {
		return fmt.Errorf("%s must not target localhost or private network addresses", fieldLabel)
	}

	return nil
}

func isLocalOrPrivateIP(ip net.IP) bool {
	if ip == nil {
		return true
	}

	if ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() ||
		ip.IsMulticast() {
		return true
	}

	if ipv4 := ip.To4(); ipv4 != nil {
		if ipv4[0] == 100 && ipv4[1] >= 64 && ipv4[1] <= 127 {
			return true
		}
	}

	return false
}

func validateResolvedOutboundHost(hostname string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := resolveSafeOutboundHostIPs(ctx, "ip", hostname, "outbound request url")
	return err
}

func resolveSafeOutboundHostIPs(ctx context.Context, network string, hostname string, fieldLabel string) ([]net.IP, error) {
	if err := validateOutboundHost(hostname, fieldLabel); err != nil {
		return nil, err
	}

	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), ".")
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

	for _, resolvedIP := range ips {
		if isLocalOrPrivateIP(resolvedIP) {
			return nil, fmt.Errorf("%s resolves to localhost or private network addresses", fieldLabel)
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

func doNotificationRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("invalid outbound request")
	}

	if client == nil {
		client = NewSafeOutboundHTTPClient(nil, 15*time.Second)
	}

	proxyMediated := clientUsesOutboundProxy(client)
	if err := validateOutboundRequestHost(req.URL.Hostname(), proxyMediated); err != nil {
		return nil, err
	}

	checkedClient := *client
	originalCheckRedirect := client.CheckRedirect
	checkedClient.CheckRedirect = func(redirectReq *http.Request, via []*http.Request) error {
		if redirectReq == nil || redirectReq.URL == nil {
			return errors.New("invalid outbound request")
		}
		if err := validateOutboundRequestHost(redirectReq.URL.Hostname(), proxyMediated); err != nil {
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

func validateOutboundRequestHost(hostname string, proxyMediated bool) error {
	if proxyMediated {
		return validateOutboundHost(hostname, "outbound request url")
	}
	return validateResolvedOutboundHost(hostname)
}

func (s *NotificationService) newNotificationHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return NewSafeOutboundHTTPClient(s.DB, timeout)
}
