package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

type NotificationService struct {
	DB               *gorm.DB
	templateService  *NotificationTemplateService
	templateRenderer *TemplateRenderer
}

const maxNotificationDaysBefore = 10

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
	Enabled *bool   `json:"enabled"`
	Config  *string `json:"config"`
}

type UpdatePolicyInput struct {
	DaysBefore     *int  `json:"days_before"`
	NotifyOnDueDay *bool `json:"notify_on_due_day"`
}

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

	if err := validateChannelConfig(channelType, input.Config); err != nil {
		return nil, err
	}

	channel := model.NotificationChannel{
		UserID:  userID,
		Type:    channelType,
		Enabled: input.Enabled,
		Config:  input.Config,
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
		updates["enabled"] = *input.Enabled
	}
	if input.Config != nil {
		if err := validateChannelConfig(channel.Type, *input.Config); err != nil {
			return nil, err
		}
		updates["config"] = *input.Config
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

	// Build template data for test notification
	testSubName := "Test Subscription"
	testBillingDate := time.Now().AddDate(0, 0, 3)
	testSubscription := &model.Subscription{
		Name:     testSubName,
		Amount:   9.99,
		Currency: "USD",
		Category: "Entertainment",
		URL:      "https://example.com/subscription",
		Notes:    "Test notification",
	}

	templateData := s.buildTemplateData(
		testSubscription,
		&user,
		testBillingDate,
		3,
	)

	// Render message from template
	message, err := s.renderNotificationMessage(userID, channel.Type, templateData)
	if err != nil {
		return fmt.Errorf("failed to render notification message: %w", err)
	}

	switch channel.Type {
	case "smtp":
		return s.sendSMTP(channel, user.Email, message)
	case "resend":
		return s.sendResend(channel, message)
	case "telegram":
		return s.sendTelegram(channel, message)
	case "webhook":
		return s.sendWebhook(channel, message)
	case "gotify":
		return s.sendGotify(channel, message)
	case "ntfy":
		return s.sendNtfy(channel, message, testSubscription.URL)
	case "bark":
		return s.sendBark(channel, message)
	case "serverchan":
		return s.sendServerChan(channel, message)
	case "feishu":
		return s.sendFeishu(channel, message)
	case "wecom":
		return s.sendWeCom(channel, message)
	case "dingtalk":
		return s.sendDingTalk(channel, message)
	case "pushdeer":
		return s.sendPushDeer(channel, message)
	case "pushplus":
		return s.sendPushplus(channel, message)
	case "pushover":
		return s.sendPushover(channel, message)
	case "napcat":
		return s.sendNapCat(channel, message)
	default:
		return errors.New("unsupported channel type")
	}
}

func (s *NotificationService) GetPolicy(userID uint) (*model.NotificationPolicy, error) {
	var policy model.NotificationPolicy
	if err := s.DB.Where("user_id = ?", userID).First(&policy).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &model.NotificationPolicy{
				UserID:         userID,
				DaysBefore:     3,
				NotifyOnDueDay: true,
			}, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (s *NotificationService) UpdatePolicy(userID uint, input UpdatePolicyInput) (*model.NotificationPolicy, error) {
	var policy model.NotificationPolicy
	err := s.DB.Where("user_id = ?", userID).First(&policy).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		policy = model.NotificationPolicy{
			UserID:         userID,
			DaysBefore:     3,
			NotifyOnDueDay: true,
		}
	}

	if input.DaysBefore != nil {
		if *input.DaysBefore < 0 || *input.DaysBefore > maxNotificationDaysBefore {
			return nil, fmt.Errorf("days_before must be between 0 and %d", maxNotificationDaysBefore)
		}
		policy.DaysBefore = *input.DaysBefore
	}
	if input.NotifyOnDueDay != nil {
		policy.NotifyOnDueDay = *input.NotifyOnDueDay
	}

	if policy.ID == 0 {
		if err := s.DB.Create(&policy).Error; err != nil {
			return nil, err
		}
	} else {
		if err := s.DB.Save(&policy).Error; err != nil {
			return nil, err
		}
	}

	return &policy, nil
}

func (s *NotificationService) ListLogs(userID uint, limit int) ([]model.NotificationLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var logs []model.NotificationLog
	err := s.DB.Where("user_id = ?", userID).
		Order("sent_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

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

func (s *NotificationService) ProcessPendingNotifications() error {
	var channelUserIDs []uint
	if err := s.DB.Model(&model.NotificationChannel{}).
		Where("enabled = ?", true).
		Distinct("user_id").
		Pluck("user_id", &channelUserIDs).Error; err != nil {
		return fmt.Errorf("failed to query notification channels: %w", err)
	}

	for _, userID := range channelUserIDs {
		if err := s.processUserNotifications(userID); err != nil {
			fmt.Printf("notification error for user %d: %v\n", userID, err)
		}
	}

	return nil
}

func (s *NotificationService) processUserNotifications(userID uint) error {
	if err := autoAdvanceRecurringNextBillingDatesForUser(s.DB, userID, time.Now().UTC()); err != nil {
		return err
	}

	policy, err := s.GetPolicy(userID)
	if err != nil {
		return err
	}

	var subs []model.Subscription
	if err := s.DB.Where("user_id = ? AND enabled = ? AND billing_type = ? AND next_billing_date IS NOT NULL",
		userID, true, "recurring").Find(&subs).Error; err != nil {
		return err
	}

	var enabledChannels []model.NotificationChannel
	if err := s.DB.Where("user_id = ? AND enabled = ?", userID, true).Find(&enabledChannels).Error; err != nil {
		return err
	}

	if len(enabledChannels) == 0 {
		return nil
	}

	var user model.User
	if err := s.DB.Select("email").First(&user, userID).Error; err != nil {
		return err
	}

	// Use system timezone
	systemLoc := pkg.GetSystemTimezone()

	for _, sub := range subs {
		if sub.NextBillingDate == nil {
			continue
		}

		notifyEnabled := true
		daysBefore := policy.DaysBefore
		notifyOnDueDay := policy.NotifyOnDueDay

		if sub.NotifyEnabled != nil {
			notifyEnabled = *sub.NotifyEnabled
		}
		if !notifyEnabled {
			continue
		}
		if sub.NotifyDaysBefore != nil {
			daysBefore = *sub.NotifyDaysBefore
		}

		billingDate := pkg.NormalizeDateInTimezone(*sub.NextBillingDate, systemLoc)

		shouldNotify := false
		daysUntilBilling := pkg.DaysUntil(*sub.NextBillingDate, systemLoc)

		if daysUntilBilling == daysBefore && daysBefore > 0 {
			shouldNotify = true
		}
		if daysUntilBilling == 0 && notifyOnDueDay {
			shouldNotify = true
		}

		if !shouldNotify {
			continue
		}

		for _, channel := range enabledChannels {
			var count int64
			s.DB.Model(&model.NotificationLog{}).
				Where("subscription_id = ? AND channel_type = ? AND notify_date = ? AND status = ?",
					sub.ID, channel.Type, billingDate, "sent").
				Count(&count)
			if count > 0 {
				continue
			}

			// Build template data and render message
			templateData := s.buildTemplateData(&sub, &user, billingDate, daysUntilBilling)
			message, renderErr := s.renderNotificationMessage(userID, channel.Type, templateData)
			if renderErr != nil {
				fmt.Printf("failed to render template for user %d channel %s: %v\n", userID, channel.Type, renderErr)
				continue
			}
			var sendErr error
			switch channel.Type {
			case "smtp":
				sendErr = s.sendSMTP(channel, user.Email, message)
			case "resend":
				sendErr = s.sendResend(channel, message)
			case "telegram":
				sendErr = s.sendTelegram(channel, message)
			case "webhook":
				sendErr = s.sendWebhook(channel, message)
			case "gotify":
				sendErr = s.sendGotify(channel, message)
			case "ntfy":
				sendErr = s.sendNtfy(channel, message, sub.URL)
			case "bark":
				sendErr = s.sendBark(channel, message)
			case "serverchan":
				sendErr = s.sendServerChan(channel, message)
			case "feishu":
				sendErr = s.sendFeishu(channel, message)
			case "wecom":
				sendErr = s.sendWeCom(channel, message)
			case "dingtalk":
				sendErr = s.sendDingTalk(channel, message)
			case "pushdeer":
				sendErr = s.sendPushDeer(channel, message)
			case "pushplus":
				sendErr = s.sendPushplus(channel, message)
			case "pushover":
				sendErr = s.sendPushover(channel, message)
			case "napcat":
				sendErr = s.sendNapCat(channel, message)
			}

			logEntry := model.NotificationLog{
				UserID:         userID,
				SubscriptionID: sub.ID,
				ChannelType:    channel.Type,
				NotifyDate:     billingDate,
				SentAt:         time.Now().UTC(),
			}

			if sendErr != nil {
				logEntry.Status = "failed"
				logEntry.Error = sendErr.Error()
			} else {
				logEntry.Status = "sent"
			}

			s.DB.Create(&logEntry)
		}
	}

	return nil
}

func (s *NotificationService) sendSMTP(channel model.NotificationChannel, toEmail, message string) error {
	var cfg struct {
		Host          string `json:"host"`
		Port          int64  `json:"port"`
		Username      string `json:"username"`
		Password      string `json:"password"`
		FromEmail     string `json:"from_email"`
		FromName      string `json:"from_name"`
		ToEmail       string `json:"to_email"`
		Encryption    string `json:"encryption"`
		SkipTLSVerify bool   `json:"skip_tls_verify"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid smtp config")
	}
	if cfg.Host == "" {
		return errors.New("smtp config requires host")
	}
	if cfg.FromEmail == "" {
		return errors.New("smtp config requires from_email")
	}
	if cfg.ToEmail == "" {
		return errors.New("smtp config requires to_email")
	}
	port := cfg.Port
	if port <= 0 || port > 65535 {
		port = 587
	}
	encryption := strings.ToLower(strings.TrimSpace(cfg.Encryption))
	if encryption == "" {
		encryption = "starttls"
	}
	rtCfg := smtpRuntimeConfig{
		Host:           cfg.Host,
		Port:           port,
		Username:       cfg.Username,
		Password:       cfg.Password,
		FromEmail:      cfg.FromEmail,
		FromName:       cfg.FromName,
		Encryption:     encryption,
		AuthMethod:     "auto",
		TimeoutSeconds: 10,
		SkipTLSVerify:  cfg.SkipTLSVerify,
	}

	subject := "Subscription Reminder"
	body := message
	smtpMessage := buildSMTPMessage(rtCfg.FromEmail, rtCfg.FromName, cfg.ToEmail, subject, body)

	return sendSMTPMessage(rtCfg, cfg.ToEmail, smtpMessage)
}

func (s *NotificationService) sendResend(channel model.NotificationChannel, message string) error {
	var cfg struct {
		APIKey    string `json:"api_key"`
		FromEmail string `json:"from_email"`
		ToEmail   string `json:"to_email"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid resend config")
	}
	if cfg.APIKey == "" || cfg.FromEmail == "" || cfg.ToEmail == "" {
		return errors.New("resend config requires api_key, from_email, and to_email")
	}
	payload := map[string]interface{}{
		"from":    cfg.FromEmail,
		"to":      []string{cfg.ToEmail},
		"subject": "Subscription Reminder",
		"text":    message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("resend request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("resend API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (s *NotificationService) sendTelegram(channel model.NotificationChannel, message string) error {
	var cfg struct {
		BotToken string `json:"bot_token"`
		ChatID   string `json:"chat_id"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid telegram config")
	}
	if cfg.BotToken == "" || cfg.ChatID == "" {
		return errors.New("telegram config requires bot_token and chat_id")
	}
	payload := map[string]interface{}{
		"chat_id":    cfg.ChatID,
		"text":       message,
		"parse_mode": "Markdown",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("telegram API error %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

func (s *NotificationService) sendWebhook(channel model.NotificationChannel, message string) error {
	var cfg struct {
		URL     string            `json:"url"`
		Secret  string            `json:"secret"`
		Method  string            `json:"method"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid webhook config")
	}
	if cfg.URL == "" {
		return errors.New("webhook config requires url")
	}

	method, err := normalizeWebhookMethod(cfg.Method)
	if err != nil {
		return err
	}
	if method == http.MethodGet && strings.TrimSpace(cfg.Secret) != "" {
		return errors.New("webhook secret is not supported when method is GET")
	}
	normalizedHeaders, err := normalizeWebhookHeaders(cfg.Headers)
	if err != nil {
		return err
	}

	sentAt := time.Now().UTC().Format(time.RFC3339)
	payload := map[string]interface{}{
		"event":   "subscription.reminder",
		"message": message,
		"sent_at": sentAt,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	requestURL := cfg.URL
	var requestBody io.Reader
	if method == http.MethodGet {
		parsedURL, err := url.Parse(cfg.URL)
		if err != nil {
			return fmt.Errorf("invalid webhook url: %w", err)
		}
		query := parsedURL.Query()
		query.Set("event", "subscription.reminder")
		query.Set("message", message)
		query.Set("sent_at", sentAt)
		parsedURL.RawQuery = query.Encode()
		requestURL = parsedURL.String()
	} else {
		requestBody = bytes.NewReader(body)
	}

	req, err := http.NewRequest(method, requestURL, requestBody)
	if err != nil {
		return err
	}
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range normalizedHeaders {
		req.Header.Set(key, value)
	}

	if cfg.Secret != "" {
		mac := hmac.New(sha256.New, []byte(cfg.Secret))
		mac.Write(body)
		sig := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-Signature-256", "sha256="+sig)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("webhook error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func normalizeWebhookMethod(method string) (string, error) {
	normalized := strings.ToUpper(strings.TrimSpace(method))
	if normalized == "" {
		return http.MethodPost, nil
	}

	switch normalized {
	case http.MethodGet, http.MethodPost, http.MethodPut:
		return normalized, nil
	default:
		return "", errors.New("webhook method must be one of: GET, POST, PUT")
	}
}

func normalizeWebhookHeaders(headers map[string]string) (map[string]string, error) {
	normalized := make(map[string]string, len(headers))

	for key, value := range headers {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, errors.New("webhook headers cannot contain empty key")
		}
		if trimmedKey != key {
			return nil, errors.New("webhook header name must not contain leading or trailing spaces")
		}
		if !isValidHTTPHeaderName(trimmedKey) {
			return nil, errors.New("webhook header name contains invalid characters")
		}
		if strings.ContainsAny(trimmedKey, "\r\n") {
			return nil, errors.New("webhook header name contains invalid newline characters")
		}
		if strings.ContainsAny(value, "\r\n") {
			return nil, errors.New("webhook header value contains invalid newline characters")
		}

		normalized[trimmedKey] = value
	}

	return normalized, nil
}

func isValidHTTPHeaderName(name string) bool {
	for _, r := range name {
		if (r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') {
			continue
		}

		switch r {
		case '!', '#', '$', '%', '&', '\'', '*', '+', '-', '.', '^', '_', '`', '|', '~':
			continue
		default:
			return false
		}
	}

	return true
}

func (s *NotificationService) sendGotify(channel model.NotificationChannel, message string) error {
	var cfg struct {
		URL   string `json:"url"`
		Token string `json:"token"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid gotify config")
	}
	if cfg.URL == "" || cfg.Token == "" {
		return errors.New("gotify config requires url and token")
	}

	payload := map[string]interface{}{
		"title":    "Subscription Reminder",
		"message":  message,
		"priority": 5,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(cfg.URL, "/") + "/message?token=" + cfg.Token
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gotify request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("gotify API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *NotificationService) sendNtfy(channel model.NotificationChannel, message, subscriptionURL string) error {
	var cfg struct {
		URL      string `json:"url"`
		Topic    string `json:"topic"`
		Token    string `json:"token"`
		Username string `json:"username"`
		Password string `json:"password"`
		Priority string `json:"priority"`
		Tags     string `json:"tags"`
		Click    string `json:"click"`
		Icon     string `json:"icon"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid ntfy config")
	}

	serverURL := cfg.URL
	if serverURL == "" {
		serverURL = "https://ntfy.sh"
	}

	if cfg.Topic == "" {
		return errors.New("ntfy config requires topic")
	}

	endpoint := strings.TrimRight(serverURL, "/") + "/" + cfg.Topic

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Title", "Subscription Reminder")
	priority := strings.TrimSpace(cfg.Priority)
	if priority == "" {
		priority = "default"
	}
	req.Header.Set("Priority", priority)

	tags := strings.TrimSpace(cfg.Tags)
	if tags == "" {
		tags = "calendar"
	}
	req.Header.Set("Tags", tags)

	click := strings.TrimSpace(subscriptionURL)
	if click == "" {
		click = strings.TrimSpace(cfg.Click)
	}
	if click != "" {
		req.Header.Set("Click", click)
		req.Header.Set("X-Click", click)
	}

	if icon := strings.TrimSpace(cfg.Icon); icon != "" {
		req.Header.Set("Icon", icon)
	}

	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	} else if cfg.Username != "" && cfg.Password != "" {
		req.SetBasicAuth(cfg.Username, cfg.Password)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("ntfy request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("ntfy API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *NotificationService) sendBark(channel model.NotificationChannel, message string) error {
	var cfg struct {
		URL       string `json:"url"`
		DeviceKey string `json:"device_key"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid bark config")
	}

	serverURL := cfg.URL
	if serverURL == "" {
		serverURL = "https://api.day.app"
	}

	if cfg.DeviceKey == "" {
		return errors.New("bark config requires device_key")
	}

	payload := map[string]interface{}{
		"title": "Subscription Reminder",
		"body":  message,
		"group": "Subdux",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(serverURL, "/") + "/" + cfg.DeviceKey
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("bark request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("bark API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *NotificationService) sendServerChan(channel model.NotificationChannel, message string) error {
	var cfg struct {
		SendKey string `json:"send_key"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid serverchan config")
	}
	if cfg.SendKey == "" {
		return errors.New("serverchan config requires send_key")
	}

	payload := map[string]string{
		"title": "Subscription Reminder",
		"desp":  message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := buildServerChanEndpoint(cfg.SendKey)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("serverchan request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("serverchan API error %d: %s", resp.StatusCode, string(respBody))
	}
	if err := validateServerChanBusinessResponse(respBody); err != nil {
		return err
	}

	return nil
}

var serverChanUIDPattern = regexp.MustCompile(`^sctp(\d+)t`)

func buildServerChanEndpoint(sendKey string) string {
	sendKey = strings.TrimSpace(sendKey)
	if m := serverChanUIDPattern.FindStringSubmatch(sendKey); len(m) == 2 {
		return fmt.Sprintf("https://%s.push.ft07.com/send/%s.send", m[1], sendKey)
	}
	return fmt.Sprintf("https://sctapi.ftqq.com/%s.send", sendKey)
}

func validateServerChanBusinessResponse(body []byte) error {
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Info    string `json:"info"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("invalid serverchan response: %w", err)
	}
	if resp.Code == 0 {
		return nil
	}

	message := strings.TrimSpace(resp.Message)
	if message == "" {
		message = strings.TrimSpace(resp.Info)
	}
	if message == "" {
		message = strings.TrimSpace(resp.Error)
	}
	if message == "" {
		message = "unknown error"
	}

	return fmt.Errorf("serverchan business error code %d: %s", resp.Code, message)
}

func (s *NotificationService) sendFeishu(channel model.NotificationChannel, message string) error {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
		Secret     string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid feishu config")
	}
	if cfg.WebhookURL == "" {
		return errors.New("feishu config requires webhook_url")
	}

	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": message,
		},
	}

	if cfg.Secret != "" {
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		stringToSign := timestamp + "\n" + cfg.Secret
		h := hmac.New(sha256.New, []byte(cfg.Secret))
		h.Write([]byte(stringToSign))
		sign := base64.StdEncoding.EncodeToString(h.Sum(nil))
		payload["timestamp"] = timestamp
		payload["sign"] = sign
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("feishu request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("feishu API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *NotificationService) sendWeCom(channel model.NotificationChannel, message string) error {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid wecom config")
	}
	if cfg.WebhookURL == "" {
		return errors.New("wecom config requires webhook_url")
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, cfg.WebhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("wecom request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("wecom API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *NotificationService) sendDingTalk(channel model.NotificationChannel, message string) error {
	var cfg struct {
		WebhookURL string `json:"webhook_url"`
		Secret     string `json:"secret"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid dingtalk config")
	}
	if cfg.WebhookURL == "" {
		return errors.New("dingtalk config requires webhook_url")
	}

	webhookURL := cfg.WebhookURL
	if cfg.Secret != "" {
		timestampMs := time.Now().UnixMilli()
		timestamp := strconv.FormatInt(timestampMs, 10)
		stringToSign := timestamp + "\n" + cfg.Secret
		h := hmac.New(sha256.New, []byte(cfg.Secret))
		h.Write([]byte(stringToSign))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(h.Sum(nil)))
		webhookURL = fmt.Sprintf("%s&timestamp=%s&sign=%s", webhookURL, timestamp, sign)
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": message,
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("dingtalk request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("dingtalk API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *NotificationService) sendPushDeer(channel model.NotificationChannel, message string) error {
	var cfg struct {
		PushKey   string `json:"push_key"`
		ServerURL string `json:"server_url"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid pushdeer config")
	}
	if cfg.PushKey == "" {
		return errors.New("pushdeer config requires push_key")
	}

	serverURL := strings.TrimSpace(cfg.ServerURL)
	if serverURL == "" {
		serverURL = "https://api2.pushdeer.com"
	}

	payload := url.Values{}
	payload.Set("pushkey", cfg.PushKey)
	payload.Set("text", "Subscription Reminder")
	payload.Set("desp", message)
	payload.Set("type", "markdown")

	endpoint := strings.TrimRight(serverURL, "/") + "/message/push"
	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(payload.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pushdeer request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("pushdeer API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

func (s *NotificationService) sendPushplus(channel model.NotificationChannel, message string) error {
	var cfg struct {
		Token    string `json:"token"`
		Endpoint string `json:"endpoint"`
		Template string `json:"template"`
		Channel  string `json:"channel"`
		Topic    string `json:"topic"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid pushplus config")
	}
	if cfg.Token == "" {
		return errors.New("pushplus config requires token")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://www.pushplus.plus/send"
	}

	template := strings.TrimSpace(cfg.Template)
	if template == "" {
		template = "markdown"
	}

	payload := map[string]string{
		"token":    cfg.Token,
		"title":    "Subscription Reminder",
		"content":  message,
		"template": template,
	}
	if strings.TrimSpace(cfg.Channel) != "" {
		payload["channel"] = strings.TrimSpace(cfg.Channel)
	}
	if strings.TrimSpace(cfg.Topic) != "" {
		payload["topic"] = strings.TrimSpace(cfg.Topic)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pushplus request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("pushplus API error %d: %s", resp.StatusCode, string(respBody))
	}

	var pushplusResp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(respBody, &pushplusResp); err == nil {
		if pushplusResp.Code != 0 && pushplusResp.Code != 200 {
			errMsg := pushplusResp.Msg
			if errMsg == "" {
				errMsg = string(respBody)
			}
			return fmt.Errorf("pushplus API error %d: %s", pushplusResp.Code, errMsg)
		}
	}

	return nil
}

func (s *NotificationService) sendPushover(channel model.NotificationChannel, message string) error {
	var cfg struct {
		Token    string `json:"token"`
		User     string `json:"user"`
		Device   string `json:"device"`
		Priority *int   `json:"priority"`
		Sound    string `json:"sound"`
		Endpoint string `json:"endpoint"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid pushover config")
	}
	if strings.TrimSpace(cfg.Token) == "" {
		return errors.New("pushover config requires token")
	}
	if strings.TrimSpace(cfg.User) == "" {
		return errors.New("pushover config requires user")
	}

	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.pushover.net/1/messages.json"
	}

	payload := url.Values{}
	payload.Set("token", strings.TrimSpace(cfg.Token))
	payload.Set("user", strings.TrimSpace(cfg.User))
	payload.Set("title", "Subscription Reminder")
	payload.Set("message", message)
	if strings.TrimSpace(cfg.Device) != "" {
		payload.Set("device", strings.TrimSpace(cfg.Device))
	}
	if cfg.Priority != nil {
		payload.Set("priority", strconv.Itoa(*cfg.Priority))
	}
	if strings.TrimSpace(cfg.Sound) != "" {
		payload.Set("sound", strings.TrimSpace(cfg.Sound))
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(payload.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("pushover request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("pushover API error %d: %s", resp.StatusCode, string(respBody))
	}

	var pushoverResp struct {
		Status int      `json:"status"`
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal(respBody, &pushoverResp); err == nil {
		if pushoverResp.Status != 1 {
			errMsg := strings.TrimSpace(strings.Join(pushoverResp.Errors, ", "))
			if errMsg == "" {
				errMsg = string(respBody)
			}
			return fmt.Errorf("pushover API error: %s", errMsg)
		}
	}

	return nil
}

func (s *NotificationService) sendNapCat(channel model.NotificationChannel, message string) error {
	var cfg struct {
		URL         string `json:"url"`
		AccessToken string `json:"access_token"`
		MessageType string `json:"message_type"`
		UserID      string `json:"user_id"`
		GroupID     string `json:"group_id"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid napcat config")
	}
	if cfg.URL == "" {
		return errors.New("napcat config requires url")
	}

	msgType := strings.ToLower(strings.TrimSpace(cfg.MessageType))
	if msgType == "" {
		msgType = "private"
	}

	if msgType == "private" && cfg.UserID == "" {
		return errors.New("napcat config requires user_id for private messages")
	}
	if msgType == "group" && cfg.GroupID == "" {
		return errors.New("napcat config requires group_id for group messages")
	}

	payload := map[string]interface{}{
		"message_type": msgType,
		"message":      message,
	}
	if msgType == "private" {
		payload["user_id"] = cfg.UserID
	} else {
		payload["group_id"] = cfg.GroupID
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	endpoint := strings.TrimRight(cfg.URL, "/") + "/send_msg"
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.AccessToken != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.AccessToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("napcat request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("napcat API error %d: %s", resp.StatusCode, string(respBody))
	}

	var napcatResp struct {
		Status  string `json:"status"`
		Retcode int    `json:"retcode"`
		Message string `json:"message"`
		Wording string `json:"wording"`
	}
	if err := json.Unmarshal(respBody, &napcatResp); err == nil {
		if napcatResp.Retcode != 0 {
			errMsg := napcatResp.Message
			if errMsg == "" {
				errMsg = napcatResp.Wording
			}
			if errMsg == "" {
				errMsg = string(respBody)
			}
			return fmt.Errorf("napcat API error %d: %s", napcatResp.Retcode, errMsg)
		}
	}

	return nil
}

func isValidChannelType(t string) bool {
	switch t {
	case "smtp", "resend", "telegram", "webhook", "gotify", "ntfy", "bark", "serverchan", "feishu", "wecom", "dingtalk", "pushdeer", "pushplus", "pushover", "napcat":
		return true
	default:
		return false
	}
}

func validateChannelConfig(channelType, config string) error {
	if strings.TrimSpace(config) == "" {
		config = "{}"
	}

	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(config), &raw); err != nil {
		return errors.New("config must be valid JSON")
	}

	switch channelType {
	case "smtp":
		var cfg struct {
			Host      string `json:"host"`
			Port      int64  `json:"port"`
			FromEmail string `json:"from_email"`
			ToEmail   string `json:"to_email"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid smtp config format")
		}
		if cfg.Host == "" {
			return errors.New("smtp channel requires host")
		}
		if cfg.FromEmail == "" {
			return errors.New("smtp channel requires from_email")
		}
		if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
			return errors.New("invalid from_email address")
		}
		if cfg.ToEmail == "" {
			return errors.New("smtp channel requires to_email")
		}
		if _, err := mail.ParseAddress(cfg.ToEmail); err != nil {
			return errors.New("invalid to_email address")
		}
		if cfg.Port < 0 || cfg.Port > 65535 {
			return errors.New("smtp port must be between 0 and 65535")
		}
		return nil
	case "resend":
		var cfg struct {
			APIKey    string `json:"api_key"`
			FromEmail string `json:"from_email"`
			ToEmail   string `json:"to_email"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid resend config format")
		}
		if cfg.APIKey == "" {
			return errors.New("resend channel requires api_key")
		}
		if cfg.FromEmail == "" {
			return errors.New("resend channel requires from_email")
		}
		if cfg.ToEmail == "" {
			return errors.New("resend channel requires to_email")
		}
		if _, err := mail.ParseAddress(cfg.FromEmail); err != nil {
			return errors.New("invalid from_email address")
		}
		if _, err := mail.ParseAddress(cfg.ToEmail); err != nil {
			return errors.New("invalid to_email address")
		}
		return nil
	case "telegram":
		var cfg struct {
			BotToken string `json:"bot_token"`
			ChatID   string `json:"chat_id"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid telegram config format")
		}
		if cfg.BotToken == "" {
			return errors.New("telegram channel requires bot_token")
		}
		if cfg.ChatID == "" {
			return errors.New("telegram channel requires chat_id")
		}
		return nil
	case "webhook":
		var cfg struct {
			URL     string            `json:"url"`
			Secret  string            `json:"secret"`
			Method  string            `json:"method"`
			Headers map[string]string `json:"headers"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid webhook config format")
		}
		if cfg.URL == "" {
			return errors.New("webhook channel requires url")
		}
		if !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
			return errors.New("webhook url must start with http:// or https://")
		}
		method, err := normalizeWebhookMethod(cfg.Method)
		if err != nil {
			return err
		}
		if method == http.MethodGet && strings.TrimSpace(cfg.Secret) != "" {
			return errors.New("webhook secret is not supported when method is GET")
		}
		if _, err := normalizeWebhookHeaders(cfg.Headers); err != nil {
			return err
		}
		return nil
	case "gotify":
		var cfg struct {
			URL   string `json:"url"`
			Token string `json:"token"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid gotify config format")
		}
		if cfg.URL == "" {
			return errors.New("gotify channel requires url")
		}
		if !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
			return errors.New("gotify url must start with http:// or https://")
		}
		if cfg.Token == "" {
			return errors.New("gotify channel requires token")
		}
		return nil
	case "ntfy":
		var cfg struct {
			Topic string `json:"topic"`
			URL   string `json:"url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid ntfy config format")
		}
		if cfg.Topic == "" {
			return errors.New("ntfy channel requires topic")
		}
		if cfg.URL != "" && !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
			return errors.New("ntfy url must start with http:// or https://")
		}
		return nil
	case "bark":
		var cfg struct {
			DeviceKey string `json:"device_key"`
			URL       string `json:"url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid bark config format")
		}
		if cfg.DeviceKey == "" {
			return errors.New("bark channel requires device_key")
		}
		if cfg.URL != "" && !strings.HasPrefix(cfg.URL, "http://") && !strings.HasPrefix(cfg.URL, "https://") {
			return errors.New("bark url must start with http:// or https://")
		}
		return nil
	case "serverchan":
		var cfg struct {
			SendKey string `json:"send_key"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid serverchan config format")
		}
		if cfg.SendKey == "" {
			return errors.New("serverchan channel requires send_key")
		}
		return nil
	case "feishu":
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid feishu config format")
		}
		if cfg.WebhookURL == "" {
			return errors.New("feishu channel requires webhook_url")
		}
		if !strings.HasPrefix(cfg.WebhookURL, "https://") {
			return errors.New("feishu webhook_url must start with https://")
		}
		return nil
	case "wecom":
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid wecom config format")
		}
		if cfg.WebhookURL == "" {
			return errors.New("wecom channel requires webhook_url")
		}
		if !strings.HasPrefix(cfg.WebhookURL, "https://") {
			return errors.New("wecom webhook_url must start with https://")
		}
		return nil
	case "dingtalk":
		var cfg struct {
			WebhookURL string `json:"webhook_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid dingtalk config format")
		}
		if cfg.WebhookURL == "" {
			return errors.New("dingtalk channel requires webhook_url")
		}
		if !strings.HasPrefix(cfg.WebhookURL, "https://") {
			return errors.New("dingtalk webhook_url must start with https://")
		}
		return nil
	case "pushdeer":
		var cfg struct {
			PushKey   string `json:"push_key"`
			ServerURL string `json:"server_url"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid pushdeer config format")
		}
		if strings.TrimSpace(cfg.PushKey) == "" {
			return errors.New("pushdeer channel requires push_key")
		}
		serverURL := strings.TrimSpace(cfg.ServerURL)
		if serverURL != "" && !strings.HasPrefix(serverURL, "http://") && !strings.HasPrefix(serverURL, "https://") {
			return errors.New("pushdeer server_url must start with http:// or https://")
		}
		return nil
	case "pushplus":
		var cfg struct {
			Token    string `json:"token"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid pushplus config format")
		}
		if strings.TrimSpace(cfg.Token) == "" {
			return errors.New("pushplus channel requires token")
		}
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint != "" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			return errors.New("pushplus endpoint must start with http:// or https://")
		}
		return nil
	case "pushover":
		var cfg struct {
			Token    string `json:"token"`
			User     string `json:"user"`
			Endpoint string `json:"endpoint"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid pushover config format")
		}
		if strings.TrimSpace(cfg.Token) == "" {
			return errors.New("pushover channel requires token")
		}
		if strings.TrimSpace(cfg.User) == "" {
			return errors.New("pushover channel requires user")
		}
		endpoint := strings.TrimSpace(cfg.Endpoint)
		if endpoint != "" && !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
			return errors.New("pushover endpoint must start with http:// or https://")
		}
		return nil
	case "napcat":
		var cfg struct {
			URL         string `json:"url"`
			MessageType string `json:"message_type"`
			UserID      string `json:"user_id"`
			GroupID     string `json:"group_id"`
		}
		if err := json.Unmarshal([]byte(config), &cfg); err != nil {
			return errors.New("invalid napcat config format")
		}
		if strings.TrimSpace(cfg.URL) == "" {
			return errors.New("napcat channel requires url")
		}
		napcatURL := strings.TrimSpace(cfg.URL)
		if !strings.HasPrefix(napcatURL, "http://") && !strings.HasPrefix(napcatURL, "https://") {
			return errors.New("napcat url must start with http:// or https://")
		}
		msgType := strings.ToLower(strings.TrimSpace(cfg.MessageType))
		if msgType == "" {
			msgType = "private"
		}
		if msgType != "private" && msgType != "group" {
			return errors.New("napcat message_type must be private or group")
		}
		if msgType == "private" && strings.TrimSpace(cfg.UserID) == "" {
			return errors.New("napcat channel requires user_id for private messages")
		}
		if msgType == "group" && strings.TrimSpace(cfg.GroupID) == "" {
			return errors.New("napcat channel requires group_id for group messages")
		}
		return nil
	default:
		return errors.New("unsupported channel type")
	}
}
