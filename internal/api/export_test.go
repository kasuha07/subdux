package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func newExportAPITestDB(t *testing.T) *gorm.DB {
	t.Helper()
	t.Setenv("JWT_SECRET", "export-api-test-jwt-secret-0123456789")
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "export-api-test-settings-key")

	dbPath := filepath.Join(t.TempDir(), "subdux-export-api-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}

	if err := db.AutoMigrate(
		&model.User{},
		&model.SystemSetting{},
		&model.APIKey{},
		&model.UserPreference{},
		&model.UserCurrency{},
		&model.Category{},
		&model.PaymentMethod{},
		&model.Subscription{},
		&model.NotificationChannel{},
		&model.NotificationPolicy{},
		&model.NotificationTemplate{},
		&model.CalendarToken{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	if err := pkg.InitJWTSecret(db); err != nil {
		t.Fatalf("failed to initialize jwt secret: %v", err)
	}

	return db
}

func createExportAPITestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	user := model.User{
		Username: "export-user",
		Email:    "export@example.com",
		Password: "hashed-password",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func seedExportAPITestChannel(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()

	config, err := pkg.EncryptNotificationChannelConfig(`{"api_key":"resend-secret","from_email":"from@example.com","to_email":"to@example.com"}`)
	if err != nil {
		t.Fatalf("EncryptNotificationChannelConfig() error = %v", err)
	}
	if err := db.Create(&model.NotificationChannel{
		UserID:  userID,
		Type:    "resend",
		Enabled: true,
		Config:  config,
	}).Error; err != nil {
		t.Fatalf("failed to create notification channel: %v", err)
	}
}

func newExportAPITestServer(t *testing.T, db *gorm.DB) *echo.Echo {
	t.Helper()

	e := echo.New()
	SetupRoutes(context.Background(), e, db, service.NewBackgroundTaskMonitor())
	return e
}

func TestExportBlocksAPIKeyPrincipal(t *testing.T) {
	db := newExportAPITestDB(t)
	user := createExportAPITestUser(t, db)
	seedExportAPITestChannel(t, db, user.ID)
	apiKeyResp, err := service.NewAPIKeyService(db).Create(user.ID, user.Role, service.CreateAPIKeyInput{
		Name:    "Read only",
		KeyKind: service.APIKeyKindAPIIntegration,
		Scopes:  []string{service.APIKeyScopeRead},
	})
	if err != nil {
		t.Fatalf("failed to create api key: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/export", nil)
	req.Header.Set("X-API-Key", apiKeyResp.Key)
	rec := httptest.NewRecorder()
	newExportAPITestServer(t, db).ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "resend-secret") {
		t.Fatalf("api key export response leaked secret: %s", rec.Body.String())
	}
}

func TestExportRedactsSecretsUnlessConfirmed(t *testing.T) {
	db := newExportAPITestDB(t)
	user := createExportAPITestUser(t, db)
	seedExportAPITestChannel(t, db, user.ID)
	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	e := newExportAPITestServer(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "resend-secret") {
		t.Fatalf("default export leaked secret: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"secrets_included": false`) {
		t.Fatalf("default export missing secrets_included=false marker: %s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/export?include_secrets=1&confirm=include_secrets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("confirmed status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "resend-secret") {
		t.Fatalf("confirmed export did not include secret: %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"secrets_included": true`) {
		t.Fatalf("confirmed export missing secrets_included=true marker: %s", rec.Body.String())
	}
}

func TestExportRequiresConfirmationToIncludeSecrets(t *testing.T) {
	db := newExportAPITestDB(t)
	user := createExportAPITestUser(t, db)
	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/export?include_secrets=1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	newExportAPITestServer(t, db).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
}
