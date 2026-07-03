package notification

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/kasuha07/subdux/internal/service/outbound"
	"gorm.io/gorm"
)

type ssrfProtectionConfig = outbound.Policy

const (
	ssrfFilterModeBlacklist = outbound.FilterModeBlacklist
	ssrfFilterModeWhitelist = outbound.FilterModeWhitelist
)

var errRestrictedOutboundTarget = outbound.ErrRestrictedOutboundTarget

func validateOutboundChannelURL(rawURL string, fieldLabel string, requireHTTPS bool, db *gorm.DB) error {
	return outbound.ValidateChannelURL(rawURL, fieldLabel, requireHTTPS, db)
}

func validateOutboundHost(hostname string, fieldLabel string, db *gorm.DB) error {
	return outbound.ValidateHost(hostname, fieldLabel, db)
}

func validateResolvedOutboundHost(hostname string, db *gorm.DB) error {
	return outbound.ValidateResolvedHost(hostname, db)
}

func validateOutboundHostWithConfig(hostname string, fieldLabel string, cfg ssrfProtectionConfig) error {
	return outbound.ValidateHostWithConfig(hostname, fieldLabel, cfg)
}

func resolveSafeOutboundHostIPs(ctx context.Context, network string, hostname string, fieldLabel string, db *gorm.DB) ([]net.IP, error) {
	return outbound.ResolveSafeHostIPs(ctx, network, hostname, fieldLabel, db)
}

func doNotificationRequest(client *http.Client, req *http.Request, db *gorm.DB) (*http.Response, error) {
	return outbound.DoNotificationRequest(client, req, db)
}

func (s *Service) newNotificationHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client, err := outbound.BuildHTTPClientWithTimeout(context.Background(), s.DB, outbound.PurposeNotification, timeout)
	if err != nil {
		return outbound.NewSafeOutboundHTTPClient(s.DB, timeout)
	}
	return client
}

func (s *Service) newFixedNotificationHTTPClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client, err := outbound.BuildHTTPClientWithTimeout(context.Background(), s.DB, outbound.PurposeFixedNotification, timeout)
	if err != nil {
		return outbound.NewOutboundHTTPClient(s.DB, timeout)
	}
	return client
}
