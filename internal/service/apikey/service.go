package apikey

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/kasuha07/subdux/internal/service/userstatus"
	"gorm.io/gorm"
)

var (
	ErrAPIKeyNotFound     = serviceerr.New(serviceerr.KindNotFound, "api_key_not_found", "api key not found")
	ErrAPIKeyNameRequired = serviceerr.New(serviceerr.KindInvalid, "api_key_name_is_required", "api key name is required")
	ErrAPIKeyNameTooLong  = serviceerr.New(serviceerr.KindInvalid, "api_key_name_must_be_100_characters_or_less", "api key name must be 100 characters or less")
	ErrAPIKeyExpired      = serviceerr.New(serviceerr.KindUnauthorized, "api_key_has_expired", "api key has expired")
	ErrAPIKeyInvalid      = serviceerr.New(serviceerr.KindUnauthorized, "invalid_api_key", "invalid api key")
	ErrAPIKeyLimitReached = serviceerr.New(serviceerr.KindConflict, "maximum_number_of_api_keys_reached", "maximum number of api keys reached")
	ErrAPIKeyScopeInvalid = serviceerr.New(serviceerr.KindInvalid, "invalid_api_key_scopes", "invalid api key scopes")
	ErrAPIKeyKindRequired = serviceerr.New(serviceerr.KindInvalid, "api_key_kind_is_required", "api key kind is required")
	ErrAPIKeyKindInvalid  = serviceerr.New(serviceerr.KindInvalid, "invalid_api_key_kind", "invalid api key kind")
)

const maxKeysPerUser = 5

const (
	APIKeyScopeRead  = "read"
	APIKeyScopeWrite = "write"
)

const (
	APIKeyKindMCPClient      = "mcp_client"
	APIKeyKindAPIIntegration = "api_integration"
)

var (
	errUserInactive = userstatus.ErrUserNotActive

	defaultScopes = []string{APIKeyScopeRead}
	validScopes   = map[string]struct{}{
		APIKeyScopeRead:  {},
		APIKeyScopeWrite: {},
	}
	validKinds = map[string]struct{}{
		APIKeyKindMCPClient:      {},
		APIKeyKindAPIIntegration: {},
	}
)

type ActiveUserChecker interface {
	EnsureActiveUser(db *gorm.DB, userID uint) error
}

type GormActiveUserChecker struct{}

func (GormActiveUserChecker) EnsureActiveUser(db *gorm.DB, userID uint) error {
	return userstatus.EnsureActive(db, userID)
}

type Service struct {
	db                *gorm.DB
	activeUserChecker ActiveUserChecker
}

func NewService(db *gorm.DB) *Service {
	return NewServiceWithActiveUserChecker(db, GormActiveUserChecker{})
}

func NewServiceWithActiveUserChecker(db *gorm.DB, checker ActiveUserChecker) *Service {
	if checker == nil {
		checker = GormActiveUserChecker{}
	}
	return &Service{db: db, activeUserChecker: checker}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	if s.db != nil {
		clone.db = s.db.WithContext(ctx)
	}
	return &clone
}

type CreateInput struct {
	Name      string     `json:"name"`
	KeyKind   string     `json:"key_kind"`
	ExpiresAt *time.Time `json:"expires_at"`
	Scopes    []string   `json:"scopes"`
}

type CreateResponse struct {
	APIKey model.APIKey `json:"api_key"`
	Key    string       `json:"key"`
}

type Principal struct {
	UserID  uint
	KeyID   uint
	KeyKind string
	Scopes  []string
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
		return append([]string{}, defaultScopes...)
	}

	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	scopes := make([]string, 0, len(parts))

	for _, part := range parts {
		scope := strings.ToLower(strings.TrimSpace(part))
		if scope == "" {
			continue
		}
		if _, ok := validScopes[scope]; !ok {
			continue
		}
		if _, exists := seen[scope]; exists {
			continue
		}
		seen[scope] = struct{}{}
		scopes = append(scopes, scope)
	}

	if len(scopes) == 0 {
		return append([]string{}, defaultScopes...)
	}

	sort.Strings(scopes)
	return scopes
}

func normalizeScopes(input []string) ([]string, error) {
	if len(input) == 0 {
		return append([]string{}, defaultScopes...), nil
	}

	seen := make(map[string]struct{}, len(input))
	scopes := make([]string, 0, len(input))
	for _, scope := range input {
		canonical := strings.ToLower(strings.TrimSpace(scope))
		if canonical == "" {
			continue
		}
		if _, ok := validScopes[canonical]; !ok {
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
	if len(scopes) == 1 && scopes[0] == APIKeyScopeWrite {
		return nil, ErrAPIKeyScopeInvalid
	}

	sort.Strings(scopes)
	return scopes, nil
}

func NormalizeAPIKeyKind(value string) (string, error) {
	kind := strings.ToLower(strings.TrimSpace(value))
	if kind == "" {
		return "", ErrAPIKeyKindRequired
	}
	if _, ok := validKinds[kind]; !ok {
		return "", ErrAPIKeyKindInvalid
	}
	return kind, nil
}

func NormalizePersistedAPIKeyKind(value string) string {
	kind := strings.ToLower(strings.TrimSpace(value))
	if _, ok := validKinds[kind]; ok {
		return kind
	}
	return APIKeyKindAPIIntegration
}

func (s *Service) Create(userID uint, role string, input CreateInput) (*CreateResponse, error) {
	if input.Name == "" {
		return nil, ErrAPIKeyNameRequired
	}
	if len(input.Name) > 100 {
		return nil, ErrAPIKeyNameTooLong
	}
	keyKind, err := NormalizeAPIKeyKind(input.KeyKind)
	if err != nil {
		return nil, err
	}

	if role != "admin" {
		var count int64
		s.db.Model(&model.APIKey{}).Where("user_id = ?", userID).Count(&count)
		if count >= maxKeysPerUser {
			return nil, ErrAPIKeyLimitReached
		}
	}

	scopes, err := normalizeScopes(input.Scopes)
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
		KeyKind:   keyKind,
		Scopes:    strings.Join(scopes, ","),
		ExpiresAt: input.ExpiresAt,
	}

	if err := s.db.Create(&apiKey).Error; err != nil {
		return nil, fmt.Errorf("failed to create api key: %w", err)
	}

	return &CreateResponse{
		APIKey: apiKey,
		Key:    rawKey,
	}, nil
}

func (s *Service) List(userID uint) ([]model.APIKey, error) {
	var keys []model.APIKey
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&keys).Error; err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *Service) Delete(userID uint, keyID uint) error {
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
func (s *Service) ValidateKey(rawKey string) (*Principal, error) {
	keyHash := hashAPIKey(rawKey)

	var apiKey model.APIKey
	if err := s.db.Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		return nil, ErrAPIKeyInvalid
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(pkg.Now()) {
		return nil, ErrAPIKeyExpired
	}

	if err := s.activeUserChecker.EnsureActiveUser(s.db, apiKey.UserID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, errUserInactive) {
			return nil, ErrAPIKeyInvalid
		}
		return nil, err
	}

	now := pkg.NowUTC()
	s.db.Model(&apiKey).Update("last_used_at", now)

	return &Principal{
		UserID:  apiKey.UserID,
		KeyID:   apiKey.ID,
		KeyKind: NormalizePersistedAPIKeyKind(apiKey.KeyKind),
		Scopes:  ParseAPIKeyScopes(apiKey.Scopes),
	}, nil
}
