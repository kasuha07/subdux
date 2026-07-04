package service

import (
	auditservice "github.com/kasuha07/subdux/internal/service/audit"
	"gorm.io/gorm"
)

const (
	AuditTransportMCP = auditservice.TransportMCP

	AuditStatusSuccess = auditservice.StatusSuccess
	AuditStatusError   = auditservice.StatusError

	AuditResourceSubscription = auditservice.ResourceSubscription
)

type AuditService = auditservice.Service

func NewAuditService(db *gorm.DB) *AuditService {
	return auditservice.NewService(db)
}

type CreateAuditEventInput = auditservice.CreateEventInput
type AuditEventFilter = auditservice.EventFilter
