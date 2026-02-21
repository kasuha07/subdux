package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type NotificationService struct {
	DB *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{DB: db}
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
		return nil, errors.New("invalid channel type, must be one of: smtp, resend, telegram, webhook")
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

	testSubName := "Test Subscription"
	testBillingDate := time.Now().AddDate(0, 0, 3).Format("2006-01-02")

	switch channel.Type {
	case "smtp":
		return s.sendSMTP(channel, user.Email, testSubName, testBillingDate)
	case "resend":
		return s.sendResend(channel, testSubName, testBillingDate)
	case "telegram":
		return s.sendTelegram(channel, testSubName, testBillingDate)
	case "webhook":
		return s.sendWebhook(channel, testSubName, testBillingDate)
	case "gotify":
		return s.sendGotify(channel, testSubName, testBillingDate)
	case "ntfy":
		return s.sendNtfy(channel, testSubName, testBillingDate)
	case "bark":
		return s.sendBark(channel, testSubName, testBillingDate)
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
		if *input.DaysBefore < 0 || *input.DaysBefore > 90 {
			return nil, errors.New("days_before must be between 0 and 90")
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

func (s *NotificationService) ProcessPendingNotifications() error {
	today := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.UTC)

	var channelUserIDs []uint
	if err := s.DB.Model(&model.NotificationChannel{}).
		Where("enabled = ?", true).
		Distinct("user_id").
		Pluck("user_id", &channelUserIDs).Error; err != nil {
		return fmt.Errorf("failed to query notification channels: %w", err)
	}

	for _, userID := range channelUserIDs {
		if err := s.processUserNotifications(userID, today); err != nil {
			fmt.Printf("notification error for user %d: %v\n", userID, err)
		}
	}

	return nil
}

func (s *NotificationService) processUserNotifications(userID uint, today time.Time) error {
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

		billingDate := time.Date(
			sub.NextBillingDate.Year(), sub.NextBillingDate.Month(), sub.NextBillingDate.Day(),
			0, 0, 0, 0, time.UTC,
		)

		shouldNotify := false
		daysUntilBilling := int(billingDate.Sub(today).Hours() / 24)

		if daysUntilBilling == daysBefore && daysBefore > 0 {
			shouldNotify = true
		}
		if daysUntilBilling == 0 && notifyOnDueDay {
			shouldNotify = true
		}

		if !shouldNotify {
			continue
		}

		billingDateStr := billingDate.Format("2006-01-02")

		for _, channel := range enabledChannels {
			var count int64
			s.DB.Model(&model.NotificationLog{}).
				Where("subscription_id = ? AND channel_type = ? AND notify_date = ? AND status = ?",
					sub.ID, channel.Type, billingDate, "sent").
				Count(&count)
			if count > 0 {
				continue
			}

			var sendErr error
			switch channel.Type {
			case "smtp":
				sendErr = s.sendSMTP(channel, user.Email, sub.Name, billingDateStr)
			case "resend":
				sendErr = s.sendResend(channel, sub.Name, billingDateStr)
			case "telegram":
				sendErr = s.sendTelegram(channel, sub.Name, billingDateStr)
			case "webhook":
				sendErr = s.sendWebhook(channel, sub.Name, billingDateStr)
			case "gotify":
				sendErr = s.sendGotify(channel, sub.Name, billingDateStr)
			case "ntfy":
				sendErr = s.sendNtfy(channel, sub.Name, billingDateStr)
			case "bark":
				sendErr = s.sendBark(channel, sub.Name, billingDateStr)
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

func (s *NotificationService) sendSMTP(channel model.NotificationChannel, _, subName, billingDate string) error {
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

	subject := fmt.Sprintf("Subscription Reminder: %s", subName)
	body := fmt.Sprintf(
		"This is a reminder that your subscription \"%s\" is due on %s.\r\n\r\nSent by Subdux.",
		subName, billingDate,
	)
	message := buildSMTPMessage(rtCfg.FromEmail, rtCfg.FromName, cfg.ToEmail, subject, body)

	return sendSMTPMessage(rtCfg, cfg.ToEmail, message)
}

func (s *NotificationService) sendResend(channel model.NotificationChannel, subName, billingDate string) error {
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
		"subject": fmt.Sprintf("Subscription Reminder: %s", subName),
		"text": fmt.Sprintf(
			"This is a reminder that your subscription \"%s\" is due on %s.\n\nSent by Subdux.",
			subName, billingDate,
		),
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

func (s *NotificationService) sendTelegram(channel model.NotificationChannel, subName, billingDate string) error {
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

	text := fmt.Sprintf(
		"📋 *Subscription Reminder*\n\n*%s* is due on *%s*\n\n_Sent by Subdux_",
		subName, billingDate,
	)

	payload := map[string]interface{}{
		"chat_id":    cfg.ChatID,
		"text":       text,
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

func (s *NotificationService) sendWebhook(channel model.NotificationChannel, subName, billingDate string) error {
	var cfg struct {
		URL    string `json:"url"`
		Secret string `json:"secret"`
		Method string `json:"method"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &cfg); err != nil {
		return errors.New("invalid webhook config")
	}
	if cfg.URL == "" {
		return errors.New("webhook config requires url")
	}

	method := strings.ToUpper(strings.TrimSpace(cfg.Method))
	if method == "" {
		method = http.MethodPost
	}

	payload := map[string]interface{}{
		"event":        "subscription.reminder",
		"subscription": subName,
		"billing_date": billingDate,
		"sent_at":      time.Now().UTC().Format(time.RFC3339),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(method, cfg.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

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

func (s *NotificationService) sendGotify(channel model.NotificationChannel, subName, billingDate string) error {
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
		"title":    fmt.Sprintf("Subscription Reminder: %s", subName),
		"message":  fmt.Sprintf("Your subscription \"%s\" is due on %s.\n\nSent by Subdux.", subName, billingDate),
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

func (s *NotificationService) sendNtfy(channel model.NotificationChannel, subName, billingDate string) error {
	var cfg struct {
		URL      string `json:"url"`
		Topic    string `json:"topic"`
		Token    string `json:"token"`
		Username string `json:"username"`
		Password string `json:"password"`
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
	message := fmt.Sprintf("Your subscription \"%s\" is due on %s.\n\nSent by Subdux.", subName, billingDate)

	req, err := http.NewRequest(http.MethodPost, endpoint, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Title", fmt.Sprintf("Subscription Reminder: %s", subName))
	req.Header.Set("Priority", "default")
	req.Header.Set("Tags", "calendar")

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

func (s *NotificationService) sendBark(channel model.NotificationChannel, subName, billingDate string) error {
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
		"title": fmt.Sprintf("Subscription Reminder: %s", subName),
		"body":  fmt.Sprintf("Your subscription \"%s\" is due on %s.\n\nSent by Subdux.", subName, billingDate),
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

func isValidChannelType(t string) bool {
	switch t {
	case "smtp", "resend", "telegram", "webhook", "gotify", "ntfy", "bark":
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
			URL string `json:"url"`
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
	default:
		return errors.New("unsupported channel type")
	}
}
