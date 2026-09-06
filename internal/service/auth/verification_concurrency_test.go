package auth

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// TestVerifyBackupCodeSingleUseUnderConcurrency proves a backup code can be
// spent exactly once even when several requests present the same code at the
// same time. The read-then-delete sequence races between requests; only the
// request whose conditional delete removes the row may succeed.
func TestVerifyBackupCodeSingleUseUnderConcurrency(t *testing.T) {
	svc, user := newTOTPTestService(t)
	configureSingleConnPool(t, svc.DB)

	const plainCode = "abcd1234"
	hash, err := bcrypt.GenerateFromPassword([]byte(plainCode), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash backup code: %v", err)
	}
	if err := svc.DB.Create(&model.UserBackupCode{UserID: user.ID, CodeHash: string(hash)}).Error; err != nil {
		t.Fatalf("seed backup code: %v", err)
	}

	const attempts = 8
	results := make([]bool, attempts)
	var wg sync.WaitGroup
	var start sync.WaitGroup
	start.Add(1)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start.Wait()
			results[i] = svc.VerifyBackupCode(user.ID, plainCode)
		}(i)
	}
	start.Done()
	wg.Wait()

	succeeded := 0
	for _, ok := range results {
		if ok {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("backup code accepted %d times across %d concurrent attempts, want exactly 1", succeeded, attempts)
	}

	var remaining int64
	if err := svc.DB.Model(&model.UserBackupCode{}).Where("user_id = ?", user.ID).Count(&remaining).Error; err != nil {
		t.Fatalf("count backup codes: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("backup code rows after race = %d, want 0", remaining)
	}
}

// TestConsumeVerificationCodeSingleConsumeUnderConcurrency proves a correct
// verification code is consumed exactly once: concurrent requests reading the
// same unconsumed row all pass the bcrypt check, but only the one whose
// conditional claim update matches a row wins; the losers get the invalid-code
// error instead of silently succeeding.
func TestConsumeVerificationCodeSingleConsumeUnderConcurrency(t *testing.T) {
	db := newTestDB(t)
	configureSingleConnPool(t, db)
	s := &Service{DB: db}

	const correctCode = "123456"
	hash, err := bcrypt.GenerateFromPassword([]byte(correctCode), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash verification code: %v", err)
	}
	verification := model.EmailVerificationCode{
		Email:     "single-consume@example.com",
		Purpose:   verificationPurposePasswordReset,
		CodeHash:  string(hash),
		ExpiresAt: time.Now().Add(verificationCodeTTL),
	}
	if err := s.DB.Create(&verification).Error; err != nil {
		t.Fatalf("seed verification code: %v", err)
	}

	const attempts = 8
	errs := make([]error, attempts)
	var wg sync.WaitGroup
	var start sync.WaitGroup
	start.Add(1)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start.Wait()
			errs[i] = s.consumeVerificationCode(nil, verification.Email, verificationPurposePasswordReset, correctCode)
		}(i)
	}
	start.Done()
	wg.Wait()

	succeeded := 0
	for _, err := range errs {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("verification code consumed %d times across %d concurrent attempts, want exactly 1", succeeded, attempts)
	}

	var stored model.EmailVerificationCode
	if err := s.DB.First(&stored, verification.ID).Error; err != nil {
		t.Fatalf("reload verification code: %v", err)
	}
	if stored.ConsumedAt == nil {
		t.Fatal("verification code consumed_at = nil after race, want set")
	}
}

// TestConsumeVerificationCodeFailureCountingUnderConcurrency proves concurrent
// wrong guesses each land on the failure counter: the atomic increment must
// not lose updates, and once the counter reaches the cap no further attempt
// may return the plain invalid-code error.
func TestConsumeVerificationCodeFailureCountingUnderConcurrency(t *testing.T) {
	db := newTestDB(t)
	configureSingleConnPool(t, db)
	s := &Service{DB: db}

	const correctCode = "123456"
	const wrongCode = "000000"
	hash, err := bcrypt.GenerateFromPassword([]byte(correctCode), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash verification code: %v", err)
	}
	verification := model.EmailVerificationCode{
		Email:     "failure-count@example.com",
		Purpose:   verificationPurposePasswordReset,
		CodeHash:  string(hash),
		ExpiresAt: time.Now().Add(verificationCodeTTL),
	}
	if err := s.DB.Create(&verification).Error; err != nil {
		t.Fatalf("seed verification code: %v", err)
	}

	// Attempt count is double the cap: half the guesses race past the cap
	// check while the counter is still below it.
	attempts := verificationCodeMaxFailures * 2
	errs := make([]error, attempts)
	var wg sync.WaitGroup
	var start sync.WaitGroup
	start.Add(1)
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			start.Wait()
			errs[i] = s.consumeVerificationCode(nil, verification.Email, verificationPurposePasswordReset, wrongCode)
		}(i)
	}
	start.Done()
	wg.Wait()

	var stored model.EmailVerificationCode
	if err := s.DB.First(&stored, verification.ID).Error; err != nil {
		t.Fatalf("reload verification code: %v", err)
	}
	if stored.FailedAttempts < verificationCodeMaxFailures {
		t.Fatalf("failed_attempts after %d concurrent wrong guesses = %d, want at least %d (lost updates)",
			attempts, stored.FailedAttempts, verificationCodeMaxFailures)
	}
	if stored.ConsumedAt == nil {
		t.Fatalf("consumed_at = nil after failure cap was reached, want set")
	}
	for i, err := range errs {
		if err == nil {
			t.Fatalf("attempt %d with wrong code returned nil error, want failure", i)
		}
		if !errors.Is(err, ErrVerificationCodeInvalid) && !errors.Is(err, ErrVerificationCodeTooManyAttempts) {
			t.Fatalf("attempt %d error = %v, want invalid or too-many-attempts", i, err)
		}
	}

	// After the cap the code is dead: a later attempt finds no unconsumed
	// row and reports the generic invalid-code error (same as the pre-fix
	// behavior for an already-consumed code).
	if err := s.consumeVerificationCode(nil, verification.Email, verificationPurposePasswordReset, correctCode); !errors.Is(err, ErrVerificationCodeInvalid) {
		t.Fatalf("consume after cap error = %v, want ErrVerificationCodeInvalid", err)
	}
}

// configureSingleConnPool mirrors the production SQLite pool (one connection)
// so the regression tests exercise the same statement-interleaving behavior as
// the deployed server.
func configureSingleConnPool(t *testing.T, db *gorm.DB) {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access sql DB handle: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
}
