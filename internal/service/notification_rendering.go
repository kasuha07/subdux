package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

func (s *NotificationService) buildTemplateData(sub *model.Subscription, user *model.User, billingDate time.Time, daysUntil int) TemplateData {
	paymentMethodName := ""
	if sub.PaymentMethodID != nil {
		var paymentMethod model.PaymentMethod
		err := s.DB.Select("name").
			Where("id = ? AND user_id = ?", *sub.PaymentMethodID, sub.UserID).
			First(&paymentMethod).Error
		if err == nil {
			paymentMethodName = paymentMethod.Name
		}
	}

	return TemplateData{
		SubscriptionName: sub.Name,
		BillingDate:      billingDate.Format("2006-01-02"),
		Amount:           sub.Amount,
		Currency:         sub.Currency,
		DaysUntil:        daysUntil,
		Category:         sub.Category,
		PaymentMethod:    paymentMethodName,
		URL:              sub.URL,
		Remark:           sub.Notes,
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
