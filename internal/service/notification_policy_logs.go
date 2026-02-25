package service

import (
	"errors"
	"fmt"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func (s *NotificationService) GetPolicy(userID uint) (*model.NotificationPolicy, error) {
	var policy model.NotificationPolicy
	if err := s.DB.Where("user_id = ?", userID).First(&policy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.NotificationPolicy{
				UserID:         userID,
				DaysBefore:     3,
				NotifyOnDueDay: true,
			}, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (s *NotificationService) UpdatePolicy(userID uint, input UpdatePolicyInput) (*model.NotificationPolicy, error) {
	var policy model.NotificationPolicy
	err := s.DB.Where("user_id = ?", userID).First(&policy).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		policy = model.NotificationPolicy{
			UserID:         userID,
			DaysBefore:     3,
			NotifyOnDueDay: true,
		}
	}

	if input.DaysBefore != nil {
		if *input.DaysBefore < 0 || *input.DaysBefore > maxNotificationDaysBefore {
			return nil, fmt.Errorf("days_before must be between 0 and %d", maxNotificationDaysBefore)
		}
		policy.DaysBefore = *input.DaysBefore
	}
	if input.NotifyOnDueDay != nil {
		policy.NotifyOnDueDay = *input.NotifyOnDueDay
	}

	if policy.ID == 0 {
		if err := s.DB.Create(&policy).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.DB.Save(&policy).Error; err != nil {
			return nil, err
		}
	}

	return &policy, nil
}

func (s *NotificationService) ListLogs(userID uint, limit int) ([]model.NotificationLog, error) {
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
