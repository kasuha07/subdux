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
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

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
	resp, err := doNotificationRequest(client, req)
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
	requestURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", cfg.BotToken)
	req, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
	resp, err := doNotificationRequest(client, req)
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
