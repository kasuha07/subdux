package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
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
