package service

import (
	"gorm.io/gorm"
)

type NotificationService struct {
	DB               *gorm.DB
	templateService  *NotificationTemplateService
	templateRenderer *TemplateRenderer
}

const maxNotificationDaysBefore = 10
const maxEnabledNotificationChannels = 3
const maxParallelUserNotificationChecks = 4
const maxParallelNotificationDispatchesPerUser = 4

func NewNotificationService(db *gorm.DB, templateService *NotificationTemplateService, templateRenderer *TemplateRenderer) *NotificationService {
	return &NotificationService{
		DB:               db,
		templateService:  templateService,
		templateRenderer: templateRenderer,
	}
}

type CreateChannelInput struct {
	Type    string `json:"type"`
	Enabled bool   `json:"enabled"`
	Config  string `json:"config"`
}

type UpdateChannelInput struct {
	Enabled                 *bool    `json:"enabled"`
	Config                  *string  `json:"config"`
	ClearedSecretFields     []string `json:"cleared_secret_fields"`
	ClearedWebhookHeaderKeys []string `json:"cleared_webhook_header_keys"`
}

type UpdatePolicyInput struct {
	DaysBefore     *int  `json:"days_before"`
	NotifyOnDueDay *bool `json:"notify_on_due_day"`
}
