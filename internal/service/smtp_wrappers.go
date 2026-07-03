package service

import (
	"github.com/kasuha07/subdux/internal/service/smtp"
	"gorm.io/gorm"
)

type smtpRuntimeConfig = smtp.RuntimeConfig

var (
	ErrInvalidSMTPRateLimit = smtp.ErrInvalidSMTPRateLimit
	ErrSMTPRateLimited      = smtp.ErrSMTPRateLimited
)

func (s *AdminService) loadSMTPRuntimeConfig() (*smtpRuntimeConfig, error) {
	return smtp.LoadRuntimeConfig(s.DB)
}

func loadSMTPRuntimeConfig(db *gorm.DB) (*smtpRuntimeConfig, error) {
	return smtp.LoadRuntimeConfig(db)
}

func normalizeSMTPRateLimitSeconds(value int64) (int64, error) {
	return smtp.NormalizeRateLimitSeconds(value)
}

func buildSMTPMessage(fromEmail string, fromName string, toEmail string, subject string, body string) []byte {
	return smtp.BuildMessage(fromEmail, fromName, toEmail, subject, body)
}

func sendSMTPMessage(cfg smtpRuntimeConfig, recipient string, message []byte) error {
	return smtp.Send(cfg, recipient, message)
}

func reserveSMTPRateLimitSlot(cfg smtpRuntimeConfig) error {
	return smtp.ReserveRateLimitSlot(cfg)
}
