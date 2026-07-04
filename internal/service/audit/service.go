package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

const (
	TransportMCP = "mcp"

	StatusSuccess = "success"
	StatusError   = "error"

	ResourceSubscription = "subscription"

	defaultKeyKind   = "mcp_client"
	defaultScopeUsed = "write"

	maxJSONBytes  = 8 << 10
	maxErrorBytes = 2 << 10
)

type Service struct {
	DB *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	clone := *s
	if s.DB != nil {
		clone.DB = s.DB.WithContext(ctx)
	}
	return &clone
}

type CreateEventInput struct {
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

type EventFilter struct {
	UserID       *uint
	Limit        int
	Before       *time.Time
	Status       string
	ResourceType string
}

func (s *Service) IsEnabled() (bool, error) {
	return settings.GetBool(context.Background(), s.DB, "audit_enabled", true)
}

func (s *Service) Create(input CreateEventInput) (*model.AuditEvent, error) {
	eventID, err := generateEventID()
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
		Error:               truncateString(input.Error, maxErrorBytes),
		LatencyMS:           input.LatencyMS,
		ClientName:          truncateString(input.ClientName, 120),
		ClientVersion:       truncateString(input.ClientVersion, 80),
		RequestID:           truncateString(input.RequestID, 120),
		RequestArgsRedacted: marshalCappedJSON(redactValue(input.RequestArgsRedacted), maxJSONBytes),
		BeforeSnapshot:      marshalCappedJSON(input.BeforeSnapshot, maxJSONBytes),
		AfterSnapshot:       marshalCappedJSON(input.AfterSnapshot, maxJSONBytes),
	}

	if event.KeyKind == "" {
		event.KeyKind = defaultKeyKind
	}
	if event.ScopeUsed == "" {
		event.ScopeUsed = defaultScopeUsed
	}
	if event.Transport == "" {
		event.Transport = TransportMCP
	}
	if event.Status == "" {
		event.Status = StatusSuccess
	}

	if err := s.DB.Create(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func (s *Service) List(filter EventFilter) ([]model.AuditEvent, error) {
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

func generateEventID() (string, error) {
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

func redactValue(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		redacted := make(map[string]interface{}, len(typed))
		for key, item := range typed {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "password") || strings.Contains(lower, "secret") || strings.Contains(lower, "token") || strings.Contains(lower, "key") {
				redacted[key] = "[redacted]"
				continue
			}
			redacted[key] = redactValue(item)
		}
		return redacted
	case []interface{}:
		redacted := make([]interface{}, len(typed))
		for i, item := range typed {
			redacted[i] = redactValue(item)
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
