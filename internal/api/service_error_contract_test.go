package api

import (
	"net/http"
	"testing"

	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	servicesmtp "github.com/kasuha07/subdux/internal/service/smtp"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
)

// TestServiceErrorContract locks the HTTP status that representative typed
// service errors resolve to through the single central mapper. It is the
// regression guard for the message-substring → typed-Kind migration: if a
// sentinel's Kind is ever changed (and with it the response status), this test
// fails loudly rather than silently shifting the external contract.
func TestServiceErrorContract(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want int
	}{
		{"subscription not found", subscriptionservice.ErrSubscriptionNotFound, http.StatusNotFound},
		{"subscription category not found (input)", subscriptionservice.ErrCategoryNotFound, http.StatusBadRequest},
		{"subscription invalid url", subscriptionservice.ErrInvalidSubscriptionURL, http.StatusBadRequest},
		{"currency not found", catalogservice.ErrCurrencyNotFound, http.StatusNotFound},
		{"currency code exists", catalogservice.ErrCurrencyCodeExists, http.StatusConflict},
		{"currency in use", catalogservice.ErrCurrencyInUse, http.StatusBadRequest},
		{"category name exists", catalogservice.ErrCategoryNameExists, http.StatusConflict},
		{"payment method not found", catalogservice.ErrPaymentMethodNotFound, http.StatusNotFound},
		{"user not found", serviceauth.ErrUserNotFound, http.StatusNotFound},
		{"registration disabled", serviceauth.ErrRegistrationDisabled, http.StatusForbidden},
		{"email already registered", serviceauth.ErrEmailAlreadyRegistered, http.StatusConflict},
		{"verification code too frequent", serviceauth.ErrVerificationCodeTooFrequent, http.StatusTooManyRequests},
		{"invalid refresh token", serviceauth.ErrInvalidRefreshToken, http.StatusUnauthorized},
		{"account disabled", serviceauth.ErrAccountDisabled, http.StatusUnauthorized},
		{"api key limit reached", apikeyservice.ErrAPIKeyLimitReached, http.StatusConflict},
		{"api key not found", apikeyservice.ErrAPIKeyNotFound, http.StatusNotFound},
		{"api key expired", apikeyservice.ErrAPIKeyExpired, http.StatusUnauthorized},
		{"smtp rate limited", servicesmtp.ErrSMTPRateLimited, http.StatusTooManyRequests},
		{"backup invalid password", servicebackup.ErrBackupInvalidPassword, http.StatusBadRequest},
		{"backup encryption password required", servicebackup.ErrBackupEncryptionPasswordRequired, http.StatusBadRequest},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, ok := serviceerr.KindOf(tc.err)
			if !ok {
				t.Fatalf("%v is not a typed service error", tc.err)
			}
			if got := statusForServiceError(kind); got != tc.want {
				t.Fatalf("status for %q = %d, want %d", tc.err, got, tc.want)
			}
		})
	}
}
