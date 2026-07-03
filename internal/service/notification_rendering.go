package service

import (
	"errors"
	"fmt"

	"github.com/kasuha07/subdux/internal/model"
)

func (s *NotificationService) buildTemplateData(
	candidate subscriptionReminderCandidate,
	user *model.User,
) TemplateData {
	template := candidate.Template
	paymentMethodName := ""
	if template.PaymentMethodID != nil {
		var paymentMethod model.PaymentMethod
		err := s.DB.Select("name").
			Where("id = ? AND user_id = ?", *template.PaymentMethodID, candidate.UserID).
			First(&paymentMethod).Error
		if err == nil {
			paymentMethodName = paymentMethod.Name
		}
	}

	return TemplateData{
		SubscriptionName: template.Name,
		BillingDate:      candidate.NotifyDate.Format("2006-01-02"),
		Amount:           template.Amount,
		Currency:         template.Currency,
		DaysUntil:        candidate.DaysUntil,
		EventType:        candidate.EventType,
		RenewalMode:      template.RenewalMode,
		Status:           template.Status,
		Category:         template.Category,
		PaymentMethod:    paymentMethodName,
		URL:              template.URL,
		Remark:           template.Notes,
		UserEmail:        user.Email,
	}
}

func (s *NotificationService) renderNotificationMessage(userID uint, channelType string, templateData TemplateData) (string, error) {
	template, err := s.templateService.GetTemplateForChannel(userID, channelType)
	if err != nil {
		return "", fmt.Errorf("failed to get template: %w", err)
	}
	if template == nil {
		return "", errors.New("no template found for channel")
	}
	message, err := s.templateRenderer.RenderTemplate(template.Template, templateData)
	if err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}
	return message, nil
}
