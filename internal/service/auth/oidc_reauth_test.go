package auth

import (
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
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
