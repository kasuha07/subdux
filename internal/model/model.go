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

type SystemSetting struct {
	Key   string `gorm:"primaryKey;size:100" json:"key"`
	Value string `gorm:"size:500" json:"value"`
}

type Subscription struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"index;not null" json:"user_id"`
	Name             string         `gorm:"not null;size:255" json:"name"`
	Amount           float64        `gorm:"not null;check:chk_subscriptions_amount_non_negative,amount >= 0" json:"amount"`
	Currency         string         `gorm:"not null;size:10;default:'USD'" json:"currency"`
	Enabled          bool           `gorm:"default:true" json:"enabled"`
	Status           string         `gorm:"not null;size:30;default:'active';check:chk_subscriptions_status,status IN ('active','ended')" json:"status"`
	RenewalMode      string         `gorm:"not null;size:30;default:'auto_renew';check:chk_subscriptions_renewal_mode,renewal_mode IN ('auto_renew','manual_renew','cancel_at_period_end')" json:"renewal_mode"`
	EndsAt           *time.Time     `json:"ends_at"`
	BillingType      string         `gorm:"not null;size:30;default:'recurring'" json:"billing_type"`
	RecurrenceType   string         `gorm:"size:30" json:"recurrence_type"`
	IntervalCount    *int           `json:"interval_count"`
	IntervalUnit     string         `gorm:"size:10" json:"interval_unit"`
	MonthlyDay       *int           `json:"monthly_day"`
	YearlyMonth      *int           `json:"yearly_month"`
	YearlyDay        *int           `json:"yearly_day"`
	NextBillingDate  *time.Time     `json:"next_billing_date"`
	Category         string         `gorm:"size:100" json:"category"`
	CategoryID       *uint          `gorm:"index" json:"category_id"`
	PaymentMethodID  *uint          `gorm:"index" json:"payment_method_id"`
	NotifyEnabled    *bool          `json:"notify_enabled"`
	NotifyDaysBefore *int           `gorm:"check:chk_subscriptions_notify_days_before,notify_days_before IS NULL OR (notify_days_before >= 0 AND notify_days_before <= 10)" json:"notify_days_before"`
	Icon             string         `gorm:"size:500" json:"icon"`
	URL              string         `json:"url"`
	Notes            string         `json:"notes"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	User             *User          `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	CategoryRef      *Category      `gorm:"foreignKey:CategoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
	PaymentMethodRef *PaymentMethod `gorm:"foreignKey:PaymentMethodID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
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

type Category struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	UserID         uint      `gorm:"not null;index;uniqueIndex:idx_user_category_name;uniqueIndex:idx_user_category_system_key" json:"user_id"`
	Name           string    `gorm:"not null;size:30;uniqueIndex:idx_user_category_name" json:"name"`
	SystemKey      *string   `gorm:"size:100;uniqueIndex:idx_user_category_system_key" json:"system_key"`
	NameCustomized bool      `gorm:"default:false" json:"name_customized"`
	DisplayOrder   int       `gorm:"default:0" json:"display_order"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	User           *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
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
	User           *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

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

type CalendarToken struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"-"`
	Token     string    `gorm:"uniqueIndex;not null;size:64" json:"token,omitempty"`
	Name      string    `gorm:"not null;size:100" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	User      *User     `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
}

func (t *CalendarToken) MaskToken() {
	if len(t.Token) > 8 {
		t.Token = t.Token[:4] + "..." + t.Token[len(t.Token)-4:]
	}
}
