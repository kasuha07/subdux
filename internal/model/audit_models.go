package model

import "time"

type AuditEvent struct {
	EventID             string    `gorm:"primaryKey;size:36" json:"event_id"`
	OccurredAt          time.Time `gorm:"not null;index;index:idx_audit_user_occurred,priority:2" json:"occurred_at"`
	UserID              uint      `gorm:"not null;index:idx_audit_user_occurred,priority:1" json:"user_id"`
	KeyID               uint      `gorm:"not null;index" json:"key_id"`
	KeyKind             string    `gorm:"not null;size:30" json:"key_kind"`
	ScopeUsed           string    `gorm:"not null;size:20" json:"scope_used"`
	Transport           string    `gorm:"not null;size:20;index" json:"transport"`
	ToolName            string    `gorm:"not null;size:100" json:"tool_name"`
	ResourceType        string    `gorm:"not null;size:50;index" json:"resource_type"`
	ResourceID          string    `gorm:"size:64" json:"resource_id"`
	Action              string    `gorm:"not null;size:50" json:"action"`
	Status              string    `gorm:"not null;size:20;index" json:"status"`
	Error               string    `gorm:"size:2048" json:"error"`
	LatencyMS           int64     `gorm:"not null;default:0" json:"latency_ms"`
	ClientName          string    `gorm:"size:120" json:"client_name"`
	ClientVersion       string    `gorm:"size:80" json:"client_version"`
	RequestID           string    `gorm:"size:120" json:"request_id"`
	RequestArgsRedacted string    `gorm:"type:text" json:"request_args_redacted"`
	BeforeSnapshot      string    `gorm:"type:text" json:"before_snapshot"`
	AfterSnapshot       string    `gorm:"type:text" json:"after_snapshot"`
	CreatedAt           time.Time `json:"created_at"`
}
