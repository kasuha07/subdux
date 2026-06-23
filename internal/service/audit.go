package service

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"gorm.io/gorm"
)

const (
	AuditTransportMCP = "mcp"

	AuditStatusSuccess = "success"
	AuditStatusError   = "error"

	AuditResourceSubscription = "subscription"

	maxAuditJSONBytes  = 8 << 10
	maxAuditErrorBytes = 2 << 10
)

type AuditService struct {
	DB *gorm.DB
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{DB: db}
}

type CreateAuditEventInput struct {
	UserID              uint
	KeyID               uint
	KeyKind             string
	ScopeUsed           string
	Transport           string
	ToolName            string
	ResourceType        string
	ResourceID          string
	Action              string
	Status              string
	Error               string
	LatencyMS           int64
	ClientName          string
	ClientVersion       string
	RequestID           string
	RequestArgsRedacted interface{}
	BeforeSnapshot      interface{}
	AfterSnapshot       interface{}
}

type AuditEventFilter struct {
	UserID       *uint
	Limit        int
	Before       *time.Time
	Status       string
	ResourceType string
}

func (s *AuditService) IsEnabled() (bool, error) {
	return getBoolSystemSettingValue(s.DB, "audit_enabled", true)
}

func (s *AuditService) Create(input CreateAuditEventInput) (*model.AuditEvent, error) {
	eventID, err := generateAuditEventID()
	if err != nil {
		return nil, err
	}

	event := &model.AuditEvent{
		EventID:             eventID,
		OccurredAt:          pkg.NowUTC(),
		UserID:              input.UserID,
		KeyID:               input.KeyID,
		KeyKind:             strings.TrimSpace(input.KeyKind),
		ScopeUsed:           strings.TrimSpace(input.ScopeUsed),
		Transport:           strings.TrimSpace(input.Transport),
		ToolName:            strings.TrimSpace(input.ToolName),
		ResourceType:        strings.TrimSpace(input.ResourceType),
		ResourceID:          strings.TrimSpace(input.ResourceID),
		Action:              strings.TrimSpace(input.Action),
		Status:              strings.TrimSpace(input.Status),
		Error:               truncateString(input.Error, maxAuditErrorBytes),
		LatencyMS:           input.LatencyMS,
		ClientName:          truncateString(input.ClientName, 120),
		ClientVersion:       truncateString(input.ClientVersion, 80),
		RequestID:           truncateString(input.RequestID, 120),
		RequestArgsRedacted: marshalCappedJSON(redactAuditValue(input.RequestArgsRedacted), maxAuditJSONBytes),
		BeforeSnapshot:      marshalCappedJSON(input.BeforeSnapshot, maxAuditJSONBytes),
		AfterSnapshot:       marshalCappedJSON(input.AfterSnapshot, maxAuditJSONBytes),
	}

	if event.KeyKind == "" {
		event.KeyKind = APIKeyKindMCPClient
	}
	if event.ScopeUsed == "" {
		event.ScopeUsed = APIKeyScopeWrite
	}
	if event.Transport == "" {
		event.Transport = AuditTransportMCP
	}
	if event.Status == "" {
		event.Status = AuditStatusSuccess
	}

	if err := s.DB.Create(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (s *AuditService) List(filter AuditEventFilter) ([]model.AuditEvent, error) {
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}

	query := s.DB.Model(&model.AuditEvent{})
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	if filter.Before != nil {
		query = query.Where("occurred_at < ?", *filter.Before)
	}
	if strings.TrimSpace(filter.Status) != "" {
		query = query.Where("status = ?", strings.TrimSpace(filter.Status))
	}
	if strings.TrimSpace(filter.ResourceType) != "" {
		query = query.Where("resource_type = ?", strings.TrimSpace(filter.ResourceType))
	}

	var events []model.AuditEvent
	if err := query.Order("occurred_at DESC").Limit(limit).Find(&events).Error; err != nil {
		return nil, err
	}
	return events, nil
}

func generateAuditEventID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func marshalCappedJSON(value interface{}, maxBytes int) string {
	if value == nil {
		return ""
	}
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	if len(data) > maxBytes {
		return `{"truncated":true}`
	}
	return string(data)
}

func redactAuditValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		redacted := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "key") {
				redacted[key] = "[redacted]"
				continue
			}
			redacted[key] = redactAuditValue(item)
		}
		return redacted
	case []interface{}:
		redacted := make([]interface{}, len(typed))
		for i, item := range typed {
			redacted[i] = redactAuditValue(item)
		}
		return redacted
	default:
		return value
	}
}

func truncateString(value string, maxBytes int) string {
	if len(value) <= maxBytes {
		return value
	}
	return value[:maxBytes]
}
