package notification

import (
	"context"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	"gorm.io/gorm"
)

type ReminderCandidateProvider interface {
	ListReminderCandidates(context.Context, subscriptionservice.ListReminderCandidatesInput) ([]subscriptionservice.ReminderCandidate, error)
	ListEndedManualRenewNotificationCandidates(context.Context, uint, time.Time) ([]subscriptionservice.ReminderCandidate, error)
	ValidateNotificationOutboxJob(context.Context, subscriptionservice.ValidateNotificationOutboxJobInput) (bool, string, error)
}

type Service struct {
	DB                 *gorm.DB
	templateService    *NotificationTemplateService
	templateRenderer   *TemplateRenderer
	reminderCandidates ReminderCandidateProvider
	ownerID            string
}

const (
	maxNotificationDaysBefore            = subscriptionservice.MaxNotificationDaysBefore
	maxEnabledNotificationChannels       = 3
	maxParallelUserNotificationChecks    = 4
	maxParallelNotificationDispatches    = 4
	notificationOutboxDefaultMaxAttempts = 5
	maxNotificationOutboxClaimBatch      = 20

	billingTypeRecurring = subscriptionservice.BillingTypeRecurring

	subscriptionStatusActive = subscriptionservice.StatusActive
	subscriptionStatusEnded  = subscriptionservice.StatusEnded

	renewalModeAutoRenew         = subscriptionservice.RenewalModeAutoRenew
	renewalModeManualRenew       = subscriptionservice.RenewalModeManualRenew
	renewalModeCancelAtPeriodEnd = subscriptionservice.RenewalModeCancelAtPeriodEnd

	recurrenceTypeInterval = "interval"
	intervalUnitMonth      = "month"
	intervalUnitYear       = "year"

	notificationLogStatusFailed = "failed"
	notificationLogStatusSent   = "sent"

	notificationTriggerDaysBefore  = subscriptionservice.TriggerDaysBefore
	notificationTriggerDueDay      = subscriptionservice.TriggerDueDay
	notificationTriggerManualDaily = subscriptionservice.TriggerManualDaily
	notificationTriggerManualEnded = subscriptionservice.TriggerManualEnded
	notificationTriggerEndingSoon  = subscriptionservice.TriggerEndingSoon
)

func NewService(db *gorm.DB, templateService *NotificationTemplateService, templateRenderer *TemplateRenderer) *Service {
	return &Service{
		DB:                 db,
		templateService:    templateService,
		templateRenderer:   templateRenderer,
		reminderCandidates: subscriptionservice.NewService(db),
		ownerID:            newNotificationOwnerID(),
	}
}

// WithContext returns a shallow copy of the service whose database handle is
// bound to ctx, so GORM cancels in-flight queries when ctx is cancelled. The
// embedded template service is rebound to the same context; the renderer and
// owner id are stateless and shared.
func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	clone.DB = s.DB.WithContext(ctx)
	if s.templateService != nil {
		clone.templateService = s.templateService.WithContext(ctx)
	}
	clone.reminderCandidates = subscriptionservice.NewService(clone.DB)
	return &clone
}

func (s *Service) notificationOwnerID() string {
	if s.ownerID == "" {
		s.ownerID = newNotificationOwnerID()
	}
	return s.ownerID
}

func newNotificationOwnerID() string {
	return serviceutil.NewBackgroundTaskOwnerID()
}

func (s *Service) subscriptionReminderProvider() ReminderCandidateProvider {
	if s.reminderCandidates == nil {
		s.reminderCandidates = subscriptionservice.NewService(s.DB)
	}
	return s.reminderCandidates
}

func (s *Service) acquireBackgroundTaskLease(taskKey string, ttl time.Duration) (bool, error) {
	return serviceutil.AcquireBackgroundTaskLease(s.DB, s.notificationOwnerID(), taskKey, ttl)
}

func (s *Service) withBackgroundTaskLease(taskKey string, ttl time.Duration, run func() error) error {
	return serviceutil.WithBackgroundTaskLease(s.DB, s.notificationOwnerID(), taskKey, ttl, run)
}

func normalizeDateUTC(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func reminderPolicyFromModel(policy *model.NotificationPolicy) subscriptionservice.ReminderPolicy {
	if policy == nil {
		return subscriptionservice.ReminderPolicy{}
	}
	return subscriptionservice.ReminderPolicy{
		DaysBefore:             policy.DaysBefore,
		NotifyOnDueDay:         policy.NotifyOnDueDay,
		NotifyManualRenewDaily: policy.NotifyManualRenewDaily,
	}
}

type CreateChannelInput struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Config  string `json:"config"`
}

type UpdateChannelInput struct {
	Enabled                  *bool    `json:"enabled"`
	Config                   *string  `json:"config"`
	ClearedSecretFields      []string `json:"cleared_secret_fields"`
	ClearedWebhookHeaderKeys []string `json:"cleared_webhook_header_keys"`
}

type UpdatePolicyInput struct {
	DaysBefore             *int  `json:"days_before"`
	NotifyOnDueDay         *bool `json:"notify_on_due_day"`
	NotifyManualRenewDaily *bool `json:"notify_manual_renew_daily"`
}
