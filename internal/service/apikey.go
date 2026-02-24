package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

var (
	ErrAPIKeyNotFound     = errors.New("api key not found")
	ErrAPIKeyNameRequired = errors.New("api key name is required")
	ErrAPIKeyNameTooLong  = errors.New("api key name must be 100 characters or less")
	ErrAPIKeyExpired      = errors.New("api key has expired")
	ErrAPIKeyInvalid      = errors.New("invalid api key")
	ErrAPIKeyLimitReached = errors.New("maximum number of api keys reached")
	ErrAPIKeyScopeInvalid = errors.New("invalid api key scopes")
)

const maxAPIKeysPerUser = 5

const (
	APIKeyScopeRead  = "read"
	APIKeyScopeWrite = "write"
)

var (
	defaultAPIKeyScopes = []string{APIKeyScopeRead, APIKeyScopeWrite}
	validAPIKeyScopes   = map[string]struct{}{
		APIKeyScopeRead:  {},
		APIKeyScopeWrite: {},
	}
)

type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

type CreateAPIKeyInput struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at"`
	Scopes    []string   `json:"scopes"`
}

type CreateAPIKeyResponse struct {
	APIKey model.APIKey `json:"api_key"`
	Key    string       `json:"key"`
}

type APIKeyPrincipal struct {
	UserID uint
	KeyID  uint
	Scopes []string
}

func generateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("sdx_%s", hex.EncodeToString(b)), nil
}

func hashAPIKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func ParseAPIKeyScopes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return append([]string{}, defaultAPIKeyScopes...)
	}

	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	scopes := make([]string, 0, len(parts))

	for _, part := range parts {
		scope := strings.ToLower(strings.TrimSpace(part))
		if scope == "" {
			continue
		}
		if _, ok := validAPIKeyScopes[scope]; !ok {
			continue
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}

	if len(scopes) == 0 {
		return append([]string{}, defaultAPIKeyScopes...)
	}

	sort.Strings(scopes)
	return scopes
}

func normalizeAPIKeyScopes(input []string) ([]string, error) {
	if len(input) == 0 {
		return append([]string{}, defaultAPIKeyScopes...), nil
	}

	seen := make(map[string]struct{}, len(input))
	scopes := make([]string, 0, len(input))
	for _, scope := range input {
		canonical := strings.ToLower(strings.TrimSpace(scope))
		if canonical == "" {
			continue
		}
		if _, ok := validAPIKeyScopes[canonical]; !ok {
			return nil, ErrAPIKeyScopeInvalid
		}
		if _, exists := seen[canonical]; exists {
			continue
		}
		seen[canonical] = struct{}{}
		scopes = append(scopes, canonical)
	}

	if len(scopes) == 0 {
		return nil, ErrAPIKeyScopeInvalid
	}

	sort.Strings(scopes)
	return scopes, nil
}

func (s *APIKeyService) Create(userID uint, role string, input CreateAPIKeyInput) (*CreateAPIKeyResponse, error) {
	if input.Name == "" {
		return nil, ErrAPIKeyNameRequired
	}
	if len(input.Name) > 100 {
		return nil, ErrAPIKeyNameTooLong
	}

	if role != "admin" {
		var count int64
		s.db.Model(&model.APIKey{}).Where("user_id = ?", userID).Count(&count)
		if count >= maxAPIKeysPerUser {
			return nil, ErrAPIKeyLimitReached
		}
	}

	scopes, err := normalizeAPIKeyScopes(input.Scopes)
	if err != nil {
		return nil, err
	}

	rawKey, err := generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate api key: %w", err)
	}

	prefix := rawKey[:12]

	apiKey := model.APIKey{
		UserID:    userID,
		Name:      input.Name,
		KeyHash:   hashAPIKey(rawKey),
		Prefix:    prefix,
		Scopes:    strings.Join(scopes, ","),
		ExpiresAt: input.ExpiresAt,
	}

	if err := s.db.Create(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	return &CreateAPIKeyResponse{
		APIKey: apiKey,
		Key:    rawKey,
	}, nil
}

func (s *APIKeyService) List(userID uint) ([]model.APIKey, error) {
	var keys []model.APIKey
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *APIKeyService) Delete(userID uint, keyID uint) error {
	result := s.db.Where("id = ? AND user_id = ?", keyID, userID).Delete(&model.APIKey{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

// ValidateKey checks a raw API key string and returns the authenticated principal.
func (s *APIKeyService) ValidateKey(rawKey string) (*APIKeyPrincipal, error) {
	keyHash := hashAPIKey(rawKey)

	var apiKey model.APIKey
	if err := s.db.Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		return nil, ErrAPIKeyInvalid
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return nil, ErrAPIKeyExpired
	}

	// Update last used timestamp
	now := time.Now().UTC()
	s.db.Model(&apiKey).Update("last_used_at", now)

	return &APIKeyPrincipal{
		UserID: apiKey.UserID,
		KeyID:  apiKey.ID,
		Scopes: ParseAPIKeyScopes(apiKey.Scopes),
	}, nil
}
