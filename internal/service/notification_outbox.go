package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/pkg/logging"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	notificationOutboxStatusPending    = "pending"
	notificationOutboxStatusProcessing = "processing"
	notificationOutboxStatusSent       = "sent"
	notificationOutboxStatusFailed     = "failed"
	notificationOutboxStatusCancelled  = "cancelled"
	notificationOutboxStatusExpired    = "expired"

	notificationTriggerDaysBefore  = "days_before"
	notificationTriggerDueDay      = "due_day"
	notificationTriggerManualDaily = "manual_renew_daily"
	notificationTriggerManualEnded = "manual_renew_ended"
	notificationTriggerEndingSoon  = "ending_soon"

	notificationOutboxVersion      = "v1"
	notificationOutboxLeaseTTL     = 5 * time.Minute
	notificationOutboxExpiryWindow = 36 * time.Hour
)

type notificationOutboxJob struct {
	userID          uint
	subscriptionID  uint
	channel         model.NotificationChannel
	triggerType     string
	notifyDate      time.Time
	dedupeDate      time.Time
	message         string
	targetEmail     string
	subscriptionURL string
}

type NotificationDispatchSummary struct {
	Claimed   int
	Sent      int
	Failed    int
	Retried   int
	Cancelled int
	Expired   int
}

func notificationOutboxDedupeKey(userID, subscriptionID uint, channelType, triggerType string, notifyDate time.Time) string {
	return notificationOutboxDedupeKeyForTrigger(userID, subscriptionID, channelType, triggerType, notifyDate, notifyDate)
}

func notificationOutboxDedupeKeyForTrigger(userID, subscriptionID uint, channelType, triggerType string, notifyDate, dedupeDate time.Time) string {
	notifyDateKey := normalizeDateUTC(notifyDate).Format("2006-01-02")
	if notificationTriggerUsesDedupeDate(triggerType) {
		return fmt.Sprintf(
			"%s:%d:%d:%s:%s:%s:%s",
			notificationOutboxVersion,
			userID,
			subscriptionID,
			channelType,
			triggerType,
			notifyDateKey,
			normalizeDateUTC(dedupeDate).Format("2006-01-02"),
		)
	}

	return fmt.Sprintf(
		"%s:%d:%d:%s:%s:%s",
		notificationOutboxVersion,
		userID,
		subscriptionID,
		channelType,
		triggerType,
		notifyDateKey,
	)
}

func notificationTriggerUsesDedupeDate(triggerType string) bool {
	switch triggerType {
	case notificationTriggerManualDaily, notificationTriggerEndingSoon:
		return true
	default:
		return false
	}
}

func notificationTriggerRequiresExactSentLog(triggerType string) bool {
	switch triggerType {
	case notificationTriggerManualEnded, notificationTriggerEndingSoon:
		return true
	default:
		return false
	}
}

func notificationTriggerTypes(daysUntilBilling, daysBefore int, notifyOnDueDay bool) []string {
	triggers := make([]string, 0, 2)
	if daysUntilBilling == daysBefore && daysBefore > 0 {
		triggers = append(triggers, notificationTriggerDaysBefore)
	}
	if daysUntilBilling == 0 && notifyOnDueDay {
		triggers = append(triggers, notificationTriggerDueDay)
	}
	return triggers
}

func notificationTriggerTypesForSubscription(
	renewalMode string,
	daysUntilBilling int,
	daysBefore int,
	notifyOnDueDay bool,
	notifyManualRenewDaily bool,
) []string {
	triggers := notificationTriggerTypes(daysUntilBilling, daysBefore, notifyOnDueDay)
	if normalizeRenewalMode(renewalMode) == renewalModeManualRenew &&
		notifyManualRenewDaily &&
		daysBefore > 0 &&
		daysUntilBilling >= 0 &&
		daysUntilBilling < daysBefore &&
		(daysUntilBilling != 0 || !notifyOnDueDay) {
		triggers = append(triggers, notificationTriggerManualDaily)
	}
	return triggers
}

func notificationSentDateCandidates(notifyDate, originalNotifyDate time.Time) []time.Time {
	candidates := make([]time.Time, 0, 2)
	candidates = appendUniqueNotificationDate(candidates, normalizeDateUTC(notifyDate))
	if !originalNotifyDate.IsZero() {
		candidates = appendUniqueNotificationDate(candidates, pkg.NormalizeDateInTimezone(originalNotifyDate, originalNotifyDate.Location()))
	}
	return candidates
}

func appendUniqueNotificationDate(dates []time.Time, candidate time.Time) []time.Time {
	for _, existing := range dates {
		if existing.Equal(candidate) {
			return dates
		}
	}
	return append(dates, candidate)
}

func shouldScheduleNotificationOutbox(
	scheduled map[string]struct{},
	subscriptionID uint,
	channelType string,
	triggerType string,
	notifyDate time.Time,
	dedupeDate time.Time,
) bool {
	if dedupeDate.IsZero() {
		dedupeDate = notifyDate
	}
	key := fmt.Sprintf("%d|%s|%s|%s|%s",
		subscriptionID,
		channelType,
		triggerType,
		notifyDate.UTC().Format(time.RFC3339Nano),
		dedupeDate.UTC().Format(time.RFC3339Nano),
	)
	if _, exists := scheduled[key]; exists {
		return false
	}
	scheduled[key] = struct{}{}
	return true
}

func (s *NotificationService) enqueueNotificationOutbox(job notificationOutboxJob) error {
	notifyDate := normalizeDateUTC(job.notifyDate)
	dedupeDate := job.dedupeDate
	if dedupeDate.IsZero() {
		dedupeDate = notifyDate
	}
	if sent, err := s.notificationAlreadySent(job.subscriptionID, job.channel.Type, job.triggerType, notifyDate, job.notifyDate, dedupeDate); err != nil || sent {
		return err
	}

	channelID := job.channel.ID
	expiresAt := notifyDate.Add(notificationOutboxExpiryWindow)
	now := pkg.NowUTC()
	if job.triggerType == notificationTriggerManualEnded {
		expiresAt = now.Add(notificationOutboxExpiryWindow)
	}
	outbox := model.NotificationOutbox{
		DedupeKey:       notificationOutboxDedupeKeyForTrigger(job.userID, job.subscriptionID, job.channel.Type, job.triggerType, notifyDate, dedupeDate),
		UserID:          job.userID,
		SubscriptionID:  job.subscriptionID,
		ChannelID:       &channelID,
		ChannelType:     job.channel.Type,
		TriggerType:     job.triggerType,
		NotifyDate:      notifyDate,
		ScheduledFor:    now,
		ExpiresAt:       &expiresAt,
		Status:          notificationOutboxStatusPending,
		MaxAttempts:     notificationOutboxDefaultMaxAttempts,
		NextAttemptAt:   now,
		Message:         job.message,
		TargetEmail:     job.targetEmail,
		SubscriptionURL: job.subscriptionURL,
	}

	return s.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&outbox).Error
}

func (s *NotificationService) notificationAlreadySent(subscriptionID uint, channelType, triggerType string, notifyDate, originalNotifyDate, dedupeDate time.Time) (bool, error) {
	var count int64
	notifyDates := notificationSentDateCandidates(notifyDate, originalNotifyDate)
	query := s.DB.Model(&model.NotificationLog{}).
		Where("subscription_id = ? AND channel_type = ? AND notify_date IN ? AND status = ?",
			subscriptionID, channelType, notifyDates, notificationLogStatusSent)
	if notificationTriggerRequiresExactSentLog(triggerType) {
		query = query.Where("trigger_type = ?", triggerType)
	} else {
		query = query.Where("trigger_type = ? OR trigger_type = ? OR trigger_type IS NULL", triggerType, "")
	}
	if notificationTriggerUsesDedupeDate(triggerType) {
		systemLoc := pkg.GetSystemTimezone()
		sentDateStart := pkg.NormalizeDateInTimezone(dedupeDate, systemLoc)
		sentDateEnd := sentDateStart.AddDate(0, 0, 1)
		query = query.Where("sent_at >= ? AND sent_at < ?", sentDateStart.UTC(), sentDateEnd.UTC())
	}
	err := query.Count(&count).Error
	return count > 0, err
}

func (s *NotificationService) DispatchDueNotificationOutbox(ctx context.Context) (*NotificationDispatchSummary, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	summary := &NotificationDispatchSummary{}
	expired, err := s.expireNotificationOutbox()
	if err != nil {
		return summary, err
	}
	summary.Expired = expired

	jobs, err := s.claimDueNotificationOutbox(ctx, maxNotificationOutboxClaimBatch, notificationOutboxLeaseTTL)
	if err != nil {
		return summary, err
	}
	summary.Claimed = len(jobs)
	if len(jobs) == 0 {
		return summary, nil
	}

	workerCount := notificationDispatchWorkerCount(len(jobs))
	jobCh := make(chan model.NotificationOutbox, len(jobs))
	resultCh := make(chan string, len(jobs))
	var wg sync.WaitGroup

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobCh {
				if ctx.Err() != nil {
					resultCh <- notificationOutboxStatusPending
					continue
				}
				resultCh <- s.dispatchNotificationOutboxJob(job)
			}
		}()
	}

	for _, job := range jobs {
		jobCh <- job
	}
	close(jobCh)
	wg.Wait()
	close(resultCh)

	for status := range resultCh {
		switch status {
		case notificationOutboxStatusSent:
			summary.Sent++
		case notificationOutboxStatusFailed:
			summary.Failed++
		case notificationOutboxStatusPending:
			summary.Retried++
		case notificationOutboxStatusCancelled:
			summary.Cancelled++
		case notificationOutboxStatusExpired:
			summary.Expired++
		}
	}

	return summary, nil
}

func (s *NotificationService) expireNotificationOutbox() (int, error) {
	now := pkg.NowUTC()
	result := s.DB.Model(&model.NotificationOutbox{}).
		Where("status IN ? AND expires_at IS NOT NULL AND expires_at <= ?",
			[]string{notificationOutboxStatusPending, notificationOutboxStatusProcessing}, now).
		Updates(map[string]interface{}{
			"status":       notificationOutboxStatusExpired,
			"locked_by":    "",
			"locked_until": nil,
			"updated_at":   now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return int(result.RowsAffected), nil
}

func (s *NotificationService) claimDueNotificationOutbox(ctx context.Context, batchSize int, leaseTTL time.Duration) ([]model.NotificationOutbox, error) {
	if batchSize <= 0 {
		return nil, nil
	}

	now := pkg.NowUTC()
	var candidates []model.NotificationOutbox
	err := s.DB.Select("id").
		Where("status IN ? AND next_attempt_at <= ? AND (expires_at IS NULL OR expires_at > ?) AND (locked_until IS NULL OR locked_until <= ?)",
			[]string{notificationOutboxStatusPending, notificationOutboxStatusProcessing}, now, now, now).
		Order("next_attempt_at ASC, id ASC").
		Limit(batchSize).
		Find(&candidates).Error
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	ownerID := s.notificationOwnerID()
	leaseUntil := now.Add(leaseTTL)
	claimedIDs := make([]uint, 0, len(candidates))
	for _, candidate := range candidates {
		if ctx != nil && ctx.Err() != nil {
			break
		}

		result := s.DB.Model(&model.NotificationOutbox{}).
			Where("id = ? AND status IN ? AND next_attempt_at <= ? AND (expires_at IS NULL OR expires_at > ?) AND (locked_until IS NULL OR locked_until <= ?)",
				candidate.ID,
				[]string{notificationOutboxStatusPending, notificationOutboxStatusProcessing},
				now,
				now,
				now,
			).
			Updates(map[string]interface{}{
				"status":          notificationOutboxStatusProcessing,
				"locked_by":       ownerID,
				"locked_until":    leaseUntil,
				"last_attempt_at": now,
				"attempt_count":   gorm.Expr("attempt_count + ?", 1),
				"updated_at":      now,
			})
		if result.Error != nil {
			return nil, result.Error
		}
		if result.RowsAffected == 1 {
			claimedIDs = append(claimedIDs, candidate.ID)
		}
	}
	if len(claimedIDs) == 0 {
		return nil, nil
	}

	var jobs []model.NotificationOutbox
	if err := s.DB.Where("id IN ? AND locked_by = ? AND status = ?", claimedIDs, ownerID, notificationOutboxStatusProcessing).
		Order("id ASC").
		Find(&jobs).Error; err != nil {
		return nil, err
	}
	return jobs, nil
}

// logOutboxPersistError records a failure to persist a notification outbox
// state transition. These errors are non-fatal: the job is left in place for a
// later retry. They are emitted at error level (with the job id and the
// attempted action) so operators can spot stuck jobs.
func logOutboxPersistError(job model.NotificationOutbox, action string, err error) {
	logging.Error("failed to persist notification outbox state",
		slog.Uint64("job_id", uint64(job.ID)),
		slog.String("action", action),
		slog.Any("error", err),
	)
}

func (s *NotificationService) dispatchNotificationOutboxJob(job model.NotificationOutbox) string {
	if status := s.cancelOutboxIfNoLongerDeliverable(job); status != "" {
		return status
	}

	channel, status := s.loadOutboxChannel(job)
	if status != "" {
		return status
	}

	sendErr := s.dispatchNotificationChannel(*channel, job.TargetEmail, job.Message, job.SubscriptionURL)
	if sendErr != nil {
		if err := s.markNotificationOutboxFailed(job, sendErr); err != nil {
			logOutboxPersistError(job, "mark_failed", err)
		}
		if job.AttemptCount >= effectiveNotificationOutboxMaxAttempts(job) {
			return notificationOutboxStatusFailed
		}
		return notificationOutboxStatusPending
	}

	if err := s.markNotificationOutboxSent(job); err != nil {
		logOutboxPersistError(job, "mark_sent", err)
		return notificationOutboxStatusProcessing
	}
	return notificationOutboxStatusSent
}

func (s *NotificationService) cancelOutboxIfNoLongerDeliverable(job model.NotificationOutbox) string {
	now := pkg.NowUTC()
	if job.ExpiresAt != nil && !job.ExpiresAt.After(now) {
		if err := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusExpired, ""); err != nil {
			logOutboxPersistError(job, "expire", err)
		}
		return notificationOutboxStatusExpired
	}

	var sub model.Subscription
	err := s.DB.Select("id", "user_id", "status", "billing_type", "renewal_mode", "ends_at", "next_billing_date", "notify_enabled", "notify_days_before").
		Where("id = ? AND user_id = ?", job.SubscriptionID, job.UserID).
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if updateErr := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, "subscription not found"); updateErr != nil {
			logOutboxPersistError(job, "cancel_subscription_missing", updateErr)
		}
		return notificationOutboxStatusCancelled
	}
	if err != nil {
		if updateErr := s.releaseNotificationOutboxForRetry(job, err); updateErr != nil {
			logOutboxPersistError(job, "release_subscription_lookup", updateErr)
		}
		return notificationOutboxStatusPending
	}

	notifyEnabled := true
	if sub.NotifyEnabled != nil {
		notifyEnabled = *sub.NotifyEnabled
	}
	if job.TriggerType == notificationTriggerManualEnded {
		if normalizeStatus(sub.Status) != subscriptionStatusEnded ||
			normalizeRenewalMode(sub.RenewalMode) != renewalModeManualRenew ||
			sub.BillingType != billingTypeRecurring ||
			sub.EndsAt == nil ||
			!notifyEnabled ||
			!normalizeDateUTC(job.NotifyDate).Equal(normalizeDateUTC(*sub.EndsAt)) {
			if err := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, "manual-renew end notification no longer deliverable"); err != nil {
				logOutboxPersistError(job, "cancel_manual_end_not_deliverable", err)
			}
			return notificationOutboxStatusCancelled
		}
		return ""
	}

	if job.TriggerType == notificationTriggerEndingSoon {
		if normalizeStatus(sub.Status) != subscriptionStatusActive ||
			normalizeRenewalMode(sub.RenewalMode) != renewalModeCancelAtPeriodEnd ||
			sub.BillingType != billingTypeRecurring ||
			!notifyEnabled ||
			cancelAtPeriodEndBoundary(sub) == nil {
			if err := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, "ending notification no longer deliverable"); err != nil {
				logOutboxPersistError(job, "cancel_ending_not_deliverable", err)
			}
			return notificationOutboxStatusCancelled
		}

		matches, reason, err := s.outboxMatchesCurrentEndingSoonReminder(job, sub)
		if err != nil {
			if updateErr := s.releaseNotificationOutboxForRetry(job, err); updateErr != nil {
				logOutboxPersistError(job, "release_ending_validation", updateErr)
			}
			return notificationOutboxStatusPending
		}
		if !matches {
			if err := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, reason); err != nil {
				logOutboxPersistError(job, "cancel_stale_ending", err)
			}
			return notificationOutboxStatusCancelled
		}
		return ""
	}

	if normalizeStatus(sub.Status) != subscriptionStatusActive ||
		sub.BillingType != billingTypeRecurring ||
		sub.NextBillingDate == nil ||
		!notifyEnabled ||
		!subscriptionHasFutureCharge(sub) {
		if err := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, "notification no longer deliverable"); err != nil {
			logOutboxPersistError(job, "cancel_not_deliverable", err)
		}
		return notificationOutboxStatusCancelled
	}

	matches, reason, err := s.outboxMatchesCurrentReminder(job, sub)
	if err != nil {
		if updateErr := s.releaseNotificationOutboxForRetry(job, err); updateErr != nil {
			logOutboxPersistError(job, "release_reminder_validation", updateErr)
		}
		return notificationOutboxStatusPending
	}
	if !matches {
		if err := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, reason); err != nil {
			logOutboxPersistError(job, "cancel_stale_reminder", err)
		}
		return notificationOutboxStatusCancelled
	}

	return ""
}

func (s *NotificationService) outboxMatchesCurrentReminder(job model.NotificationOutbox, sub model.Subscription) (bool, string, error) {
	policy, err := s.GetPolicy(job.UserID)
	if err != nil {
		return false, "", err
	}

	daysBefore := policy.DaysBefore
	notifyOnDueDay := policy.NotifyOnDueDay
	if sub.NotifyDaysBefore != nil {
		daysBefore = *sub.NotifyDaysBefore
	}

	systemLoc := pkg.GetSystemTimezone()
	billingDate := pkg.NormalizeDateInTimezone(*sub.NextBillingDate, systemLoc)
	if !normalizeDateUTC(job.NotifyDate).Equal(normalizeDateUTC(billingDate)) {
		return false, "queued reminder no longer matches billing date", nil
	}

	daysUntilBilling := pkg.DaysUntil(*sub.NextBillingDate, systemLoc)
	if job.TriggerType == notificationTriggerManualDaily {
		scheduledDate := pkg.NormalizeDateInTimezone(job.ScheduledFor, systemLoc)
		today := pkg.TodayInTimezone(systemLoc)
		if !scheduledDate.Equal(today) {
			return false, "queued daily manual-renew reminder is stale", nil
		}
	}

	for _, triggerType := range notificationTriggerTypesForSubscription(
		sub.RenewalMode,
		daysUntilBilling,
		daysBefore,
		notifyOnDueDay,
		policy.NotifyManualRenewDaily,
	) {
		if triggerType == job.TriggerType {
			return true, "", nil
		}
	}

	return false, "queued reminder no longer matches reminder timing", nil
}

func (s *NotificationService) outboxMatchesCurrentEndingSoonReminder(job model.NotificationOutbox, sub model.Subscription) (bool, string, error) {
	policy, err := s.GetPolicy(job.UserID)
	if err != nil {
		return false, "", err
	}

	daysBefore := policy.DaysBefore
	notifyOnDueDay := policy.NotifyOnDueDay
	if sub.NotifyDaysBefore != nil {
		daysBefore = *sub.NotifyDaysBefore
	}

	boundary := cancelAtPeriodEndBoundary(sub)
	if boundary == nil {
		return false, "queued ending reminder no longer has an ending date", nil
	}

	systemLoc := pkg.GetSystemTimezone()
	endDate := pkg.NormalizeDateInTimezone(*boundary, systemLoc)
	if !normalizeDateUTC(job.NotifyDate).Equal(normalizeDateUTC(endDate)) {
		return false, "queued ending reminder no longer matches ending date", nil
	}

	daysUntilEnd := pkg.DaysUntil(endDate, systemLoc)
	if len(notificationTriggerTypes(daysUntilEnd, daysBefore, notifyOnDueDay)) == 0 {
		return false, "queued ending reminder no longer matches reminder timing", nil
	}

	scheduledDate := pkg.NormalizeDateInTimezone(job.ScheduledFor, systemLoc)
	today := pkg.TodayInTimezone(systemLoc)
	if !scheduledDate.Equal(today) {
		return false, "queued ending reminder is stale", nil
	}

	return true, "", nil
}

func (s *NotificationService) loadOutboxChannel(job model.NotificationOutbox) (*model.NotificationChannel, string) {
	if job.ChannelID == nil {
		if updateErr := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, "notification channel not found or disabled"); updateErr != nil {
			logOutboxPersistError(job, "cancel_channel_unset", updateErr)
		}
		return nil, notificationOutboxStatusCancelled
	}

	var channel model.NotificationChannel
	err := s.DB.Where("id = ? AND user_id = ? AND type = ? AND enabled = ?", *job.ChannelID, job.UserID, job.ChannelType, true).
		First(&channel).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if updateErr := s.updateNotificationOutboxTerminal(job, notificationOutboxStatusCancelled, "notification channel not found or disabled"); updateErr != nil {
			logOutboxPersistError(job, "cancel_channel_missing", updateErr)
		}
		return nil, notificationOutboxStatusCancelled
	}
	if err != nil {
		if updateErr := s.releaseNotificationOutboxForRetry(job, err); updateErr != nil {
			logOutboxPersistError(job, "release_channel_lookup", updateErr)
		}
		return nil, notificationOutboxStatusPending
	}
	return &channel, ""
}

func (s *NotificationService) markNotificationOutboxSent(job model.NotificationOutbox) error {
	now := pkg.NowUTC()
	status := notificationOutboxStatusSent
	return s.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.NotificationOutbox{}).
			Where("id = ? AND locked_by = ? AND status = ?", job.ID, s.notificationOwnerID(), notificationOutboxStatusProcessing).
			Updates(map[string]interface{}{
				"status":       status,
				"sent_at":      now,
				"last_error":   "",
				"locked_by":    "",
				"locked_until": nil,
				"updated_at":   now,
			})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		outboxID := job.ID
		return tx.Create(&model.NotificationLog{
			OutboxID:       &outboxID,
			UserID:         job.UserID,
			SubscriptionID: job.SubscriptionID,
			ChannelType:    job.ChannelType,
			TriggerType:    job.TriggerType,
			NotifyDate:     job.NotifyDate,
			Status:         notificationLogStatusSent,
			SentAt:         now,
		}).Error
	})
}

func (s *NotificationService) markNotificationOutboxFailed(job model.NotificationOutbox, sendErr error) error {
	now := pkg.NowUTC()
	sanitizedErr := sanitizeNotificationError(sendErr.Error())
	maxAttempts := effectiveNotificationOutboxMaxAttempts(job)
	status := notificationOutboxStatusPending
	nextAttemptAt := now.Add(notificationOutboxBackoff(job.AttemptCount))
	if job.AttemptCount >= maxAttempts {
		status = notificationOutboxStatusFailed
	}

	return s.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{
			"status":       status,
			"last_error":   sanitizedErr,
			"locked_by":    "",
			"locked_until": nil,
			"updated_at":   now,
		}
		if status == notificationOutboxStatusPending {
			updates["next_attempt_at"] = nextAttemptAt
		}

		result := tx.Model(&model.NotificationOutbox{}).
			Where("id = ? AND locked_by = ? AND status = ?", job.ID, s.notificationOwnerID(), notificationOutboxStatusProcessing).
			Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}

		outboxID := job.ID
		return tx.Create(&model.NotificationLog{
			OutboxID:       &outboxID,
			UserID:         job.UserID,
			SubscriptionID: job.SubscriptionID,
			ChannelType:    job.ChannelType,
			TriggerType:    job.TriggerType,
			NotifyDate:     job.NotifyDate,
			Status:         notificationLogStatusFailed,
			Error:          sanitizedErr,
			SentAt:         now,
		}).Error
	})
}

func (s *NotificationService) updateNotificationOutboxTerminal(job model.NotificationOutbox, status, reason string) error {
	now := pkg.NowUTC()
	return s.DB.Model(&model.NotificationOutbox{}).
		Where("id = ? AND locked_by = ? AND status = ?", job.ID, s.notificationOwnerID(), notificationOutboxStatusProcessing).
		Updates(map[string]interface{}{
			"status":       status,
			"last_error":   reason,
			"locked_by":    "",
			"locked_until": nil,
			"updated_at":   now,
		}).Error
}

func (s *NotificationService) releaseNotificationOutboxForRetry(job model.NotificationOutbox, err error) error {
	now := pkg.NowUTC()
	return s.DB.Model(&model.NotificationOutbox{}).
		Where("id = ? AND locked_by = ? AND status = ?", job.ID, s.notificationOwnerID(), notificationOutboxStatusProcessing).
		Updates(map[string]interface{}{
			"status":          notificationOutboxStatusPending,
			"last_error":      sanitizeNotificationError(err.Error()),
			"next_attempt_at": now.Add(notificationOutboxBackoff(job.AttemptCount)),
			"locked_by":       "",
			"locked_until":    nil,
			"updated_at":      now,
		}).Error
}

func effectiveNotificationOutboxMaxAttempts(job model.NotificationOutbox) int {
	if job.MaxAttempts <= 0 {
		return notificationOutboxDefaultMaxAttempts
	}
	return job.MaxAttempts
}

func notificationOutboxBackoff(attemptCount int) time.Duration {
	switch {
	case attemptCount <= 1:
		return 15 * time.Minute
	case attemptCount == 2:
		return 30 * time.Minute
	case attemptCount == 3:
		return time.Hour
	default:
		return 3 * time.Hour
	}
}
