package service

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"github.com/shiroha/subdux/internal/pkg"
	"net"
	"net/mail"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

const (
	smtpRateLimitLastAttemptKey = "smtp_rate_limit_last_attempt_at"
	smtpRateLimitMaxSeconds     = 86400
)

var (
	ErrInvalidSMTPRateLimit = errors.New("smtp rate limit must be between 0 and 86400 seconds")
	ErrSMTPRateLimited      = errors.New("smtp send rate limit exceeded, please wait before trying again")
	smtpRateLimitMu         sync.Mutex
)

type smtpRuntimeConfig struct {
	Host             string
	Port             int64
	Username         string
	Password         string
	FromEmail        string
	FromName         string
	Encryption       string
	AuthMethod       string
	HeloName         string
	TimeoutSeconds   int64
	RateLimitSeconds int64
	RateLimitDB      *gorm.DB
	SkipTLSVerify    bool
	DialContext      func(context.Context, string, string) (net.Conn, error)
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
	body := fmt.Sprintf("This is a test email from Subdux.\r\nSent at: %s", pkg.Now().Format(time.RFC3339))
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
		"smtp_enabled":            "false",
		"smtp_host":               "",
		"smtp_port":               "587",
		"smtp_username":           "",
		"smtp_password":           "",
		"smtp_from_email":         "",
		"smtp_from_name":          "",
		"smtp_encryption":         "starttls",
		"smtp_auth_method":        "auto",
		"smtp_helo_name":          "",
		"smtp_timeout_seconds":    "10",
		"smtp_rate_limit_seconds": "0",
		"smtp_skip_tls_verify":    "false",
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

	rateLimitSeconds := parseSMTPRateLimitSeconds(values["smtp_rate_limit_seconds"])
	username := strings.TrimSpace(values["smtp_username"])
	password := values["smtp_password"]

	if authMethod != "auto" && authMethod != "none" && (username == "" || strings.TrimSpace(password) == "") {
		return nil, errors.New("smtp username and password are required for selected auth method")
	}

	return &smtpRuntimeConfig{
		Host:             host,
		Port:             port,
		Username:         username,
		Password:         password,
		FromEmail:        fromEmail,
		FromName:         strings.TrimSpace(values["smtp_from_name"]),
		Encryption:       encryption,
		AuthMethod:       authMethod,
		HeloName:         strings.TrimSpace(values["smtp_helo_name"]),
		TimeoutSeconds:   timeoutSeconds,
		RateLimitSeconds: rateLimitSeconds,
		RateLimitDB:      db,
		SkipTLSVerify:    values["smtp_skip_tls_verify"] == "true",
		DialContext:      NewOutboundDialContext(db, time.Duration(timeoutSeconds)*time.Second),
	}, nil
}

func normalizeSMTPRateLimitSeconds(value int64) (int64, error) {
	if value < 0 || value > smtpRateLimitMaxSeconds {
		return 0, ErrInvalidSMTPRateLimit
	}
	return value, nil
}

func parseSMTPRateLimitSeconds(raw string) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	if value > smtpRateLimitMaxSeconds {
		return smtpRateLimitMaxSeconds
	}
	return value
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
	if err := reserveSMTPRateLimitSlot(cfg); err != nil {
		return err
	}

	address := net.JoinHostPort(cfg.Host, strconv.FormatInt(cfg.Port, 10))
	dialContext := cfg.DialContext
	if dialContext == nil {
		dialer := net.Dialer{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		}
		dialContext = dialer.DialContext
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.TimeoutSeconds)*time.Second)
	defer cancel()

	var client *smtp.Client
	if cfg.Encryption == "ssl_tls" {
		rawConn, err := dialContext(ctx, "tcp", address)
		if err != nil {
			return fmt.Errorf("failed to connect to smtp server: %w", err)
		}
		conn := tls.Client(rawConn, &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: cfg.SkipTLSVerify, // #nosec G402 -- Not a security issue: default false; admin-only compatibility switch for trusted self-signed SMTP servers.
		})
		if err := conn.HandshakeContext(ctx); err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to connect to smtp server: %w", err)
		}
		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			_ = conn.Close()
			return fmt.Errorf("failed to initialize smtp client: %w", err)
		}
	} else {
		conn, err := dialContext(ctx, "tcp", address)
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
			InsecureSkipVerify: cfg.SkipTLSVerify, // #nosec G402 -- Not a security issue: default false; admin-only compatibility switch for trusted self-signed SMTP servers.
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

func reserveSMTPRateLimitSlot(cfg smtpRuntimeConfig) error {
	if cfg.RateLimitSeconds <= 0 || cfg.RateLimitDB == nil {
		return nil
	}

	smtpRateLimitMu.Lock()
	defer smtpRateLimitMu.Unlock()

	now := pkg.NowUTC()
	interval := time.Duration(cfg.RateLimitSeconds) * time.Second

	return cfg.RateLimitDB.Transaction(func(tx *gorm.DB) error {
		var setting model.SystemSetting
		err := tx.Where("key = ?", smtpRateLimitLastAttemptKey).First(&setting).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		if err == nil {
			lastAttempt, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(setting.Value))
			if parseErr == nil && now.Sub(lastAttempt) < interval {
				return ErrSMTPRateLimited
			}
		}

		return tx.Where("key = ?", smtpRateLimitLastAttemptKey).
			Assign(model.SystemSetting{Value: now.Format(time.RFC3339Nano)}).
			FirstOrCreate(&model.SystemSetting{Key: smtpRateLimitLastAttemptKey}).Error
	})
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
