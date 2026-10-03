package mcpoauth

import (
	"crypto/subtle"
	"errors"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"golang.org/x/oauth2"
	"gorm.io/gorm"
)

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

type Principal struct {
	UserID     uint
	GrantID    uint
	ClientID   string
	ClientName string
	Scopes     []string
	ExpiresAt  time.Time
}

func verifierValid(value string) bool {
	if len(value) < 43 || len(value) > 128 {
		return false
	}
	for _, c := range value {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("-._~", c)) {
			return false
		}
	}
	return true
}

func (s *Service) Exchange(params url.Values) (*TokenResponse, error) {
	resource, err := s.Resource()
	if err != nil {
		return nil, err
	}
	if params.Get("resource") != resource {
		return nil, problem("invalid_target", "resource must match this MCP server")
	}
	if params.Get("client_secret") != "" {
		return nil, problem("invalid_client", "only public PKCE clients are supported")
	}
	var response *TokenResponse
	var protocolErr error
	err = s.db.Transaction(func(tx *gorm.DB) error {
		txService := *s
		txService.db = tx
		var client model.MCPOAuthClient
		if err := tx.Where("id = ?", params.Get("client_id")).Take(&client).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return problem("invalid_client", "unknown client")
			}
			return err
		}
		var grant model.MCPOAuthGrant
		switch params.Get("grant_type") {
		case "authorization_code":
			var request model.MCPOAuthRequest
			if err := tx.Where("code_hash = ?", hash(params.Get("code"))).Take(&request).Error; err != nil {
				return problem("invalid_grant", "invalid authorization code")
			}
			verifier := params.Get("code_verifier")
			challenge := oauth2.S256ChallengeFromVerifier(verifier)
			if request.ClientID != client.ID || request.Resource != resource || request.RedirectURI != params.Get("redirect_uri") || request.UserID == nil || !verifierValid(verifier) || subtle.ConstantTimeCompare([]byte(challenge), []byte(request.CodeChallenge)) != 1 {
				return problem("invalid_grant", "authorization code binding or PKCE validation failed")
			}
			// Conditional consumption serializes concurrent exchanges on SQLite
			// and PostgreSQL without issuing two token families for one code.
			// Claim before expiry/account/quota checks so a stale code read cannot
			// bypass replay revocation. Rejected unused codes roll back the claim.
			claim := tx.Model(&model.MCPOAuthRequest{}).Where("id = ? AND consumed_at IS NULL", request.ID).Update("consumed_at", time.Now())
			if claim.Error != nil {
				return claim.Error
			}
			if claim.RowsAffected != 1 {
				if err := txService.revokeCodeGrant(request.ID); err != nil {
					return err
				}
				protocolErr = problem("invalid_grant", "authorization code was already used")
				return nil // Commit the competing exchange's token-family revocation.
			}
			if time.Now().After(request.ExpiresAt) {
				return problem("invalid_grant", "authorization code expired")
			}
			if err := active(tx, *request.UserID); err != nil {
				return err
			}
			var count int64
			if err := tx.Model(&model.MCPOAuthGrant{}).Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", *request.UserID, time.Now()).Count(&count).Error; err != nil {
				return err
			}
			if count >= 50 {
				return problem("invalid_grant", "revoke an existing MCP connection before adding another")
			}
			grant = model.MCPOAuthGrant{UserID: *request.UserID, ClientID: client.ID, ClientName: request.ClientName, Scopes: request.Scopes, Resource: resource, ExpiresAt: time.Now().Add(GrantTTL)}
			if err := tx.Create(&grant).Error; err != nil {
				return err
			}
			if err := tx.Model(&model.MCPOAuthRequest{}).Where("id = ?", request.ID).Update("grant_id", grant.ID).Error; err != nil {
				return err
			}
		case "refresh_token":
			var token model.MCPOAuthToken
			if err := tx.Where("hash = ? AND kind = ?", hash(params.Get("refresh_token")), "refresh").Take(&token).Error; err != nil {
				return problem("invalid_grant", "invalid refresh token")
			}
			if err := tx.Where("id = ? AND client_id = ? AND resource = ?", token.GrantID, client.ID, resource).Take(&grant).Error; err != nil {
				return problem("invalid_grant", "invalid refresh token binding")
			}
			if token.ConsumedAt != nil {
				if err := tx.Model(&model.MCPOAuthGrant{}).Where("id = ?", grant.ID).Update("revoked_at", time.Now()).Error; err != nil {
					return err
				}
				protocolErr = problem("invalid_grant", "refresh token replay revoked this authorization")
				return nil
			}
			if grant.RevokedAt != nil || time.Now().After(grant.ExpiresAt) || time.Now().After(token.ExpiresAt) {
				return problem("invalid_grant", "refresh token expired or revoked")
			}
			if err := active(tx, grant.UserID); err != nil {
				return err
			}
			if raw := params.Get("scope"); raw != "" {
				scopes, err := scopeSet(raw)
				if err != nil {
					return err
				}
				for _, scope := range strings.Fields(scopes) {
					if !slices.Contains(strings.Fields(grant.Scopes), scope) {
						return problem("invalid_scope", "refresh cannot expand granted scopes")
					}
				}
				grant.Scopes = scopes
				if err := tx.Model(&model.MCPOAuthGrant{}).Where("id = ?", grant.ID).Update("scopes", scopes).Error; err != nil {
					return err
				}
			}
			claim := tx.Model(&model.MCPOAuthToken{}).Where("hash = ? AND consumed_at IS NULL", token.Hash).Update("consumed_at", time.Now())
			if claim.Error != nil {
				return claim.Error
			}
			if claim.RowsAffected != 1 {
				if err := tx.Model(&model.MCPOAuthGrant{}).Where("id = ?", grant.ID).Update("revoked_at", time.Now()).Error; err != nil {
					return err
				}
				protocolErr = problem("invalid_grant", "refresh token was already used")
				return nil
			}
		default:
			return problem("unsupported_grant_type", "only authorization_code and refresh_token are supported")
		}
		var err error
		response, err = txService.issue(grant)
		return err
	})
	if err != nil {
		return nil, err
	}
	if protocolErr != nil {
		return nil, protocolErr
	}
	return response, nil
}

func (s *Service) revokeCodeGrant(requestID string) error {
	// Read the association afresh: a competing exchange can attach its grant
	// after this transaction reads the code but before conditional consumption.
	grantIDs := s.db.Model(&model.MCPOAuthRequest{}).Select("grant_id").Where("id = ? AND consumed_at IS NOT NULL", requestID)
	return s.db.Model(&model.MCPOAuthGrant{}).Where("id IN (?)", grantIDs).Update("revoked_at", time.Now()).Error
}

func (s *Service) issue(grant model.MCPOAuthGrant) (*TokenResponse, error) {
	a, err := secret()
	if err != nil {
		return nil, err
	}
	r, err := secret()
	if err != nil {
		return nil, err
	}
	access := "sdx_mcpa_" + a
	refresh := "sdx_mcpr_" + r
	expiration := time.Now().Add(AccessTTL)
	if grant.ExpiresAt.Before(expiration) {
		expiration = grant.ExpiresAt
	}
	tokens := []model.MCPOAuthToken{
		{Hash: hash(access), GrantID: grant.ID, Kind: "access", ExpiresAt: expiration},
		{Hash: hash(refresh), GrantID: grant.ID, Kind: "refresh", ExpiresAt: grant.ExpiresAt},
	}
	if err := s.db.Create(&tokens).Error; err != nil {
		return nil, err
	}
	return &TokenResponse{access, "Bearer", int64(time.Until(expiration).Seconds()), refresh, grant.Scopes}, nil
}

func (s *Service) ValidateToken(raw string) (*Principal, error) {
	if !strings.HasPrefix(raw, "sdx_mcpa_") || len(raw) != 52 {
		return nil, problem("invalid_token", "invalid MCP access token")
	}
	resource, err := s.Resource()
	if err != nil {
		return nil, err
	}
	var token model.MCPOAuthToken
	if err := s.db.Where("hash = ? AND kind = ? AND expires_at > ?", hash(raw), "access", time.Now()).Take(&token).Error; err != nil {
		return nil, problem("invalid_token", "MCP access token expired or invalid")
	}
	var grant model.MCPOAuthGrant
	if err := s.db.Where("id = ? AND resource = ? AND revoked_at IS NULL AND expires_at > ?", token.GrantID, resource, time.Now()).Take(&grant).Error; err != nil {
		return nil, problem("invalid_token", "MCP authorization expired or revoked")
	}
	if err := active(s.db, grant.UserID); err != nil {
		return nil, problem("invalid_token", "account is unavailable")
	}
	now := time.Now()
	if err := s.db.Model(&model.MCPOAuthGrant{}).Where("id = ? AND (last_used_at IS NULL OR last_used_at < ?)", grant.ID, now.Add(-time.Minute)).Update("last_used_at", now).Error; err != nil {
		return nil, err
	}
	return &Principal{grant.UserID, grant.ID, grant.ClientID, grant.ClientName, strings.Fields(grant.Scopes), token.ExpiresAt}, nil
}

// RFC 7009 revocation deliberately returns success for unknown or mismatched
// tokens; possession plus the matching client ID can only revoke that family.
func (s *Service) RevokeToken(raw, clientID string) error {
	var token model.MCPOAuthToken
	if err := s.db.Where("hash = ?", hash(raw)).Take(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	return s.db.Model(&model.MCPOAuthGrant{}).Where("id = ? AND client_id = ?", token.GrantID, clientID).Update("revoked_at", time.Now()).Error
}
