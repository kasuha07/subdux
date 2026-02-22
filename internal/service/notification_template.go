package service

import (
	"errors"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type NotificationTemplateService struct {
	DB        *gorm.DB
	validator *TemplateValidator
}

func NewNotificationTemplateService(db *gorm.DB, validator *TemplateValidator) *NotificationTemplateService {
	return &NotificationTemplateService{
		DB:        db,
		validator: validator,
	}
}

type CreateTemplateInput struct {
	ChannelType *string `json:"channel_type"`
	Format      string  `json:"format"`
	Template    string  `json:"template"`
}

type UpdateTemplateInput struct {
	Format   *string `json:"format"`
	Template *string `json:"template"`
}

// ListTemplates returns all templates for a user
func (s *NotificationTemplateService) ListTemplates(userID uint) ([]model.NotificationTemplate, error) {
	var templates []model.NotificationTemplate
	err := s.DB.Where("user_id = ?", userID).Order("channel_type ASC NULLS FIRST").Find(&templates).Error
	return templates, err
}

// GetTemplate retrieves a specific template
func (s *NotificationTemplateService) GetTemplate(userID, templateID uint) (*model.NotificationTemplate, error) {
	var tmpl model.NotificationTemplate
	if err := s.DB.Where("id = ? AND user_id = ?", templateID, userID).First(&tmpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}
	return &tmpl, nil
}

// CreateTemplate creates a new template
func (s *NotificationTemplateService) CreateTemplate(userID uint, input CreateTemplateInput) (*model.NotificationTemplate, error) {
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if err := s.validator.ValidateFormat(format); err != nil {
		return nil, err
	}

	if err := s.validator.ValidateTemplate(input.Template); err != nil {
		return nil, err
	}

	if input.ChannelType != nil {
		channelType := strings.ToLower(strings.TrimSpace(*input.ChannelType))
		if !isValidChannelType(channelType) {
			return nil, errors.New("invalid channel type")
		}
		input.ChannelType = &channelType
	}

	var count int64
	query := s.DB.Model(&model.NotificationTemplate{}).Where("user_id = ?", userID)
	if input.ChannelType == nil {
		query = query.Where("channel_type IS NULL")
	} else {
		query = query.Where("channel_type = ?", *input.ChannelType)
	}
	query.Count(&count)
	if count > 0 {
		if input.ChannelType == nil {
			return nil, errors.New("default template already exists")
		}
		return nil, errors.New("template for this channel type already exists")
	}

	tmpl := model.NotificationTemplate{
		UserID:      userID,
		ChannelType: input.ChannelType,
		Format:      format,
		Template:    input.Template,
	}

	if err := s.DB.Create(&tmpl).Error; err != nil {
		return nil, err
	}

	return &tmpl, nil
}

// UpdateTemplate updates an existing template
func (s *NotificationTemplateService) UpdateTemplate(userID, templateID uint, input UpdateTemplateInput) (*model.NotificationTemplate, error) {
	var tmpl model.NotificationTemplate
	if err := s.DB.Where("id = ? AND user_id = ?", templateID, userID).First(&tmpl).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("template not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})

	if input.Format != nil {
		format := strings.ToLower(strings.TrimSpace(*input.Format))
		if err := s.validator.ValidateFormat(format); err != nil {
			return nil, err
		}
		updates["format"] = format
	}

	if input.Template != nil {
		if err := s.validator.ValidateTemplate(*input.Template); err != nil {
			return nil, err
		}
		updates["template"] = *input.Template
	}

	if len(updates) > 0 {
		if err := s.DB.Model(&tmpl).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if err := s.DB.First(&tmpl, templateID).Error; err != nil {
		return nil, err
	}

	return &tmpl, nil
}

// DeleteTemplate deletes a template
func (s *NotificationTemplateService) DeleteTemplate(userID, templateID uint) error {
	result := s.DB.Where("id = ? AND user_id = ?", templateID, userID).Delete(&model.NotificationTemplate{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("template not found")
	}
	return nil
}

// GetTemplateForChannel retrieves template for specific channel (with fallback to default)
func (s *NotificationTemplateService) GetTemplateForChannel(userID uint, channelType string) (*model.NotificationTemplate, error) {
	var tmpl model.NotificationTemplate
	err := s.DB.Where("user_id = ? AND channel_type = ?", userID, channelType).First(&tmpl).Error
	if err == nil {
		return &tmpl, nil
	}

	err = s.DB.Where("user_id = ? AND channel_type IS NULL", userID).First(&tmpl).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("no template configured (default or channel-specific)")
		}
		return nil, err
	}

	return &tmpl, nil
}

// PreviewTemplate renders a template with user's first subscription data when available.
func (s *NotificationTemplateService) PreviewTemplate(userID uint, input CreateTemplateInput) (string, error) {
	format := strings.ToLower(strings.TrimSpace(input.Format))
	if err := s.validator.ValidateFormat(format); err != nil {
		return "", err
	}
	if err := s.validator.ValidateTemplate(input.Template); err != nil {
		return "", err
	}

	renderer := NewTemplateRenderer(s.validator)
	templateData := TemplateData{
		SubscriptionName: "Netflix Premium",
		BillingDate:      "2026-03-15",
		Amount:           15.99,
		Currency:         "USD",
		DaysUntil:        3,
		Category:         "Entertainment",
		PaymentMethod:    "Credit Card",
		URL:              "https://www.netflix.com",
		Remark:           "Family plan",
		UserEmail:        "user@example.com",
	}

	var sub model.Subscription
	if err := s.DB.Where("user_id = ?", userID).Order("id ASC").First(&sub).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return "", err
		}
		return renderer.RenderTemplate(input.Template, templateData)
	}

	templateData.SubscriptionName = sub.Name
	templateData.Amount = sub.Amount
	templateData.Currency = sub.Currency
	templateData.URL = sub.URL
	templateData.Remark = sub.Notes
	templateData.Category = sub.Category

	if sub.NextBillingDate != nil {
		billingDate := time.Date(
			sub.NextBillingDate.Year(),
			sub.NextBillingDate.Month(),
			sub.NextBillingDate.Day(),
			0, 0, 0, 0,
			sub.NextBillingDate.Location(),
		)
		templateData.BillingDate = billingDate.Format("2006-01-02")
		now := time.Now().In(billingDate.Location())
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, billingDate.Location())
		templateData.DaysUntil = int(billingDate.Sub(today).Hours() / 24)
	}

	if sub.CategoryID != nil && strings.TrimSpace(templateData.Category) == "" {
		var category model.Category
		if err := s.DB.Select("name").Where("id = ? AND user_id = ?", *sub.CategoryID, userID).First(&category).Error; err == nil {
			templateData.Category = category.Name
		}
	}

	if sub.PaymentMethodID != nil {
		var paymentMethod model.PaymentMethod
		if err := s.DB.Select("name").Where("id = ? AND user_id = ?", *sub.PaymentMethodID, userID).First(&paymentMethod).Error; err == nil {
			templateData.PaymentMethod = paymentMethod.Name
		}
	}

	var user model.User
	if err := s.DB.Select("email").Where("id = ?", userID).First(&user).Error; err == nil {
		templateData.UserEmail = user.Email
	}

	return renderer.RenderTemplate(input.Template, templateData)
}
