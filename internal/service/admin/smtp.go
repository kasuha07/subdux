package admin

import (
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	servicesmtp "github.com/kasuha07/subdux/internal/service/smtp"
)

func (s *Service) SendSMTPTestEmail(userID uint, recipientOverride string) error {
	cfg, err := servicesmtp.LoadRuntimeConfig(s.DB)
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
	message := servicesmtp.BuildMessage(cfg.FromEmail, cfg.FromName, recipient, subject, body)

	if err := servicesmtp.Send(*cfg, recipient, message); err != nil {
		return err
	}

	return nil
}
