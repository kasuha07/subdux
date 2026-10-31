package auth

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/pquerna/otp/totp"
)

const totpReplayTestSecret = "JBSWY3DPEHPK3PXP"

func totpCodeAt(t *testing.T, at time.Time) string {
	t.Helper()
	code, err := totp.GenerateCode(totpReplayTestSecret, at)
	if err != nil {
		t.Fatal(err)
	}
	return code
}

func TestConsumeTOTPCodeRejectsReplayAndOlderSteps(t *testing.T) {
	db := newTestDB(t)
	secret := totpReplayTestSecret
	user := model.User{Username: "totp-replay", Email: "totp-replay@example.com", Password: "x", Role: "user", Status: "active", TotpEnabled: true, TotpSecret: &secret}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 31, 12, 0, 15, 0, time.UTC)
	original := totpNow
	totpNow = func() time.Time { return now }
	t.Cleanup(func() { totpNow = original })

	current := totpCodeAt(t, now)
	previous := totpCodeAt(t, now.Add(-30*time.Second))

	if !consumeTOTPCode(db, user.ID, &secret, current) {
		t.Fatal("first use of the current code was rejected")
	}
	if consumeTOTPCode(db, user.ID, &secret, current) {
		t.Fatal("the same code was accepted twice")
	}
	if consumeTOTPCode(db, user.ID, &secret, previous) {
		t.Fatal("an older step was accepted after a newer one")
	}

	now = now.Add(30 * time.Second)
	if !consumeTOTPCode(db, user.ID, &secret, totpCodeAt(t, now)) {
		t.Fatal("the next step's code was rejected")
	}
}

func TestConsumeTOTPCodeIsSingleUseUnderConcurrency(t *testing.T) {
	db := newTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	secret := totpReplayTestSecret
	user := model.User{Username: "totp-race", Email: "totp-race@example.com", Password: "x", Role: "user", Status: "active", TotpEnabled: true, TotpSecret: &secret}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	code := totpCodeAt(t, time.Now())

	var accepted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if consumeTOTPCode(db, user.ID, &secret, code) {
				accepted.Add(1)
			}
		}()
	}
	wg.Wait()
	if got := accepted.Load(); got != 1 {
		t.Fatalf("concurrent uses accepted = %d, want 1", got)
	}
}

func TestTOTPCodeAcceptedForLoginCannotBeReusedForReauth(t *testing.T) {
	db := newTestDB(t)
	secret := totpReplayTestSecret
	user := createTestUser(t, db)
	if err := db.Model(&model.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{"totp_enabled": true, "totp_secret": secret}).Error; err != nil {
		t.Fatal(err)
	}
	code := totpCodeAt(t, time.Now())

	totpService := NewTOTPService(db)
	if !totpService.VerifyLogin(user.ID, code) {
		t.Fatal("login verification rejected a valid code")
	}
	if consumeTOTPCode(db, user.ID, &secret, code) {
		t.Fatal("a code spent on login was accepted again for step-up")
	}
}
