package auth

import (
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

const (
	totpPeriodSeconds = 30
	// totpSkewSteps matches totp.Validate's default tolerance of one step
	// either side of the current one.
	totpSkewSteps = 1
)

// totpNow is the clock TOTP verification uses; tests may replace it.
var totpNow = time.Now

// matchTOTPCounter returns the time step whose code equals code, checking the
// current step and totpSkewSteps either side, in ascending order.
func matchTOTPCounter(secret string, code string, now time.Time) (int64, bool) {
	opts := totp.ValidateOpts{Period: totpPeriodSeconds, Skew: 0, Digits: otp.DigitsSix, Algorithm: otp.AlgorithmSHA1}
	current := now.Unix() / totpPeriodSeconds
	for step := current - totpSkewSteps; step <= current+totpSkewSteps; step++ {
		at := time.Unix(step*totpPeriodSeconds, 0).UTC()
		if ok, err := totp.ValidateCustom(code, secret, at, opts); err == nil && ok {
			return step, true
		}
	}
	return 0, false
}

// consumeTOTPCode verifies code against the user's enrolled secret and spends
// its time step: the users.totp_last_counter update only succeeds for a step
// newer than the last accepted one, so the same code (or an older one) cannot
// be accepted twice, even by concurrent requests. Login and every step-up
// verification share the counter, so a code accepted for one cannot be
// replayed for the other.
func consumeTOTPCode(db *gorm.DB, userID uint, secret *string, code string) bool {
	if secret == nil || *secret == "" || code == "" {
		return false
	}
	counter, ok := matchTOTPCounter(*secret, code, totpNow())
	if !ok {
		return false
	}
	result := db.Model(&model.User{}).
		Where("id = ? AND totp_last_counter < ?", userID, counter).
		UpdateColumn("totp_last_counter", counter)
	return result.Error == nil && result.RowsAffected == 1
}
