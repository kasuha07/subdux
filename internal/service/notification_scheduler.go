package service

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/pkg/logging"
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
	if err := reconcileSubscriptionLifecycleForUser(s.DB, userID, now); err != nil {
		return err
	}

	policy, err := s.GetPolicy(userID)
	if err != nil {
		return err
	}

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND status = ? AND billing_type = ? AND (next_billing_date IS NOT NULL OR ends_at IS NOT NULL)",
		userID, subscriptionStatusActive, billingTypeRecurring).Find(&subs).Error; err != nil {
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

	systemLoc := pkg.GetSystemTimezone()
	scheduledDispatches := make(map[string]struct{})

	endedManualRenewSubs, err := s.manualRenewEndedNotificationCandidates(userID, now)
	if err != nil {
		return err
	}
	for _, sub := range endedManualRenewSubs {
		if sub.EndsAt == nil {
			continue
		}

		notifyEnabled := true
		if sub.NotifyEnabled != nil {
			notifyEnabled = *sub.NotifyEnabled
		}
		if !notifyEnabled {
			continue
		}

		endedAt := pkg.NormalizeDateInTimezone(*sub.EndsAt, systemLoc)
		for _, channel := range enabledChannels {
			if !shouldScheduleNotificationOutbox(scheduledDispatches, sub.ID, channel.Type, notificationTriggerManualEnded, endedAt, endedAt) {
				continue
			}

			templateData := s.buildTemplateData(&sub, &user, endedAt, 0, "manual_renew_ended")
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
				subscriptionID:  sub.ID,
				channel:         channel,
				triggerType:     notificationTriggerManualEnded,
				notifyDate:      endedAt,
				dedupeDate:      endedAt,
				message:         message,
				targetEmail:     user.Email,
				subscriptionURL: sub.URL,
			}); err != nil {
				return err
			}
		}
	}

	for _, sub := range subs {
		if normalizeRenewalMode(sub.RenewalMode) != renewalModeCancelAtPeriodEnd {
			continue
		}

		boundary := cancelAtPeriodEndBoundary(sub)
		if boundary == nil {
			continue
		}

		notifyEnabled := true
		daysBefore := policy.DaysBefore
		notifyOnDueDay := policy.NotifyOnDueDay

		if sub.NotifyEnabled != nil {
			notifyEnabled = *sub.NotifyEnabled
		}
		if !notifyEnabled {
			continue
		}
		if sub.NotifyDaysBefore != nil {
			daysBefore = *sub.NotifyDaysBefore
		}

		endDate := pkg.NormalizeDateInTimezone(*boundary, systemLoc)
		scanDate := pkg.NormalizeDateInTimezone(now, systemLoc)
		daysUntilEnd := pkg.DaysUntil(endDate, systemLoc)
		triggerTypes := notificationTriggerTypes(daysUntilEnd, daysBefore, notifyOnDueDay)
		if len(triggerTypes) == 0 {
			continue
		}

		for _, channel := range enabledChannels {
			if !shouldScheduleNotificationOutbox(scheduledDispatches, sub.ID, channel.Type, notificationTriggerEndingSoon, endDate, scanDate) {
				continue
			}

			templateData := s.buildTemplateData(&sub, &user, endDate, daysUntilEnd, "ending_soon")
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
				subscriptionID:  sub.ID,
				channel:         channel,
				triggerType:     notificationTriggerEndingSoon,
				notifyDate:      endDate,
				dedupeDate:      scanDate,
				message:         message,
				targetEmail:     user.Email,
				subscriptionURL: sub.URL,
			}); err != nil {
				return err
			}
		}
	}

	for _, sub := range subs {
		if sub.NextBillingDate == nil || !subscriptionHasFutureCharge(sub) {
			continue
		}

		notifyEnabled := true
		daysBefore := policy.DaysBefore
		notifyOnDueDay := policy.NotifyOnDueDay

		if sub.NotifyEnabled != nil {
			notifyEnabled = *sub.NotifyEnabled
		}
		if !notifyEnabled {
			continue
		}
		if sub.NotifyDaysBefore != nil {
			daysBefore = *sub.NotifyDaysBefore
		}

		billingDate := pkg.NormalizeDateInTimezone(*sub.NextBillingDate, systemLoc)
		scanDate := pkg.NormalizeDateInTimezone(now, systemLoc)

		daysUntilBilling := pkg.DaysUntil(*sub.NextBillingDate, systemLoc)
		triggerTypes := notificationTriggerTypesForSubscription(
			sub.RenewalMode,
			daysUntilBilling,
			daysBefore,
			notifyOnDueDay,
			policy.NotifyManualRenewDaily,
		)
		if len(triggerTypes) == 0 {
			continue
		}

		for _, channel := range enabledChannels {
			for _, triggerType := range triggerTypes {
				dedupeDate := billingDate
				if triggerType == notificationTriggerManualDaily {
					dedupeDate = scanDate
				}
				if !shouldScheduleNotificationOutbox(scheduledDispatches, sub.ID, channel.Type, triggerType, billingDate, dedupeDate) {
					continue
				}

				eventType := notificationEventTypeForSubscription(sub)
				templateData := s.buildTemplateData(&sub, &user, billingDate, daysUntilBilling, eventType)
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
					subscriptionID:  sub.ID,
					channel:         channel,
					triggerType:     triggerType,
					notifyDate:      billingDate,
					dedupeDate:      dedupeDate,
					message:         message,
					targetEmail:     user.Email,
					subscriptionURL: sub.URL,
				}); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func cancelAtPeriodEndBoundary(sub model.Subscription) *time.Time {
	if sub.EndsAt != nil {
		return sub.EndsAt
	}
	return sub.NextBillingDate
}

func (s *NotificationService) manualRenewEndedNotificationCandidates(userID uint, referenceDate time.Time) ([]model.Subscription, error) {
	today := normalizeDateUTC(referenceDate)
	var subs []model.Subscription
	if err := s.DB.Where(
		"user_id = ? AND status = ? AND renewal_mode = ? AND billing_type = ? AND ends_at IS NOT NULL AND ends_at < ?",
		userID,
		subscriptionStatusEnded,
		renewalModeManualRenew,
		billingTypeRecurring,
		today,
	).Order("ends_at ASC, id ASC").Find(&subs).Error; err != nil {
		return nil, err
	}

	candidates := make([]model.Subscription, 0, len(subs))
	for _, sub := range subs {
		if sub.EndsAt == nil {
			continue
		}
		sent, err := s.manualRenewEndedNotificationAlreadySent(sub.ID, normalizeDateUTC(*sub.EndsAt))
		if err != nil {
			return nil, err
		}
		if sent {
			continue
		}
		candidates = append(candidates, sub)
	}
	return candidates, nil
}

func (s *NotificationService) manualRenewEndedNotificationAlreadySent(subscriptionID uint, endedAt time.Time) (bool, error) {
	var count int64
	err := s.DB.Model(&model.NotificationLog{}).
		Where("subscription_id = ? AND trigger_type = ? AND notify_date = ? AND status = ?",
			subscriptionID, notificationTriggerManualEnded, normalizeDateUTC(endedAt), notificationLogStatusSent).
		Count(&count).Error
	return count > 0, err
}
