package model

import "time"

// MCPOAuthClient represents a public PKCE client. URL client IDs are CIMD
// documents; randomly generated IDs belong to dynamically registered clients.
type MCPOAuthClient struct {
	ID           string    `gorm:"primaryKey;size:2048" json:"client_id"`
	Name         string    `gorm:"not null;size:120" json:"client_name"`
	RedirectURIs string    `gorm:"type:text;not null" json:"-"`
	MetadataAt   time.Time `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"-"`
}

func (MCPOAuthClient) TableName() string { return "mcp_oauth_clients" }

// MCPOAuthRequest holds a short-lived authorization interaction and its
// single-use authorization code. Only hashes of the opaque handles are saved.
type MCPOAuthRequest struct {
	ID            string     `gorm:"primaryKey;size:64" json:"-"`
	ClientID      string     `gorm:"not null;size:2048" json:"-"`
	ClientName    string     `gorm:"not null;size:120" json:"-"`
	RedirectURI   string     `gorm:"type:text;not null" json:"-"`
	Scopes        string     `gorm:"not null;size:100" json:"-"`
	Resource      string     `gorm:"type:text;not null" json:"-"`
	State         string     `gorm:"type:text" json:"-"`
	CodeChallenge string     `gorm:"not null;size:43" json:"-"`
	UserID        *uint      `gorm:"index" json:"-"`
	CodeHash      *string    `gorm:"uniqueIndex;size:64" json:"-"`
	GrantID       *uint      `json:"-"`
	ExpiresAt     time.Time  `gorm:"not null;index" json:"-"`
	DecidedAt     *time.Time `json:"-"`
	ConsumedAt    *time.Time `json:"-"`
	CreatedAt     time.Time  `json:"-"`
	User          *User      `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (MCPOAuthRequest) TableName() string { return "mcp_oauth_requests" }

type MCPOAuthGrant struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"not null;index" json:"-"`
	ClientID   string     `gorm:"not null;size:2048" json:"client_id"`
	ClientName string     `gorm:"not null;size:120" json:"client_name"`
	Scopes     string     `gorm:"not null;size:100" json:"scope"`
	Resource   string     `gorm:"type:text;not null" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt  *time.Time `gorm:"index" json:"-"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
	User       User       `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (MCPOAuthGrant) TableName() string { return "mcp_oauth_grants" }

type MCPOAuthToken struct {
	Hash       string        `gorm:"primaryKey;size:64" json:"-"`
	GrantID    uint          `gorm:"not null;index" json:"-"`
	Kind       string        `gorm:"not null;size:10" json:"-"`
	ExpiresAt  time.Time     `gorm:"not null;index" json:"-"`
	ConsumedAt *time.Time    `json:"-"`
	CreatedAt  time.Time     `json:"-"`
	Grant      MCPOAuthGrant `gorm:"foreignKey:GrantID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (MCPOAuthToken) TableName() string { return "mcp_oauth_tokens" }
