package notification

import (
	"errors"
	"fmt"

	"github.com/kasuha07/subdux/internal/model"
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
			return nil, serviceerr.New(serviceerr.KindInvalid, fmt.Sprintf("days_before must be between 0 and %d", maxNotificationDaysBefore))
		}
		policy.DaysBefore = *input.DaysBefore
	}
	if input.NotifyOnDueDay != nil {
		policy.NotifyOnDueDay = *input.NotifyOnDueDay
	}
	if input.NotifyManualRenewDaily != nil {
		policy.NotifyManualRenewDaily = *input.NotifyManualRenewDaily
	}

	if policy.ID == 0 {
		if err := s.DB.Model(&model.NotificationPolicy{}).Create(map[string]interface{}{
			"user_id":                   policy.UserID,
			"days_before":               policy.DaysBefore,
			"notify_on_due_day":         policy.NotifyOnDueDay,
			"notify_manual_renew_daily": policy.NotifyManualRenewDaily,
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

	return &policy, nil
}

func (s *Service) ListLogs(userID uint, limit int) ([]model.NotificationLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var logs []model.NotificationLog
	err := s.DB.Where("user_id = ?", userID).
		Order("sent_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
