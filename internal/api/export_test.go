package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
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

func createExportReauthTestUser(t *testing.T, db *gorm.DB) model.User {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(reauthGateTestPassword), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user := model.User{
		Username: "export-reauth-user",
		Email:    "export-reauth@example.com",
		Password: string(hash),
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create reauth user: %v", err)
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

func exportAPITestToken(t *testing.T, user model.User) string {
	t.Helper()

	token, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
	if err != nil {
		t.Fatalf("failed to generate access token: %v", err)
	}
	return token
}

func mintExportReauthTicket(t *testing.T, e *echo.Echo, token, operation string) string {
	t.Helper()

	body := fmt.Sprintf(`{"operation":%q,"password":%q}`, operation, reauthGateTestPassword)
	req := httptest.NewRequest(http.MethodPost, "/api/reauth/password", strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("mint ticket status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp struct {
		Ticket string `json:"ticket"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode ticket response: %v; body = %s", err, rec.Body.String())
	}
	if resp.Ticket == "" {
		t.Fatalf("mint ticket returned empty ticket; body = %s", rec.Body.String())
	}
	return resp.Ticket
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
	user := createExportReauthTestUser(t, db)
	seedExportAPITestChannel(t, db, user.ID)
	token := exportAPITestToken(t, user)
	e := newExportAPITestServer(t, db)

	redactedTicket := mintExportReauthTicket(t, e, token, service.ReauthOperationExportRedacted)
	req := httptest.NewRequest(http.MethodGet, "/api/export", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(reauthTicketHeader, redactedTicket)
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

	secretsTicket := mintExportReauthTicket(t, e, token, service.ReauthOperationExportSecrets)
	req = httptest.NewRequest(http.MethodGet, "/api/export?include_secrets=1&confirm=include_secrets", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set(reauthTicketHeader, secretsTicket)
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

func TestExportRequiresValidReauthTicket(t *testing.T) {
	db := newExportAPITestDB(t)
	user := createExportReauthTestUser(t, db)
	seedExportAPITestChannel(t, db, user.ID)
	token := exportAPITestToken(t, user)
	e := newExportAPITestServer(t, db)

	t.Run("redacted export requires matching ticket", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/export", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("body = %s, want re-authentication required", rec.Body.String())
		}

		wrongTicket := mintExportReauthTicket(t, e, token, service.ReauthOperationExportSecrets)
		req = httptest.NewRequest(http.MethodGet, "/api/export", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(reauthTicketHeader, wrongTicket)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("wrong-ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("wrong-ticket body = %s, want re-authentication required", rec.Body.String())
		}

		ticket := mintExportReauthTicket(t, e, token, service.ReauthOperationExportRedacted)
		req = httptest.NewRequest(http.MethodGet, "/api/export", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(reauthTicketHeader, ticket)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}

		req = httptest.NewRequest(http.MethodGet, "/api/export", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(reauthTicketHeader, ticket)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("reused-ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused-ticket body = %s, want re-authentication required", rec.Body.String())
		}
	})

	t.Run("secret export requires secret-scoped ticket", func(t *testing.T) {
		redactedTicket := mintExportReauthTicket(t, e, token, service.ReauthOperationExportRedacted)
		req := httptest.NewRequest(http.MethodGet, "/api/export?include_secrets=1&confirm=include_secrets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(reauthTicketHeader, redactedTicket)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("wrong-ticket status = %d, want %d; body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "re-authentication required") {
			t.Fatalf("wrong-ticket body = %s, want re-authentication required", rec.Body.String())
		}

		ticket := mintExportReauthTicket(t, e, token, service.ReauthOperationExportSecrets)
		req = httptest.NewRequest(http.MethodGet, "/api/export?include_secrets=1&confirm=include_secrets", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set(reauthTicketHeader, ticket)
		rec = httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d; body = %s", rec.Code, http.StatusOK, rec.Body.String())
		}
		if !strings.Contains(rec.Body.String(), "resend-secret") {
			t.Fatalf("confirmed export did not include secret: %s", rec.Body.String())
		}
	})
}
