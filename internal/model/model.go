package model

import "time"

type User struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	Username       string    `gorm:"uniqueIndex;not null;size:255" json:"username"`
	Email          string    `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Password       string    `gorm:"not null" json:"-"`
	Role           string    `gorm:"size:20;default:'user'" json:"role"`
	Status         string    `gorm:"size:20;default:'active'" json:"status"`
	TotpSecret     *string   `gorm:"size:64" json:"-"`
	TotpEnabled    bool      `gorm:"default:false" json:"totp_enabled"`
	TotpTempSecret *string   `gorm:"size:64" json:"-"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserBackupCode struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	CodeHash  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
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
}

type SystemSetting struct {
	Key   string `gorm:"primaryKey;size:100" json:"key"`
	Value string `gorm:"size:500" json:"value"`
}

type Subscription struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserID          uint      `gorm:"index;not null" json:"user_id"`
	Name            string    `gorm:"not null;size:255" json:"name"`
	Amount          float64   `gorm:"not null" json:"amount"`
	Currency        string    `gorm:"not null;size:10;default:'USD'" json:"currency"`
	BillingCycle    string    `gorm:"not null;size:20" json:"billing_cycle"` // weekly, monthly, yearly
	NextBillingDate time.Time `json:"next_billing_date"`
	Category        string    `gorm:"size:100" json:"category"`
	CategoryID      *uint     `gorm:"index" json:"category_id"`
	Icon            string    `gorm:"size:500" json:"icon"`
	URL             string    `json:"url"`
	Notes           string    `json:"notes"`
	Status          string    `gorm:"size:20;default:'active'" json:"status"` // active, paused, cancelled
	Color           string    `gorm:"size:20" json:"color"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ExchangeRate struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	BaseCurrency   string    `gorm:"not null;size:10;uniqueIndex:idx_base_target" json:"base_currency"`
	TargetCurrency string    `gorm:"not null;size:10;uniqueIndex:idx_base_target" json:"target_currency"`
	Rate           float64   `gorm:"not null" json:"rate"`
	Source         string    `gorm:"not null;size:50" json:"source"` // "free" or "premium"
	FetchedAt      time.Time `gorm:"not null" json:"fetched_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserPreference struct {
	UserID            uint      `gorm:"primaryKey" json:"user_id"`
	PreferredCurrency string    `gorm:"size:10;default:'USD'" json:"preferred_currency"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type UserCurrency struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"not null;index;uniqueIndex:idx_user_currency" json:"user_id"`
	Code      string    `gorm:"not null;size:10;uniqueIndex:idx_user_currency" json:"code"`
	Symbol    string    `gorm:"size:10" json:"symbol"`
	Alias     string    `gorm:"size:100" json:"alias"`
	SortOrder int       `gorm:"default:0" json:"sort_order"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Category struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	UserID       uint      `gorm:"not null;index;uniqueIndex:idx_user_category_name" json:"user_id"`
	Name         string    `gorm:"not null;size:30;uniqueIndex:idx_user_category_name" json:"name"`
	DisplayOrder int       `gorm:"default:0" json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
