package model

import "time"

type SystemSetting struct {
	Key   string `gorm:"primaryKey;size:100" json:"key"`
	Value string `gorm:"size:500" json:"value"`
}

type ExchangeRate struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	BaseCurrency   string    `gorm:"not null;size:10;uniqueIndex:idx_base_target" json:"base_currency"`
	TargetCurrency string    `gorm:"not null;size:10;uniqueIndex:idx_base_target" json:"target_currency"`
	Rate           float64   `gorm:"not null" json:"rate"`
	Source         string    `gorm:"not null;size:50" json:"source"`
	FetchedAt      time.Time `gorm:"not null" json:"fetched_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UserPreference struct {
	UserID            uint      `gorm:"primaryKey" json:"user_id"`
	PreferredCurrency string    `gorm:"size:10;default:'USD'" json:"preferred_currency"`
	UpdatedAt         time.Time `json:"updated_at"`
	User              *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
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
	User      *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
