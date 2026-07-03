package serviceutil

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AcquireBackgroundTaskLease tries to claim taskKey for ownerID until now+ttl.
// It returns true only to the caller that holds the lease.
func AcquireBackgroundTaskLease(db *gorm.DB, ownerID, taskKey string, ttl time.Duration) (bool, error) {
	if taskKey == "" || ownerID == "" || ttl <= 0 {
		return false, nil
	}

	now := pkg.NowUTC()
	leaseUntil := now.Add(ttl)

	lease := model.BackgroundTaskLease{
		TaskKey:     taskKey,
		OwnerID:     ownerID,
		LeaseUntil:  leaseUntil,
		HeartbeatAt: now,
	}
	result := db.Clauses(clause.OnConflict{DoNothing: true}).Create(&lease)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}

	result = db.Model(&model.BackgroundTaskLease{}).
		Where("task_key = ? AND (lease_until <= ? OR owner_id = ?)", taskKey, now, ownerID).
		Updates(map[string]interface{}{
			"owner_id":     ownerID,
			"lease_until":  leaseUntil,
			"heartbeat_at": now,
			"updated_at":   now,
		})
	if result.Error != nil {
		return false, result.Error
	}

	return result.RowsAffected == 1, nil
}

// WithBackgroundTaskLease runs fn only if ownerID can claim taskKey.
func WithBackgroundTaskLease(db *gorm.DB, ownerID, taskKey string, ttl time.Duration, run func() error) error {
	acquired, err := AcquireBackgroundTaskLease(db, ownerID, taskKey, ttl)
	if err != nil || !acquired {
		return err
	}
	return run()
}

// NewBackgroundTaskOwnerID returns a process-stable identifier for claiming
// background-task leases.
func NewBackgroundTaskOwnerID() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "subdux"
	}

	var randomBytes [4]byte
	if _, err := rand.Read(randomBytes[:]); err != nil {
		return fmt.Sprintf("%s:%d", hostname, os.Getpid())
	}
	return fmt.Sprintf("%s:%d:%s", hostname, os.Getpid(), hex.EncodeToString(randomBytes[:]))
}
