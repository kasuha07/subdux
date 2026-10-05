package admin

import (
	"errors"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/gorm"
)

func approveSecurityChange() error {
	return nil
}

func newSecuritySettingsTestService(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := newTestDB(t)
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate system settings table: %v", err)
	}
	return NewService(db), db
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func TestUpdateSettingsRequiresAuthorizationForSecuritySettings(t *testing.T) {
	cases := []struct {
		name  string
		input UpdateSettingsInput
	}{
		{"oidc issuer", UpdateSettingsInput{OIDCIssuerURL: stringPtr("https://attacker.example.com")}},
		{"oidc client id", UpdateSettingsInput{OIDCClientID: stringPtr("other-client")}},
		{"oidc client secret", UpdateSettingsInput{OIDCClientSecret: stringPtr("new-secret")}},
		{"oidc redirect url", UpdateSettingsInput{OIDCRedirectURL: stringPtr("https://attacker.example.com/callback")}},
		{"oidc authorization endpoint", UpdateSettingsInput{OIDCAuthorizeURL: stringPtr("https://attacker.example.com/authorize")}},
		{"oidc token endpoint", UpdateSettingsInput{OIDCTokenURL: stringPtr("https://attacker.example.com/token")}},
		{"oidc userinfo endpoint", UpdateSettingsInput{OIDCUserinfoURL: stringPtr("https://attacker.example.com/userinfo")}},
		{"oidc mfa acr", UpdateSettingsInput{OIDCReauthACRMFA: stringPtr("level0")}},
		{"oidc phishing-resistant acr", UpdateSettingsInput{OIDCReauthACRPhishingResistant: stringPtr("level0")}},
		{"ssrf protection", UpdateSettingsInput{SSRFProtectionEnabled: boolPtr(false)}},
		{"ssrf private ip", UpdateSettingsInput{SSRFAllowPrivateIP: boolPtr(true)}},
		{"ssrf domain mode", UpdateSettingsInput{SSRFDomainFilterMode: stringPtr("whitelist")}},
		{"ssrf domain list", UpdateSettingsInput{SSRFDomainFilterList: stringPtr("example.com")}},
		{"ssrf ip mode", UpdateSettingsInput{SSRFIPFilterMode: stringPtr("whitelist")}},
		{"ssrf ip list", UpdateSettingsInput{SSRFIPFilterList: stringPtr("10.0.0.0/8")}},
		{"ssrf resolved ip filter", UpdateSettingsInput{SSRFFilterResolvedIPs: boolPtr(false)}},
		{"system proxy enabled", UpdateSettingsInput{SystemProxyEnabled: boolPtr(true), SystemProxyURL: stringPtr("http://proxy.example.com:8080")}},
		{"system proxy type", UpdateSettingsInput{SystemProxyType: stringPtr("socks5")}},
		{"system proxy url", UpdateSettingsInput{SystemProxyURL: stringPtr("http://proxy.example.com:8080")}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, db := newSecuritySettingsTestService(t)
			input := tc.input
			// A harmless field in the same request must roll back with it.
			input.SiteName = stringPtr("changed-site")

			if err := svc.UpdateSettings(input, nil); !errors.Is(err, ErrSecuritySettingsChangeRequiresReauth) {
				t.Fatalf("UpdateSettings(nil authorizer) error = %v, want %v", err, ErrSecuritySettingsChangeRequiresReauth)
			}
			var count int64
			if err := db.Model(&model.SystemSetting{}).Count(&count).Error; err != nil {
				t.Fatalf("failed to count settings: %v", err)
			}
			if count != 0 {
				t.Fatalf("rejected update saved %d settings", count)
			}

			denied := errors.New("denied")
			if err := svc.UpdateSettings(input, func() error { return denied }); !errors.Is(err, denied) {
				t.Fatalf("UpdateSettings(denying authorizer) error = %v, want %v", err, denied)
			}
			if err := db.Model(&model.SystemSetting{}).Count(&count).Error; err != nil {
				t.Fatalf("failed to count settings: %v", err)
			}
			if count != 0 {
				t.Fatalf("denied update saved %d settings", count)
			}

			calls := 0
			if err := svc.UpdateSettings(input, func() error { calls++; return nil }); err != nil {
				t.Fatalf("UpdateSettings(approving authorizer) error = %v", err)
			}
			if calls != 1 {
				t.Fatalf("authorizer calls = %d, want 1", calls)
			}
			settings, err := svc.GetSettings()
			if err != nil {
				t.Fatalf("GetSettings() error = %v", err)
			}
			if settings.SiteName != "changed-site" {
				t.Fatalf("SiteName = %q, want approved update saved", settings.SiteName)
			}
		})
	}
}

func TestUpdateSettingsAllowsResendingSavedSecuritySettings(t *testing.T) {
	svc, _ := newSecuritySettingsTestService(t)
	if err := svc.UpdateSettings(UpdateSettingsInput{
		OIDCIssuerURL:         stringPtr("https://idp.example.com"),
		OIDCClientID:          stringPtr("client"),
		OIDCClientSecret:      stringPtr("secret"),
		SSRFDomainFilterList:  stringPtr("example.com"),
		SSRFProtectionEnabled: boolPtr(false),
	}, approveSecurityChange); err != nil {
		t.Fatalf("UpdateSettings() seed error = %v", err)
	}

	saved, err := svc.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}

	// The admin UI re-submits a whole section with the saved values, an empty
	// write-only secret, and whatever harmless field the admin actually edited.
	input := UpdateSettingsInput{
		Revisions:                      saved.Revisions,
		OIDCEnabled:                    boolPtr(true),
		OIDCProviderName:               stringPtr("Company SSO"),
		OIDCAutoCreateUser:             boolPtr(true),
		OIDCScopes:                     stringPtr("openid email"),
		OIDCIssuerURL:                  stringPtr(saved.OIDCIssuerURL),
		OIDCClientID:                   stringPtr(saved.OIDCClientID),
		OIDCClientSecret:               stringPtr("  "),
		OIDCRedirectURL:                stringPtr(saved.OIDCRedirectURL),
		OIDCAuthorizeURL:               stringPtr(saved.OIDCAuthorizeURL),
		OIDCTokenURL:                   stringPtr(saved.OIDCTokenURL),
		OIDCUserinfoURL:                stringPtr(saved.OIDCUserinfoURL),
		OIDCReauthACRMFA:               stringPtr(saved.OIDCReauthACRMFA),
		OIDCReauthACRPhishingResistant: stringPtr(saved.OIDCReauthACRPhishingResistant),
		SSRFProtectionEnabled:          boolPtr(saved.SSRFProtectionEnabled),
		SSRFAllowPrivateIP:             boolPtr(saved.SSRFAllowPrivateIP),
		SSRFDomainFilterMode:           stringPtr(saved.SSRFDomainFilterMode),
		SSRFDomainFilterList:           stringPtr(saved.SSRFDomainFilterList),
		SSRFIPFilterMode:               stringPtr(saved.SSRFIPFilterMode),
		SSRFIPFilterList:               stringPtr(saved.SSRFIPFilterList),
		SSRFFilterResolvedIPs:          boolPtr(saved.SSRFFilterResolvedIPs),
		SystemProxyEnabled:             boolPtr(saved.SystemProxyEnabled),
		SystemProxyType:                stringPtr(saved.SystemProxyType),
	}
	if err := svc.UpdateSettings(input, nil); err != nil {
		t.Fatalf("UpdateSettings() re-sending saved values error = %v, want nil", err)
	}
}

func TestUpdateSettingsValidatesBeforeAuthorizingSecurityChange(t *testing.T) {
	svc, _ := newSecuritySettingsTestService(t)
	calls := 0
	err := svc.UpdateSettings(UpdateSettingsInput{SSRFIPFilterList: stringPtr("10.0.0.0/99")}, func() error {
		calls++
		return nil
	})
	if err == nil {
		t.Fatal("UpdateSettings() accepted an invalid ip filter list")
	}
	if calls != 0 {
		t.Fatalf("authorizer calls = %d, want 0 for invalid input", calls)
	}
}
