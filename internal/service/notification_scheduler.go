package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/logging"
)

const notificationScanTaskKey = "notification_scan"
const notificationScanLeaseTTL = 30 * time.Minute

func (s *NotificationService) ProcessPendingNotifications() error {
	if err := s.EnqueuePendingNotifications(); err != nil {
		return err
	}
	_, err := s.DispatchDueNotificationOutbox(context.Background())
	return err
}

func (s *NotificationService) EnqueuePendingNotifications() error {
	return s.withBackgroundTaskLease(notificationScanTaskKey, notificationScanLeaseTTL, s.enqueuePendingNotifications)
}

func (s *NotificationService) enqueuePendingNotifications() error {
	var channelUserIDs []uint
	if err := s.DB.Model(&model.NotificationChannel{}).
		Where("enabled = ?", true).
		Distinct("user_id").
		Pluck("user_id", &channelUserIDs).Error; err != nil {
		return fmt.Errorf("failed to query notification channels: %w", err)
	}

	userIDs := uniqueUserIDs(channelUserIDs)
	if len(userIDs) == 0 {
		return nil
	}

	workerCount := notificationWorkerCount(len(userIDs))
	userJobs := make(chan uint, len(userIDs))
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for userID := range userJobs {
				if err := s.processUserNotifications(userID); err != nil {
					logging.Error("notification processing failed for user",
						slog.Uint64("user_id", uint64(userID)), slog.Any("error", err))
				}
			}
		}()
	}

	for _, userID := range userIDs {
		userJobs <- userID
	}
	close(userJobs)
	wg.Wait()

	return nil
}

func uniqueUserIDs(userIDs []uint) []uint {
	if len(userIDs) == 0 {
		return nil
	}

	seen := make(map[uint]struct{}, len(userIDs))
	unique := make([]uint, 0, len(userIDs))
	for _, userID := range userIDs {
		if _, ok := seen[userID]; ok {
			continue
		}
		seen[userID] = struct{}{}
		unique = append(unique, userID)
	}

	return unique
}

func notificationWorkerCount(userCount int) int {
	if userCount <= 0 {
		return 0
	}
	if userCount < maxParallelUserNotificationChecks {
		return userCount
	}
	return maxParallelUserNotificationChecks
}

func notificationDispatchWorkerCount(jobCount int) int {
	if jobCount <= 0 {
		return 0
	}
	if jobCount < maxParallelNotificationDispatches {
		return jobCount
	}
	return maxParallelNotificationDispatches
}

func (s *NotificationService) processUserNotifications(userID uint) error {
	now := pkg.NowInSystemTimezone()

	policy, err := s.GetPolicy(userID)
	if err != nil {
		return err
	}
	reminderPolicy := subscriptionReminderPolicy{
		DaysBefore:             policy.DaysBefore,
		NotifyOnDueDay:         policy.NotifyOnDueDay,
		NotifyManualRenewDaily: policy.NotifyManualRenewDaily,
	}

	candidates, err := listSubscriptionReminderCandidates(context.Background(), s.DB, userID, now, reminderPolicy)
	if err != nil {
		return err
	}

	var enabledChannels []model.NotificationChannel
	if err := s.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&enabledChannels).Error; err != nil {
		return err
	}

	if len(enabledChannels) == 0 {
		return nil
	}

	var user model.User
	if err := s.DB.Select("email").First(&user, userID).Error; err != nil {
		return err
	}

	scheduledDispatches := make(map[string]struct{})

	endedManualRenewSubs, err := listEndedManualRenewNotificationCandidates(context.Background(), s.DB, userID, now)
	if err != nil {
		return err
	}
	for _, candidate := range endedManualRenewSubs {
		sent, err := s.manualRenewEndedNotificationAlreadySent(candidate.SubscriptionID, normalizeDateUTC(candidate.NotifyDate))
		if err != nil {
			return err
		}
		if sent {
			continue
		}

		for _, channel := range enabledChannels {
			if !shouldScheduleNotificationOutbox(scheduledDispatches, candidate.SubscriptionID, channel.Type, candidate.TriggerType, candidate.NotifyDate, candidate.DedupeDate) {
				continue
			}

			templateData := s.buildTemplateData(candidate, &user)
			message, renderErr := s.renderNotificationMessage(userID, channel.Type, templateData)
			if renderErr != nil {
				logging.Error("failed to render notification template",
					slog.Uint64("user_id", uint64(userID)),
					slog.String("channel", channel.Type),
					slog.Any("error", renderErr))
				continue
			}
			if err := s.enqueueNotificationOutbox(notificationOutboxJob{
				userID:          userID,
				subscriptionID:  candidate.SubscriptionID,
				channel:         channel,
				triggerType:     candidate.TriggerType,
				notifyDate:      candidate.NotifyDate,
				dedupeDate:      candidate.DedupeDate,
				message:         message,
				targetEmail:     user.Email,
				subscriptionURL: candidate.Template.URL,
			}); err != nil {
				return err
			}
		}
	}

	for _, candidate := range candidates {
		for _, channel := range enabledChannels {
			if !shouldScheduleNotificationOutbox(scheduledDispatches, candidate.SubscriptionID, channel.Type, candidate.TriggerType, candidate.NotifyDate, candidate.DedupeDate) {
				continue
			}

			templateData := s.buildTemplateData(candidate, &user)
			message, renderErr := s.renderNotificationMessage(userID, channel.Type, templateData)
			if renderErr != nil {
				logging.Error("failed to render notification template",
					slog.Uint64("user_id", uint64(userID)),
					slog.String("channel", channel.Type),
					slog.Any("error", renderErr))
				continue
			}
			if err := s.enqueueNotificationOutbox(notificationOutboxJob{
				userID:          userID,
				subscriptionID:  candidate.SubscriptionID,
				channel:         channel,
				triggerType:     candidate.TriggerType,
				notifyDate:      candidate.NotifyDate,
				dedupeDate:      candidate.DedupeDate,
				message:         message,
				targetEmail:     user.Email,
				subscriptionURL: candidate.Template.URL,
			}); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *NotificationService) manualRenewEndedNotificationAlreadySent(subscriptionID uint, endedAt time.Time) (bool, error) {
	var count int64
	err := s.DB.Model(&model.NotificationLog{}).
		Where("subscription_id = ? AND trigger_type = ? AND notify_date = ? AND status = ?",
			subscriptionID, notificationTriggerManualEnded, normalizeDateUTC(endedAt), notificationLogStatusSent).
		Count(&count).Error
	return count > 0, err
}
