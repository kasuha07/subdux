package service

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
)

func TestMergeNotificationConfigWithExistingSecretsPreservesBlankSecret(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	existingPlain := `{"url":"https://example.com/hook","secret":"keep-me"}`
	existingEncrypted, err := pkg.EncryptNotificationChannelConfig(existingPlain)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	merged, err := mergeNotificationConfigWithExistingSecrets("webhook", existingEncrypted, `{"url":"https://example.com/new-hook","secret":""}`, nil, nil)
	if err != nil {
		t.Fatalf("mergeNotificationConfigWithExistingSecrets() error = %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed["url"] != "https://example.com/new-hook" {
		t.Fatalf("merged url = %q, want %q", parsed["url"], "https://example.com/new-hook")
	}
	if parsed["secret"] != "keep-me" {
		t.Fatalf("merged secret = %q, want %q", parsed["secret"], "keep-me")
	}
}

func TestSanitizeChannelForResponseMasksSecrets(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	encrypted, err := pkg.EncryptNotificationChannelConfig(`{"api_key":"my-secret-api-key","from_email":"from@example.com","to_email":"to@example.com"}`)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	svc := NewNotificationService(nil, nil, nil)
	channel, configuredSecretFields, _ := svc.SanitizeChannelForResponse(model.NotificationChannel{
		Type:   "resend",
		Config: encrypted,
	})

	var parsed map[string]string
	if err := json.Unmarshal([]byte(channel.Config), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed["api_key"] != "" {
		t.Fatalf("api_key = %q, want empty string", parsed["api_key"])
	}
	if !slices.Contains(configuredSecretFields, "api_key") {
		t.Fatalf("configuredSecretFields = %v, want api_key", configuredSecretFields)
	}
	if parsed["from_email"] != "from@example.com" {
		t.Fatalf("from_email = %q, want from@example.com", parsed["from_email"])
	}
}

func TestSanitizeChannelForResponseMasksWebhookHeaders(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	encrypted, err := pkg.EncryptNotificationChannelConfig(`{"url":"https://example.com/webhook","headers":{"X-Token":"abc123","X-Env":"prod"}}`)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	svc := NewNotificationService(nil, nil, nil)
	channel, _, configuredWebhookHeaderKeys := svc.SanitizeChannelForResponse(model.NotificationChannel{
		Type:   "webhook",
		Config: encrypted,
	})

	var parsed struct {
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(channel.Config), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed.Headers["X-Token"] != "" {
		t.Fatalf("headers.X-Token = %q, want empty string", parsed.Headers["X-Token"])
	}
	if parsed.Headers["X-Env"] != "" {
		t.Fatalf("headers.X-Env = %q, want empty string", parsed.Headers["X-Env"])
	}
	if !slices.Contains(configuredWebhookHeaderKeys, "X-Token") || !slices.Contains(configuredWebhookHeaderKeys, "X-Env") {
		t.Fatalf("configuredWebhookHeaderKeys = %v, want X-Token and X-Env", configuredWebhookHeaderKeys)
	}
}

func TestSanitizeChannelForResponseClearsTokenBearingWebhookURLs(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	svc := NewNotificationService(nil, nil, nil)
	for _, tc := range []struct {
		channelType string
		config      string
	}{
		{channelType: "feishu", config: `{"webhook_url":"https://open.feishu.cn/hook/secret-token","secret":"hmac-key"}`},
		{channelType: "wecom", config: `{"webhook_url":"https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=abc"}`},
		{channelType: "dingtalk", config: `{"webhook_url":"https://oapi.dingtalk.com/robot/send?access_token=xyz","secret":"hmac-key"}`},
	} {
		encrypted, err := pkg.EncryptNotificationChannelConfig(tc.config)
		if err != nil {
			t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
		}

		channel, configuredSecretFields, _ := svc.SanitizeChannelForResponse(model.NotificationChannel{
			Type:   tc.channelType,
			Config: encrypted,
		})

		var parsed map[string]string
		if err := json.Unmarshal([]byte(channel.Config), &parsed); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if parsed["webhook_url"] != "" {
			t.Fatalf("%s webhook_url = %q, want empty string", tc.channelType, parsed["webhook_url"])
		}
		if !slices.Contains(configuredSecretFields, "webhook_url") {
			t.Fatalf("%s configuredSecretFields = %v, want webhook_url", tc.channelType, configuredSecretFields)
		}
	}
}

func TestSanitizeNotificationError(t *testing.T) {
	input := "telegram API error 400: POST https://api.telegram.org/bot123456:ABC/sendMessage?token=xyz Authorization: Bearer very-secret"
	sanitized := sanitizeNotificationError(input)

	if strings.Contains(sanitized, "123456:ABC") || strings.Contains(sanitized, "xyz") || strings.Contains(sanitized, "very-secret") {
		t.Fatalf("sanitizeNotificationError() leaked secret: %q", sanitized)
	}
	if !strings.Contains(sanitized, "[REDACTED]") {
		t.Fatalf("sanitizeNotificationError() = %q, want redaction marker", sanitized)
	}
}

func TestUpdateChannelPreservesExistingSecretWhenInputBlank(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	db := newNotificationChannelLimitTestDB(t)
	user := createNotificationChannelLimitTestUser(t, db)
	svc := NewNotificationService(db, nil, nil)

	initialConfig := `{"url":"https://example.com/hook","secret":"keep-me"}`
	encryptedInitial, err := pkg.EncryptNotificationChannelConfig(initialConfig)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	channel := model.NotificationChannel{
		UserID:  user.ID,
		Type:    "webhook",
		Enabled: true,
		Config:  encryptedInitial,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("db.Create() error = %v", err)
	}

	updateConfig := `{"url":"https://example.com/new-hook","secret":""}`
	updated, err := svc.UpdateChannel(user.ID, channel.ID, UpdateChannelInput{Config: &updateConfig})
	if err != nil {
		t.Fatalf("UpdateChannel() error = %v", err)
	}

	decrypted, err := pkg.DecryptNotificationChannelConfig(updated.Config)
	if err != nil {
		t.Fatalf("DecryptNotificationChannelConfig() error = %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(decrypted), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed["secret"] != "keep-me" {
		t.Fatalf("secret = %q, want %q", parsed["secret"], "keep-me")
	}
	if parsed["url"] != "https://example.com/new-hook" {
		t.Fatalf("url = %q, want %q", parsed["url"], "https://example.com/new-hook")
	}
}

func TestUpdateChannelPreservesExistingWebhookHeaderValuesWhenInputBlank(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	db := newNotificationChannelLimitTestDB(t)
	user := createNotificationChannelLimitTestUser(t, db)
	svc := NewNotificationService(db, nil, nil)

	initialConfig := `{"url":"https://example.com/hook","headers":{"X-Token":"keep-me","X-Env":"prod"}}`
	encryptedInitial, err := pkg.EncryptNotificationChannelConfig(initialConfig)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	channel := model.NotificationChannel{
		UserID:  user.ID,
		Type:    "webhook",
		Enabled: true,
		Config:  encryptedInitial,
	}
	if err := db.Create(&channel).Error; err != nil {
		t.Fatalf("db.Create() error = %v", err)
	}

	updateConfig := `{"url":"https://example.com/new-hook","headers":{"X-Token":"","X-Env":""}}`
	updated, err := svc.UpdateChannel(user.ID, channel.ID, UpdateChannelInput{Config: &updateConfig})
	if err != nil {
		t.Fatalf("UpdateChannel() error = %v", err)
	}

	decrypted, err := pkg.DecryptNotificationChannelConfig(updated.Config)
	if err != nil {
		t.Fatalf("DecryptNotificationChannelConfig() error = %v", err)
	}

	var parsed struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(decrypted), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed.URL != "https://example.com/new-hook" {
		t.Fatalf("url = %q, want %q", parsed.URL, "https://example.com/new-hook")
	}
	if parsed.Headers["X-Token"] != "keep-me" {
		t.Fatalf("headers.X-Token = %q, want %q", parsed.Headers["X-Token"], "keep-me")
	}
	if parsed.Headers["X-Env"] != "prod" {
		t.Fatalf("headers.X-Env = %q, want %q", parsed.Headers["X-Env"], "prod")
	}
}

func TestMergeNotificationConfigClearsSecretWhenExplicitlyCleared(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	existingPlain := `{"url":"https://example.com/hook","secret":"old-secret"}`
	existingEncrypted, err := pkg.EncryptNotificationChannelConfig(existingPlain)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	merged, err := mergeNotificationConfigWithExistingSecrets("webhook", existingEncrypted, `{"url":"https://example.com/hook","secret":""}`, []string{"secret"}, nil)
	if err != nil {
		t.Fatalf("mergeNotificationConfigWithExistingSecrets() error = %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed["secret"] != "" {
		t.Fatalf("secret = %q, want empty string (explicitly cleared)", parsed["secret"])
	}
	if parsed["url"] != "https://example.com/hook" {
		t.Fatalf("url = %q, want %q", parsed["url"], "https://example.com/hook")
	}
}

func TestMergeNotificationConfigPreservesSecretWhenNotInClearedList(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	existingPlain := `{"url":"https://example.com/hook","secret":"keep-me"}`
	existingEncrypted, err := pkg.EncryptNotificationChannelConfig(existingPlain)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	// ClearedFields lists a different field, so "secret" should be preserved
	merged, err := mergeNotificationConfigWithExistingSecrets("webhook", existingEncrypted, `{"url":"https://example.com/hook","secret":""}`, []string{"other_field"}, nil)
	if err != nil {
		t.Fatalf("mergeNotificationConfigWithExistingSecrets() error = %v", err)
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if parsed["secret"] != "keep-me" {
		t.Fatalf("secret = %q, want %q (should be preserved)", parsed["secret"], "keep-me")
	}
}

func TestMergeWebhookHeadersClearsWhenExplicitlyCleared(t *testing.T) {
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "notification-channel-security-test-key")

	existingPlain := `{"url":"https://example.com/hook","headers":{"X-Token":"keep-me","X-Env":"prod"}}`
	existingEncrypted, err := pkg.EncryptNotificationChannelConfig(existingPlain)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}

	merged, err := mergeNotificationConfigWithExistingSecrets("webhook", existingEncrypted, `{"url":"https://example.com/hook","headers":{"X-Token":"","X-Env":""}}`, nil, []string{"X-Token"})
	if err != nil {
		t.Fatalf("mergeNotificationConfigWithExistingSecrets() error = %v", err)
	}

	var parsed struct {
		URL     string            `json:"url"`
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal([]byte(merged), &parsed); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	// X-Token was in clearedHeaderKeys, so it should be cleared
	if parsed.Headers["X-Token"] != "" {
		t.Fatalf("headers.X-Token = %q, want empty (explicitly cleared)", parsed.Headers["X-Token"])
	}
	// X-Env was NOT in clearedHeaderKeys, so it should be preserved
	if parsed.Headers["X-Env"] != "prod" {
		t.Fatalf("headers.X-Env = %q, want %q (should be preserved)", parsed.Headers["X-Env"], "prod")
	}
}
