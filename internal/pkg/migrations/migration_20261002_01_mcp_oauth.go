package migrations

import (
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
	"time"
)

// oauthClient20261002 represents a public PKCE client. URL client IDs are CIMD
// documents; randomly generated IDs belong to dynamically registered clients.
type oauthClient20261002 struct {
	ID           string    `gorm:"primaryKey;size:2048" json:"client_id"`
	Name         string    `gorm:"not null;size:120" json:"client_name"`
	RedirectURIs string    `gorm:"type:text;not null" json:"-"`
	MetadataAt   time.Time `gorm:"not null" json:"-"`
	CreatedAt    time.Time `json:"-"`
}

// oauthRequest20261002 holds a short-lived authorization interaction and its
// single-use authorization code. Only hashes of the opaque handles are saved.
type oauthRequest20261002 struct {
	ID            string      `gorm:"primaryKey;size:64" json:"-"`
	ClientID      string      `gorm:"not null;size:2048" json:"-"`
	ClientName    string      `gorm:"not null;size:120" json:"-"`
	RedirectURI   string      `gorm:"type:text;not null" json:"-"`
	Scopes        string      `gorm:"not null;size:100" json:"-"`
	Resource      string      `gorm:"type:text;not null" json:"-"`
	State         string      `gorm:"type:text" json:"-"`
	CodeChallenge string      `gorm:"not null;size:43" json:"-"`
	UserID        *uint       `gorm:"index" json:"-"`
	CodeHash      *string     `gorm:"uniqueIndex;size:64" json:"-"`
	GrantID       *uint       `json:"-"`
	ExpiresAt     time.Time   `gorm:"not null;index" json:"-"`
	DecidedAt     *time.Time  `json:"-"`
	ConsumedAt    *time.Time  `json:"-"`
	CreatedAt     time.Time   `json:"-"`
	User          *model.User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type oauthGrant20261002 struct {
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
	User       model.User `gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type oauthToken20261002 struct {
	Hash       string             `gorm:"primaryKey;size:64" json:"-"`
	GrantID    uint               `gorm:"not null;index" json:"-"`
	Kind       string             `gorm:"not null;size:10" json:"-"`
	ExpiresAt  time.Time          `gorm:"not null;index" json:"-"`
	ConsumedAt *time.Time         `json:"-"`
	CreatedAt  time.Time          `json:"-"`
	Grant      oauthGrant20261002 `gorm:"foreignKey:GrantID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (oauthClient20261002) TableName() string { return "mcp_oauth_clients" }

func (oauthRequest20261002) TableName() string { return "mcp_oauth_requests" }

func (oauthGrant20261002) TableName() string { return "mcp_oauth_grants" }

func (oauthToken20261002) TableName() string { return "mcp_oauth_tokens" }

type oauthAuditColumns20261002 struct {
	OAuthGrantID  *uint  `gorm:"index"`
	OAuthClientID string `gorm:"size:2048"`
}

func (oauthAuditColumns20261002) TableName() string { return "audit_events" }

type oauthIdempotencyColumns20261002 struct {
	OAuthGrantID *uint `gorm:"index"`
}

func (oauthIdempotencyColumns20261002) TableName() string { return "mcp_idempotency_keys" }
func migrateMCPOAuth(db *gorm.DB) error {
	if err := db.AutoMigrate(&oauthClient20261002{}, &oauthRequest20261002{}, &oauthGrant20261002{}, &oauthToken20261002{}); err != nil {
		return err
	}
	for _, change := range []struct {
		model   interface{}
		columns []string
	}{
		{&oauthAuditColumns20261002{}, []string{"OAuthGrantID", "OAuthClientID"}},
		{&oauthIdempotencyColumns20261002{}, []string{"OAuthGrantID"}},
	} {
		for _, column := range change.columns {
			if !db.Migrator().HasColumn(change.model, column) {
				if err := db.Migrator().AddColumn(change.model, column); err != nil {
					return err
				}
			}
		}
		if !db.Migrator().HasIndex(change.model, "OAuthGrantID") {
			if err := db.Migrator().CreateIndex(change.model, "OAuthGrantID"); err != nil {
				return err
			}
		}
	}
	return nil
}
