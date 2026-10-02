package mcpoauth

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

type ConsentInfo struct {
	ClientID    string   `json:"client_id"`
	ClientName  string   `json:"client_name"`
	RedirectURI string   `json:"redirect_uri"`
	Scopes      []string `json:"scopes"`
}

func scopeSet(raw string) (string, error) {
	if len(raw) > 100 {
		return "", problem("invalid_scope", "unsupported scopes")
	}
	values := strings.Fields(raw)
	if len(values) == 0 {
		values = []string{"read"}
	}
	for _, value := range values {
		if value != "read" && value != "write" && value != "offline_access" {
			return "", problem("invalid_scope", "unsupported scopes")
		}
	}
	// Write implies read, matching the existing API-key scope contract.
	result := []string{"read"}
	if slices.Contains(values, "write") {
		result = append(result, "write")
	}
	if slices.Contains(values, "offline_access") {
		result = append(result, "offline_access")
	}
	return strings.Join(result, " "), nil
}

func (s *Service) Begin(params url.Values) (string, error) {
	if params.Get("response_type") != "code" {
		return "", problem("unsupported_response_type", "only authorization code is supported")
	}
	challenge := params.Get("code_challenge")
	decoded, err := base64.RawURLEncoding.DecodeString(challenge)
	if params.Get("code_challenge_method") != "S256" || err != nil || len(decoded) != 32 || base64.RawURLEncoding.EncodeToString(decoded) != challenge {
		return "", problem("invalid_request", "PKCE S256 is required")
	}
	if len(params.Get("state")) > 2048 {
		return "", problem("invalid_request", "state is too long")
	}
	resource, err := s.Resource()
	if err != nil {
		return "", err
	}
	if params.Get("resource") != resource {
		return "", problem("invalid_target", "resource must match this MCP server")
	}
	client, err := s.clientByID(params.Get("client_id"))
	if err != nil {
		return "", err
	}
	redirect := params.Get("redirect_uri")
	var registered []string
	if err := json.Unmarshal([]byte(client.RedirectURIs), &registered); err != nil {
		return "", err
	}
	if !slices.ContainsFunc(registered, func(raw string) bool { return redirectMatches(raw, redirect) }) {
		return "", problem("invalid_request", "redirect URI is not registered")
	}
	scopes, err := scopeSet(params.Get("scope"))
	if err != nil {
		return "", err
	}
	if err := s.Cleanup(time.Now()); err != nil {
		return "", err
	}
	var count int64
	if err := s.db.Model(&model.MCPOAuthRequest{}).Count(&count).Error; err != nil {
		return "", err
	}
	if count >= 5000 {
		return "", problem("temporarily_unavailable", "authorization request limit reached")
	}
	handle, err := secret()
	if err != nil {
		return "", err
	}
	request := model.MCPOAuthRequest{ID: hash(handle), ClientID: client.ID, ClientName: client.Name, RedirectURI: redirect,
		Scopes: scopes, Resource: resource, State: params.Get("state"), CodeChallenge: challenge, ExpiresAt: time.Now().Add(RequestTTL)}
	if err := s.db.Create(&request).Error; err != nil {
		return "", err
	}
	return handle, nil
}

// Consent binds the interaction to the first authenticated human who views it.
// Another user cannot approve a request viewed by someone else.
func (s *Service) Consent(handle string, userID uint) (*ConsentInfo, error) {
	if err := active(s.db, userID); err != nil {
		return nil, err
	}
	if len(handle) != 43 {
		return nil, problem("invalid_request", "authorization request has expired or is invalid")
	}
	if err := s.db.Model(&model.MCPOAuthRequest{}).Where("id = ? AND user_id IS NULL AND decided_at IS NULL AND expires_at > ?", hash(handle), time.Now()).Update("user_id", userID).Error; err != nil {
		return nil, err
	}
	var request model.MCPOAuthRequest
	if err := s.db.Where("id = ? AND user_id = ? AND decided_at IS NULL AND expires_at > ?", hash(handle), userID, time.Now()).Take(&request).Error; err != nil {
		return nil, problem("invalid_request", "authorization request has expired or is invalid")
	}
	return &ConsentInfo{request.ClientID, request.ClientName, request.RedirectURI, strings.Fields(request.Scopes)}, nil
}

func (s *Service) Decide(handle string, userID uint, approve, allowWrite bool) (string, error) {
	var redirect string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := active(tx, userID); err != nil {
			return err
		}
		var request model.MCPOAuthRequest
		if err := tx.Where("id = ? AND user_id = ? AND decided_at IS NULL AND expires_at > ?", hash(handle), userID, time.Now()).Take(&request).Error; err != nil {
			return problem("invalid_request", "authorization request has expired or is invalid")
		}
		txService := *s
		txService.db = tx
		issuer, err := txService.Issuer()
		if err != nil {
			return err
		}
		if request.Resource != issuer+"/mcp" {
			return problem("invalid_target", "the MCP resource has changed")
		}
		u, _ := url.Parse(request.RedirectURI)
		query := u.Query()
		for _, field := range []string{"code", "error", "error_description", "state", "iss"} {
			query.Del(field)
		}
		query.Set("state", request.State)
		query.Set("iss", issuer)
		now := time.Now()
		updates := map[string]any{"decided_at": now}
		if approve {
			code, err := secret()
			if err != nil {
				return err
			}
			query.Set("code", "sdx_mcpc_"+code)
			updates["code_hash"] = hash(query.Get("code"))
			updates["expires_at"] = now.Add(CodeTTL)
			if !allowWrite {
				updates["scopes"] = strings.ReplaceAll(request.Scopes, " write", "")
			}
		} else {
			query.Set("error", "access_denied")
			query.Set("error_description", "the user denied authorization")
		}
		result := tx.Model(&model.MCPOAuthRequest{}).Where("id = ? AND decided_at IS NULL", request.ID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return problem("invalid_request", "authorization request was already decided")
		}
		u.RawQuery = query.Encode()
		redirect = u.String()
		return nil
	})
	return redirect, err
}

func (s *Service) ListGrants(userID uint) ([]model.MCPOAuthGrant, error) {
	grants := []model.MCPOAuthGrant{}
	err := s.db.Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).Order("created_at DESC").Limit(50).Find(&grants).Error
	return grants, err
}

func (s *Service) RevokeGrant(userID, grantID uint) error {
	result := s.db.Model(&model.MCPOAuthGrant{}).Where("id = ? AND user_id = ?", grantID, userID).Update("revoked_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return problem("invalid_request", "authorization not found")
	}
	return nil
}
