package service

import (
	"context"

	"gorm.io/gorm"
)

type NotificationService struct {
	DB               *gorm.DB
	templateService  *NotificationTemplateService
	templateRenderer *TemplateRenderer
	ownerID          string
}

const maxNotificationDaysBefore = 10
const maxEnabledNotificationChannels = 3
const maxParallelUserNotificationChecks = 4
const maxParallelNotificationDispatches = 4
const notificationOutboxDefaultMaxAttempts = 5
const maxNotificationOutboxClaimBatch = 20

func NewNotificationService(db *gorm.DB, templateService *NotificationTemplateService, templateRenderer *TemplateRenderer) *NotificationService {
	return &NotificationService{
		DB:               db,
		templateService:  templateService,
		templateRenderer: templateRenderer,
		ownerID:          newNotificationOwnerID(),
	}
}

// WithContext returns a shallow copy of the service whose database handle is
// bound to ctx, so GORM cancels in-flight queries when ctx is cancelled. The
// embedded template service is rebound to the same context; the renderer and
// owner id are stateless and shared.
func (s *NotificationService) WithContext(ctx context.Context) *NotificationService {
	clone := *s
	clone.DB = s.DB.WithContext(ctx)
	if s.templateService != nil {
		clone.templateService = s.templateService.WithContext(ctx)
	}
	return &clone
}

func (s *NotificationService) notificationOwnerID() string {
	if s.ownerID == "" {
		s.ownerID = newNotificationOwnerID()
	}
	return s.ownerID
}

func newNotificationOwnerID() string {
	return NewBackgroundTaskOwnerID()
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
