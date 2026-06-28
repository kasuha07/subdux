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
	ID                     uint      `gorm:"primaryKey" json:"id"`
	UserID                 uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	DaysBefore             int       `gorm:"default:3;check:chk_notification_policies_days_before,days_before >= 0 AND days_before <= 10" json:"days_before"`
	NotifyOnDueDay         bool      `gorm:"default:true" json:"notify_on_due_day"`
	NotifyManualRenewDaily bool      `gorm:"default:false" json:"notify_manual_renew_daily"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	User                   *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type NotificationLog struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	OutboxID       *uint         `gorm:"index" json:"outbox_id"`
	UserID         uint          `gorm:"index;not null;index:idx_notification_logs_user_status_sent,priority:1;index:idx_notification_logs_user_sub_channel_sent,priority:1" json:"user_id"`
	SubscriptionID uint          `gorm:"index;not null;index:idx_notification_logs_user_sub_channel_sent,priority:2" json:"subscription_id"`
	ChannelType    string        `gorm:"not null;size:20;index:idx_notification_logs_user_sub_channel_sent,priority:3" json:"channel_type"`
	TriggerType    string        `gorm:"size:30;index;default:''" json:"trigger_type"`
	NotifyDate     time.Time     `gorm:"not null;index" json:"notify_date"`
	Status         string        `gorm:"not null;size:20;index:idx_notification_logs_user_status_sent,priority:2" json:"status"`
	Error          string        `gorm:"type:text" json:"error"`
	SentAt         time.Time     `gorm:"index:idx_notification_logs_user_status_sent,priority:3;index:idx_notification_logs_user_sub_channel_sent,priority:4" json:"sent_at"`
	User           *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Subscription   *Subscription `gorm:"foreignKey:SubscriptionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

type NotificationOutbox struct {
	ID              uint          `gorm:"primaryKey" json:"id"`
	DedupeKey       string        `gorm:"not null;size:255;uniqueIndex" json:"dedupe_key"`
	UserID          uint          `gorm:"not null;index" json:"user_id"`
	SubscriptionID  uint          `gorm:"not null;index" json:"subscription_id"`
	ChannelID       *uint         `gorm:"index" json:"channel_id"`
	ChannelType     string        `gorm:"not null;size:20" json:"channel_type"`
	TriggerType     string        `gorm:"not null;size:30;index" json:"trigger_type"`
	NotifyDate      time.Time     `gorm:"not null;index" json:"notify_date"`
	ScheduledFor    time.Time     `gorm:"not null;index" json:"scheduled_for"`
	ExpiresAt       *time.Time    `gorm:"index" json:"expires_at"`
	Status          string        `gorm:"not null;size:20;index" json:"status"`
	AttemptCount    int           `gorm:"not null;default:0" json:"attempt_count"`
	MaxAttempts     int           `gorm:"not null;default:5" json:"max_attempts"`
	NextAttemptAt   time.Time     `gorm:"not null;index" json:"next_attempt_at"`
	LockedBy        string        `gorm:"size:120;index" json:"locked_by"`
	LockedUntil     *time.Time    `gorm:"index" json:"locked_until"`
	LastAttemptAt   *time.Time    `json:"last_attempt_at"`
	SentAt          *time.Time    `json:"sent_at"`
	LastError       string        `gorm:"type:text" json:"last_error"`
	Message         string        `gorm:"type:text;not null" json:"message"`
	TargetEmail     string        `gorm:"size:255" json:"target_email"`
	SubscriptionURL string        `gorm:"type:text" json:"subscription_url"`
	CreatedAt       time.Time     `json:"created_at"`
	UpdatedAt       time.Time     `json:"updated_at"`
	User            *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Subscription    *Subscription `gorm:"foreignKey:SubscriptionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
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
