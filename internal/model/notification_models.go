package model

import "time"

type NotificationChannel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Type      string    `gorm:"not null;size:20" json:"type"`
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	Config    string    `gorm:"type:text" json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	User      *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type NotificationPolicy struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	DaysBefore     int       `gorm:"default:3;check:chk_notification_policies_days_before,days_before >= 0 AND days_before <= 10" json:"days_before"`
	NotifyOnDueDay bool      `gorm:"default:true" json:"notify_on_due_day"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	User           *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type NotificationLog struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	UserID         uint          `gorm:"index;not null" json:"user_id"`
	SubscriptionID uint          `gorm:"index;not null" json:"subscription_id"`
	ChannelType    string        `gorm:"not null;size:20" json:"channel_type"`
	NotifyDate     time.Time     `gorm:"not null;index" json:"notify_date"`
	Status         string        `gorm:"not null;size:20" json:"status"`
	Error          string        `gorm:"type:text" json:"error"`
	SentAt         time.Time     `json:"sent_at"`
	User           *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Subscription   *Subscription `gorm:"foreignKey:SubscriptionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

// NotificationTemplate stores user-customizable notification message templates
type NotificationTemplate struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	ChannelType *string   `gorm:"size:20;index" json:"channel_type"`
	Format      string    `gorm:"size:20;not null;default:'plaintext'" json:"format"`
	Template    string    `gorm:"type:text;not null" json:"template"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	User        *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}
