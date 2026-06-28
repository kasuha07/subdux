package service

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	actionTypeUpcomingRenewal    = "upcoming_renewal"
	actionTypeManualRenewalDue   = "manual_renewal_due"
	actionTypeEndingSoon         = "ending_soon"
	actionTypeNotificationFailed = "notification_failed"
	actionTypeMissingNextBilling = "missing_next_billing"
	actionTypePriceIncrease      = "price_increase"
	actionSeverityCritical       = "critical"
	actionSeverityHigh           = "high"
	actionSeverityMedium         = "medium"
	actionSeverityLow            = "low"
	actionSnoozeDefaultDays      = 7
	actionCenterUpcomingDays     = 30
	actionCenterUrgentDays       = 7
	actionCenterRecentChangeDays = 30
	actionCenterFailedLogDays    = 30
	actionCenterMaxItems         = 100
	notificationLogStatusFailed  = "failed"
	notificationLogStatusSent    = "sent"
)

type ActionCenter struct {
	GeneratedAt    time.Time            `json:"generated_at"`
	WindowDays     int                  `json:"window_days"`
	UrgentDays     int                  `json:"urgent_days"`
	Items          []SubscriptionAction `json:"items"`
	Counts         ActionCenterCounts   `json:"counts"`
	AvailableTypes []string             `json:"available_types"`
}

type ActionCenterCounts struct {
	Total          int `json:"total"`
	Critical       int `json:"critical"`
	High           int `json:"high"`
	Medium         int `json:"medium"`
	Low            int `json:"low"`
	NeedsDecision  int `json:"needs_decision"`
	NeedsRepair    int `json:"needs_repair"`
	UpcomingCharge int `json:"upcoming_charge"`
	Snoozed        int `json:"snoozed"`
}

type SubscriptionAction struct {
	Key                   string     `json:"key"`
	Type                  string     `json:"type"`
	Severity              string     `json:"severity"`
	NeedsDecision         bool       `json:"needs_decision"`
	NeedsRepair           bool       `json:"needs_repair"`
	UpcomingCharge        bool       `json:"upcoming_charge"`
	SubscriptionID        uint       `json:"subscription_id"`
	SubscriptionName      string     `json:"subscription_name"`
	SubscriptionIcon      string     `json:"subscription_icon"`
	Amount                float64    `json:"amount"`
	Currency              string     `json:"currency"`
	RenewalMode           string     `json:"renewal_mode"`
	Status                string     `json:"status"`
	DueDate               *string    `json:"due_date"`
	DaysUntil             *int       `json:"days_until"`
	EventDate             *string    `json:"event_date"`
	Message               string     `json:"message"`
	Detail                string     `json:"detail"`
	PreviousMonthlyAmount *float64   `json:"previous_monthly_amount"`
	NewMonthlyAmount      *float64   `json:"new_monthly_amount"`
	DeltaMonthlyAmount    *float64   `json:"delta_monthly_amount"`
	DeltaPercentage       *float64   `json:"delta_percentage"`
	NotificationChannel   string     `json:"notification_channel"`
	NotificationError     string     `json:"notification_error"`
	AllowedActions        []string   `json:"allowed_actions"`
	SnoozedUntil          *time.Time `json:"snoozed_until"`
}

type SnoozeSubscriptionActionInput struct {
	Key       string `json:"key"`
	Days      int    `json:"days"`
	UntilDate string `json:"until_date"`
}

func (s *SubscriptionService) GetActionCenter(userID uint) (*ActionCenter, error) {
	now := pkg.NowInSystemTimezone()
	if err := reconcileSubscriptionLifecycleForUser(s.DB, userID, now); err != nil {
		return nil, err
	}

	today := normalizeDateUTC(now)
	windowEnd := today.AddDate(0, 0, actionCenterUpcomingDays)

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ?", userID).Find(&subs).Error; err != nil {
		return nil, err
	}
	for i := range subs {
		normalizeSubscriptionForResponse(&subs[i])
	}

	snoozes, err := s.activeActionSnoozes(userID, today)
	if err != nil {
		return nil, err
	}

	items := make([]SubscriptionAction, 0, len(subs))
	for _, sub := range subs {
		items = append(items, subscriptionScheduleActions(sub, today, windowEnd)...)
	}

	notificationItems, err := s.notificationFailureActions(userID, today)
	if err != nil {
		return nil, err
	}
	items = append(items, notificationItems...)

	priceItems, err := s.priceIncreaseActions(userID, today)
	if err != nil {
		return nil, err
	}
	items = append(items, priceItems...)

	visible := make([]SubscriptionAction, 0, len(items))
	snoozedCount := 0
	for _, item := range items {
		snooze, ok := snoozes[item.Key]
		if ok && normalizeDateUTC(snooze.SnoozedUntil).After(today) {
			snoozedCount++
			continue
		}
		visible = append(visible, item)
	}

	sort.Slice(visible, func(i, j int) bool {
		leftPriority := actionSeverityRank(visible[i].Severity)
		rightPriority := actionSeverityRank(visible[j].Severity)
		if leftPriority != rightPriority {
			return leftPriority < rightPriority
		}
		leftDue := actionSortDate(visible[i])
		rightDue := actionSortDate(visible[j])
		if !leftDue.Equal(rightDue) {
			return leftDue.Before(rightDue)
		}
		if visible[i].SubscriptionName != visible[j].SubscriptionName {
			return visible[i].SubscriptionName < visible[j].SubscriptionName
		}
		return visible[i].Key < visible[j].Key
	})
	if len(visible) > actionCenterMaxItems {
		visible = visible[:actionCenterMaxItems]
	}

	return &ActionCenter{
		GeneratedAt:    now,
		WindowDays:     actionCenterUpcomingDays,
		UrgentDays:     actionCenterUrgentDays,
		Items:          visible,
		Counts:         buildActionCenterCounts(visible, snoozedCount),
		AvailableTypes: []string{actionTypeManualRenewalDue, actionTypeNotificationFailed, actionTypeMissingNextBilling, actionTypePriceIncrease, actionTypeEndingSoon, actionTypeUpcomingRenewal},
	}, nil
}

func (s *SubscriptionService) SnoozeAction(userID uint, input SnoozeSubscriptionActionInput) (*model.SubscriptionActionSnooze, error) {
	actionKey := strings.TrimSpace(input.Key)
	subscriptionID, err := parseActionSubscriptionID(actionKey)
	if err != nil {
		return nil, err
	}

	var sub model.Subscription
	if err := s.DB.Where("id = ? AND user_id = ?", subscriptionID, userID).First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("subscription not found")
		}
		return nil, err
	}

	snoozedUntil, err := resolveActionSnoozeUntil(input)
	if err != nil {
		return nil, err
	}

	snooze := model.SubscriptionActionSnooze{
		UserID:         userID,
		SubscriptionID: subscriptionID,
		ActionKey:      actionKey,
		SnoozedUntil:   snoozedUntil,
	}
	if err := s.DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "user_id"},
			{Name: "subscription_id"},
			{Name: "action_key"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"snoozed_until", "updated_at"}),
	}).Create(&snooze).Error; err != nil {
		return nil, err
	}

	var saved model.SubscriptionActionSnooze
	if err := s.DB.Where("user_id = ? AND subscription_id = ? AND action_key = ?", userID, subscriptionID, actionKey).
		First(&saved).Error; err != nil {
		return nil, err
	}
	return &saved, nil
}

func (s *SubscriptionService) activeActionSnoozes(userID uint, today time.Time) (map[string]model.SubscriptionActionSnooze, error) {
	var snoozes []model.SubscriptionActionSnooze
	if err := s.DB.Where("user_id = ? AND snoozed_until > ?", userID, today).Find(&snoozes).Error; err != nil {
		return nil, err
	}
	result := make(map[string]model.SubscriptionActionSnooze, len(snoozes))
	for _, snooze := range snoozes {
		result[snooze.ActionKey] = snooze
	}
	return result, nil
}

func subscriptionScheduleActions(sub model.Subscription, today, windowEnd time.Time) []SubscriptionAction {
	if normalizeStatus(sub.Status) != subscriptionStatusActive {
		return nil
	}

	renewalMode := normalizeRenewalMode(sub.RenewalMode)
	if sub.NextBillingDate == nil {
		return []SubscriptionAction{{
			Key:              subscriptionActionKey(sub.ID, actionTypeMissingNextBilling, ""),
			Type:             actionTypeMissingNextBilling,
			Severity:         actionSeverityHigh,
			NeedsRepair:      true,
			SubscriptionID:   sub.ID,
			SubscriptionName: sub.Name,
			SubscriptionIcon: sub.Icon,
			Amount:           sub.Amount,
			Currency:         strings.ToUpper(strings.TrimSpace(sub.Currency)),
			RenewalMode:      renewalMode,
			Status:           normalizeStatus(sub.Status),
			Message:          "missing next billing date",
			Detail:           "set a next billing date so reminders, reports, and calendar entries stay accurate",
			AllowedActions:   []string{"edit", "open_detail", "snooze"},
		}}
	}

	dueDate := normalizeDateUTC(*sub.NextBillingDate)
	daysUntil := int(dueDate.Sub(today).Hours() / 24)
	if dueDate.Before(today) || dueDate.After(windowEnd) {
		if renewalMode != renewalModeCancelAtPeriodEnd {
			return nil
		}
	}

	switch renewalMode {
	case renewalModeManualRenew:
		if dueDate.After(windowEnd) {
			return nil
		}
		severity := actionSeverityMedium
		if daysUntil <= actionCenterUrgentDays {
			severity = actionSeverityHigh
		}
		if daysUntil <= 0 {
			severity = actionSeverityCritical
		}
		return []SubscriptionAction{newScheduleAction(
			sub,
			actionTypeManualRenewalDue,
			severity,
			true,
			true,
			dueDate,
			daysUntil,
			"manual renewal needs confirmation",
			"confirm the renewal after payment or edit the next billing date",
			[]string{"mark_renewed", "cancel_at_period_end", "edit", "open_detail", "snooze"},
		)}
	case renewalModeCancelAtPeriodEnd:
		endDate := dueDate
		if sub.EndsAt != nil {
			endDate = normalizeDateUTC(*sub.EndsAt)
		}
		endDaysUntil := int(endDate.Sub(today).Hours() / 24)
		if endDate.Before(today) || endDate.After(windowEnd) {
			return nil
		}
		severity := actionSeverityLow
		if endDaysUntil <= actionCenterUrgentDays {
			severity = actionSeverityMedium
		}
		if endDaysUntil <= 0 {
			severity = actionSeverityHigh
		}
		return []SubscriptionAction{newScheduleAction(
			sub,
			actionTypeEndingSoon,
			severity,
			true,
			false,
			endDate,
			endDaysUntil,
			"subscription is ending soon",
			"review whether to keep it canceled or reactivate it",
			[]string{"edit", "open_detail", "snooze"},
		)}
	default:
		if dueDate.After(windowEnd) {
			return nil
		}
		severity := actionSeverityLow
		if daysUntil <= actionCenterUrgentDays {
			severity = actionSeverityMedium
		}
		if daysUntil <= 0 {
			severity = actionSeverityHigh
		}
		return []SubscriptionAction{newScheduleAction(
			sub,
			actionTypeUpcomingRenewal,
			severity,
			false,
			true,
			dueDate,
			daysUntil,
			"upcoming renewal",
			"review the subscription before the next charge",
			[]string{"cancel_at_period_end", "edit", "open_detail", "snooze"},
		)}
	}
}

func newScheduleAction(
	sub model.Subscription,
	actionType string,
	severity string,
	needsDecision bool,
	upcomingCharge bool,
	dueDate time.Time,
	daysUntil int,
	message string,
	detail string,
	allowedActions []string,
) SubscriptionAction {
	date := dueDate.Format("2006-01-02")
	return SubscriptionAction{
		Key:              subscriptionActionKey(sub.ID, actionType, date),
		Type:             actionType,
		Severity:         severity,
		NeedsDecision:    needsDecision,
		UpcomingCharge:   upcomingCharge,
		SubscriptionID:   sub.ID,
		SubscriptionName: sub.Name,
		SubscriptionIcon: sub.Icon,
		Amount:           sub.Amount,
		Currency:         strings.ToUpper(strings.TrimSpace(sub.Currency)),
		RenewalMode:      normalizeRenewalMode(sub.RenewalMode),
		Status:           normalizeStatus(sub.Status),
		DueDate:          &date,
		DaysUntil:        &daysUntil,
		Message:          message,
		Detail:           detail,
		AllowedActions:   allowedActions,
	}
}

func (s *SubscriptionService) notificationFailureActions(userID uint, today time.Time) ([]SubscriptionAction, error) {
	since := today.AddDate(0, 0, -actionCenterFailedLogDays)
	var logs []model.NotificationLog
	if err := s.DB.Where("user_id = ? AND status = ? AND sent_at >= ?", userID, notificationLogStatusFailed, since).
		Order("sent_at DESC, id DESC").
		Limit(100).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	recovered, err := s.notificationRecoveryIndex(userID, since)
	if err != nil {
		return nil, err
	}

	// Process logs newest-first and keep only the most recent failure per
	// (subscription, channel): once recovered, every older failure for that key
	// is necessarily recovered too, so a single check at the most recent entry
	// decides the whole key.
	type failedCandidate struct {
		log model.NotificationLog
		key string
	}
	seen := map[string]struct{}{}
	candidates := make([]failedCandidate, 0, len(logs))
	ids := make([]uint, 0, len(logs))
	for _, logEntry := range logs {
		key := subscriptionActionKey(logEntry.SubscriptionID, actionTypeNotificationFailed, logEntry.ChannelType)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if recovered(logEntry) {
			continue
		}
		candidates = append(candidates, failedCandidate{log: logEntry, key: key})
		ids = append(ids, logEntry.SubscriptionID)
	}

	subsByID, err := s.loadSubscriptionsByIDs(userID, ids)
	if err != nil {
		return nil, err
	}

	items := make([]SubscriptionAction, 0, len(candidates))
	for _, candidate := range candidates {
		logEntry := candidate.log
		sub, ok := subsByID[logEntry.SubscriptionID]
		if !ok {
			continue
		}
		if normalizeStatus(sub.Status) != subscriptionStatusActive {
			continue
		}

		eventDate := logEntry.SentAt.UTC().Format(time.RFC3339)
		notifyDate := normalizeDateUTC(logEntry.NotifyDate).Format("2006-01-02")
		items = append(items, SubscriptionAction{
			Key:                 candidate.key,
			Type:                actionTypeNotificationFailed,
			Severity:            actionSeverityHigh,
			NeedsRepair:         true,
			SubscriptionID:      sub.ID,
			SubscriptionName:    sub.Name,
			SubscriptionIcon:    sub.Icon,
			Amount:              sub.Amount,
			Currency:            strings.ToUpper(strings.TrimSpace(sub.Currency)),
			RenewalMode:         normalizeRenewalMode(sub.RenewalMode),
			Status:              normalizeStatus(sub.Status),
			EventDate:           &eventDate,
			Message:             "notification delivery failed",
			Detail:              "failed reminder for " + notifyDate,
			NotificationChannel: logEntry.ChannelType,
			NotificationError:   logEntry.Error,
			AllowedActions:      []string{"edit", "open_detail", "snooze"},
		})
	}
	return items, nil
}

// notificationRecoveryIndex returns a predicate that reports whether a failed
// log has since been superseded by a successful delivery on the same
// (subscription, channel). It collapses the former per-failure lookup into a
// single query: a failure is recovered exactly when the latest successful
// delivery for its key occurred after it. Sent logs older than `since` can
// never recover an in-window failure (whose own sent_at is >= since), so the
// same lower bound bounds this scan and is covered by
// idx_notification_logs_user_status_sent.
func (s *SubscriptionService) notificationRecoveryIndex(userID uint, since time.Time) (func(model.NotificationLog) bool, error) {
	var sentLogs []model.NotificationLog
	if err := s.DB.
		Select("subscription_id", "channel_type", "sent_at").
		Where("user_id = ? AND status = ? AND sent_at >= ?", userID, notificationLogStatusSent, since).
		Find(&sentLogs).Error; err != nil {
		return nil, err
	}

	latestSent := make(map[string]time.Time, len(sentLogs))
	for _, sent := range sentLogs {
		key := notificationRecoveryKey(sent.SubscriptionID, sent.ChannelType)
		if current, ok := latestSent[key]; !ok || sent.SentAt.After(current) {
			latestSent[key] = sent.SentAt
		}
	}

	return func(failed model.NotificationLog) bool {
		latest, ok := latestSent[notificationRecoveryKey(failed.SubscriptionID, failed.ChannelType)]
		return ok && latest.After(failed.SentAt)
	}, nil
}

func notificationRecoveryKey(subscriptionID uint, channelType string) string {
	return fmt.Sprintf("%d:%s", subscriptionID, channelType)
}

func (s *SubscriptionService) priceIncreaseActions(userID uint, today time.Time) ([]SubscriptionAction, error) {
	since := today.AddDate(0, 0, -actionCenterRecentChangeDays)
	var events []model.SubscriptionEvent
	if err := s.DB.Where(
		"user_id = ? AND previous_monthly_amount IS NOT NULL AND new_monthly_amount IS NOT NULL AND created_at >= ?",
		userID,
		since,
	).Order("created_at DESC, id DESC").Limit(100).Find(&events).Error; err != nil {
		return nil, err
	}

	items := make([]SubscriptionAction, 0, len(events))
	seen := map[uint]struct{}{}
	type priceChange struct {
		event model.SubscriptionEvent
		delta float64
	}
	// Keep only the most recent non-zero price change per subscription (events
	// arrive newest-first), mirroring the original first-seen-wins semantics: a
	// most-recent decrease suppresses the subscription entirely.
	candidates := make([]priceChange, 0, len(events))
	ids := make([]uint, 0, len(events))
	for _, event := range events {
		if event.SubscriptionID == nil || event.PreviousMonthlyAmount == nil || event.NewMonthlyAmount == nil {
			continue
		}
		delta := *event.NewMonthlyAmount - *event.PreviousMonthlyAmount
		if delta == 0 {
			continue
		}
		subscriptionID := *event.SubscriptionID
		if _, ok := seen[subscriptionID]; ok {
			continue
		}
		seen[subscriptionID] = struct{}{}
		if delta < 0 {
			continue
		}
		candidates = append(candidates, priceChange{event: event, delta: delta})
		ids = append(ids, subscriptionID)
	}

	subsByID, err := s.loadSubscriptionsByIDs(userID, ids)
	if err != nil {
		return nil, err
	}

	for _, candidate := range candidates {
		event := candidate.event
		sub, ok := subsByID[*event.SubscriptionID]
		if !ok {
			continue
		}
		if normalizeStatus(sub.Status) != subscriptionStatusActive {
			continue
		}

		eventDate := event.CreatedAt.UTC().Format(time.RFC3339)
		previous := *event.PreviousMonthlyAmount
		current := *event.NewMonthlyAmount
		deltaCopy := candidate.delta
		percentage := percentageDelta(previous, current)
		items = append(items, SubscriptionAction{
			Key:                   subscriptionActionKey(sub.ID, actionTypePriceIncrease, event.CreatedAt.Format("2006-01-02")),
			Type:                  actionTypePriceIncrease,
			Severity:              actionSeverityMedium,
			NeedsDecision:         true,
			SubscriptionID:        sub.ID,
			SubscriptionName:      sub.Name,
			SubscriptionIcon:      sub.Icon,
			Amount:                sub.Amount,
			Currency:              strings.ToUpper(strings.TrimSpace(sub.Currency)),
			RenewalMode:           normalizeRenewalMode(sub.RenewalMode),
			Status:                normalizeStatus(sub.Status),
			EventDate:             &eventDate,
			Message:               "price increased",
			Detail:                "review whether the new price is still worth keeping",
			PreviousMonthlyAmount: &previous,
			NewMonthlyAmount:      &current,
			DeltaMonthlyAmount:    &deltaCopy,
			DeltaPercentage:       &percentage,
			AllowedActions:        []string{"cancel_at_period_end", "edit", "open_detail", "snooze"},
		})
	}
	return items, nil
}

func subscriptionActionKey(subscriptionID uint, actionType, qualifier string) string {
	parts := []string{fmt.Sprintf("subscription:%d", subscriptionID), actionType}
	if strings.TrimSpace(qualifier) != "" {
		parts = append(parts, strings.TrimSpace(qualifier))
	}
	return strings.Join(parts, ":")
}

func parseActionSubscriptionID(key string) (uint, error) {
	parts := strings.Split(strings.TrimSpace(key), ":")
	if len(parts) < 3 || parts[0] != "subscription" {
		return 0, errors.New("invalid action key")
	}
	var id uint
	if _, err := fmt.Sscanf(parts[1], "%d", &id); err != nil || id == 0 {
		return 0, errors.New("invalid action key")
	}
	return id, nil
}

func resolveActionSnoozeUntil(input SnoozeSubscriptionActionInput) (time.Time, error) {
	if strings.TrimSpace(input.UntilDate) != "" {
		parsed, err := parseOptionalDateString(input.UntilDate)
		if err != nil {
			return time.Time{}, err
		}
		if parsed == nil {
			return time.Time{}, errors.New("snooze date is required")
		}
		snoozedUntil := normalizeDateUTC(*parsed)
		if !snoozedUntil.After(normalizeDateUTC(pkg.NowInSystemTimezone())) {
			return time.Time{}, errors.New("snooze date must be in the future")
		}
		return snoozedUntil, nil
	}

	days := input.Days
	if days <= 0 {
		days = actionSnoozeDefaultDays
	}
	if days > actionCenterUpcomingDays {
		days = actionCenterUpcomingDays
	}
	return normalizeDateUTC(pkg.NowInSystemTimezone()).AddDate(0, 0, days), nil
}

func actionSeverityRank(severity string) int {
	switch severity {
	case actionSeverityCritical:
		return 0
	case actionSeverityHigh:
		return 1
	case actionSeverityMedium:
		return 2
	case actionSeverityLow:
		return 3
	default:
		return 4
	}
}

func actionSortDate(item SubscriptionAction) time.Time {
	if item.DueDate != nil {
		if parsed, err := time.Parse("2006-01-02", *item.DueDate); err == nil {
			return parsed
		}
	}
	if item.EventDate != nil {
		if parsed, err := time.Parse(time.RFC3339, *item.EventDate); err == nil {
			return parsed
		}
	}
	return time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
}

func buildActionCenterCounts(items []SubscriptionAction, snoozedCount int) ActionCenterCounts {
	counts := ActionCenterCounts{
		Total:   len(items),
		Snoozed: snoozedCount,
	}
	for _, item := range items {
		switch item.Severity {
		case actionSeverityCritical:
			counts.Critical++
		case actionSeverityHigh:
			counts.High++
		case actionSeverityMedium:
			counts.Medium++
		case actionSeverityLow:
			counts.Low++
		}
		if item.NeedsDecision {
			counts.NeedsDecision++
		}
		if item.NeedsRepair {
			counts.NeedsRepair++
		}
		if item.UpcomingCharge {
			counts.UpcomingCharge++
		}
	}
	return counts
}
