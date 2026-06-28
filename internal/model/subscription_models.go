package model

import "time"

type Subscription struct {
	ID               uint           `gorm:"primaryKey" json:"id"`
	UserID           uint           `gorm:"not null;index;index:idx_subscriptions_user_status_billing,priority:1;index:idx_subscriptions_user_next_billing,priority:1" json:"user_id"`
	Name             string         `gorm:"not null;size:255" json:"name"`
	Amount           float64        `gorm:"not null;check:chk_subscriptions_amount_non_negative,amount >= 0" json:"amount"`
	Currency         string         `gorm:"not null;size:10;default:'USD'" json:"currency"`
	Enabled          bool           `gorm:"default:true" json:"enabled"`
	Status           string         `gorm:"not null;size:30;default:'active';check:chk_subscriptions_status,status IN ('active','ended');index:idx_subscriptions_user_status_billing,priority:2" json:"status"`
	RenewalMode      string         `gorm:"not null;size:30;default:'auto_renew';check:chk_subscriptions_renewal_mode,renewal_mode IN ('auto_renew','manual_renew','cancel_at_period_end')" json:"renewal_mode"`
	EndsAt           *time.Time     `json:"ends_at"`
	BillingType      string         `gorm:"not null;size:30;default:'recurring';index:idx_subscriptions_user_status_billing,priority:3" json:"billing_type"`
	RecurrenceType   string         `gorm:"size:30" json:"recurrence_type"`
	IntervalCount    *int           `json:"interval_count"`
	IntervalUnit     string         `gorm:"size:10" json:"interval_unit"`
	MonthlyDay       *int           `json:"monthly_day"`
	YearlyMonth      *int           `json:"yearly_month"`
	YearlyDay        *int           `json:"yearly_day"`
	NextBillingDate  *time.Time     `gorm:"index:idx_subscriptions_user_next_billing,priority:2" json:"next_billing_date"`
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

type SubscriptionEvent struct {
	ID                        uint          `gorm:"primaryKey" json:"id"`
	UserID                    uint          `gorm:"not null;index:idx_subscription_events_user_created,priority:1;index:idx_subscription_events_user_sub_created,priority:1" json:"user_id"`
	ActorUserID               *uint         `gorm:"index" json:"actor_user_id"`
	SubscriptionID            *uint         `gorm:"index;index:idx_subscription_events_user_sub_created,priority:2" json:"subscription_id"`
	SubscriptionName          string        `gorm:"not null;size:255" json:"subscription_name"`
	Type                      string        `gorm:"not null;size:30;index" json:"type"`
	ChangedFields             string        `gorm:"type:text;not null;default:'[]'" json:"changed_fields"`
	PreviousAmount            *float64      `json:"previous_amount"`
	NewAmount                 *float64      `json:"new_amount"`
	PreviousMonthlyAmount     *float64      `json:"previous_monthly_amount"`
	NewMonthlyAmount          *float64      `json:"new_monthly_amount"`
	PreviousCurrency          string        `gorm:"size:10" json:"previous_currency"`
	NewCurrency               string        `gorm:"size:10" json:"new_currency"`
	PreviousNextBillingDate   *time.Time    `json:"previous_next_billing_date"`
	NewNextBillingDate        *time.Time    `json:"new_next_billing_date"`
	PreviousStatus            string        `gorm:"size:30" json:"previous_status"`
	NewStatus                 string        `gorm:"size:30" json:"new_status"`
	PreviousRenewalMode       string        `gorm:"size:30" json:"previous_renewal_mode"`
	NewRenewalMode            string        `gorm:"size:30" json:"new_renewal_mode"`
	PreviousCategoryID        *uint         `json:"previous_category_id"`
	NewCategoryID             *uint         `json:"new_category_id"`
	PreviousCategoryName      string        `gorm:"size:100" json:"previous_category_name"`
	NewCategoryName           string        `gorm:"size:100" json:"new_category_name"`
	PreviousPaymentMethodID   *uint         `json:"previous_payment_method_id"`
	NewPaymentMethodID        *uint         `json:"new_payment_method_id"`
	PreviousPaymentMethodName string        `gorm:"size:50" json:"previous_payment_method_name"`
	NewPaymentMethodName      string        `gorm:"size:50" json:"new_payment_method_name"`
	CreatedAt                 time.Time     `gorm:"index:idx_subscription_events_user_created,priority:2;index:idx_subscription_events_user_sub_created,priority:3" json:"created_at"`
	User                      *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Subscription              *Subscription `gorm:"foreignKey:SubscriptionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"-"`
}

type SubscriptionActionSnooze struct {
	ID             uint          `gorm:"primaryKey" json:"id"`
	UserID         uint          `gorm:"not null;index;uniqueIndex:idx_action_snooze_user_sub_key,priority:1" json:"user_id"`
	SubscriptionID uint          `gorm:"not null;index;uniqueIndex:idx_action_snooze_user_sub_key,priority:2" json:"subscription_id"`
	ActionKey      string        `gorm:"not null;size:100;uniqueIndex:idx_action_snooze_user_sub_key,priority:3" json:"action_key"`
	SnoozedUntil   time.Time     `gorm:"not null;index" json:"snoozed_until"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	User           *User         `gorm:"foreignKey:UserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
	Subscription   *Subscription `gorm:"foreignKey:SubscriptionID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;" json:"-"`
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
