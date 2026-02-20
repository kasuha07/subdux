package model

import "time"

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null;size:255" json:"email"`
	Password  string    `gorm:"not null" json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
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
	Icon            string    `gorm:"size:100" json:"icon"`
	URL             string    `json:"url"`
	Notes           string    `json:"notes"`
	Status          string    `gorm:"size:20;default:'active'" json:"status"` // active, paused, cancelled
	Color           string    `gorm:"size:20" json:"color"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
