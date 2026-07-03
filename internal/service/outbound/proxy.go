package outbound

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/net/proxy"
	"gorm.io/gorm"
)

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

	cfg, err := LoadSystemProxyConfig(db)
	if err != nil || !cfg.Enabled {
		return directDialer.DialContext
	}

	proxyURL, err := NormalizeSystemProxyURL(cfg.Type, cfg.URL)
	if err != nil {
		return directDialer.DialContext
	}

	switch cfg.Type {
	case SystemProxyTypeHTTP:
		return func(ctx context.Context, network string, address string) (net.Conn, error) {
			return dialHTTPProxyConnect(ctx, directDialer, proxyURL, network, address)
		}
	case SystemProxyTypeSOCKS5:
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

func NewSafeOutboundDialContext(db *gorm.DB, timeout time.Duration) func(context.Context, string, string) (net.Conn, error) {
	if db != nil {
		cfg, err := LoadSystemProxyConfig(db)
		if err == nil && cfg.Enabled {
			proxyDialContext := NewOutboundDialContext(db, timeout)
			return func(ctx context.Context, network string, address string) (net.Conn, error) {
				if isTCPNetwork(network) {
					host, _, err := net.SplitHostPort(address)
					if err != nil {
						return nil, err
					}
					if err := ValidateHost(host, "outbound request url", db); err != nil {
						return nil, err
					}
				}
				return proxyDialContext(ctx, network, address)
			}
		}
	}
	return newSafeOutboundDialerWithDB(db, timeout).DialContext
}

func newOutboundHTTPTransport(options outboundHTTPClientOptions) (http.RoundTripper, error) {
	if options.DB == nil {
		if options.SecureEgress {
			transport := cloneDefaultHTTPTransport()
			transport.Proxy = nil
			transport.DialContext = options.safeDialer().DialContext
			return transport, nil
		}
		return http.DefaultTransport, nil
	}

	cfg, err := LoadSystemProxyConfig(options.DB)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		if options.SecureEgress {
			transport := cloneDefaultHTTPTransport()
			transport.Proxy = nil
			transport.DialContext = options.safeDialer().DialContext
			return transport, nil
		}
		return http.DefaultTransport, nil
	}

	transport := cloneDefaultHTTPTransport()
	switch cfg.Type {
	case SystemProxyTypeHTTP:
		proxyURL, err := NormalizeSystemProxyURL(cfg.Type, cfg.URL)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	case SystemProxyTypeSOCKS5:
		proxyURL, err := NormalizeSystemProxyURL(cfg.Type, cfg.URL)
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

	if options.SecureEgress {
		return outboundProxyRoundTripper{transport: transport, db: options.DB}, nil
	}
	return transport, nil
}

type outboundProxyRoundTripper struct {
	transport http.RoundTripper
	db        *gorm.DB
}

func (t outboundProxyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req == nil || req.URL == nil {
		return nil, errors.New("invalid outbound request")
	}
	if err := validateOutboundRequestHost(req.URL.Hostname(), true, t.db); err != nil {
		return nil, err
	}
	if t.transport == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return t.transport.RoundTrip(req)
}

func (t outboundProxyRoundTripper) outboundProxyMediated() bool {
	return true
}

func clientUsesOutboundProxy(client *http.Client) bool {
	if client == nil || client.Transport == nil {
		return false
	}

	type proxyAwareTransport interface {
		outboundProxyMediated() bool
	}
	if transport, ok := client.Transport.(proxyAwareTransport); ok {
		return transport.outboundProxyMediated()
	}
	return false
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
