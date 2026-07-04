package service

import (
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	"gorm.io/gorm"
)

var (
	ErrAPIKeyNotFound     = apikeyservice.ErrAPIKeyNotFound
	ErrAPIKeyNameRequired = apikeyservice.ErrAPIKeyNameRequired
	ErrAPIKeyNameTooLong  = apikeyservice.ErrAPIKeyNameTooLong
	ErrAPIKeyExpired      = apikeyservice.ErrAPIKeyExpired
	ErrAPIKeyInvalid      = apikeyservice.ErrAPIKeyInvalid
	ErrAPIKeyLimitReached = apikeyservice.ErrAPIKeyLimitReached
	ErrAPIKeyScopeInvalid = apikeyservice.ErrAPIKeyScopeInvalid
	ErrAPIKeyKindRequired = apikeyservice.ErrAPIKeyKindRequired
	ErrAPIKeyKindInvalid  = apikeyservice.ErrAPIKeyKindInvalid
)

const (
	APIKeyScopeRead  = apikeyservice.APIKeyScopeRead
	APIKeyScopeWrite = apikeyservice.APIKeyScopeWrite
)

const (
	APIKeyKindMCPClient      = apikeyservice.APIKeyKindMCPClient
	APIKeyKindAPIIntegration = apikeyservice.APIKeyKindAPIIntegration
)

type APIKeyService = apikeyservice.Service

func NewAPIKeyService(db *gorm.DB) *APIKeyService {
	return apikeyservice.NewService(db)
}

type CreateAPIKeyInput = apikeyservice.CreateInput
type CreateAPIKeyResponse = apikeyservice.CreateResponse
type APIKeyPrincipal = apikeyservice.Principal

func ParseAPIKeyScopes(raw string) []string {
	return apikeyservice.ParseAPIKeyScopes(raw)
}

func NormalizeAPIKeyKind(value string) (string, error) {
	return apikeyservice.NormalizeAPIKeyKind(value)
}

func NormalizePersistedAPIKeyKind(value string) string {
	return apikeyservice.NormalizePersistedAPIKeyKind(value)
}
