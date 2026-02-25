package service

import (
	"fmt"
	"sync"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func (s *NotificationService) ProcessPendingNotifications() error {
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
					fmt.Printf("notification error for user %d: %v\n", userID, err)
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
	if jobCount < maxParallelNotificationDispatchesPerUser {
		return jobCount
	}
	return maxParallelNotificationDispatchesPerUser
}

func notificationDispatchKey(subscriptionID uint, channelType string, notifyDate time.Time) string {
	return fmt.Sprintf("%d|%s|%s", subscriptionID, channelType, notifyDate.UTC().Format(time.RFC3339Nano))
}

func shouldScheduleNotificationDispatch(
	scheduled map[string]struct{},
	subscriptionID uint,
	channelType string,
	notifyDate time.Time,
) bool {
	key := notificationDispatchKey(subscriptionID, channelType, notifyDate)
	if _, exists := scheduled[key]; exists {
		return false
	}
	scheduled[key] = struct{}{}
	return true
}

func (s *NotificationService) processUserNotifications(userID uint) error {
	now := time.Now().In(pkg.GetSystemTimezone())
	if err := autoAdvanceRecurringNextBillingDatesForUser(s.DB, userID, now); err != nil {
		return err
	}

	policy, err := s.GetPolicy(userID)
	if err != nil {
		return err
	}

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND enabled = ? AND billing_type = ? AND next_billing_date IS NOT NULL",
		userID, true, "recurring").Find(&subs).Error; err != nil {
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
	dispatchJobs := make([]notificationDispatchJob, 0)
	scheduledDispatches := make(map[string]struct{})

	for _, sub := range subs {
		if sub.NextBillingDate == nil {
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

		shouldNotify := false
		daysUntilBilling := pkg.DaysUntil(*sub.NextBillingDate, systemLoc)

		if daysUntilBilling == daysBefore && daysBefore > 0 {
			shouldNotify = true
		}
		if daysUntilBilling == 0 && notifyOnDueDay {
			shouldNotify = true
		}

		if !shouldNotify {
			continue
		}

		for _, channel := range enabledChannels {
			if !shouldScheduleNotificationDispatch(scheduledDispatches, sub.ID, channel.Type, billingDate) {
				continue
			}

			var count int64
			s.DB.Model(&model.NotificationLog{}).
				Where("subscription_id = ? AND channel_type = ? AND notify_date = ? AND status = ?",
					sub.ID, channel.Type, billingDate, "sent").
				Count(&count)
			if count > 0 {
				continue
			}

			templateData := s.buildTemplateData(&sub, &user, billingDate, daysUntilBilling)
			message, renderErr := s.renderNotificationMessage(userID, channel.Type, templateData)
			if renderErr != nil {
				fmt.Printf("failed to render template for user %d channel %s: %v\n", userID, channel.Type, renderErr)
				continue
			}
			dispatchJobs = append(dispatchJobs, notificationDispatchJob{
				subscriptionID:  sub.ID,
				channel:         channel,
				notifyDate:      billingDate,
				message:         message,
				targetEmail:     user.Email,
				subscriptionURL: sub.URL,
			})
		}
	}

	return s.dispatchNotificationJobs(userID, dispatchJobs)
}
