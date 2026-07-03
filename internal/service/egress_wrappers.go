package service

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/kasuha07/subdux/internal/service/outbound"
	"gorm.io/gorm"
)

type outboundPurpose = outbound.Purpose
type outboundPolicy = outbound.Policy
type ssrfProtectionConfig = outbound.Policy
type systemProxyConfig = outbound.SystemProxyConfig

const (
	outboundPurposeOIDC              = outbound.PurposeOIDC
	outboundPurposeNotification      = outbound.PurposeNotification
	outboundPurposeFixedNotification = outbound.PurposeFixedNotification
	outboundPurposeIconProxy         = outbound.PurposeIconProxy
	outboundPurposeAdminTest         = outbound.PurposeAdminTest
	outboundPurposeExchangeRate      = outbound.PurposeExchangeRate

	systemProxyTypeHTTP   = outbound.SystemProxyTypeHTTP
	systemProxyTypeSOCKS5 = outbound.SystemProxyTypeSOCKS5

	ssrfFilterModeBlacklist = outbound.FilterModeBlacklist
	ssrfFilterModeWhitelist = outbound.FilterModeWhitelist

	ssrfProtectionEnabledKey = outbound.ProtectionEnabledKey
	ssrfAllowPrivateIPKey    = outbound.AllowPrivateIPKey
	ssrfDomainFilterModeKey  = outbound.DomainFilterModeKey
	ssrfDomainFilterListKey  = outbound.DomainFilterListKey
	ssrfIPFilterModeKey      = outbound.IPFilterModeKey
	ssrfIPFilterListKey      = outbound.IPFilterListKey
	ssrfFilterResolvedIPsKey = outbound.FilterResolvedIPsKey
)

var (
	ErrInvalidSystemProxyType      = outbound.ErrInvalidSystemProxyType
	ErrInvalidSystemProxyURL       = outbound.ErrInvalidSystemProxyURL
	ErrInvalidSSRFFilterMode       = outbound.ErrInvalidSSRFFilterMode
	ErrInvalidSSRFDomainFilterList = outbound.ErrInvalidSSRFDomainFilterList
	ErrSSRFDomainFilterListTooLong = outbound.ErrSSRFDomainFilterListTooLong
	ErrInvalidSSRFIPFilterList     = outbound.ErrInvalidSSRFIPFilterList
	ErrSSRFIPFilterListTooLong     = outbound.ErrSSRFIPFilterListTooLong
	ErrInvalidSSRFTestTarget       = outbound.ErrInvalidSSRFTestTarget
	errRestrictedOutboundTarget    = outbound.ErrRestrictedOutboundTarget
)

func NewOutboundHTTPClient(db *gorm.DB, timeout time.Duration) *http.Client {
	return outbound.NewOutboundHTTPClient(db, timeout)
}

func NewSafeOutboundHTTPClient(db *gorm.DB, timeout time.Duration) *http.Client {
	return outbound.NewSafeOutboundHTTPClient(db, timeout)
}

func NewOutboundDialContext(db *gorm.DB, timeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return outbound.NewOutboundDialContext(db, timeout)
}

func NewSafeOutboundDialContext(db *gorm.DB, timeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	return outbound.NewSafeOutboundDialContext(db, timeout)
}

func buildOutboundHTTPClient(ctx context.Context, db *gorm.DB, purpose outboundPurpose) (*http.Client, error) {
	return outbound.BuildHTTPClient(ctx, db, purpose)
}

func buildOutboundHTTPClientWithTimeout(ctx context.Context, db *gorm.DB, purpose outboundPurpose, timeout time.Duration) (*http.Client, error) {
	return outbound.BuildHTTPClientWithTimeout(ctx, db, purpose, timeout)
}

func validateHTTPURL(rawURL string, fieldLabel string, requireHTTPS bool) (*url.URL, error) {
	return outbound.ValidateHTTPURL(rawURL, fieldLabel, requireHTTPS)
}

func validateOutboundChannelURL(rawURL string, fieldLabel string, requireHTTPS bool, db *gorm.DB) error {
	return outbound.ValidateChannelURL(rawURL, fieldLabel, requireHTTPS, db)
}

func validateOutboundURL(ctx context.Context, db *gorm.DB, rawURL string, purpose outboundPurpose) error {
	return outbound.ValidateURL(ctx, db, rawURL, purpose)
}

func validateOutboundHost(hostname string, fieldLabel string, db *gorm.DB) error {
	return outbound.ValidateHost(hostname, fieldLabel, db)
}

func validateOutboundHostForPurpose(hostname string, fieldLabel string, db *gorm.DB, purpose outboundPurpose) error {
	return outbound.ValidateHostForPurpose(hostname, fieldLabel, db, purpose)
}

func validateOutboundHostWithConfig(hostname string, fieldLabel string, cfg outboundPolicy) error {
	return outbound.ValidateHostWithConfig(hostname, fieldLabel, cfg)
}

func validateResolvedOutboundHost(hostname string, db *gorm.DB) error {
	return outbound.ValidateResolvedHost(hostname, db)
}

func resolveSafeOutboundHostIPs(ctx context.Context, network string, hostname string, fieldLabel string, db *gorm.DB) ([]net.IP, error) {
	return outbound.ResolveSafeHostIPs(ctx, network, hostname, fieldLabel, db)
}

func doNotificationRequest(client *http.Client, req *http.Request, db *gorm.DB) (*http.Response, error) {
	return outbound.DoNotificationRequest(client, req, db)
}

func doOutboundRequest(client *http.Client, req *http.Request, db *gorm.DB, purpose outboundPurpose) (*http.Response, error) {
	return outbound.DoRequest(client, req, db, purpose)
}

func loadSystemProxyConfig(db *gorm.DB) (systemProxyConfig, error) {
	return outbound.LoadSystemProxyConfig(db)
}

func validateIncomingSystemProxySettings(tx *gorm.DB, input UpdateSettingsInput) error {
	return outbound.ValidateIncomingSystemProxySettings(tx, input.SystemProxyEnabled, input.SystemProxyType, input.SystemProxyURL)
}

func validateSystemProxySettings(proxyType string, proxyURL string, enabled bool) error {
	return outbound.ValidateSystemProxySettings(proxyType, proxyURL, enabled)
}

func normalizeSystemProxyType(proxyType string) (string, error) {
	return outbound.NormalizeSystemProxyType(proxyType)
}

func normalizeSystemProxyURL(proxyType string, rawURL string) (*url.URL, error) {
	return outbound.NormalizeSystemProxyURL(proxyType, rawURL)
}

func loadOutboundPolicy(ctx context.Context, db *gorm.DB) (outboundPolicy, error) {
	return outbound.LoadPolicyContext(ctx, db)
}

func outboundPolicyForDB(db *gorm.DB) outboundPolicy {
	return outbound.PolicyForDB(db)
}

func normalizeSSRFFilterMode(mode string) (string, error) {
	return outbound.NormalizeFilterMode(mode)
}

func normalizeSSRFDomainFilterList(raw string) (string, error) {
	return outbound.NormalizeDomainFilterList(raw)
}

func normalizeSSRFIPFilterList(raw string) (string, error) {
	return outbound.NormalizeIPFilterList(raw)
}

func validateSSRFProtectionSettings(input UpdateSettingsInput) error {
	return outbound.ValidatePolicyUpdate(input.SSRFDomainFilterMode, input.SSRFIPFilterMode, input.SSRFDomainFilterList, input.SSRFIPFilterList)
}

func normalizeSSRFTestTarget(raw string) (string, error) {
	return outbound.NormalizeTestTarget(raw)
}

func ssrfTestUsesOutboundProxy(db *gorm.DB) bool {
	return outbound.TestUsesOutboundProxy(db)
}

func ssrfTestAllowedReason(cfg outboundPolicy) string {
	return outbound.TestAllowedReason(cfg)
}

func stringifyIPs(ips []net.IP) []string {
	return outbound.StringifyIPs(ips)
}
