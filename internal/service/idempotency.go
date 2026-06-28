package service

import (
	"errors"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

// IdempotencyService persists and replays the outcome of MCP write tool calls.
// It is deliberately thin: the orchestration that decides whether to replay,
// reject, or execute lives in the MCP handler so the mutation, audit event, and
// idempotency record all commit inside a single transaction.
type IdempotencyService struct {
	DB *gorm.DB
}

func NewIdempotencyService(db *gorm.DB) *IdempotencyService {
	return &IdempotencyService{DB: db}
}

// Lookup returns the stored record for a user-scoped idempotency key, or
// (nil, nil) when no record exists yet. A non-nil error indicates a real
// lookup failure rather than a cache miss.
func (s *IdempotencyService) Lookup(userID uint, key string) (*model.MCPIdempotencyKey, error) {
	var record model.MCPIdempotencyKey
	err := s.DB.Where("user_id = ? AND idempotency_key = ?", userID, key).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// Save persists a new idempotency record. The caller is expected to run this
// inside the same transaction as the mutation it describes so that a committed
// mutation always has a matching record and a rolled-back mutation has none.
func (s *IdempotencyService) Save(record *model.MCPIdempotencyKey) error {
	return s.DB.Create(record).Error
}
