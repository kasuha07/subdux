package idempotency

import (
	"context"
	"errors"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

// Service persists and replays the outcome of MCP write tool calls.
// It is deliberately thin: the orchestration that decides whether to replay,
// reject, or execute lives in the MCP handler so the mutation, audit event, and
// idempotency record all commit inside a single transaction.
type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	if s.DB != nil {
		clone.DB = s.DB.WithContext(ctx)
	}
	return &clone
}

// Lookup returns the stored record for a user-scoped idempotency key, or
// (nil, nil) when no record exists yet. A non-nil error indicates a real
// lookup failure rather than a cache miss.
func (s *Service) Lookup(userID uint, key string) (*model.MCPIdempotencyKey, error) {
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
func (s *Service) Save(record *model.MCPIdempotencyKey) error {
	return s.DB.Create(record).Error
}
