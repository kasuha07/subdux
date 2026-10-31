package admin

import (
	"strings"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

// ErrSecuritySettingsChangeRequiresReauth rejects a security-sensitive settings
// change that no step-up authorized. It shares the reauth service's public error
// code so clients handle it like any other missing ticket.
var ErrSecuritySettingsChangeRequiresReauth = serviceerr.New(serviceerr.KindInvalid, "re_authentication_required", "re-authentication required")

// securitySettingsChanged reports whether input changes a setting that a
// session-only admin must not be able to change without stepping up:
//
//   - which OIDC provider is trusted and which client it is (issuer, client ID,
//     client secret): an attacker-controlled issuer can assert any subject and
//     any amr/acr, which would sign in as linked users without their second
//     factor and satisfy OIDC step-up for linked admins;
//   - where the authorization code, client secret, and identity claims are sent
//     (redirect, authorization, token, and userinfo endpoints);
//   - how strong an OIDC login counts for step-up (the reauth acr allowlists);
//   - how outbound requests are filtered or routed (SSRF policy, system proxy);
//   - where and how the stored SMTP password is sent (host, port, encryption,
//     auth method, username, TLS verification), but only while a stored
//     password would be reused: an empty password field keeps the saved one,
//     so pointing SMTP at another server would hand it the old credential.
//     Re-entering the password alongside the change needs no step-up.
//
// Only actual changes count: the admin UI submits whole sections, so a field
// re-sent with its saved value must not demand a ticket. Write-only secrets
// count whenever a replacement is supplied, since an empty value keeps them.
func securitySettingsChanged(current *SystemSettings, input UpdateSettingsInput) bool {
	stringChanged := func(next *string, saved string) bool {
		return next != nil && *next != saved
	}
	boolChanged := func(next *bool, saved bool) bool {
		return next != nil && *next != saved
	}
	secretReplaced := func(next *string) bool {
		return next != nil && strings.TrimSpace(*next) != ""
	}

	return stringChanged(input.OIDCIssuerURL, current.OIDCIssuerURL) ||
		stringChanged(input.OIDCClientID, current.OIDCClientID) ||
		secretReplaced(input.OIDCClientSecret) ||
		stringChanged(input.OIDCRedirectURL, current.OIDCRedirectURL) ||
		stringChanged(input.OIDCAuthorizeURL, current.OIDCAuthorizeURL) ||
		stringChanged(input.OIDCTokenURL, current.OIDCTokenURL) ||
		stringChanged(input.OIDCUserinfoURL, current.OIDCUserinfoURL) ||
		stringChanged(input.OIDCReauthACRMFA, current.OIDCReauthACRMFA) ||
		stringChanged(input.OIDCReauthACRPhishingResistant, current.OIDCReauthACRPhishingResistant) ||
		boolChanged(input.SSRFProtectionEnabled, current.SSRFProtectionEnabled) ||
		boolChanged(input.SSRFAllowPrivateIP, current.SSRFAllowPrivateIP) ||
		stringChanged(input.SSRFDomainFilterMode, current.SSRFDomainFilterMode) ||
		stringChanged(input.SSRFDomainFilterList, current.SSRFDomainFilterList) ||
		stringChanged(input.SSRFIPFilterMode, current.SSRFIPFilterMode) ||
		stringChanged(input.SSRFIPFilterList, current.SSRFIPFilterList) ||
		boolChanged(input.SSRFFilterResolvedIPs, current.SSRFFilterResolvedIPs) ||
		boolChanged(input.SystemProxyEnabled, current.SystemProxyEnabled) ||
		stringChanged(input.SystemProxyType, current.SystemProxyType) ||
		secretReplaced(input.SystemProxyURL) ||
		smtpCredentialDestinationChanged(current, input)
}

// smtpCredentialDestinationChanged reports whether input changes where or how
// the saved SMTP password is presented while that saved password is retained.
func smtpCredentialDestinationChanged(current *SystemSettings, input UpdateSettingsInput) bool {
	if !current.SMTPPasswordSet {
		return false
	}
	if input.SMTPPassword != nil && strings.TrimSpace(*input.SMTPPassword) != "" {
		return false
	}
	trimmedChanged := func(next *string, saved string) bool {
		return next != nil && strings.TrimSpace(*next) != strings.TrimSpace(saved)
	}
	foldedChanged := func(next *string, saved string) bool {
		return next != nil && !strings.EqualFold(strings.TrimSpace(*next), strings.TrimSpace(saved))
	}
	return foldedChanged(input.SMTPHost, current.SMTPHost) ||
		(input.SMTPPort != nil && *input.SMTPPort != current.SMTPPort) ||
		foldedChanged(input.SMTPEncryption, current.SMTPEncryption) ||
		foldedChanged(input.SMTPAuthMethod, current.SMTPAuthMethod) ||
		trimmedChanged(input.SMTPUsername, current.SMTPUsername) ||
		(input.SMTPSkipTLSVerify != nil && *input.SMTPSkipTLSVerify != current.SMTPSkipTLSVerify)
}
