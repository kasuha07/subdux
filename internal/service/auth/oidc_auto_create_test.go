package auth

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
)

func autoCreateOIDCSettings() oidcSettings {
	return oidcSettings{Enabled: true, AutoCreateUser: true}
}

func countOIDCUsers(t *testing.T, svc *Service, email string) (users int64, connections int64) {
	t.Helper()
	if err := svc.DB.Model(&model.User{}).Where("LOWER(email) = ?", email).Count(&users).Error; err != nil {
		t.Fatalf("failed to count users: %v", err)
	}
	if err := svc.DB.Model(&model.OIDCConnection{}).Where("LOWER(email) = ?", email).Count(&connections).Error; err != nil {
		t.Fatalf("failed to count oidc connections: %v", err)
	}
	return users, connections
}

func TestFinishOIDCLoginAutoCreateRejectsUnverifiedEmail(t *testing.T) {
	svc := NewService(newTestDB(t))

	claims := &oidcIdentityClaims{Subject: "attacker-subject", Email: "victim@example.com", EmailVerified: false}
	_, err := svc.finishOIDCLogin(autoCreateOIDCSettings(), claims)
	if !errors.Is(err, ErrOIDCEmailNotVerified) {
		t.Fatalf("finishOIDCLogin() error = %v, want %v", err, ErrOIDCEmailNotVerified)
	}

	users, connections := countOIDCUsers(t, svc, "victim@example.com")
	if users != 0 || connections != 0 {
		t.Fatalf("unverified auto-create left users=%d connections=%d, want 0/0", users, connections)
	}
}

func TestFinishOIDCLoginAutoCreateRespectsEmailDomainWhitelist(t *testing.T) {
	db := newTestDB(t)
	seedSystemSetting(t, db, "email_domain_whitelist", "corp.com")
	svc := NewService(db)

	claims := &oidcIdentityClaims{Subject: "outsider-subject", Email: "outsider@blocked.net", EmailVerified: true}
	_, err := svc.finishOIDCLogin(autoCreateOIDCSettings(), claims)
	if !errors.Is(err, ErrEmailDomainNotAllowed) {
		t.Fatalf("finishOIDCLogin() error = %v, want %v", err, ErrEmailDomainNotAllowed)
	}

	users, connections := countOIDCUsers(t, svc, "outsider@blocked.net")
	if users != 0 || connections != 0 {
		t.Fatalf("blocked-domain auto-create left users=%d connections=%d, want 0/0", users, connections)
	}
}

func TestFinishOIDCLoginAutoCreatesVerifiedAllowedEmail(t *testing.T) {
	t.Setenv("JWT_SECRET", "oidc-auto-create-test-secret-at-least-32-chars")
	db := newTestDB(t)
	seedSystemSetting(t, db, "email_domain_whitelist", "corp.com")
	svc := NewService(db)

	claims := &oidcIdentityClaims{Subject: "member-subject", Email: "Member@Team.Corp.com", EmailVerified: true}
	result, err := svc.finishOIDCLogin(autoCreateOIDCSettings(), claims)
	if err != nil {
		t.Fatalf("finishOIDCLogin() error = %v, want nil", err)
	}
	if result.User == nil || result.User.Email != "member@team.corp.com" || result.Token == "" {
		t.Fatalf("finishOIDCLogin() result = %+v, want issued session for member@team.corp.com", result)
	}

	var connection model.OIDCConnection
	if err := db.Where("provider = ? AND subject = ?", oidcProviderKey, "member-subject").First(&connection).Error; err != nil {
		t.Fatalf("failed to load created oidc connection: %v", err)
	}
	if connection.UserID != result.User.ID {
		t.Fatalf("connection.UserID = %d, want %d", connection.UserID, result.User.ID)
	}
}

func TestFinishOIDCLoginExistingConnectionDoesNotRequireVerifiedEmail(t *testing.T) {
	t.Setenv("JWT_SECRET", "oidc-auto-create-test-secret-at-least-32-chars")
	db := newTestDB(t)
	seedSystemSetting(t, db, "email_domain_whitelist", "corp.com")
	svc := NewService(db)

	user := model.User{Username: "linked", Email: "linked@elsewhere.net", Password: "x", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	if err := db.Create(&model.OIDCConnection{
		UserID: user.ID, Provider: oidcProviderKey, Subject: "linked-subject", Email: user.Email,
	}).Error; err != nil {
		t.Fatalf("failed to create connection: %v", err)
	}

	// The verification and whitelist gates guard account creation only; an
	// identity the user already linked from an authenticated session keeps working.
	claims := &oidcIdentityClaims{Subject: "linked-subject", Email: user.Email, EmailVerified: false}
	result, err := svc.finishOIDCLogin(oidcSettings{Enabled: true}, claims)
	if err != nil {
		t.Fatalf("finishOIDCLogin() error = %v, want nil", err)
	}
	if result.User == nil || result.User.ID != user.ID {
		t.Fatalf("finishOIDCLogin() user = %+v, want %d", result.User, user.ID)
	}
}

func TestOIDCBoolClaimUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want bool
	}{
		{name: "bool true", raw: `{"email_verified":true}`, want: true},
		{name: "bool false", raw: `{"email_verified":false}`, want: false},
		{name: "string true", raw: `{"email_verified":"true"}`, want: true},
		{name: "string true mixed case", raw: `{"email_verified":" True "}`, want: true},
		{name: "string false", raw: `{"email_verified":"false"}`, want: false},
		{name: "unexpected string", raw: `{"email_verified":"yes"}`, want: false},
		{name: "number", raw: `{"email_verified":1}`, want: false},
		{name: "null", raw: `{"email_verified":null}`, want: false},
		{name: "missing", raw: `{}`, want: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var claims oidcIdentityClaims
			if err := json.Unmarshal([]byte(tc.raw), &claims); err != nil {
				t.Fatalf("json.Unmarshal() error = %v, want nil", err)
			}
			if bool(claims.EmailVerified) != tc.want {
				t.Fatalf("EmailVerified = %v, want %v", claims.EmailVerified, tc.want)
			}
		})
	}
}

func TestMergeOIDCUserInfoClaimsCarriesEmailVerification(t *testing.T) {
	t.Run("email from userinfo takes userinfo verification", func(t *testing.T) {
		claims := &oidcIdentityClaims{Subject: "sub-1", EmailVerified: true}
		userInfo := &oidcIdentityClaims{Subject: "sub-1", Email: " other@example.com ", EmailVerified: false}
		if err := mergeOIDCUserInfoClaims(claims, userInfo); err != nil {
			t.Fatalf("mergeOIDCUserInfoClaims() error = %v, want nil", err)
		}
		if claims.Email != "other@example.com" || claims.EmailVerified {
			t.Fatalf("merged email/verified = %q/%v, want other@example.com/false", claims.Email, claims.EmailVerified)
		}
	})

	t.Run("id token email keeps id token verification", func(t *testing.T) {
		claims := &oidcIdentityClaims{Subject: "sub-1", Email: "user@example.com", EmailVerified: false}
		userInfo := &oidcIdentityClaims{Subject: "sub-1", Email: "user@example.com", EmailVerified: true, Name: "User"}
		if err := mergeOIDCUserInfoClaims(claims, userInfo); err != nil {
			t.Fatalf("mergeOIDCUserInfoClaims() error = %v, want nil", err)
		}
		if claims.EmailVerified {
			t.Fatal("EmailVerified = true, want ID token value false")
		}
		if claims.Name != "User" {
			t.Fatalf("Name = %q, want User", claims.Name)
		}
	})

	t.Run("subject mismatch is rejected", func(t *testing.T) {
		claims := &oidcIdentityClaims{Subject: "sub-1"}
		userInfo := &oidcIdentityClaims{Subject: "sub-2", Email: "user@example.com", EmailVerified: true}
		if err := mergeOIDCUserInfoClaims(claims, userInfo); err == nil {
			t.Fatal("mergeOIDCUserInfoClaims() error = nil, want subject mismatch")
		}
		if claims.Email != "" {
			t.Fatalf("Email = %q, want unchanged empty value", claims.Email)
		}
	})
}
