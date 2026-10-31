package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kasuha07/subdux/internal/api/apimw"
	"github.com/kasuha07/subdux/internal/model"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/labstack/echo/v4"
)

func deleteOIDCConnection(t *testing.T, e *echo.Echo, token string, connectionID uint, ticket string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/auth/oidc/connections/%d", connectionID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	if ticket != "" {
		req.Header.Set(apimw.ReauthTicketHeader, ticket)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

func TestDeleteOIDCConnectionRequiresValidReauthTicket(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	user := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, user)

	connection := model.OIDCConnection{UserID: user.ID, Provider: "oidc", Issuer: "https://idp.example.com", Subject: "delete-gate", Email: user.Email}
	if err := db.Create(&connection).Error; err != nil {
		t.Fatalf("create oidc connection: %v", err)
	}
	countConnection := func() int64 {
		t.Helper()
		var count int64
		if err := db.Model(&model.OIDCConnection{}).Where("id = ?", connection.ID).Count(&count).Error; err != nil {
			t.Fatal(err)
		}
		return count
	}

	for name, ticket := range map[string]func() string{
		"missing ticket":         func() string { return "" },
		"wrong-operation ticket": func() string { return mintReauthTicket(t, e, token, servicereauth.ReauthOperationConnectOIDC) },
	} {
		t.Run(name, func(t *testing.T) {
			rec := deleteOIDCConnection(t, e, token, connection.ID, ticket())
			if rec.Code != http.StatusBadRequest || !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
				t.Fatalf("status = %d body = %s, want re-authentication required", rec.Code, rec.Body.String())
			}
			if countConnection() != 1 {
				t.Fatal("connection was deleted without a valid ticket")
			}
		})
	}

	t.Run("valid ticket deletes and is single-use", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationDeleteOIDCConnection)
		if rec := deleteOIDCConnection(t, e, token, connection.ID, ticket); rec.Code != http.StatusOK {
			t.Fatalf("status = %d body = %s, want 200", rec.Code, rec.Body.String())
		}
		if countConnection() != 0 {
			t.Fatal("connection was not deleted")
		}
		rec := deleteOIDCConnection(t, e, token, connection.ID, ticket)
		if rec.Code != http.StatusBadRequest || !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("reused ticket status = %d body = %s", rec.Code, rec.Body.String())
		}
	})
}
