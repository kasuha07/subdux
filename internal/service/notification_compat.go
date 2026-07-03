package service

import (
	notificationservice "github.com/kasuha07/subdux/internal/service/notification"
	"gorm.io/gorm"
)

type NotificationService = notificationservice.Service
type NotificationTemplateService = notificationservice.NotificationTemplateService
type CreateChannelInput = notificationservice.CreateChannelInput
type UpdateChannelInput = notificationservice.UpdateChannelInput
type UpdatePolicyInput = notificationservice.UpdatePolicyInput
type NotificationDispatchSummary = notificationservice.NotificationDispatchSummary
type CreateTemplateInput = notificationservice.CreateTemplateInput
type UpdateTemplateInput = notificationservice.UpdateTemplateInput
type TemplateData = notificationservice.TemplateData
type TemplateRenderer = notificationservice.TemplateRenderer
type TemplateValidator = notificationservice.TemplateValidator

func NewNotificationService(db *gorm.DB, templateService *NotificationTemplateService, templateRenderer *TemplateRenderer) *NotificationService {
	return notificationservice.NewService(db, templateService, templateRenderer)
}

func NewNotificationTemplateService(db *gorm.DB, validator *TemplateValidator) *NotificationTemplateService {
	return notificationservice.NewNotificationTemplateService(db, validator)
}

func NewTemplateRenderer(validator *TemplateValidator) *TemplateRenderer {
	return notificationservice.NewTemplateRenderer(validator)
}

func NewTemplateValidator() *TemplateValidator {
	return notificationservice.NewTemplateValidator()
}

func decryptNotificationChannelConfig(config string) (string, error) {
	return notificationservice.DecryptNotificationChannelConfig(config)
}

func encryptNotificationChannelConfig(config string) (string, error) {
	return notificationservice.EncryptNotificationChannelConfig(config)
}

func getNotificationChannelSecretFields(channelType string) map[string]struct{} {
	return notificationservice.GetNotificationChannelSecretFields(channelType)
}

func parseNotificationConfigMap(raw string) (map[string]interface{}, error) {
	return notificationservice.ParseNotificationConfigMap(raw)
}

func sanitizeNotificationConfig(channelType, config string) (string, []string, []string) {
	return notificationservice.SanitizeNotificationConfig(channelType, config)
}

func isValidChannelType(t string) bool {
	return notificationservice.IsValidChannelType(t)
}

func validateChannelConfig(channelType, config string, db *gorm.DB) error {
	return notificationservice.ValidateChannelConfig(channelType, config, db)
}
