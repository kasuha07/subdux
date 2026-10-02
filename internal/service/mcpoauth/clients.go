package mcpoauth

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ClientMetadata struct {
	ClientID                          string   `json:"client_id,omitempty"`
	ClientName                        string   `json:"client_name"`
	RedirectURIs                      []string `json:"redirect_uris"`
	GrantTypes                        []string `json:"grant_types,omitempty"`
	ResponseTypes                     []string `json:"response_types,omitempty"`
	TokenEndpointAuthMethod           string   `json:"token_endpoint_auth_method,omitempty"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported,omitempty"`
}

func validateMetadata(m ClientMetadata) error {
	if len(m.ClientName) > 120 || strings.IndexFunc(m.ClientName, unicode.IsControl) >= 0 || len(m.RedirectURIs) == 0 || len(m.RedirectURIs) > 10 {
		return problem("invalid_client_metadata", "invalid client name or redirect URIs")
	}
	if (len(m.TokenEndpointAuthMethodsSupported) > 0 && !slices.Contains(m.TokenEndpointAuthMethodsSupported, "none")) || (len(m.TokenEndpointAuthMethodsSupported) == 0 && m.TokenEndpointAuthMethod != "" && m.TokenEndpointAuthMethod != "none") {
		return problem("invalid_client_metadata", "only public PKCE clients using token endpoint authentication none are supported")
	}
	for _, value := range m.GrantTypes {
		if value != "authorization_code" && value != "refresh_token" {
			return problem("invalid_client_metadata", "unsupported grant type")
		}
	}
	for _, value := range m.ResponseTypes {
		if value != "code" {
			return problem("invalid_client_metadata", "only the code response type is supported")
		}
	}
	for _, raw := range m.RedirectURIs {
		if !validRedirect(raw) {
			return problem("invalid_redirect_uri", "redirect URIs must use HTTPS or HTTP loopback without credentials or fragments")
		}
	}
	return nil
}

func validRedirect(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && len(raw) <= 2048 && u.Host != "" && u.User == nil && !strings.Contains(raw, "#") && u.Opaque == "" &&
		(u.Scheme == "https" || (u.Scheme == "http" && isLoopback(u.Hostname())))
}

func redirectMatches(registered, requested string) bool {
	if registered == requested {
		return true
	}
	if !validRedirect(requested) {
		return false
	}
	a, _ := url.Parse(registered)
	b, _ := url.Parse(requested)
	// RFC 8252 permits any port for native HTTP loopback redirects. Every
	// other component (including host, encoded path and query) matches exactly.
	if a == nil || a.Scheme != "http" || !isLoopback(a.Hostname()) || a.Hostname() != b.Hostname() {
		return false
	}
	a.Host = b.Host
	return a.String() == b.String()
}

func (s *Service) Register(m ClientMetadata) (*ClientMetadata, error) {
	if err := validateMetadata(m); err != nil {
		return nil, err
	}
	id, err := secret()
	if err != nil {
		return nil, err
	}
	m.ClientID = "sdx_client_" + id
	if m.ClientName == "" {
		m.ClientName = "MCP client"
	}
	m.GrantTypes = []string{"authorization_code", "refresh_token"}
	m.ResponseTypes = []string{"code"}
	m.TokenEndpointAuthMethod = "none"
	m.TokenEndpointAuthMethodsSupported = nil
	if err := s.saveClient(m, false); err != nil {
		return nil, err
	}
	return &m, nil
}

func (s *Service) saveClient(m ClientMetadata, replace bool) error {
	redirects, err := json.Marshal(m.RedirectURIs)
	if err != nil {
		return err
	}
	client := model.MCPOAuthClient{ID: m.ClientID, Name: m.ClientName, RedirectURIs: string(redirects), MetadataAt: time.Now()}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&model.MCPOAuthClient{}).Count(&count).Error; err != nil {
			return err
		}
		if count >= 1000 {
			var existing int64
			if err := tx.Model(&model.MCPOAuthClient{}).Where("id = ?", m.ClientID).Count(&existing).Error; err != nil {
				return err
			}
			if existing == 0 {
				return problem("temporarily_unavailable", "client registration limit reached")
			}
		}
		if replace {
			return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "id"}}, DoUpdates: clause.AssignmentColumns([]string{"name", "redirect_uris", "metadata_at"})}).Create(&client).Error
		}
		return tx.Create(&client).Error
	})
}

func (s *Service) clientByID(id string) (*model.MCPOAuthClient, error) {
	if len(id) > 2048 {
		return nil, problem("invalid_client", "invalid client ID")
	}
	var client model.MCPOAuthClient
	err := s.db.Where("id = ?", id).Take(&client).Error
	u, parseErr := url.Parse(id)
	isCIMD := parseErr == nil && u.Scheme == "https" && u.Host != "" && u.Path != "" && u.Path != "/" && u.User == nil && u.RawQuery == "" && u.Fragment == ""
	if err == nil && (!isCIMD || time.Since(client.MetadataAt) < time.Hour) {
		return &client, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if !isCIMD {
		return nil, problem("invalid_client", "unknown client")
	}
	req, err := http.NewRequestWithContext(s.db.Statement.Context, http.MethodGet, id, nil)
	if err != nil {
		return nil, problem("invalid_client", "invalid client metadata URL")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, problem("invalid_client", "unable to retrieve public HTTPS client metadata")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, problem("invalid_client", "client metadata must return HTTP 200 without redirects")
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, (64<<10)+1))
	if err != nil || len(body) > 64<<10 {
		return nil, problem("invalid_client", "invalid client metadata response")
	}
	var metadata ClientMetadata
	if json.Unmarshal(body, &metadata) != nil || metadata.ClientID != id {
		return nil, problem("invalid_client", "client metadata ID does not match its URL")
	}
	if err := validateMetadata(metadata); err != nil {
		return nil, err
	}
	if metadata.ClientName == "" {
		metadata.ClientName = u.Hostname()
	}
	if err := s.saveClient(metadata, true); err != nil {
		return nil, err
	}
	if err := s.db.Where("id = ?", id).Take(&client).Error; err != nil {
		return nil, err
	}
	return &client, nil
}
