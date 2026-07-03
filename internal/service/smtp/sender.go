package smtp

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/smtp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
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

func NormalizeRateLimitSeconds(value int64) (int64, error) {
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

func BuildMessage(fromEmail string, fromName string, toEmail string, subject string, body string) []byte {
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

func Send(cfg RuntimeConfig, recipient string, message []byte) error {
	if err := ReserveRateLimitSlot(cfg); err != nil {
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

func ReserveRateLimitSlot(cfg RuntimeConfig) error {
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

func buildSMTPAuth(cfg RuntimeConfig) (smtp.Auth, error) {
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
