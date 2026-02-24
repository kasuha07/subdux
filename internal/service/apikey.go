package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
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
)

const maxAPIKeysPerUser = 5

type APIKeyService struct {
	db *gorm.DB
}

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return &APIKeyService{db: db}
}

type CreateAPIKeyInput struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at"`
}

type CreateAPIKeyResponse struct {
	APIKey model.APIKey `json:"api_key"`
	Key    string       `json:"key"`
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

// ValidateKey checks a raw API key string and returns the associated user ID.
func (s *APIKeyService) ValidateKey(rawKey string) (uint, error) {
	keyHash := hashAPIKey(rawKey)

	var apiKey model.APIKey
	if err := s.db.Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
		return 0, ErrAPIKeyInvalid
	}

	if apiKey.ExpiresAt != nil && apiKey.ExpiresAt.Before(time.Now()) {
		return 0, ErrAPIKeyExpired
	}

	// Update last used timestamp
	s.db.Model(&apiKey).Update("last_used_at", time.Now())

	return apiKey.UserID, nil
}
