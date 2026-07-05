package admin

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	servicesmtp "github.com/kasuha07/subdux/internal/service/smtp"
)

func (s *Service) SendSMTPTestEmail(userID uint, recipientOverride string) error {
	cfg, err := servicesmtp.LoadRuntimeConfig(s.DB)
	if err != nil {
		return normalizeSMTPTestError(err)
	}

	recipient := strings.TrimSpace(recipientOverride)
	if recipient == "" {
		var user model.User
		if err := s.DB.Select("email").First(&user, userID).Error; err != nil {
			return serviceerr.Wrap(
				serviceerr.KindInternal,
				"smtp_test_recipient_lookup_failed",
				"failed to load current user email",
				err,
			)
		}
		recipient = strings.TrimSpace(user.Email)
	}

	if recipient == "" {
		return serviceerr.New(serviceerr.KindInvalid, "recipient_email_is_required_for_smtp_test", "recipient email is required for smtp test")
	}

	if _, err := mail.ParseAddress(recipient); err != nil {
		return serviceerr.New(serviceerr.KindInvalid, "invalid_recipient_email", "invalid recipient email")
	}

	subject := "Subdux SMTP Test"
	body := fmt.Sprintf("This is a test email from Subdux.\r\nSent at: %s", pkg.Now().Format(time.RFC3339))
	message := servicesmtp.BuildMessage(cfg.FromEmail, cfg.FromName, recipient, subject, body)

	if err := servicesmtp.Send(*cfg, recipient, message); err != nil {
		return normalizeSMTPTestError(err)
	}

	return nil
}

// Preserve the SMTP test route's client-facing contract under the centralized
// API error handler: SMTP misconfiguration and delivery/debug failures should
// stay visible as 4xx, while already-typed errors (for example rate limits)
// keep their original status mapping.
func normalizeSMTPTestError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := serviceerr.KindOf(err); ok {
		return err
	}
	return serviceerr.Wrap(serviceerr.KindInvalid, "smtp_test_failed", err.Error(), err)
}
