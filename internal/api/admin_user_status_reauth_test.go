package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func putAdminUserStatus(t *testing.T, e *echo.Echo, token string, userID uint, status, ticket string) *httptest.ResponseRecorder {
	t.Helper()

	body := fmt.Sprintf(`{"status":%q}`, status)
	req := httptest.NewRequest(http.MethodPut, fmt.Sprintf("/api/admin/users/%d/status", userID), strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestChangeUserStatusRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	target := model.User{
		Username: "status-target",
		Email:    "status-target@example.com",
		Password: "x",
		Role:     "user",
		Status:   "active",
	}
	if err := db.Create(&target).Error; err != nil {
		t.Fatalf("failed to create target user: %v", err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	assertStatus := func(t *testing.T, want string) {
		t.Helper()
		var reloaded model.User
		if err := db.Select("id", "status").First(&reloaded, target.ID).Error; err != nil {
			t.Fatalf("failed to reload target user: %v", err)
		}
		if reloaded.Status != want {
			t.Fatalf("target status = %q, want %q", reloaded.Status, want)
		}
	}

	t.Run("missing ticket is refused", func(t *testing.T) {
		rec := putAdminUserStatus(t, e, token, target.ID, "disabled", "")
		if rec.Code != http.StatusBadRequest || !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("status = %d body = %s, want 400 re-authentication required", rec.Code, rec.Body.String())
		}
		assertStatus(t, "active")
	})

	t.Run("wrong-operation ticket is refused", func(t *testing.T) {
		roleTicket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationChangeUserRole)
		rec := putAdminUserStatus(t, e, token, target.ID, "disabled", roleTicket)
		if rec.Code != http.StatusBadRequest || !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("status = %d body = %s, want 400 re-authentication required", rec.Code, rec.Body.String())
		}
		assertStatus(t, "active")
	})

	t.Run("invalid status is rejected before the ticket is spent", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationChangeUserStatus)
		rec := putAdminUserStatus(t, e, token, target.ID, "suspended", ticket)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "invalid_status") {
			t.Fatalf("status = %d body = %s, want 400 invalid_status", rec.Code, rec.Body.String())
		}
		rec = putAdminUserStatus(t, e, token, target.ID, "disabled", ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s, want 200 with the unspent ticket", rec.Code, rec.Body.String())
		}
		assertStatus(t, "disabled")
	})

	t.Run("valid ticket reactivates and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationChangeUserStatus)
		rec := putAdminUserStatus(t, e, token, target.ID, "active", ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s, want 200", rec.Code, rec.Body.String())
		}
		assertStatus(t, "active")

		rec = putAdminUserStatus(t, e, token, target.ID, "disabled", ticket)
		if rec.Code != http.StatusBadRequest || !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket status = %d body = %s, want 400 re-authentication required", rec.Code, rec.Body.String())
		}
		assertStatus(t, "active")
	})
}
