package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shiroha/subdux/internal/pkg"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

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

	sentAt := pkg.NowUTC().Format(time.RFC3339)
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
