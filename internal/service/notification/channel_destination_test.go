package notification

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

const destinationTestEncryptionKey = "notification-channel-destination-test-key"

func encryptDestinationTestConfig(t *testing.T, plain string) string {
	t.Helper()
	encrypted, err := pkg.EncryptNotificationChannelConfig(plain)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}
	return encrypted
}

func requireSecretsReentryError(t *testing.T, err error, wantFields ...string) {
	t.Helper()
	var svcErr *serviceerr.Error
	if !errors.As(err, &svcErr) {
		t.Fatalf("error = %v, want *serviceerr.Error", err)
	}
	if svcErr.Kind != serviceerr.KindInvalid || svcErr.Code != "notification_secrets_must_be_reentered_when_destination_changes" {
		t.Fatalf("error kind/code = %v/%q, want invalid/notification_secrets_must_be_reentered_when_destination_changes", svcErr.Kind, svcErr.Code)
	}
	fields, _ := svcErr.Params["fields"].(string)
	if fields != strings.Join(wantFields, ", ") {
		t.Fatalf("error fields = %q, want %q", fields, strings.Join(wantFields, ", "))
	}
}

func TestMergeNotificationConfigRejectsRetainedSecretsWhenDestinationChanges(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", destinationTestEncryptionKey)

	tests := []struct {
		name       string
		chType     string
		existing   string
		incoming   string
		wantFields []string
	}{
		{
			name:       "webhook url blank secret",
			chType:     "webhook",
			existing:   `{"url":"https://example.com/hook","secret":"s3cret"}`,
			incoming:   `{"url":"https://attacker.example/hook","secret":""}`,
			wantFields: []string{"secret"},
		},
		{
			name:       "webhook path on shared relay host",
			chType:     "webhook",
			existing:   `{"url":"https://relay.example/owner-a","secret":"s3cret"}`,
			incoming:   `{"url":"https://relay.example/owner-b"}`,
			wantFields: []string{"secret"},
		},
		{
			name:       "webhook headers omitted entirely",
			chType:     "webhook",
			existing:   `{"url":"https://example.com/hook","headers":{"Authorization":"Bearer abc","X-Env":"prod"}}`,
			incoming:   `{"url":"https://attacker.example/hook"}`,
			wantFields: []string{"headers.Authorization", "headers.X-Env"},
		},
		{
			name:       "webhook headers blank",
			chType:     "webhook",
			existing:   `{"url":"https://example.com/hook","headers":{"Authorization":"Bearer abc"}}`,
			incoming:   `{"url":"https://attacker.example/hook","headers":{"Authorization":""}}`,
			wantFields: []string{"headers.Authorization"},
		},
		{
			name:       "smtp host",
			chType:     "smtp",
			existing:   `{"host":"smtp.example.com","port":587,"username":"u","password":"pw","from_email":"a@example.com","to_email":"b@example.com"}`,
			incoming:   `{"host":"smtp.attacker.example","port":587,"username":"u","password":"","from_email":"a@example.com","to_email":"b@example.com"}`,
			wantFields: []string{"password"},
		},
		{
			name:       "smtp port",
			chType:     "smtp",
			existing:   `{"host":"smtp.example.com","port":587,"password":"pw","from_email":"a@example.com","to_email":"b@example.com"}`,
			incoming:   `{"host":"smtp.example.com","port":2525,"from_email":"a@example.com","to_email":"b@example.com"}`,
			wantFields: []string{"password"},
		},
		{
			name:       "smtp encryption downgrade",
			chType:     "smtp",
			existing:   `{"host":"smtp.example.com","port":587,"encryption":"starttls","password":"pw","from_email":"a@example.com","to_email":"b@example.com"}`,
			incoming:   `{"host":"smtp.example.com","port":587,"encryption":"none","password":"","from_email":"a@example.com","to_email":"b@example.com"}`,
			wantFields: []string{"password"},
		},
		{
			name:       "smtp skip tls verify enabled",
			chType:     "smtp",
			existing:   `{"host":"smtp.example.com","port":587,"password":"pw","from_email":"a@example.com","to_email":"b@example.com"}`,
			incoming:   `{"host":"smtp.example.com","port":587,"skip_tls_verify":true,"password":"","from_email":"a@example.com","to_email":"b@example.com"}`,
			wantFields: []string{"password"},
		},
		{
			name:       "gotify url",
			chType:     "gotify",
			existing:   `{"url":"https://gotify.example.com","token":"tok"}`,
			incoming:   `{"url":"https://attacker.example","token":""}`,
			wantFields: []string{"token"},
		},
		{
			name:       "ntfy url",
			chType:     "ntfy",
			existing:   `{"url":"https://ntfy.example.com","topic":"t","token":"tok","password":"pw"}`,
			incoming:   `{"url":"https://attacker.example","topic":"t"}`,
			wantFields: []string{"password", "token"},
		},
		{
			name:       "bark custom server",
			chType:     "bark",
			existing:   `{"device_key":"dk"}`,
			incoming:   `{"url":"https://attacker.example","device_key":""}`,
			wantFields: []string{"device_key"},
		},
		{
			name:       "pushdeer server",
			chType:     "pushdeer",
			existing:   `{"push_key":"pk"}`,
			incoming:   `{"server_url":"https://attacker.example","push_key":""}`,
			wantFields: []string{"push_key"},
		},
		{
			name:       "pushplus endpoint",
			chType:     "pushplus",
			existing:   `{"token":"tok"}`,
			incoming:   `{"endpoint":"https://attacker.example/send","token":""}`,
			wantFields: []string{"token"},
		},
		{
			name:       "pushover endpoint",
			chType:     "pushover",
			existing:   `{"token":"tok","user":"usr"}`,
			incoming:   `{"endpoint":"https://attacker.example/1/messages.json","token":"","user":""}`,
			wantFields: []string{"token", "user"},
		},
		{
			name:       "napcat url",
			chType:     "napcat",
			existing:   `{"url":"https://napcat.example.com","access_token":"tok","message_type":"private","user_id":"1"}`,
			incoming:   `{"url":"https://attacker.example","access_token":"","message_type":"private","user_id":"1"}`,
			wantFields: []string{"access_token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := encryptDestinationTestConfig(t, tt.existing)
			merged, err := mergeNotificationConfigWithExistingSecrets(tt.chType, existing, tt.incoming, nil, nil)
			if err == nil {
				t.Fatalf("merge succeeded with %s, want secrets re-entry error", merged)
			}
			requireSecretsReentryError(t, err, tt.wantFields...)
		})
	}
}

func TestMergeNotificationConfigAllowsDestinationChangeWithReenteredOrClearedSecrets(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", destinationTestEncryptionKey)

	existing := encryptDestinationTestConfig(t, `{"url":"https://example.com/hook","secret":"old-secret","headers":{"Authorization":"Bearer old","X-Env":"prod"}}`)
	merged, err := mergeNotificationConfigWithExistingSecrets(
		"webhook",
		existing,
		`{"url":"https://new.example/hook","secret":"new-secret","headers":{"Authorization":"","X-Env":"staging"}}`,
		nil,
		[]string{"Authorization"},
	)
	if err != nil {
		t.Fatalf("mergeNotificationConfigWithExistingSecrets() error = %v", err)
	}

	var parsed struct {
		URL     string            `json:"url"`
		Secret  string            `json:"secret"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed.URL != "https://new.example/hook" || parsed.Secret != "new-secret" {
		t.Fatalf("merged = %s, want new url and re-entered secret", merged)
	}
	if parsed.Headers["Authorization"] != "" || parsed.Headers["X-Env"] != "staging" {
		t.Fatalf("merged headers = %v, want Authorization cleared and X-Env re-entered", parsed.Headers)
	}
	if strings.Contains(merged, "old") {
		t.Fatalf("merged = %s, must not carry any old credential", merged)
	}

	clearedExisting := encryptDestinationTestConfig(t, `{"host":"smtp.example.com","port":587,"password":"pw","from_email":"a@example.com","to_email":"b@example.com"}`)
	merged, err = mergeNotificationConfigWithExistingSecrets(
		"smtp",
		clearedExisting,
		`{"host":"smtp.other.example","port":587,"password":"","from_email":"a@example.com","to_email":"b@example.com"}`,
		[]string{"password"},
		nil,
	)
	if err != nil {
		t.Fatalf("mergeNotificationConfigWithExistingSecrets(smtp cleared) error = %v", err)
	}
	if strings.Contains(merged, `"pw"`) {
		t.Fatalf("merged = %s, must not carry cleared smtp password", merged)
	}
}

func TestMergeNotificationConfigPreservesSecretsForEquivalentDestination(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", destinationTestEncryptionKey)

	tests := []struct {
		name     string
		chType   string
		existing string
		incoming string
		field    string
		want     string
	}{
		{
			name:     "webhook host case and trailing slash",
			chType:   "webhook",
			existing: `{"url":"https://example.com/hook","secret":"keep"}`,
			incoming: `{"url":"  HTTPS://Example.COM/hook/  ","secret":""}`,
			field:    "secret",
			want:     "keep",
		},
		{
			name:     "smtp default port and encryption",
			chType:   "smtp",
			existing: `{"host":"smtp.example.com","password":"keep","from_email":"a@example.com","to_email":"b@example.com"}`,
			incoming: `{"host":"SMTP.example.com","port":587,"encryption":"starttls","skip_tls_verify":false,"password":"","from_email":"a@example.com","to_email":"b@example.com"}`,
			field:    "password",
			want:     "keep",
		},
		{
			name:     "non-destination field change",
			chType:   "ntfy",
			existing: `{"url":"https://ntfy.example.com","topic":"a","token":"keep"}`,
			incoming: `{"url":"https://ntfy.example.com","topic":"b","priority":"4"}`,
			field:    "token",
			want:     "keep",
		},
		{
			name:     "fixed-endpoint channel",
			chType:   "telegram",
			existing: `{"bot_token":"keep","chat_id":"1"}`,
			incoming: `{"bot_token":"","chat_id":"2"}`,
			field:    "bot_token",
			want:     "keep",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := encryptDestinationTestConfig(t, tt.existing)
			merged, err := mergeNotificationConfigWithExistingSecrets(tt.chType, existing, tt.incoming, nil, nil)
			if err != nil {
				t.Fatalf("mergeNotificationConfigWithExistingSecrets() error = %v", err)
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
				t.Fatalf("json.Unmarshal() error = %v", err)
			}
			if got, _ := parsed[tt.field].(string); got != tt.want {
				t.Fatalf("%s = %q, want %q (merged=%s)", tt.field, got, tt.want, merged)
			}
		})
	}
}

func TestUpdateChannelDestinationChangeLeavesStoredConfigUntouched(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", destinationTestEncryptionKey)

	db := newNotificationChannelLimitTestDB(t)
	user := createNotificationChannelLimitTestUser(t, db)
	svc := NewService(db, nil, nil)

	initialConfig := `{"url":"https://example.com/hook","headers":{"Authorization":"Bearer keep-me"}}`
	channel := model.NotificationChannel{
		UserID:  user.ID,
		Type:    "webhook",
		Enabled: true,
		Config:  encryptDestinationTestConfig(t, initialConfig),
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("db.Create() error = %v", err)
	}

	updateConfig := `{"url":"https://attacker.example/collect","headers":{"Authorization":""}}`
	_, err := svc.UpdateChannel(user.ID, channel.ID, UpdateChannelInput{Config: &updateConfig, Revision: channel.Revision})
	requireSecretsReentryError(t, err, "headers.Authorization")

	var stored model.NotificationChannel
	if err := db.First(&stored, channel.ID).Error; err != nil {
		t.Fatalf("db.First() error = %v", err)
	}
	if stored.Revision != channel.Revision {
		t.Fatalf("revision = %d, want unchanged %d", stored.Revision, channel.Revision)
	}
	decrypted, err := pkg.DecryptNotificationChannelConfig(stored.Config)
	if err != nil {
		t.Fatalf("DecryptNotificationChannelConfig() error = %v", err)
	}
	if !strings.Contains(decrypted, "https://example.com/hook") || !strings.Contains(decrypted, "Bearer keep-me") {
		t.Fatalf("stored config = %s, want original destination and header", decrypted)
	}
}

func TestNotificationChannelDestinationFieldsOnlyCoverSecretBearingTypes(t *testing.T) {
	for channelType := range notificationChannelDestinationFields {
		if !isValidChannelType(channelType) {
			t.Fatalf("destination fields registered for unknown channel type %q", channelType)
		}
		if len(getNotificationChannelSecretFields(channelType)) == 0 {
			t.Fatalf("channel type %q has destination fields but no secret fields; merge would skip the check", channelType)
		}
	}
	if !slices.Contains(notificationChannelDestinationFields["smtp"], "host") {
		t.Fatal("smtp destination must include host")
	}
}
