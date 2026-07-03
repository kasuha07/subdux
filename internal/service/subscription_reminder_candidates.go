package service

import (
	"context"
	"errors"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

type subscriptionReminderPolicy struct {
	DaysBefore             int
	NotifyOnDueDay         bool
	NotifyManualRenewDaily bool
}

type subscriptionReminderCandidate struct {
	SubscriptionID uint
	UserID         uint
	NotifyDate     time.Time
	DedupeDate     time.Time
	DaysUntil      int
	TriggerType    string
	EventType      string
	Template       subscriptionReminderTemplateSnapshot
}

// subscriptionReminderTemplateSnapshot is the bounded subset of subscription
// fields exposed to notification templates. Keep scheduling/dedupe data on
// subscriptionReminderCandidate and add template variables here deliberately.
type subscriptionReminderTemplateSnapshot struct {
	Name            string
	Amount          float64
	Currency        string
	Status          string
	RenewalMode     string
	Category        string
	PaymentMethodID *uint
	URL             string
	Notes           string
}

func listSubscriptionReminderCandidates(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	now time.Time,
	policy subscriptionReminderPolicy,
) ([]subscriptionReminderCandidate, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	db = db.WithContext(ctx)

	if err := reconcileSubscriptionLifecycleForUser(db, userID, now); err != nil {
		return nil, err
	}

	var subs []model.Subscription
	if err := db.Where("user_id = ? AND status = ? AND billing_type = ? AND (next_billing_date IS NOT NULL OR ends_at IS NOT NULL)",
		userID, subscriptionStatusActive, billingTypeRecurring).Find(&subs).Error; err != nil {
		return nil, err
	}

	systemLoc := pkg.GetSystemTimezone()
	scanDate := pkg.NormalizeDateInTimezone(now, systemLoc)
	candidates := make([]subscriptionReminderCandidate, 0, len(subs))
	for _, sub := range subs {
		if !subscriptionNotificationEnabled(sub) {
			continue
		}

		subPolicy := subscriptionReminderPolicyForSubscription(sub, policy)
		candidates = append(candidates, subscriptionEndingReminderCandidates(sub, subPolicy, systemLoc, scanDate)...)
		candidates = append(candidates, subscriptionBillingReminderCandidates(sub, subPolicy, systemLoc, scanDate)...)
	}

	return candidates, nil
}

func listEndedManualRenewNotificationCandidates(
	ctx context.Context,
	db *gorm.DB,
	userID uint,
	referenceDate time.Time,
) ([]subscriptionReminderCandidate, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	today := normalizeDateUTC(referenceDate)
	var subs []model.Subscription
	if err := db.WithContext(ctx).Where(
		"user_id = ? AND status = ? AND renewal_mode = ? AND billing_type = ? AND ends_at IS NOT NULL AND ends_at < ?",
		userID,
		subscriptionStatusEnded,
		renewalModeManualRenew,
		billingTypeRecurring,
		today,
	).Order("ends_at ASC, id ASC").Find(&subs).Error; err != nil {
		return nil, err
	}

	systemLoc := pkg.GetSystemTimezone()
	candidates := make([]subscriptionReminderCandidate, 0, len(subs))
	for _, sub := range subs {
		if sub.EndsAt == nil || !subscriptionNotificationEnabled(sub) {
			continue
		}

		endedAt := pkg.NormalizeDateInTimezone(*sub.EndsAt, systemLoc)
		candidates = append(candidates, subscriptionReminderCandidateFromSubscription(
			sub,
			endedAt,
			endedAt,
			0,
			notificationTriggerManualEnded,
			"manual_renew_ended",
		))
	}
	return candidates, nil
}

func validateSubscriptionNotificationOutboxJob(
	ctx context.Context,
	db *gorm.DB,
	job model.NotificationOutbox,
	policy subscriptionReminderPolicy,
) (bool, string, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var sub model.Subscription
	err := db.WithContext(ctx).
		Select("id", "user_id", "name", "amount", "currency", "status", "billing_type", "renewal_mode", "ends_at", "next_billing_date", "category", "payment_method_id", "notify_enabled", "notify_days_before", "url", "notes").
		Where("id = ? AND user_id = ?", job.SubscriptionID, job.UserID).
		First(&sub).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, "subscription not found", nil
	}
	if err != nil {
		return false, "", err
	}

	if !subscriptionNotificationEnabled(sub) {
		return false, subscriptionOutboxNotDeliverableReason(job.TriggerType), nil
	}

	switch job.TriggerType {
	case notificationTriggerManualEnded:
		return subscriptionManualEndedOutboxStillDeliverable(job, sub), "manual-renew end notification no longer deliverable", nil
	case notificationTriggerEndingSoon:
		if normalizeStatus(sub.Status) != subscriptionStatusActive ||
			normalizeRenewalMode(sub.RenewalMode) != renewalModeCancelAtPeriodEnd ||
			sub.BillingType != billingTypeRecurring ||
			cancelAtPeriodEndBoundary(sub) == nil {
			return false, "ending notification no longer deliverable", nil
		}
		return subscriptionOutboxMatchesCurrentEndingReminder(job, sub, policy)
	default:
		if normalizeStatus(sub.Status) != subscriptionStatusActive ||
			sub.BillingType != billingTypeRecurring ||
			sub.NextBillingDate == nil ||
			!subscriptionHasFutureCharge(sub) {
			return false, "notification no longer deliverable", nil
		}
		return subscriptionOutboxMatchesCurrentBillingReminder(job, sub, policy)
	}
}

func subscriptionNotificationEnabled(sub model.Subscription) bool {
	if sub.NotifyEnabled == nil {
		return true
	}
	return *sub.NotifyEnabled
}

func subscriptionReminderPolicyForSubscription(sub model.Subscription, policy subscriptionReminderPolicy) subscriptionReminderPolicy {
	if sub.NotifyDaysBefore != nil {
		policy.DaysBefore = *sub.NotifyDaysBefore
	}
	return policy
}

func subscriptionEndingReminderCandidates(
	sub model.Subscription,
	policy subscriptionReminderPolicy,
	systemLoc *time.Location,
	scanDate time.Time,
) []subscriptionReminderCandidate {
	if normalizeRenewalMode(sub.RenewalMode) != renewalModeCancelAtPeriodEnd {
		return nil
	}

	boundary := cancelAtPeriodEndBoundary(sub)
	if boundary == nil {
		return nil
	}

	endDate := pkg.NormalizeDateInTimezone(*boundary, systemLoc)
	daysUntilEnd := pkg.DaysUntil(endDate, systemLoc)
	if len(subscriptionReminderTriggerTypes(daysUntilEnd, policy.DaysBefore, policy.NotifyOnDueDay)) == 0 {
		return nil
	}

	return []subscriptionReminderCandidate{
		subscriptionReminderCandidateFromSubscription(
			sub,
			endDate,
			scanDate,
			daysUntilEnd,
			notificationTriggerEndingSoon,
			"ending_soon",
		),
	}
}

func subscriptionBillingReminderCandidates(
	sub model.Subscription,
	policy subscriptionReminderPolicy,
	systemLoc *time.Location,
	scanDate time.Time,
) []subscriptionReminderCandidate {
	if sub.NextBillingDate == nil || !subscriptionHasFutureCharge(sub) {
		return nil
	}

	billingDate := pkg.NormalizeDateInTimezone(*sub.NextBillingDate, systemLoc)
	daysUntilBilling := pkg.DaysUntil(*sub.NextBillingDate, systemLoc)
	triggerTypes := subscriptionReminderTriggerTypesForSubscription(
		sub.RenewalMode,
		daysUntilBilling,
		policy.DaysBefore,
		policy.NotifyOnDueDay,
		policy.NotifyManualRenewDaily,
	)
	if len(triggerTypes) == 0 {
		return nil
	}

	candidates := make([]subscriptionReminderCandidate, 0, len(triggerTypes))
	for _, triggerType := range triggerTypes {
		dedupeDate := billingDate
		if triggerType == notificationTriggerManualDaily {
			dedupeDate = scanDate
		}

		candidates = append(candidates, subscriptionReminderCandidateFromSubscription(
			sub,
			billingDate,
			dedupeDate,
			daysUntilBilling,
			triggerType,
			subscriptionNotificationEventType(sub),
		))
	}
	return candidates
}

func subscriptionReminderCandidateFromSubscription(
	sub model.Subscription,
	notifyDate time.Time,
	dedupeDate time.Time,
	daysUntil int,
	triggerType string,
	eventType string,
) subscriptionReminderCandidate {
	return subscriptionReminderCandidate{
		SubscriptionID: sub.ID,
		UserID:         sub.UserID,
		NotifyDate:     notifyDate,
		DedupeDate:     dedupeDate,
		DaysUntil:      daysUntil,
		TriggerType:    triggerType,
		EventType:      eventType,
		Template: subscriptionReminderTemplateSnapshot{
			Name:            sub.Name,
			Amount:          sub.Amount,
			Currency:        sub.Currency,
			Status:          normalizeStatus(sub.Status),
			RenewalMode:     normalizeRenewalMode(sub.RenewalMode),
			Category:        sub.Category,
			PaymentMethodID: copyUintPointer(sub.PaymentMethodID),
			URL:             sub.URL,
			Notes:           sub.Notes,
		},
	}
}

func subscriptionReminderCandidateFromPreviewSubscription(
	sub model.Subscription,
	notifyDate time.Time,
	daysUntil int,
) subscriptionReminderCandidate {
	hasSubscriptionDate := false
	if normalizeRenewalMode(sub.RenewalMode) == renewalModeCancelAtPeriodEnd {
		if boundary := cancelAtPeriodEndBoundary(sub); boundary != nil {
			notifyDate = *boundary
			hasSubscriptionDate = true
		}
	} else if sub.NextBillingDate != nil {
		notifyDate = *sub.NextBillingDate
		hasSubscriptionDate = true
	}

	notifyDate = time.Date(
		notifyDate.Year(),
		notifyDate.Month(),
		notifyDate.Day(),
		0, 0, 0, 0,
		notifyDate.Location(),
	)
	now := pkg.Now().In(notifyDate.Location())
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, notifyDate.Location())
	if hasSubscriptionDate {
		daysUntil = int(notifyDate.Sub(today).Hours() / 24)
	}

	return subscriptionReminderCandidateFromSubscription(
		sub,
		notifyDate,
		notifyDate,
		daysUntil,
		"",
		subscriptionNotificationEventType(sub),
	)
}

func subscriptionNotificationEventType(sub model.Subscription) string {
	switch normalizeRenewalMode(sub.RenewalMode) {
	case renewalModeManualRenew:
		return "manual_renew_reminder"
	case renewalModeCancelAtPeriodEnd:
		return "ending_soon"
	default:
		return "auto_renew_reminder"
	}
}

func subscriptionReminderTriggerTypes(daysUntilBilling, daysBefore int, notifyOnDueDay bool) []string {
	triggers := make([]string, 0, 2)
	if daysUntilBilling == daysBefore && daysBefore > 0 {
		triggers = append(triggers, notificationTriggerDaysBefore)
	}
	if daysUntilBilling == 0 && notifyOnDueDay {
		triggers = append(triggers, notificationTriggerDueDay)
	}
	return triggers
}

func subscriptionReminderTriggerTypesForSubscription(
	renewalMode string,
	daysUntilBilling int,
	daysBefore int,
	notifyOnDueDay bool,
	notifyManualRenewDaily bool,
) []string {
	triggers := subscriptionReminderTriggerTypes(daysUntilBilling, daysBefore, notifyOnDueDay)
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

func subscriptionManualEndedOutboxStillDeliverable(job model.NotificationOutbox, sub model.Subscription) bool {
	return normalizeStatus(sub.Status) == subscriptionStatusEnded &&
		normalizeRenewalMode(sub.RenewalMode) == renewalModeManualRenew &&
		sub.BillingType == billingTypeRecurring &&
		sub.EndsAt != nil &&
		normalizeDateUTC(job.NotifyDate).Equal(normalizeDateUTC(*sub.EndsAt))
}

func subscriptionOutboxMatchesCurrentBillingReminder(
	job model.NotificationOutbox,
	sub model.Subscription,
	policy subscriptionReminderPolicy,
) (bool, string, error) {
	policy = subscriptionReminderPolicyForSubscription(sub, policy)

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

	for _, triggerType := range subscriptionReminderTriggerTypesForSubscription(
		sub.RenewalMode,
		daysUntilBilling,
		policy.DaysBefore,
		policy.NotifyOnDueDay,
		policy.NotifyManualRenewDaily,
	) {
		if triggerType == job.TriggerType {
			return true, "", nil
		}
	}

	return false, "queued reminder no longer matches reminder timing", nil
}

func subscriptionOutboxMatchesCurrentEndingReminder(
	job model.NotificationOutbox,
	sub model.Subscription,
	policy subscriptionReminderPolicy,
) (bool, string, error) {
	policy = subscriptionReminderPolicyForSubscription(sub, policy)

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
	if len(subscriptionReminderTriggerTypes(daysUntilEnd, policy.DaysBefore, policy.NotifyOnDueDay)) == 0 {
		return false, "queued ending reminder no longer matches reminder timing", nil
	}

	scheduledDate := pkg.NormalizeDateInTimezone(job.ScheduledFor, systemLoc)
	today := pkg.TodayInTimezone(systemLoc)
	if !scheduledDate.Equal(today) {
		return false, "queued ending reminder is stale", nil
	}

	return true, "", nil
}

func subscriptionOutboxNotDeliverableReason(triggerType string) string {
	switch triggerType {
	case notificationTriggerManualEnded:
		return "manual-renew end notification no longer deliverable"
	case notificationTriggerEndingSoon:
		return "ending notification no longer deliverable"
	default:
		return "notification no longer deliverable"
	}
}
