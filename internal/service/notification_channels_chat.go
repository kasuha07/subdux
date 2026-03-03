package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
)

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
