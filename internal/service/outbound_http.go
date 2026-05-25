package service

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"golang.org/x/net/proxy"
	"gorm.io/gorm"
)

const (
	systemProxyTypeHTTP   = "http"
	systemProxyTypeSOCKS5 = "socks5"
)

var (
	ErrInvalidSystemProxyType = errors.New("system proxy type must be http or socks5")
	ErrInvalidSystemProxyURL  = errors.New("system proxy url must include a host")
)

type outboundHTTPClientOptions struct {
	Timeout time.Duration
	DB      *gorm.DB
}

type systemProxyConfig struct {
	Enabled  bool
	Type     string
	URL      string
	HasValue bool
}

func NewOutboundHTTPClient(db *gorm.DB, timeout time.Duration) *http.Client {
	return newOutboundHTTPClient(outboundHTTPClientOptions{
		DB:      db,
		Timeout: timeout,
	})
}

func NewOutboundDialContext(db *gorm.DB, timeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	directDialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}
	if timeout <= 0 {
		directDialer.Timeout = 15 * time.Second
	}

	if db == nil {
		return directDialer.DialContext
	}

	cfg, err := loadSystemProxyConfig(db)
	if err != nil || !cfg.Enabled {
		return directDialer.DialContext
	}

	proxyURL, err := normalizeSystemProxyURL(cfg.Type, cfg.URL)
	if err != nil {
		return directDialer.DialContext
	}

	switch cfg.Type {
	case systemProxyTypeHTTP:
		return func(ctx context.Context, network string, address string) (net.Conn, error) {
			return dialHTTPProxyConnect(ctx, directDialer, proxyURL, network, address)
		}
	case systemProxyTypeSOCKS5:
		dialer, err := proxy.FromURL(proxyURL, directDialer)
		if err != nil {
			return directDialer.DialContext
		}
		return func(ctx context.Context, network string, address string) (net.Conn, error) {
			if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
				return contextDialer.DialContext(ctx, network, address)
			}
			return proxyDialContext(ctx, dialer, network, address)
		}
	default:
		return directDialer.DialContext
	}
}

func newOutboundHTTPClient(options outboundHTTPClientOptions) *http.Client {
	timeout := options.Timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	client := &http.Client{Timeout: timeout}
	transport, err := newOutboundHTTPTransport(options.DB)
	if err != nil {
		return client
	}
	client.Transport = transport
	return client
}

func newOutboundHTTPTransport(db *gorm.DB) (http.RoundTripper, error) {
	if db == nil {
		return http.DefaultTransport, nil
	}

	cfg, err := loadSystemProxyConfig(db)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		return http.DefaultTransport, nil
	}

	transport := cloneDefaultHTTPTransport()
	switch cfg.Type {
	case systemProxyTypeHTTP:
		proxyURL, err := normalizeSystemProxyURL(cfg.Type, cfg.URL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	case systemProxyTypeSOCKS5:
		proxyURL, err := normalizeSystemProxyURL(cfg.Type, cfg.URL)
		if err != nil {
			return nil, err
		}
		dialer, err := proxy.FromURL(proxyURL, proxy.Direct)
		if err != nil {
			return nil, err
		}
		transport.Proxy = nil
		transport.DialContext = func(ctx context.Context, network string, address string) (net.Conn, error) {
			if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
				return contextDialer.DialContext(ctx, network, address)
			}
			return proxyDialContext(ctx, dialer, network, address)
		}
	default:
		return nil, ErrInvalidSystemProxyType
	}

	return transport, nil
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

func proxyDialContext(ctx context.Context, dialer proxy.Dialer, network string, address string) (net.Conn, error) {
	type dialResult struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan dialResult, 1)
	go func() {
		conn, err := dialer.Dial(network, address)
		resultCh <- dialResult{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.conn != nil && ctx.Err() != nil {
			_ = result.conn.Close()
			return nil, ctx.Err()
		}
		return result.conn, result.err
	}
}

func dialHTTPProxyConnect(
	ctx context.Context,
	dialer *net.Dialer,
	proxyURL *url.URL,
	network string,
	address string,
) (net.Conn, error) {
	if network != "tcp" && network != "tcp4" && network != "tcp6" {
		return nil, fmt.Errorf("http proxy only supports tcp network")
	}

	proxyAddress := proxyURL.Host
	if proxyURL.Port() == "" {
		proxyAddress = net.JoinHostPort(proxyURL.Hostname(), "80")
	}

	conn, err := dialer.DialContext(ctx, network, proxyAddress)
	if err != nil {
		return nil, err
	}

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
		defer conn.SetDeadline(time.Time{})
	}

	req := &http.Request{
		Method: http.MethodConnect,
		URL:    &url.URL{Opaque: address},
		Host:   address,
		Header: make(http.Header),
	}
	if proxyURL.User != nil {
		username := proxyURL.User.Username()
		password, _ := proxyURL.User.Password()
		token := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
		req.Header.Set("Proxy-Authorization", "Basic "+token)
	}

	if err := req.Write(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, req)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = conn.Close()
		return nil, fmt.Errorf("http proxy CONNECT failed with status %d", resp.StatusCode)
	}

	return &bufferedConn{Conn: conn, reader: reader}, nil
}

type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (c *bufferedConn) Read(p []byte) (int, error) {
	if c.reader != nil && c.reader.Buffered() > 0 {
		return c.reader.Read(p)
	}
	return c.Conn.Read(p)
}

func loadSystemProxyConfig(db *gorm.DB) (systemProxyConfig, error) {
	cfg := systemProxyConfig{
		Enabled: false,
		Type:    systemProxyTypeHTTP,
		URL:     "",
	}
	if db == nil {
		return cfg, nil
	}

	var items []model.SystemSetting
	if err := db.Where("key IN ?", []string{
		"system_proxy_enabled",
		"system_proxy_type",
		"system_proxy_url",
	}).Find(&items).Error; err != nil {
		return cfg, err
	}

	for _, item := range items {
		settingValue := item.Value
		decryptedValue, decryptErr := decryptSystemSettingValueIfNeeded(item.Key, item.Value)
		if decryptErr == nil {
			settingValue = decryptedValue
		}

		switch item.Key {
		case "system_proxy_enabled":
			cfg.Enabled = settingValue == "true"
		case "system_proxy_type":
			cfg.Type = strings.TrimSpace(strings.ToLower(settingValue))
		case "system_proxy_url":
			cfg.URL = settingValue
			cfg.HasValue = strings.TrimSpace(settingValue) != ""
		}
	}

	if strings.TrimSpace(cfg.Type) == "" {
		cfg.Type = systemProxyTypeHTTP
	}

	return cfg, nil
}

func validateSystemProxySettings(proxyType string, proxyURL string, enabled bool) error {
	normalizedType, err := normalizeSystemProxyType(proxyType)
	if err != nil {
		return err
	}

	trimmedURL := strings.TrimSpace(proxyURL)
	if trimmedURL == "" {
		if enabled {
			return ErrInvalidSystemProxyURL
		}
		return nil
	}

	_, err = normalizeSystemProxyURL(normalizedType, trimmedURL)
	return err
}

func normalizeSystemProxyType(proxyType string) (string, error) {
	normalized := strings.TrimSpace(strings.ToLower(proxyType))
	if normalized == "" {
		return systemProxyTypeHTTP, nil
	}
	switch normalized {
	case systemProxyTypeHTTP, systemProxyTypeSOCKS5:
		return normalized, nil
	default:
		return "", ErrInvalidSystemProxyType
	}
}

func normalizeSystemProxyURL(proxyType string, rawURL string) (*url.URL, error) {
	normalizedType, err := normalizeSystemProxyType(proxyType)
	if err != nil {
		return nil, err
	}

	trimmedURL := strings.TrimSpace(rawURL)
	if trimmedURL == "" {
		return nil, ErrInvalidSystemProxyURL
	}
	if !strings.Contains(trimmedURL, "://") {
		trimmedURL = normalizedType + "://" + trimmedURL
	}

	parsed, err := url.Parse(trimmedURL)
	if err != nil || parsed.Hostname() == "" {
		return nil, ErrInvalidSystemProxyURL
	}

	switch normalizedType {
	case systemProxyTypeHTTP:
		if parsed.Scheme != "http" {
			return nil, fmt.Errorf("system proxy url must start with http://")
		}
	case systemProxyTypeSOCKS5:
		if parsed.Scheme != "socks5" && parsed.Scheme != "socks5h" {
			return nil, fmt.Errorf("system proxy url must start with socks5://")
		}
		parsed.Scheme = "socks5"
	}

	if parsed.Port() != "" {
		port, err := net.LookupPort("tcp", parsed.Port())
		if err != nil || port < 1 || port > 65535 {
			return nil, fmt.Errorf("system proxy url port is invalid")
		}
	}

	return parsed, nil
}
