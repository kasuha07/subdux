package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

const testOtherOIDCIssuer = "https://attacker-idp.example.com"

func createOIDCBindingTestUser(t *testing.T, svc *Service, username string, role string) model.User {
	t.Helper()
	user := model.User{Username: username, Email: username + "@example.com", Password: "x", Role: role, Status: "active"}
	if err := svc.DB.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	return user
}

func linkOIDCBindingTestUser(t *testing.T, svc *Service, user model.User, issuer string, subject string) model.OIDCConnection {
	t.Helper()
	connection := model.OIDCConnection{UserID: user.ID, Provider: oidcProviderKey, Issuer: issuer, Subject: subject, Email: user.Email}
	if err := svc.DB.Create(&connection).Error; err != nil {
		t.Fatalf("failed to link user: %v", err)
	}
	return connection
}

func requireServiceErrorCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatalf("error = nil, want %s", code)
	}
	var serviceErr *serviceerr.Error
	if !errors.As(err, &serviceErr) || serviceErr.Code != code {
		t.Fatalf("error = %v, want code %s", err, code)
	}
}

func TestFinishOIDCLoginRequiresMatchingIssuer(t *testing.T) {
	t.Setenv("JWT_SECRET", "oidc-issuer-test-secret-at-least-32-characters")
	svc := NewService(newTestDB(t))
	admin := createOIDCBindingTestUser(t, svc, "victim-admin", "admin")
	linkOIDCBindingTestUser(t, svc, admin, testOIDCIssuer, "shared-subject")

	forged := &oidcIdentityClaims{Issuer: testOtherOIDCIssuer, Subject: "shared-subject", Email: "attacker@example.com", EmailVerified: true}

	t.Run("without auto-create the foreign identity is not linked", func(t *testing.T) {
		_, err := svc.finishOIDCLogin(oidcSettings{AutoCreateUser: false}, forged)
		requireServiceErrorCode(t, err, "oidc_account_is_not_linked")
	})

	t.Run("with auto-create the foreign identity gets its own account", func(t *testing.T) {
		result, err := svc.finishOIDCLogin(oidcSettings{AutoCreateUser: true}, forged)
		if err != nil {
			t.Fatalf("finishOIDCLogin() error = %v", err)
		}
		if result.User == nil || result.User.ID == admin.ID || result.User.Role != "user" {
			t.Fatalf("foreign identity logged in as %+v, want a new regular user", result.User)
		}
		var created model.OIDCConnection
		if err := svc.DB.Where("user_id = ?", result.User.ID).First(&created).Error; err != nil {
			t.Fatalf("failed to load created connection: %v", err)
		}
		if created.Issuer != testOtherOIDCIssuer || created.Subject != "shared-subject" {
			t.Fatalf("created connection = %+v, want issuer-bound link", created)
		}
	})

	t.Run("the linked issuer still logs in", func(t *testing.T) {
		claims := &oidcIdentityClaims{Issuer: testOIDCIssuer, Subject: "shared-subject", Email: admin.Email}
		result, err := svc.finishOIDCLogin(oidcSettings{}, claims)
		if err != nil {
			t.Fatalf("finishOIDCLogin() error = %v", err)
		}
		if result.User == nil || result.User.ID != admin.ID {
			t.Fatalf("logged in as %+v, want user %d", result.User, admin.ID)
		}
	})
}

func TestFinishOIDCReauthRequiresMatchingIssuer(t *testing.T) {
	svc := NewService(newTestDB(t))
	admin := createOIDCBindingTestUser(t, svc, "admin", "admin")
	linkOIDCBindingTestUser(t, svc, admin, testOIDCIssuer, "admin-subject")

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	defer restoreClock()

	// A foreign issuer asserting the admin's subject and the strongest amr must
	// not mint a step-up result.
	forged := &oidcIdentityClaims{Issuer: testOtherOIDCIssuer, Subject: "admin-subject", AuthTime: now.Unix(), AMR: []string{"hwk"}}
	_, err := svc.finishOIDCReauth(admin.ID, testReauthOperationBackup, forged, now.Add(-time.Second))
	requireServiceErrorCode(t, err, "oidc_identity_is_not_linked_to_this_account")
}

func TestFinishOIDCConnectIsScopedToIssuer(t *testing.T) {
	t.Run("an identity linked under the current issuer stays exclusive", func(t *testing.T) {
		svc := NewService(newTestDB(t))
		owner := createOIDCBindingTestUser(t, svc, "owner", "user")
		other := createOIDCBindingTestUser(t, svc, "other", "user")
		linkOIDCBindingTestUser(t, svc, owner, testOIDCIssuer, "subject")

		_, err := svc.finishOIDCConnect(other.ID, &oidcIdentityClaims{Issuer: testOIDCIssuer, Subject: "subject"})
		requireServiceErrorCode(t, err, "this_oidc_account_is_already_linked_to_another_user")
	})

	t.Run("the same subject from another issuer is a different identity", func(t *testing.T) {
		svc := NewService(newTestDB(t))
		owner := createOIDCBindingTestUser(t, svc, "owner", "user")
		other := createOIDCBindingTestUser(t, svc, "other", "user")
		linkOIDCBindingTestUser(t, svc, owner, testOIDCIssuer, "subject")

		result, err := svc.finishOIDCConnect(other.ID, &oidcIdentityClaims{Issuer: testOtherOIDCIssuer, Subject: "subject"})
		if err != nil || !result.Connected {
			t.Fatalf("finishOIDCConnect() = %+v, %v; want connected", result, err)
		}
	})

	t.Run("a second identity of the current issuer is rejected", func(t *testing.T) {
		svc := NewService(newTestDB(t))
		user := createOIDCBindingTestUser(t, svc, "user", "user")
		linkOIDCBindingTestUser(t, svc, user, testOIDCIssuer, "first")

		_, err := svc.finishOIDCConnect(user.ID, &oidcIdentityClaims{Issuer: testOIDCIssuer, Subject: "second"})
		requireServiceErrorCode(t, err, "you_have_already_connected_another_oidc_account")
	})

	t.Run("a link to a previous issuer is replaced", func(t *testing.T) {
		svc := NewService(newTestDB(t))
		user := createOIDCBindingTestUser(t, svc, "user", "user")
		stale := linkOIDCBindingTestUser(t, svc, user, testOtherOIDCIssuer, "old-subject")

		result, err := svc.finishOIDCConnect(user.ID, &oidcIdentityClaims{Issuer: testOIDCIssuer, Subject: "new-subject", Email: "new@example.com"})
		if err != nil || !result.Connected {
			t.Fatalf("finishOIDCConnect() = %+v, %v; want connected", result, err)
		}
		var connections []model.OIDCConnection
		if err := svc.DB.Where("user_id = ?", user.ID).Find(&connections).Error; err != nil {
			t.Fatalf("failed to load connections: %v", err)
		}
		if len(connections) != 1 {
			t.Fatalf("connections = %+v, want exactly one", connections)
		}
		got := connections[0]
		if got.ID != stale.ID || got.Issuer != testOIDCIssuer || got.Subject != "new-subject" || got.Email != "new@example.com" {
			t.Fatalf("connection = %+v, want stale link replaced in place", got)
		}
	})
}

func TestHasOIDCConnectionOnlyCountsCurrentIssuer(t *testing.T) {
	svc := NewService(newTestDB(t))
	user := createOIDCBindingTestUser(t, svc, "user", "user")
	linkOIDCBindingTestUser(t, svc, user, testOtherOIDCIssuer, "subject")

	if has, err := svc.HasOIDCConnection(user.ID); err != nil || has {
		t.Fatalf("HasOIDCConnection() without configured issuer = %v, %v; want false", has, err)
	}

	seedSystemSetting(t, svc.DB, "oidc_issuer_url", testOIDCIssuer)
	if has, err := svc.HasOIDCConnection(user.ID); err != nil || has {
		t.Fatalf("HasOIDCConnection() with stale link = %v, %v; want false", has, err)
	}

	seedSystemSetting(t, svc.DB, "oidc_issuer_url", " "+testOtherOIDCIssuer+" ")
	if has, err := svc.HasOIDCConnection(user.ID); err != nil || !has {
		t.Fatalf("HasOIDCConnection() with current link = %v, %v; want true", has, err)
	}
}
