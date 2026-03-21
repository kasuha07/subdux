package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func (s *NotificationService) ListChannels(userID uint) ([]model.NotificationChannel, error) {
	var channels []model.NotificationChannel
	err := s.DB.Where("user_id = ?", userID).Order("id ASC").Find(&channels).Error
	return channels, err
}

func (s *NotificationService) CreateChannel(userID uint, input CreateChannelInput) (*model.NotificationChannel, error) {
	channelType := strings.ToLower(strings.TrimSpace(input.Type))
	if !isValidChannelType(channelType) {
		return nil, errors.New("invalid channel type, must be one of: smtp, resend, telegram, webhook, gotify, ntfy, bark, serverchan, feishu, wecom, dingtalk, pushdeer, pushplus, pushover, napcat")
	}
	if input.Enabled {
		if err := s.ensureEnabledChannelLimit(userID); err != nil {
			return nil, err
		}
	}

	canonicalConfig, err := parseAndNormalizeConfig(input.Config)
	if err != nil {
		return nil, err
	}

	if err := validateChannelConfig(channelType, canonicalConfig); err != nil {
		return nil, err
	}

	encryptedConfig, err := encryptNotificationChannelConfig(canonicalConfig)
	if err != nil {
		return nil, err
	}

	channel := model.NotificationChannel{
		UserID:  userID,
		Type:    channelType,
		Enabled: input.Enabled,
		Config:  encryptedConfig,
	}

	if err := s.DB.Create(&channel).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}

func (s *NotificationService) UpdateChannel(userID, channelID uint, input UpdateChannelInput) (*model.NotificationChannel, error) {
	var channel model.NotificationChannel
	if err := s.DB.Where("id = ? AND user_id = ?", channelID, userID).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("channel not found")
		}
		return nil, err
	}

	updates := make(map[string]interface{})
	if input.Enabled != nil {
		if *input.Enabled && !channel.Enabled {
			if err := s.ensureEnabledChannelLimit(userID); err != nil {
				return nil, err
			}
		}
		updates["enabled"] = *input.Enabled
	}
	if input.Config != nil {
		mergedConfig, err := mergeNotificationConfigWithExistingSecrets(channel.Type, channel.Config, *input.Config, input.ClearedSecretFields, input.ClearedWebhookHeaderKeys)
		if err != nil {
			return nil, err
		}

		if err := validateChannelConfig(channel.Type, mergedConfig); err != nil {
			return nil, err
		}

		encryptedConfig, err := encryptNotificationChannelConfig(mergedConfig)
		if err != nil {
			return nil, err
		}
		updates["config"] = encryptedConfig
	}

	if len(updates) > 0 {
		if err := s.DB.Model(&channel).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	if err := s.DB.First(&channel, channelID).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}

func parseAndNormalizeConfig(config string) (string, error) {
	parsed, err := parseNotificationConfigMap(config)
	if err != nil {
		return "", err
	}

	encoded, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func (s *NotificationService) ensureEnabledChannelLimit(userID uint) error {
	var enabledCount int64
	if err := s.DB.Model(&model.NotificationChannel{}).
		Where("user_id = ? AND enabled = ?", userID, true).
		Count(&enabledCount).Error; err != nil {
		return err
	}
	if enabledCount >= maxEnabledNotificationChannels {
		return fmt.Errorf("you can enable at most %d notification channels", maxEnabledNotificationChannels)
	}
	return nil
}

func (s *NotificationService) DeleteChannel(userID, channelID uint) error {
	result := s.DB.Where("id = ? AND user_id = ?", channelID, userID).Delete(&model.NotificationChannel{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("channel not found")
	}
	return nil
}

func (s *NotificationService) TestChannel(userID, channelID uint) error {
	var channel model.NotificationChannel
	if err := s.DB.Where("id = ? AND user_id = ?", channelID, userID).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("channel not found")
		}
		return err
	}

	var user model.User
	if err := s.DB.Select("email").First(&user, userID).Error; err != nil {
		return errors.New("failed to load user")
	}

	testSubName := "Test Subscription"
	testBillingDate := time.Now().AddDate(0, 0, 3)
	testSubscription := &model.Subscription{
		Name:        testSubName,
		Amount:      9.99,
		Currency:    "USD",
		Status:      subscriptionStatusActive,
		RenewalMode: renewalModeAutoRenew,
		Category:    "Entertainment",
		URL:         "https://example.com/subscription",
		Notes:       "Test notification",
	}

	templateData := s.buildTemplateData(
		testSubscription,
		&user,
		testBillingDate,
		3,
		notificationEventTypeForSubscription(*testSubscription),
	)

	message, err := s.renderNotificationMessage(userID, channel.Type, templateData)
	if err != nil {
		return fmt.Errorf("failed to render notification message: %w", err)
	}

	return s.dispatchNotificationChannel(channel, user.Email, message, testSubscription.URL)
}
