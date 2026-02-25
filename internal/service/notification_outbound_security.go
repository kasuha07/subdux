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
	if err := validateOutboundHost(hostname, "outbound request url"); err != nil {
		return err
	}

	normalized := strings.TrimSuffix(strings.ToLower(strings.TrimSpace(hostname)), ".")
	if net.ParseIP(normalized) != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ips, err := lookupOutboundHostIPs(ctx, "ip", normalized)
	if err != nil {
		return fmt.Errorf("failed to resolve outbound request host: %w", err)
	}
	if len(ips) == 0 {
		return errors.New("outbound request url host resolves to no addresses")
	}

	for _, resolvedIP := range ips {
		if isLocalOrPrivateIP(resolvedIP) {
			return errors.New("outbound request url resolves to localhost or private network addresses")
		}
	}

	return nil
}

func doNotificationRequest(client *http.Client, req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("invalid outbound request")
	}

	if err := validateResolvedOutboundHost(req.URL.Hostname()); err != nil {
		return nil, err
	}

	if client == nil {
		client = http.DefaultClient
	}

	checkedClient := *client
	originalCheckRedirect := client.CheckRedirect
	checkedClient.CheckRedirect = func(redirectReq *http.Request, via []*http.Request) error {
		if redirectReq == nil || redirectReq.URL == nil {
			return errors.New("invalid outbound request")
		}
		if err := validateResolvedOutboundHost(redirectReq.URL.Hostname()); err != nil {
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
