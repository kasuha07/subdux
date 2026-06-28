package model

import "time"

// MCPIdempotencyKey records the outcome of an MCP write tool call so that a
// retried or replayed request with the same idempotency key returns the
// original result instead of executing the mutation a second time.
//
// The key is scoped per user via the composite unique index so two users may
// independently reuse the same client-supplied string. RequestHash is a
// fingerprint of the tool name and arguments (excluding the key itself); a
// matching key with a different fingerprint signals the key was reused for a
// different request and is rejected rather than silently replayed.
type MCPIdempotencyKey struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;uniqueIndex:idx_mcp_idempotency_user_key,priority:1" json:"user_id"`
	IdempotencyKey string    `gorm:"not null;size:255;uniqueIndex:idx_mcp_idempotency_user_key,priority:2" json:"idempotency_key"`
	KeyID          uint      `gorm:"not null;index" json:"key_id"`
	ToolName       string    `gorm:"not null;size:100" json:"tool_name"`
	RequestHash    string    `gorm:"not null;size:64" json:"request_hash"`
	ResourceType   string    `gorm:"size:50" json:"resource_type"`
	ResourceID     string    `gorm:"size:64" json:"resource_id"`
	Response       string    `gorm:"type:text" json:"response"`
	CreatedAt      time.Time `gorm:"not null;index" json:"created_at"`
	User           *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
