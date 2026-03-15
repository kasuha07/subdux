package service

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func TestRecordPasskeyLoginMetadataAsyncDoesNotBlockResponse(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "subdux-test.db")

	openDB := func() *gorm.DB {
		t.Helper()

		db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
		if err != nil {
			t.Fatalf("failed to open test database: %v", err)
		}
		if err := db.AutoMigrate(&model.User{}, &model.PasskeyCredential{}); err != nil {
			t.Fatalf("failed to migrate test database: %v", err)
		}
		return db
	}

	db := openDB()
	lockDB := openDB()

	user := createTestUser(t, db)
	credentialID := []byte("slow-passkey")
	record := model.PasskeyCredential{
		UserID:       user.ID,
		Name:         "Primary",
		CredentialID: encodeCredentialID(credentialID),
		Credential:   []byte(`{"stale":true}`),
	}
	if err := db.Create(&record).Error; err != nil {
		t.Fatalf("failed to create passkey record: %v", err)
	}

	tx := lockDB.Begin()
	if tx.Error != nil {
		t.Fatalf("failed to begin locking transaction: %v", tx.Error)
	}
	if err := tx.Model(&model.PasskeyCredential{}).
		Where("id = ?", record.ID).
		Update("name", "Locked").
		Error; err != nil {
		t.Fatalf("failed to acquire passkey write lock: %v", err)
	}

	releaseDone := make(chan struct{})
	go func() {
		time.Sleep(300 * time.Millisecond)
		if err := tx.Commit().Error; err != nil {
			t.Errorf("failed to release write lock: %v", err)
		}
		close(releaseDone)
	}()

	authService := &AuthService{DB: db}

	start := time.Now()
	authService.recordPasskeyLoginMetadataAsync(user.ID, &webauthn.Credential{ID: credentialID}, time.Now().UTC())
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Fatalf("recordPasskeyLoginMetadataAsync() blocked for %v under sqlite write lock", elapsed)
	}

	<-releaseDone

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		var updated model.PasskeyCredential
		if err := db.First(&updated, record.ID).Error; err != nil {
			t.Fatalf("failed to reload passkey record: %v", err)
		}
		if updated.LastUsedAt != nil {
			return
		}
		time.Sleep(25 * time.Millisecond)
	}

	t.Fatal("passkey metadata update did not complete after lock release")
}
