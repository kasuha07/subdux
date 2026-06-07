package model

import "time"

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Username       string    `gorm:"uniqueIndex;not null;size:255" json:"username"`
	Email          string    `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Password       string    `gorm:"not null" json:"-"`
	Role           string    `gorm:"size:20;default:'user';check:chk_users_role,role IN ('admin','user')" json:"role"`
	Status         string    `gorm:"size:20;default:'active';check:chk_users_status,status IN ('active','disabled')" json:"status"`
	TotpSecret     *string   `gorm:"size:64" json:"-"`
	TotpEnabled    bool      `gorm:"default:false" json:"totp_enabled"`
	TotpTempSecret *string   `gorm:"size:64" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type EmailVerificationCode struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	UserID         *uint      `gorm:"index;default:null" json:"user_id"`
	Email          string     `gorm:"size:255;not null;index:idx_email_verification_lookup,priority:2" json:"email"`
	Purpose        string     `gorm:"size:50;not null;index:idx_email_verification_lookup,priority:1" json:"purpose"`
	CodeHash       string     `gorm:"not null" json:"-"`
	FailedAttempts int        `gorm:"default:0" json:"-"`
	ExpiresAt      time.Time  `gorm:"not null;index" json:"expires_at"`
	ConsumedAt     *time.Time `gorm:"index" json:"consumed_at"`
	CreatedAt      time.Time  `json:"created_at"`
	User           *User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
}

type UserBackupCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	CodeHash  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type PasskeyCredential struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	UserID       uint       `gorm:"index;not null" json:"user_id"`
	Name         string     `gorm:"size:255;not null" json:"name"`
	CredentialID string     `gorm:"size:1024;not null;uniqueIndex:idx_passkey_credential_id" json:"credential_id"`
	Credential   []byte     `gorm:"type:blob;not null" json:"-"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	User         *User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type OIDCConnection struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null;uniqueIndex:idx_oidc_user_provider" json:"user_id"`
	Provider  string    `gorm:"size:100;not null;uniqueIndex:idx_oidc_user_provider;uniqueIndex:idx_oidc_provider_subject" json:"provider"`
	Subject   string    `gorm:"size:255;not null;uniqueIndex:idx_oidc_provider_subject" json:"subject"`
	Email     string    `gorm:"size:255" json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type APIKey struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	Name       string     `gorm:"not null;size:100" json:"name"`
	KeyHash    string     `gorm:"not null;uniqueIndex:idx_api_key_hash" json:"-"`
	Prefix     string     `gorm:"not null;size:12" json:"prefix"`
	Scopes     string     `gorm:"type:text;not null;default:'read,write'" json:"-"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	User       *User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type RefreshToken struct {
	ID         uint       `gorm:"primaryKey" json:"id"`
	UserID     uint       `gorm:"index;not null" json:"user_id"`
	TokenHash  string     `gorm:"not null;uniqueIndex:idx_refresh_token_hash;size:64" json:"-"`
	ExpiresAt  time.Time  `gorm:"not null;index" json:"expires_at"`
	LastUsedAt *time.Time `json:"last_used_at"`
	RevokedAt  *time.Time `gorm:"index" json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	User       *User      `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
