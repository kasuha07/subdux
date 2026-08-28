package notification

import (
	"errors"
	"fmt"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
)

func (s *Service) GetPolicy(userID uint) (*model.NotificationPolicy, error) {
	var policy model.NotificationPolicy
	if err := s.DB.Where("user_id = ?", userID).First(&policy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.NotificationPolicy{
				UserID:                 userID,
				DaysBefore:             3,
				NotifyOnDueDay:         true,
				NotifyManualRenewDaily: false,
			}, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (s *Service) UpdatePolicy(userID uint, input UpdatePolicyInput) (*model.NotificationPolicy, error) {
	quietHoursChanged := input.QuietHoursEnabled != nil || input.QuietHoursStart != nil || input.QuietHoursEnd != nil
	var policy model.NotificationPolicy
	err := s.DB.Where("user_id = ?", userID).First(&policy).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		policy = model.NotificationPolicy{
			UserID:                 userID,
			DaysBefore:             3,
			NotifyOnDueDay:         true,
			NotifyManualRenewDaily: false,
		}
	}

	if input.DaysBefore != nil {
		if *input.DaysBefore < 0 || *input.DaysBefore > maxNotificationDaysBefore {
			return nil, serviceerr.NewCode(
				serviceerr.KindInvalid,
				"days_before_must_be_between_0_and_max",
				fmt.Sprintf("days_before must be between 0 and %d", maxNotificationDaysBefore),
				map[string]any{"max": maxNotificationDaysBefore},
			)
		}
		policy.DaysBefore = *input.DaysBefore
	}
	if input.NotifyOnDueDay != nil {
		policy.NotifyOnDueDay = *input.NotifyOnDueDay
	}
	if input.NotifyManualRenewDaily != nil {
		policy.NotifyManualRenewDaily = *input.NotifyManualRenewDaily
	}
	if input.QuietHoursStart != nil {
		if *input.QuietHoursStart != "" && !ValidQuietHoursTime(*input.QuietHoursStart) {
			return nil, invalidQuietHoursTimeError()
		}
		policy.QuietHoursStart = *input.QuietHoursStart
	}
	if input.QuietHoursEnd != nil {
		if *input.QuietHoursEnd != "" && !ValidQuietHoursTime(*input.QuietHoursEnd) {
			return nil, invalidQuietHoursTimeError()
		}
		policy.QuietHoursEnd = *input.QuietHoursEnd
	}
	if input.QuietHoursEnabled != nil {
		policy.QuietHoursEnabled = *input.QuietHoursEnabled
	}

	if policy.ID == 0 {
		if err := s.DB.Model(&model.NotificationPolicy{}).Create(map[string]interface{}{
			"user_id":                   policy.UserID,
			"days_before":               policy.DaysBefore,
			"notify_on_due_day":         policy.NotifyOnDueDay,
			"notify_manual_renew_daily": policy.NotifyManualRenewDaily,
			"quiet_hours_enabled":       policy.QuietHoursEnabled,
			"quiet_hours_start":         policy.QuietHoursStart,
			"quiet_hours_end":           policy.QuietHoursEnd,
		}).Error; err != nil {
			return nil, err
		}
		if err := s.DB.Where("user_id = ?", userID).First(&policy).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.DB.Save(&policy).Error; err != nil {
			return nil, err
		}
	}
	if quietHoursChanged {
		if err := s.reschedulePendingQuietHoursOutbox(userID, policy); err != nil {
			return nil, err
		}
	}

	return &policy, nil
}

func (s *Service) reschedulePendingQuietHoursOutbox(userID uint, policy model.NotificationPolicy) error {
	now := pkg.NowUTC()
	nextAttemptAt := now
	if policy.QuietHoursEnabled {
		if inWindow, until := quietHoursDeferUntil(now, policy.QuietHoursStart, policy.QuietHoursEnd, pkg.GetSystemTimezone()); inWindow {
			nextAttemptAt = until.UTC()
		}
	}

	// An empty last_error distinguishes quiet-hours deferral from delivery
	// backoff. Keep failed-send retry schedules intact.
	return s.DB.Model(&model.NotificationOutbox{}).
		Where("user_id = ? AND status = ? AND last_error = ? AND next_attempt_at > ?", userID, notificationOutboxStatusPending, "", now).
		Updates(map[string]interface{}{
			"scheduled_for":   nextAttemptAt,
			"next_attempt_at": nextAttemptAt,
			"updated_at":      now,
		}).Error
}

func (s *Service) ListLogs(userID uint, limit int) ([]model.NotificationLog, error) {
	if limit <= 0 {
		limit = notificationLogRetentionLimit
	}
	if limit > notificationLogRetentionLimit {
		limit = notificationLogRetentionLimit
	}
	var logs []model.NotificationLog
	err := s.DB.Where("user_id = ?", userID).
		Order("sent_at DESC, id DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
