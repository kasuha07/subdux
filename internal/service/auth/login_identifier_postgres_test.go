package auth

import (
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/migrations"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresLoginIdentifierConstraints(t *testing.T) {
	db := openAuthPostgresTestDB(t)
	first := model.User{Username: "owner", Email: "Owner@Example.com", Password: "unused", Role: "user", Status: "active"}
	if err := db.Create(&first).Error; err != nil {
		t.Fatalf("create first user: %v", err)
	}

	caseDuplicate := model.User{Username: "second", Email: "OWNER@example.com", Password: "unused", Role: "user", Status: "active"}
	if err := db.Create(&caseDuplicate).Error; !errors.Is(mapLoginIdentifierWriteError(err), ErrEmailAlreadyRegistered) {
		t.Fatalf("case-folded email error = %v, want ErrEmailAlreadyRegistered", err)
	}

	emailClaimsUsername := model.User{Username: "owner@example.com", Email: "other@example.com", Password: "unused", Role: "user", Status: "active"}
	if err := db.Create(&emailClaimsUsername).Error; !errors.Is(mapLoginIdentifierWriteError(err), ErrUsernameAlreadyTaken) {
		t.Fatalf("username claim error = %v, want ErrUsernameAlreadyTaken", err)
	}

	usernameOwner := model.User{Username: "reserved@example.com", Email: "reserved-owner@example.com", Password: "unused", Role: "user", Status: "active"}
	if err := db.Create(&usernameOwner).Error; err != nil {
		t.Fatalf("create username owner: %v", err)
	}
	emailClaimsUsername = model.User{Username: "third", Email: "RESERVED@example.com", Password: "unused", Role: "user", Status: "active"}
	if err := db.Create(&emailClaimsUsername).Error; !errors.Is(mapLoginIdentifierWriteError(err), ErrEmailAlreadyRegistered) {
		t.Fatalf("email claim error = %v, want ErrEmailAlreadyRegistered", err)
	}

	seedSystemSetting(t, db, "registration_enabled", "true")
	if _, err := NewService(db).Register(RegisterInput{
		Username: "OWNER@example.com", Email: "registration@example.com", Password: "valid-password",
	}); !errors.Is(err, ErrUsernameAlreadyTaken) {
		t.Fatalf("registration cross-field error = %v, want ErrUsernameAlreadyTaken", err)
	}
}

func TestPostgresConcurrentCrossFieldClaims(t *testing.T) {
	db := openAuthPostgresTestDB(t)
	users := []model.User{
		{Username: "shared@example.com", Email: "first@example.com", Password: "unused", Role: "user", Status: "active"},
		{Username: "second", Email: "SHARED@example.com", Password: "unused", Role: "user", Status: "active"},
	}
	start := make(chan struct{})
	ready := make(chan struct{}, len(users))
	errs := make([]error, len(users))
	var wg sync.WaitGroup
	for i := range users {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx := db.Begin()
			if tx.Error != nil {
				errs[i] = tx.Error
				ready <- struct{}{}
				return
			}
			ready <- struct{}{}
			<-start
			if err := tx.Create(&users[i]).Error; err != nil {
				_ = tx.Rollback().Error
				errs[i] = err
				return
			}
			errs[i] = tx.Commit().Error
		}()
	}
	for range users {
		<-ready
	}
	close(start)
	wg.Wait()

	succeeded, conflicted := 0, 0
	for _, err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		if LoginIdentifierConflictField(err) == "" {
			t.Fatalf("concurrent insert returned non-identifier error: %v", err)
		}
		conflicted++
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("concurrent results: succeeded=%d conflicted=%d, want 1 each", succeeded, conflicted)
	}
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count committed users: %v", err)
	}
	if count != 1 {
		t.Fatalf("committed user count = %d, want 1", count)
	}
}

func TestPostgresLegacyLoginCollisionSurvivesMigration(t *testing.T) {
	db := openAuthPostgresTestDB(t)
	t.Setenv("JWT_SECRET", "postgres-login-test-secret-at-least-32-characters")
	hash := func(password string) string {
		encoded, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
		if err != nil {
			t.Fatal(err)
		}
		return string(encoded)
	}
	// Simulate rows written before the identifier trigger was installed.
	if err := db.Exec(`ALTER TABLE users DISABLE TRIGGER trg_users_login_identifiers`).Error; err != nil {
		t.Fatalf("disable identifier trigger for legacy fixture: %v", err)
	}
	older := model.User{Username: "legacy@example.com", Email: "older@example.com", Password: hash("older-password"), Role: "user", Status: "active"}
	victim := model.User{Username: "victim", Email: "legacy@example.com", Password: hash("victim-password"), Role: "user", Status: "active"}
	for _, user := range []*model.User{&older, &victim} {
		if err := db.Create(user).Error; err != nil {
			t.Fatalf("create legacy user: %v", err)
		}
	}
	if err := db.Exec(`ALTER TABLE users ENABLE TRIGGER trg_users_login_identifiers`).Error; err != nil {
		t.Fatalf("enable identifier trigger: %v", err)
	}
	if err := db.Model(&older).Update("status", "disabled").Error; err != nil {
		t.Fatalf("update unrelated field on legacy collision: %v", err)
	}
	if err := db.Model(&older).Update("status", "active").Error; err != nil {
		t.Fatalf("restore legacy user status: %v", err)
	}

	login, err := NewService(db).Login(LoginInput{Identifier: "LEGACY@example.com", Password: "victim-password"})
	if err != nil {
		t.Fatalf("email owner login: %v", err)
	}
	if login.User == nil || login.User.ID != victim.ID {
		t.Fatalf("email owner login: user=%v, want user %d", login.User, victim.ID)
	}
	if _, err := NewService(db).Login(LoginInput{Identifier: "legacy@example.com", Password: "older-password"}); err == nil {
		t.Fatal("legacy username bypassed email owner's password")
	}
}

func openAuthPostgresTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := strings.TrimSpace(os.Getenv("SUBDUX_TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("set SUBDUX_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}
	u, err := url.Parse(dsn)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatalf("SUBDUX_TEST_POSTGRES_DSN must be a postgres:// or postgresql:// URL")
	}
	admin, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open PostgreSQL admin connection: %v", err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatalf("access PostgreSQL admin connection: %v", err)
	}
	schema := "subdux_auth_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec("CREATE SCHEMA \"" + schema + "\"").Error; err != nil {
		_ = adminSQL.Close()
		t.Fatalf("create isolated PostgreSQL schema: %v", err)
	}
	t.Cleanup(func() {
		if err := admin.Exec("DROP SCHEMA IF EXISTS \"" + schema + "\" CASCADE").Error; err != nil {
			t.Errorf("drop isolated PostgreSQL schema: %v", err)
		}
		_ = adminSQL.Close()
	})
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatalf("open isolated PostgreSQL database: %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("access isolated PostgreSQL database: %v", err)
	}
	if err := migrations.Run(db, migrations.SecretCodec{
		Encrypt: pkg.EncryptSystemSettingValue,
		Decrypt: pkg.DecryptSystemSettingValue,
	}); err != nil {
		_ = sqlDB.Close()
		t.Fatalf("migrate isolated PostgreSQL database: %v", err)
	}
	t.Cleanup(func() {
		if err := sqlDB.Close(); err != nil {
			t.Errorf("close isolated PostgreSQL database: %v", err)
		}
	})
	return db
}
