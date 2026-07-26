package outbound

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"
)

type outboundHTTPClientOptions struct {
	Timeout      time.Duration
	DB           *gorm.DB
	SecureEgress bool
	SecureDialer *safeOutboundDialer
}

type Purpose string

const (
	PurposeOIDC              Purpose = "oidc"
	PurposeNotification      Purpose = "notification"
	PurposeFixedNotification Purpose = "fixed_notification"
	PurposeIconProxy         Purpose = "icon_proxy"
	PurposeAdminTest         Purpose = "admin_test"
	PurposeExchangeRate      Purpose = "exchange_rate"
	// PurposeBackupDestination covers admin-configured backup transports (S3,
	// WebDAV). The endpoint is chosen by an administrator, so it is trusted as
	// administrator policy and relies on the configured proxy/network ACL
	// boundary rather than the user-facing SSRF filter, mirroring OIDC.
	PurposeBackupDestination Purpose = "backup_destination"
)

func NewOutboundHTTPClient(db *gorm.DB, timeout time.Duration) *http.Client {
	return newOutboundHTTPClient(outboundHTTPClientOptions{
		DB:      db,
		Timeout: timeout,
	})
}

func NewSafeOutboundHTTPClient(db *gorm.DB, timeout time.Duration) *http.Client {
	return newOutboundHTTPClient(outboundHTTPClientOptions{
		DB:           db,
		Timeout:      timeout,
		SecureEgress: true,
	})
}

func BuildHTTPClient(ctx context.Context, db *gorm.DB, purpose Purpose) (*http.Client, error) {
	return BuildHTTPClientWithTimeout(ctx, db, purpose, 15*time.Second)
}

func BuildHTTPClientWithTimeout(_ context.Context, db *gorm.DB, purpose Purpose, timeout time.Duration) (*http.Client, error) {
	switch purpose {
	case PurposeNotification:
		return NewSafeOutboundHTTPClient(db, timeout), nil
	default:
		return NewOutboundHTTPClient(db, timeout), nil
	}
}

func newOutboundHTTPClient(options outboundHTTPClientOptions) *http.Client {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	client := &http.Client{Timeout: timeout}
	transport, err := newOutboundHTTPTransport(options)
	if err != nil {
		return client
	}
	client.Transport = transport
	return client
}

func (options outboundHTTPClientOptions) safeDialer() *safeOutboundDialer {
	if options.SecureDialer != nil {
		return options.SecureDialer
	}
	return newSafeOutboundDialerWithDB(options.DB, options.Timeout)
}

type safeOutboundDialer struct {
	dialer      *net.Dialer
	dialContext func(context.Context, string, string) (net.Conn, error)
	db          *gorm.DB
}

func newSafeOutboundDialer(timeout time.Duration) *safeOutboundDialer {
	return newSafeOutboundDialerWithDB(nil, timeout)
}

func newSafeOutboundDialerWithDB(db *gorm.DB, timeout time.Duration) *safeOutboundDialer {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &safeOutboundDialer{
		dialer: &net.Dialer{
			Timeout:   timeout,
			KeepAlive: 30 * time.Second,
		},
		db: db,
	}
}

func (d *safeOutboundDialer) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	if d == nil || d.dialer == nil {
		d = newSafeOutboundDialer(15 * time.Second)
	}
	if !isTCPNetwork(network) {
		return d.dial(ctx, network, address)
	}

	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}

	ips, err := ResolveSafeHostIPs(ctx, network, host, "outbound request url", d.db)
	if err != nil {
		return nil, err
	}

	var lastErr error
	for _, ip := range ips {
		conn, err := d.dial(ctx, network, net.JoinHostPort(ip.String(), port))
		if err == nil {
			return conn, nil
		}
		lastErr = err
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("outbound request url host resolves to no addresses")
}

func (d *safeOutboundDialer) dial(ctx context.Context, network string, address string) (net.Conn, error) {
	if d != nil && d.dialContext != nil {
		return d.dialContext(ctx, network, address)
	}
	return d.dialer.DialContext(ctx, network, address)
}

func isTCPNetwork(network string) bool {
	switch strings.ToLower(strings.TrimSpace(network)) {
	case "tcp", "tcp4", "tcp6":
		return true
	default:
		return false
	}
}

func cloneDefaultHTTPTransport() *http.Transport {
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		return transport.Clone()
	}

	return &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
