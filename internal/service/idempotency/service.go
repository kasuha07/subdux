package idempotency

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg/logging"
	"gorm.io/gorm"
)

const (
	// RetentionWindow is how long a write's idempotency record (and therefore
	// its replayable result) is kept. A retry with the same key after the
	// window executes as a new request.
	RetentionWindow = 24 * time.Hour
	// MaxRecordsPerUser bounds the records one user can accumulate inside the
	// retention window; the oldest are evicted first.
	MaxRecordsPerUser = 10000

	cleanupInterval = time.Hour
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
//
// A record older than RetentionWindow is treated as absent and removed, so the
// key can be reused and the caller's Save does not collide with it.
func (s *Service) Lookup(userID uint, key string) (*model.MCPIdempotencyKey, error) {
	var record model.MCPIdempotencyKey
	err := s.DB.Where("user_id = ? AND idempotency_key = ?", userID, key).First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	if record.CreatedAt.Before(time.Now().Add(-RetentionWindow)) {
		if err := s.DB.Delete(&model.MCPIdempotencyKey{}, record.ID).Error; err != nil {
			return nil, err
		}
		return nil, nil
	}
	return &record, nil
}

// Save persists a new idempotency record. The caller is expected to run this
// inside the same transaction as the mutation it describes so that a committed
// mutation always has a matching record and a rolled-back mutation has none.
//
// Save keeps each user at or below MaxRecordsPerUser by evicting that user's
// oldest records first.
func (s *Service) Save(record *model.MCPIdempotencyKey) error {
	var count int64
	if err := s.DB.Model(&model.MCPIdempotencyKey{}).Where("user_id = ?", record.UserID).Count(&count).Error; err != nil {
		return err
	}
	if overflow := count - MaxRecordsPerUser + 1; overflow > 0 {
		var ids []uint
		if err := s.DB.Model(&model.MCPIdempotencyKey{}).
			Where("user_id = ?", record.UserID).
			Order("created_at ASC").Order("id ASC").
			Limit(int(overflow)).
			Pluck("id", &ids).Error; err != nil {
			return err
		}
		if len(ids) > 0 {
			if err := s.DB.Where("id IN ?", ids).Delete(&model.MCPIdempotencyKey{}).Error; err != nil {
				return err
			}
		}
	}
	return s.DB.Create(record).Error
}

// Cleanup deletes every record older than RetentionWindow.
func (s *Service) Cleanup(now time.Time) error {
	return s.DB.Where("created_at < ?", now.Add(-RetentionWindow)).Delete(&model.MCPIdempotencyKey{}).Error
}

// StartCleanupLoop purges expired records now and then hourly until ctx ends.
func (s *Service) StartCleanupLoop(ctx context.Context) {
	if s == nil || s.DB == nil || ctx == nil {
		return
	}
	go func() {
		run := func() {
			if err := s.WithContext(ctx).Cleanup(time.Now()); err != nil && ctx.Err() == nil {
				logging.Warn("failed to clean up MCP idempotency records", slog.Any("error", err))
			}
		}
		run()
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				run()
			case <-ctx.Done():
				return
			}
		}
	}()
}
