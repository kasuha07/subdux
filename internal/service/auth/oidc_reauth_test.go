package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

const testReauthOperationBackup = "backup"

func TestFinishOIDCReauthOwnership(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	user := model.User{Username: "admin", Email: "admin@example.com", Password: "x", Role: "admin", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	defer restoreClock()
	startedAt := now.Add(-30 * time.Second)

	other := model.User{Username: "other", Email: "other@example.com", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&other).Error; err != nil {
		t.Fatalf("failed to create other user: %v", err)
	}
	if err := db.Create(&model.OIDCConnection{
		UserID: other.ID, Provider: oidcProviderKey, Subject: "shared-subject", Email: other.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	claims := &oidcIdentityClaims{Subject: "shared-subject", Email: other.Email, AuthTime: now.Unix()}
	if _, err := svc.finishOIDCReauth(user.ID, testReauthOperationBackup, claims, startedAt); err == nil {
		t.Fatal("finishOIDCReauth() error = nil for another user's identity, want non-nil")
	}

	if err := db.Create(&model.OIDCConnection{
		UserID: user.ID, Provider: oidcProviderKey, Subject: "own-subject", Email: user.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create own connection: %v", err)
	}
	ownClaims := &oidcIdentityClaims{Subject: "own-subject", Email: user.Email, AuthTime: now.Unix()}
	result, err := svc.finishOIDCReauth(user.ID, testReauthOperationBackup, ownClaims, startedAt)
	if err != nil {
		t.Fatalf("finishOIDCReauth() error = %v, want nil", err)
	}
	if result.Purpose != oidcPurposeReauth || result.UserID != user.ID || result.Operation != testReauthOperationBackup {
		t.Fatalf("finishOIDCReauth() result = %+v, want reauth/%d/%s", result, user.ID, testReauthOperationBackup)
	}
}

func TestGradeOIDCReauth(t *testing.T) {
	mfaACR := []string{"http://schemas.openid.net/pape/policies/2007/06/multi-factor", "mfa"}
	prACR := []string{"phishing-resistant", "urn:acr:fido"}

	cases := []struct {
		name   string
		claims *oidcIdentityClaims
		want   OIDCReauthGrade
	}{
		{"no evidence stays fresh", &oidcIdentityClaims{}, OIDCGradeFresh},
		{"amr otp is mfa", &oidcIdentityClaims{AMR: []string{"pwd", "otp"}}, OIDCGradeMFA},
		{"amr mfa is mfa", &oidcIdentityClaims{AMR: []string{"mfa"}}, OIDCGradeMFA},
		{"amr fido is phishing-resistant", &oidcIdentityClaims{AMR: []string{"pwd", "fido"}}, OIDCGradePhishingResistant},
		{"amr hwk is phishing-resistant", &oidcIdentityClaims{AMR: []string{"hwk"}}, OIDCGradePhishingResistant},
		{"amr webauthn is phishing-resistant", &oidcIdentityClaims{AMR: []string{"webauthn"}}, OIDCGradePhishingResistant},
		{"configured mfa acr is mfa", &oidcIdentityClaims{ACR: "mfa"}, OIDCGradeMFA},
		{"configured pr acr is phishing-resistant", &oidcIdentityClaims{ACR: "phishing-resistant"}, OIDCGradePhishingResistant},
		{"pr amr beats mfa acr", &oidcIdentityClaims{ACR: "mfa", AMR: []string{"fido"}}, OIDCGradePhishingResistant},
		{"space separated acr matches", &oidcIdentityClaims{ACR: "level1 urn:acr:fido"}, OIDCGradePhishingResistant},
		{"case-insensitive amr", &oidcIdentityClaims{AMR: []string{"FIDO"}}, OIDCGradePhishingResistant},
		{"unrelated acr stays fresh", &oidcIdentityClaims{ACR: "level0"}, OIDCGradeFresh},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := gradeOIDCReauth(tc.claims, mfaACR, prACR); got != tc.want {
				t.Fatalf("gradeOIDCReauth() = %d, want %d", got, tc.want)
			}
		})
	}
}

func TestFinishOIDCReauthRequiresFreshLogin(t *testing.T) {
	db := newTestDB(t)
	svc := NewService(db)
	user := model.User{Username: "admin", Email: "admin@example.com", Password: "x", Role: "admin", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := db.Create(&model.OIDCConnection{
		UserID: user.ID, Provider: oidcProviderKey, Subject: "own-subject", Email: user.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create own connection: %v", err)
	}

	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	restoreClock := pkg.SetNowForTest(now)
	defer restoreClock()
	startedAt := now.Add(-30 * time.Second)

	t.Run("missing auth_time is rejected", func(t *testing.T) {
		claims := &oidcIdentityClaims{Subject: "own-subject", Email: user.Email}
		if _, err := svc.finishOIDCReauth(user.ID, testReauthOperationBackup, claims, startedAt); err == nil {
			t.Fatal("finishOIDCReauth() error = nil, want missing auth_time rejection")
		}
	})

	t.Run("stale auth_time is rejected", func(t *testing.T) {
		claims := &oidcIdentityClaims{
			Subject:  "own-subject",
			Email:    user.Email,
			AuthTime: startedAt.Add(-oidcReauthAuthSkew - time.Second).Unix(),
		}
		if _, err := svc.finishOIDCReauth(user.ID, testReauthOperationBackup, claims, startedAt); err == nil {
			t.Fatal("finishOIDCReauth() error = nil, want stale auth_time rejection")
		}
	})

	t.Run("fresh auth_time is accepted", func(t *testing.T) {
		claims := &oidcIdentityClaims{
			Subject:  "own-subject",
			Email:    user.Email,
			AuthTime: now.Unix(),
		}
		if _, err := svc.finishOIDCReauth(user.ID, testReauthOperationBackup, claims, startedAt); err != nil {
			t.Fatalf("finishOIDCReauth() error = %v, want nil", err)
		}
	})
}

func TestCreateOIDCCallbackErrorResultPreservesErrorKind(t *testing.T) {
	svc := NewService(nil)

	callback, err := svc.createOIDCCallbackErrorResult(
		oidcPurposeLogin,
		serviceerr.NewCode(
			serviceerr.KindConflict,
			"email_already_registered_connect_oidc_from_account_settings",
			"email already registered, connect oidc from account settings",
			map[string]any{"provider": "OIDC"},
		),
	)
	if err != nil {
		t.Fatalf("createOIDCCallbackErrorResult() error = %v, want nil", err)
	}

	result, err := svc.ConsumeOIDCSessionResult(callback.SessionID)
	if err != nil {
		t.Fatalf("ConsumeOIDCSessionResult() error = %v, want nil", err)
	}
	if result.ErrorKind != serviceerr.KindConflict {
		t.Fatalf("ErrorKind = %v, want %v", result.ErrorKind, serviceerr.KindConflict)
	}
	if result.ErrorCode != "email_already_registered_connect_oidc_from_account_settings" {
		t.Fatalf("ErrorCode = %q, want email_already_registered_connect_oidc_from_account_settings", result.ErrorCode)
	}
	if result.ErrorParams["provider"] != "OIDC" {
		t.Fatalf("ErrorParams[provider] = %v, want OIDC", result.ErrorParams["provider"])
	}
}

func TestConsumeOIDCReauthResultPreservesStoredErrorKind(t *testing.T) {
	svc := NewService(nil)

	callback, err := svc.createOIDCCallbackErrorResult(
		oidcPurposeReauth,
		serviceerr.New(serviceerr.KindConflict, "oidc_identity_is_not_linked_to_this_account", "oidc identity is not linked to this account"),
	)
	if err != nil {
		t.Fatalf("createOIDCCallbackErrorResult() error = %v, want nil", err)
	}

	_, err = svc.ConsumeOIDCReauthResult(callback.SessionID, 1, testReauthOperationBackup)
	if err == nil {
		t.Fatal("ConsumeOIDCReauthResult() error = nil, want non-nil")
	}

	var typed *serviceerr.Error
	if !errors.As(err, &typed) || typed == nil {
		t.Fatalf("ConsumeOIDCReauthResult() error = %T, want *serviceerr.Error", err)
	}
	if typed.Kind != serviceerr.KindConflict {
		t.Fatalf("Kind = %v, want %v", typed.Kind, serviceerr.KindConflict)
	}
	if typed.Code != "oidc_identity_is_not_linked_to_this_account" {
		t.Fatalf("Code = %q, want oidc_identity_is_not_linked_to_this_account", typed.Code)
	}
}
