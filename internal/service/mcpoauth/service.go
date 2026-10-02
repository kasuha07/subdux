// Package mcpoauth implements Subdux's bounded public-client MCP authorization
// service. Human sessions authorize grants; opaque MCP tokens never authenticate
// REST requests. All state survives process restarts and secrets are hashed.
package mcpoauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/service/userstatus"
	"gorm.io/gorm"
)

const (
	RequestTTL = 10 * time.Minute
	CodeTTL    = time.Minute
	AccessTTL  = 15 * time.Minute
	GrantTTL   = 30 * 24 * time.Hour
)

type ProtocolError struct {
	Code        string
	Description string
}

func (e *ProtocolError) Error() string       { return e.Description }
func problem(code, description string) error { return &ProtocolError{code, description} }

type Service struct {
	db     *gorm.DB
	client *http.Client
}

func NewService(db *gorm.DB) *Service {
	// CIMD is unauthenticated, attacker-controlled input. Always use the strict
	// default egress policy, even when administrators relax notification SSRF.
	client := outbound.NewSafeOutboundHTTPClient(nil, 5*time.Second)
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	return &Service{db: db, client: client}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	clone.db = s.db.WithContext(ctx)
	return &clone
}

// Issuer comes only from the administrator-configured site URL, never Host or
// forwarded headers. HTTP is allowed solely for local development.
func (s *Service) Issuer() (string, error) {
	var setting model.SystemSetting
	if err := s.db.Where("key = ?", "site_url").Take(&setting).Error; err != nil {
		return "", problem("temporarily_unavailable", "configure the public site URL to enable MCP OAuth")
	}
	u, err := url.Parse(strings.TrimSpace(setting.Value))
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.RawPath != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && isLoopback(u.Hostname()))) {
		return "", problem("temporarily_unavailable", "MCP OAuth requires an HTTPS site origin (HTTP loopback is allowed for development)")
	}
	return strings.ToLower(u.Scheme + "://" + u.Host), nil
}

func (s *Service) Resource() (string, error) {
	issuer, err := s.Issuer()
	if err != nil {
		return "", err
	}
	return issuer + "/mcp", nil
}

func secret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func isLoopback(host string) bool {
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func active(db *gorm.DB, userID uint) error {
	if err := userstatus.EnsureActive(db, userID); err != nil {
		return problem("invalid_grant", "the authorization is no longer valid")
	}
	return nil
}

// Cleanup retains used refresh tokens for the entire grant lifetime so replay
// still revokes the token family. Expired clients/requests are bounded too.
func (s *Service) Cleanup(now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("expires_at < ?", now.Add(-time.Hour)).Delete(&model.MCPOAuthRequest{}).Error; err != nil {
			return err
		}
		if err := tx.Where("expires_at < ?", now.Add(-time.Hour)).Delete(&model.MCPOAuthToken{}).Error; err != nil {
			return err
		}
		if err := tx.Where("expires_at < ?", now.Add(-time.Hour)).Delete(&model.MCPOAuthGrant{}).Error; err != nil {
			return err
		}
		// Only orphaned DCR clients may be removed; a long-lived CIMD client can
		// be refreshed on demand. Grants and pending requests remain authoritative.
		return tx.Where("created_at < ? AND id NOT IN (?) AND id NOT IN (?)", now.Add(-GrantTTL),
			tx.Model(&model.MCPOAuthGrant{}).Select("client_id"), tx.Model(&model.MCPOAuthRequest{}).Select("client_id")).Delete(&model.MCPOAuthClient{}).Error
	})
}

func (s *Service) StartCleanupLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = s.WithContext(ctx).Cleanup(time.Now())
			}
		}
	}()
}
