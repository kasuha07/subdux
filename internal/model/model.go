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

type OIDCConnection struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null;uniqueIndex:idx_oidc_user_provider" json:"user_id"`
	Provider  string    `gorm:"size:100;not null;uniqueIndex:idx_oidc_user_provider;uniqueIndex:idx_oidc_provider_subject" json:"provider"`
	Subject   string    `gorm:"size:255;not null;uniqueIndex:idx_oidc_provider_subject" json:"subject"`
	Email     string    `gorm:"size:255" json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SystemSetting struct {
	Key   string `gorm:"primaryKey;size:100" json:"key"`
	Value string `gorm:"size:500" json:"value"`
}

type Subscription struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	UserID           uint       `gorm:"index;not null" json:"user_id"`
	Name             string     `gorm:"not null;size:255" json:"name"`
	Amount           float64    `gorm:"not null" json:"amount"`
	Currency         string     `gorm:"not null;size:10;default:'USD'" json:"currency"`
	Enabled          bool       `gorm:"default:true" json:"enabled"`
	BillingType      string     `gorm:"not null;size:30;default:'recurring'" json:"billing_type"` // recurring, one_time
	RecurrenceType   string     `gorm:"size:30" json:"recurrence_type"`                           // interval, monthly_date, yearly_date
	IntervalCount    *int       `json:"interval_count"`
	IntervalUnit     string     `gorm:"size:10" json:"interval_unit"` // day, week, month, year
	MonthlyDay       *int       `json:"monthly_day"`
	YearlyMonth      *int       `json:"yearly_month"`
	YearlyDay        *int       `json:"yearly_day"`
	NextBillingDate  *time.Time `json:"next_billing_date"`
	Category         string     `gorm:"size:100" json:"category"`
	CategoryID       *uint      `gorm:"index" json:"category_id"`
	PaymentMethodID  *uint      `gorm:"index" json:"payment_method_id"`
	NotifyEnabled    *bool      `json:"notify_enabled"`
	NotifyDaysBefore *int       `json:"notify_days_before"`
	Icon             string     `gorm:"size:500" json:"icon"`
	URL              string     `json:"url"`
	Notes            string     `json:"notes"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
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
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index;uniqueIndex:idx_user_category_name;uniqueIndex:idx_user_category_system_key" json:"user_id"`
	Name           string    `gorm:"not null;size:30;uniqueIndex:idx_user_category_name" json:"name"`
	SystemKey      *string   `gorm:"size:100;uniqueIndex:idx_user_category_system_key" json:"system_key"`
	NameCustomized bool      `gorm:"default:false" json:"name_customized"`
	DisplayOrder   int       `gorm:"default:0" json:"display_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type PaymentMethod struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index;uniqueIndex:idx_user_payment_method_name;uniqueIndex:idx_user_payment_method_system_key" json:"user_id"`
	Name           string    `gorm:"not null;size:50;uniqueIndex:idx_user_payment_method_name" json:"name"`
	SystemKey      *string   `gorm:"size:100;uniqueIndex:idx_user_payment_method_system_key" json:"system_key"`
	NameCustomized bool      `gorm:"default:false" json:"name_customized"`
	Icon           string    `gorm:"size:500" json:"icon"`
	SortOrder      int       `gorm:"default:0" json:"sort_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type NotificationChannel struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Type      string    `gorm:"not null;size:20" json:"type"`
	Enabled   bool      `gorm:"default:false" json:"enabled"`
	Config    string    `gorm:"type:text" json:"config"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type NotificationPolicy struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"uniqueIndex;not null" json:"user_id"`
	DaysBefore     int       `gorm:"default:3" json:"days_before"`
	NotifyOnDueDay bool      `gorm:"default:true" json:"notify_on_due_day"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type NotificationLog struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"index;not null" json:"user_id"`
	SubscriptionID uint      `gorm:"index;not null" json:"subscription_id"`
	ChannelType    string    `gorm:"not null;size:20" json:"channel_type"`
	NotifyDate     time.Time `gorm:"not null;index" json:"notify_date"`
	Status         string    `gorm:"not null;size:20" json:"status"`
	Error          string    `gorm:"type:text" json:"error"`
	SentAt         time.Time `json:"sent_at"`
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
}
