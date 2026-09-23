package model

// ApplicationModels returns the application's persistent models in a parent
// before child order. Migration utilities and PostgreSQL's logical backup
// adapter use this registry to operate on the same application table set.
// Historical migration model lists must remain fixed in their own migrations.
func ApplicationModels() []any {
	return []any{
		&User{},
		&Category{},
		&PaymentMethod{},
		&Subscription{},
		&UserPreference{},
		&SystemSetting{},
		&UserCurrency{},
		&UserJevSetting{},
		&EmailVerificationCode{},
		&UserBackupCode{},
		&PasskeyCredential{},
		&OIDCConnection{},
		&APIKey{},
		&RefreshToken{},
		&CalendarToken{},
		&NotificationChannel{},
		&NotificationPolicy{},
		&NotificationTemplate{},
		&NotificationOutbox{},
		&NotificationLog{},
		&SubscriptionEvent{},
		&SubscriptionActionSnooze{},
		&AuditEvent{},
		&MCPIdempotencyKey{},
		&BackgroundTaskLease{},
		&ExchangeRate{},
		&BackupDestination{},
		&BackupRun{},
		&BackupRunDestination{},
	}
}
