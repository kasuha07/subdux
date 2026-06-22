package service

import (
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm/clause"
)

func (s *NotificationService) acquireBackgroundTaskLease(taskKey string, ttl time.Duration) (bool, error) {
	if taskKey == "" || ttl <= 0 {
		return false, nil
	}

	now := pkg.NowUTC()
	leaseUntil := now.Add(ttl)
	ownerID := s.notificationOwnerID()

	lease := model.BackgroundTaskLease{
		TaskKey:     taskKey,
		OwnerID:     ownerID,
		LeaseUntil:  leaseUntil,
		HeartbeatAt: now,
	}
	result := s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&lease)
	if result.Error != nil {
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}

	result = s.DB.Model(&model.BackgroundTaskLease{}).
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

func (s *NotificationService) withBackgroundTaskLease(taskKey string, ttl time.Duration, run func() error) error {
	acquired, err := s.acquireBackgroundTaskLease(taskKey, ttl)
	if err != nil || !acquired {
		return err
	}
	return run()
}
