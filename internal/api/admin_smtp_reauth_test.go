package api

import (
	"net/http"
	"testing"

	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
)

func TestUpdateSettingsSMTPDestinationChangeWithStoredPasswordRequiresReauth(t *testing.T) {
	db := newHumanOnlyRouteTestDB(t)
	admin := createReauthGateTestAdmin(t, db)
	e := newHumanOnlyRouteTestServer(t, db)
	token := reauthGateTestToken(t, admin)

	smtpHost := func(t *testing.T) string {
		t.Helper()
		settings, err := adminservice.NewService(db).GetSettings()
		if err != nil {
			t.Fatalf("GetSettings() error = %v", err)
		}
		return settings.SMTPHost
	}

	// Without a stored password, SMTP endpoint changes are ordinary settings.
	if rec := putAdminSettings(t, e, token, `{"smtp_host":"smtp.example.com","smtp_username":"mailer"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("initial SMTP save status = %d, body = %s", rec.Code, rec.Body.String())
	}
	// Supplying the password itself needs no step-up either.
	if rec := putAdminSettings(t, e, token, `{"smtp_password":"stored-secret"}`, ""); rec.Code != http.StatusOK {
		t.Fatalf("password save status = %d, body = %s", rec.Code, rec.Body.String())
	}

	for _, body := range []string{
		`{"smtp_host":"smtp.attacker.example"}`,
		`{"smtp_port":2525}`,
		`{"smtp_encryption":"none"}`,
		`{"smtp_auth_method":"login"}`,
		`{"smtp_username":"other"}`,
		`{"smtp_skip_tls_verify":true}`,
		`{"smtp_host":"smtp.attacker.example","smtp_password":"   "}`,
	} {
		rec := putAdminSettings(t, e, token, body, "")
		if rec.Code != http.StatusBadRequest || !hasErrorCodeForMessage(rec.Body.String(), "re-authentication required") {
			t.Fatalf("%s: status = %d, body = %s; want re-authentication required", body, rec.Code, rec.Body.String())
		}
	}
	if got := smtpHost(t); got != "smtp.example.com" {
		t.Fatalf("SMTP host = %q after refused changes, want smtp.example.com", got)
	}

	t.Run("re-sending saved values and harmless fields needs no ticket", func(t *testing.T) {
		rec := putAdminSettings(t, e, token, `{"smtp_host":"smtp.example.com","smtp_username":"mailer","smtp_from_name":"Subdux"}`, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
	})

	t.Run("valid ticket authorizes the change", func(t *testing.T) {
		ticket := mintReauthTicket(t, e, token, servicereauth.ReauthOperationAdminSecuritySettings)
		rec := putAdminSettings(t, e, token, `{"smtp_host":"smtp.new.example"}`, ticket)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if got := smtpHost(t); got != "smtp.new.example" {
			t.Fatalf("SMTP host = %q, want smtp.new.example", got)
		}
	})

	t.Run("re-entering the password with the change needs no ticket", func(t *testing.T) {
		rec := putAdminSettings(t, e, token, `{"smtp_host":"smtp.other.example","smtp_password":"new-secret"}`, "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
		}
		if got := smtpHost(t); got != "smtp.other.example" {
			t.Fatalf("SMTP host = %q, want smtp.other.example", got)
		}
	})
}
