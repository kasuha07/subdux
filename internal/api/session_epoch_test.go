package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/labstack/echo/v4"
)

func sessionEpochRequest(t *testing.T, e *echo.Echo, method, target, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestLogoutAllRevokesIssuedAccessTokens(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	copied := reauthGateTestToken(t, user)
	current := reauthGateTestToken(t, user)

	if rec := sessionEpochRequest(t, e, http.MethodGet, "/api/auth/me", copied, ""); rec.Code != http.StatusOK {
		t.Fatalf("pre-revocation /auth/me status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if rec := sessionEpochRequest(t, e, http.MethodPost, "/api/auth/logout-all", current, ""); rec.Code != http.StatusNoContent {
		t.Fatalf("logout-all status = %d, body = %s", rec.Code, rec.Body.String())
	}

	for _, probe := range []struct{ method, target, body string }{
		{http.MethodGet, "/api/auth/me", ""},
		{http.MethodGet, "/api/subscriptions", ""},
		{http.MethodPost, "/api/calendar/tokens", `{"name":"exfil"}`},
		{http.MethodGet, "/api/admin/users", ""},
		{http.MethodGet, "/api/reauth/methods", ""},
	} {
		rec := sessionEpochRequest(t, e, probe.method, probe.target, copied, probe.body)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s with pre-revocation token status = %d, want 401; body = %s", probe.method, probe.target, rec.Code, rec.Body.String())
		}
	}
	var calendarTokens int64
	if err := db.Model(&model.CalendarToken{}).Where("user_id = ?", user.ID).Count(&calendarTokens).Error; err != nil {
		t.Fatal(err)
	}
	if calendarTokens != 0 {
		t.Fatalf("calendar tokens = %d, want 0 after revocation", calendarTokens)
	}
}

func TestChangePasswordRevokesOldTokensAndReturnsFreshSession(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	copied := reauthGateTestToken(t, user)

	body := fmt.Sprintf(`{"current_password":%q,"new_password":"a-brand-new-password"}`, reauthGateTestPassword)
	rec := sessionEpochRequest(t, e, http.MethodPut, "/api/auth/password", copied, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("change password status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		MessageCode string `json:"message_code"`
		AccessToken string `json:"access_token"`
		Token       string `json:"token"`
		User        struct {
			ID uint `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.MessageCode != "password_changed_successfully" || resp.AccessToken == "" || resp.Token != resp.AccessToken || resp.User.ID != user.ID {
		t.Fatalf("response = %s, want message code plus a fresh session", rec.Body.String())
	}
	cookieSet := false
	for _, cookie := range rec.Result().Cookies() {
		if cookie.Value != "" && cookie.MaxAge >= 0 && strings.Contains(strings.ToLower(cookie.Name), "refresh") {
			cookieSet = true
		}
	}
	if !cookieSet {
		t.Fatalf("change password did not set a fresh refresh cookie: %q", rec.Header().Values(echo.HeaderSetCookie))
	}

	if rec := sessionEpochRequest(t, e, http.MethodGet, "/api/auth/me", copied, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("old token after password change status = %d, want 401", rec.Code)
	}
	if rec := sessionEpochRequest(t, e, http.MethodGet, "/api/auth/me", resp.AccessToken, ""); rec.Code != http.StatusOK {
		t.Fatalf("fresh token status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestAdminDisableRevokesAccessTokensAcrossReactivation(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	target := model.User{Username: "epoch-target", Email: "epoch-target@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&target).Error; err != nil {
		t.Fatal(err)
	}
	e := newHumanOnlyRouteTestServer(t, db)
	adminToken := reauthGateTestToken(t, admin)
	targetToken := reauthGateTestToken(t, target)

	for _, status := range []string{"disabled", "active"} {
		ticket := mintReauthTicket(t, e, adminToken, "change_user_status")
		if rec := putAdminUserStatus(t, e, adminToken, target.ID, status, ticket); rec.Code != http.StatusOK {
			t.Fatalf("set status %s = %d, body = %s", status, rec.Code, rec.Body.String())
		}
	}
	if rec := sessionEpochRequest(t, e, http.MethodGet, "/api/auth/me", targetToken, ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token issued before disable still works after reactivation: status = %d", rec.Code)
	}
}
