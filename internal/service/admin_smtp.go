package service

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type smtpRuntimeConfig struct {
	Host           string
	Port           int64
	Username       string
	Password       string
	FromEmail      string
	FromName       string
	Encryption     string
	AuthMethod     string
	HeloName       string
	TimeoutSeconds int64
	SkipTLSVerify  bool
}

type smtpLoginAuth struct {
	username string
	password string
}

func (a *smtpLoginAuth) Start(_ *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *smtpLoginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	prompt := strings.ToLower(strings.TrimSpace(string(fromServer)))
	switch {
	case strings.Contains(prompt, "username"):
		return []byte(a.username), nil
	case strings.Contains(prompt, "password"):
		return []byte(a.password), nil
	default:
		return nil, errors.New("unsupported smtp login challenge")
	}
}

func (s *AdminService) SendSMTPTestEmail(userID uint, recipientOverride string) error {
	cfg, err := loadSMTPRuntimeConfig(s.DB)
	if err != nil {
		return err
	}

	recipient := strings.TrimSpace(recipientOverride)
	if recipient == "" {
		var user model.User
		if err := s.DB.Select("email").First(&user, userID).Error; err != nil {
			return errors.New("failed to load current user email")
		}
		recipient = strings.TrimSpace(user.Email)
	}

	if recipient == "" {
		return errors.New("recipient email is required for smtp test")
	}

	if _, err := mail.ParseAddress(recipient); err != nil {
		return errors.New("invalid recipient email")
	}

	subject := "Subdux SMTP Test"
	body := fmt.Sprintf("This is a test email from Subdux.\r\nSent at: %s", time.Now().Format(time.RFC3339))
	message := buildSMTPMessage(cfg.FromEmail, cfg.FromName, recipient, subject, body)

	if err := sendSMTPMessage(*cfg, recipient, message); err != nil {
		return err
	}

	return nil
}

func (s *AdminService) loadSMTPRuntimeConfig() (*smtpRuntimeConfig, error) {
	return loadSMTPRuntimeConfig(s.DB)
}

func loadSMTPRuntimeConfig(db *gorm.DB) (*smtpRuntimeConfig, error) {
	if db == nil {
		return nil, errors.New("failed to load smtp settings")
	}

	defaults := map[string]string{
		"smtp_enabled":         "false",
		"smtp_host":            "",
		"smtp_port":            "587",
		"smtp_username":        "",
		"smtp_password":        "",
		"smtp_from_email":      "",
		"smtp_from_name":       "",
		"smtp_encryption":      "starttls",
		"smtp_auth_method":     "auto",
		"smtp_helo_name":       "",
		"smtp_timeout_seconds": "10",
		"smtp_skip_tls_verify": "false",
	}

	keys := make([]string, 0, len(defaults))
	values := make(map[string]string, len(defaults))
	for key, value := range defaults {
		keys = append(keys, key)
		values[key] = value
	}

	var items []model.SystemSetting
	if err := db.Where("key IN ?", keys).Find(&items).Error; err != nil {
		return nil, errors.New("failed to load smtp settings")
	}
	for _, item := range items {
		settingValue, err := decryptSystemSettingValueIfNeeded(item.Key, item.Value)
		if err != nil {
			return nil, errors.New("failed to decrypt smtp settings")
		}
		values[item.Key] = settingValue
	}

	if values["smtp_enabled"] != "true" {
		return nil, errors.New("smtp is disabled")
	}

	host := strings.TrimSpace(values["smtp_host"])
	if host == "" {
		return nil, errors.New("smtp host is required")
	}

	port, err := strconv.ParseInt(strings.TrimSpace(values["smtp_port"]), 10, 64)
	if err != nil || port < 1 || port > 65535 {
		return nil, errors.New("smtp port must be between 1 and 65535")
	}

	fromEmail := strings.TrimSpace(values["smtp_from_email"])
	if fromEmail == "" {
		return nil, errors.New("smtp from email is required")
	}

	encryption := strings.ToLower(strings.TrimSpace(values["smtp_encryption"]))
	if encryption == "" {
		encryption = "starttls"
	}
	switch encryption {
	case "starttls", "ssl_tls", "none":
	default:
		return nil, errors.New("unsupported smtp encryption mode")
	}

	authMethod := strings.ToLower(strings.TrimSpace(values["smtp_auth_method"]))
	if authMethod == "" {
		authMethod = "auto"
	}
	switch authMethod {
	case "auto", "plain", "login", "cram_md5", "none":
	default:
		return nil, errors.New("unsupported smtp auth method")
	}

	timeoutSeconds, err := strconv.ParseInt(strings.TrimSpace(values["smtp_timeout_seconds"]), 10, 64)
	if err != nil || timeoutSeconds <= 0 {
		timeoutSeconds = 10
	}

	username := strings.TrimSpace(values["smtp_username"])
	password := values["smtp_password"]

	if authMethod != "auto" && authMethod != "none" && (username == "" || strings.TrimSpace(password) == "") {
		return nil, errors.New("smtp username and password are required for selected auth method")
	}

	return &smtpRuntimeConfig{
		Host:           host,
		Port:           port,
		Username:       username,
		Password:       password,
		FromEmail:      fromEmail,
		FromName:       strings.TrimSpace(values["smtp_from_name"]),
		Encryption:     encryption,
		AuthMethod:     authMethod,
		HeloName:       strings.TrimSpace(values["smtp_helo_name"]),
		TimeoutSeconds: timeoutSeconds,
		SkipTLSVerify:  values["smtp_skip_tls_verify"] == "true",
	}, nil
}

func buildSMTPMessage(fromEmail string, fromName string, toEmail string, subject string, body string) []byte {
	escapedName := strings.ReplaceAll(fromName, "\"", "'")
	fromHeader := fromEmail
	if strings.TrimSpace(escapedName) != "" {
		fromHeader = fmt.Sprintf("\"%s\" <%s>", escapedName, fromEmail)
	}

	headers := []string{
		fmt.Sprintf("From: %s", fromHeader),
		fmt.Sprintf("To: %s", toEmail),
		fmt.Sprintf("Subject: %s", subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
	}

	return []byte(strings.Join(headers, "\r\n") + "\r\n\r\n" + body + "\r\n")
}

func sendSMTPMessage(cfg smtpRuntimeConfig, recipient string, message []byte) error {
	address := net.JoinHostPort(cfg.Host, strconv.FormatInt(cfg.Port, 10))
	dialer := net.Dialer{
		Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
	}

	var client *smtp.Client
	if cfg.Encryption == "ssl_tls" {
		conn, err := tls.DialWithDialer(&dialer, "tcp", address, &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: cfg.SkipTLSVerify,
		})
		if err != nil {
			return fmt.Errorf("failed to connect to smtp server: %w", err)
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to initialize smtp client: %w", err)
		}
	} else {
		conn, err := dialer.Dial("tcp", address)
		if err != nil {
			return fmt.Errorf("failed to connect to smtp server: %w", err)
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to initialize smtp client: %w", err)
		}
	}
	defer client.Close()

	if cfg.HeloName != "" {
		if err := client.Hello(cfg.HeloName); err != nil {
			return fmt.Errorf("smtp HELO/EHLO failed: %w", err)
		}
	}

	if cfg.Encryption == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("smtp server does not support STARTTLS")
		}
		if err := client.StartTLS(&tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: cfg.SkipTLSVerify,
		}); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	auth, err := buildSMTPAuth(cfg)
	if err != nil {
		return err
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("smtp authentication failed: %w", err)
		}
	}

	if err := client.Mail(cfg.FromEmail); err != nil {
		return fmt.Errorf("smtp MAIL FROM failed: %w", err)
	}
	if err := client.Rcpt(recipient); err != nil {
		return fmt.Errorf("smtp RCPT TO failed: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA failed: %w", err)
	}

	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("failed to write smtp message: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to finalize smtp message: %w", err)
	}

	if err := client.Quit(); err != nil {
		return fmt.Errorf("failed to close smtp session: %w", err)
	}

	return nil
}

func buildSMTPAuth(cfg smtpRuntimeConfig) (smtp.Auth, error) {
	username := strings.TrimSpace(cfg.Username)
	password := strings.TrimSpace(cfg.Password)

	switch cfg.AuthMethod {
	case "none":
		return nil, nil
	case "auto":
		if username == "" || password == "" {
			return nil, nil
		}
		return smtp.PlainAuth("", username, password, cfg.Host), nil
	case "plain":
		if username == "" || password == "" {
			return nil, errors.New("smtp username and password are required for PLAIN auth")
		}
		return smtp.PlainAuth("", username, password, cfg.Host), nil
	case "login":
		if username == "" || password == "" {
			return nil, errors.New("smtp username and password are required for LOGIN auth")
		}
		return &smtpLoginAuth{username: username, password: password}, nil
	case "cram_md5":
		if username == "" || password == "" {
			return nil, errors.New("smtp username and password are required for CRAM-MD5 auth")
		}
		return smtp.CRAMMD5Auth(username, password), nil
	default:
		return nil, errors.New("unsupported smtp auth method")
	}
}
