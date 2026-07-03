package service

import (
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"gorm.io/gorm"
)

type SubscriptionService = subscriptionservice.Service
type CurrencyConverter = subscriptionservice.CurrencyConverter
type CreateSubscriptionInput = subscriptionservice.CreateSubscriptionInput
type UpdateSubscriptionInput = subscriptionservice.UpdateSubscriptionInput
type billingDraft = subscriptionservice.BillingDraft
type lifecycleDraft = subscriptionservice.LifecycleDraft
type DashboardSummary = subscriptionservice.DashboardSummary
type ActionCenter = subscriptionservice.ActionCenter
type ActionCenterCounts = subscriptionservice.ActionCenterCounts
type SubscriptionAction = subscriptionservice.SubscriptionAction
type SnoozeSubscriptionActionInput = subscriptionservice.SnoozeSubscriptionActionInput
type SubscriptionDetail = subscriptionservice.SubscriptionDetail
type SubscriptionDetailEvent = subscriptionservice.SubscriptionDetailEvent
type SubscriptionDetailPriceHistoryItem = subscriptionservice.SubscriptionDetailPriceHistoryItem
type SubscriptionDetailNotificationLog = subscriptionservice.SubscriptionDetailNotificationLog
type SubscriptionDetailUpcomingCharge = subscriptionservice.SubscriptionDetailUpcomingCharge
type SubscriptionDetailCalendar = subscriptionservice.SubscriptionDetailCalendar
type AnalyticsReport = subscriptionservice.AnalyticsReport
type AnalyticsReportKPIs = subscriptionservice.AnalyticsReportKPIs
type MonthlyForecastItem = subscriptionservice.MonthlyForecastItem
type ReportBreakdownItem = subscriptionservice.ReportBreakdownItem
type ReportSubscriptionSpend = subscriptionservice.ReportSubscriptionSpend
type ReportUpcomingRenewal = subscriptionservice.ReportUpcomingRenewal
type ReportPriceIncrease = subscriptionservice.ReportPriceIncrease
type ReportSubscriptionEvent = subscriptionservice.ReportSubscriptionEvent
type ReportAnnualGrowthItem = subscriptionservice.ReportAnnualGrowthItem

var ErrInvalidSubscriptionURL = subscriptionservice.ErrInvalidSubscriptionURL

func NewSubscriptionService(db *gorm.DB) *SubscriptionService {
	return subscriptionservice.NewService(db)
}

const (
	billingTypeRecurring = subscriptionservice.BillingTypeRecurring

	subscriptionStatusActive = subscriptionservice.StatusActive
	subscriptionStatusEnded  = subscriptionservice.StatusEnded

	renewalModeAutoRenew         = subscriptionservice.RenewalModeAutoRenew
	renewalModeManualRenew       = subscriptionservice.RenewalModeManualRenew
	renewalModeCancelAtPeriodEnd = subscriptionservice.RenewalModeCancelAtPeriodEnd

	recurrenceTypeInterval    = subscriptionservice.RecurrenceTypeInterval
	recurrenceTypeMonthlyDate = subscriptionservice.RecurrenceTypeMonthlyDate
	recurrenceTypeYearlyDate  = subscriptionservice.RecurrenceTypeYearlyDate

	intervalUnitDay   = subscriptionservice.IntervalUnitDay
	intervalUnitWeek  = subscriptionservice.IntervalUnitWeek
	intervalUnitMonth = subscriptionservice.IntervalUnitMonth
	intervalUnitYear  = subscriptionservice.IntervalUnitYear

	maxNotificationDaysBefore = subscriptionservice.MaxNotificationDaysBefore
)

func normalizeStatus(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeRenewalMode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidSubscriptionStatus(value string) bool {
	switch value {
	case subscriptionStatusActive, subscriptionStatusEnded:
		return true
	default:
		return false
	}
}

func isValidRenewalMode(value string) bool {
	switch value {
	case renewalModeAutoRenew, renewalModeManualRenew, renewalModeCancelAtPeriodEnd:
		return true
	default:
		return false
	}
}

func normalizeDateUTC(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func presentActiveSubscriptions(subs []model.Subscription, now time.Time) []model.Subscription {
	return subscriptionservice.PresentActiveSubscriptions(subs, now)
}

func isRecurringScheduleValid(sub model.Subscription) bool {
	return subscriptionservice.IsRecurringScheduleValid(sub)
}

func managedIconFilePath(icon string) (string, bool) {
	return subscriptionservice.ManagedIconFilePath(icon)
}

func normalizeBillingDraft(draft billingDraft) (billingDraft, *time.Time, error) {
	return subscriptionservice.NormalizeBillingDraft(draft)
}

func normalizeLifecycleDraft(draft lifecycleDraft, nextBillingDate *time.Time, now time.Time) (lifecycleDraft, error) {
	return subscriptionservice.NormalizeLifecycleDraft(draft, nextBillingDate, now)
}

func deriveLegacyLifecycle(enabled bool, nextBillingDate, endsAt *time.Time, updatedAt time.Time) lifecycleDraft {
	return subscriptionservice.DeriveLegacyLifecycle(enabled, nextBillingDate, endsAt, updatedAt)
}

func syncLegacyEnabledForLifecycle(sub *model.Subscription) {
	subscriptionservice.SyncLegacyEnabledForLifecycle(sub)
}

func normalizeSubscriptionURL(raw string) (string, error) {
	return subscriptionservice.NormalizeSubscriptionURL(raw)
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func copyTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := normalizeDateUTC(*value)
	return &copied
}
