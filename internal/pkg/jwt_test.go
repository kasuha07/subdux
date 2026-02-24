package pkg

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/golang-jwt/jwt/v5"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

func newJWTTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "subdux-jwt-test.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	if err := db.AutoMigrate(&model.SystemSetting{}); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return db
}

func TestInitJWTSecretGeneratesAndPersistsSecretOnFirstBoot(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	jwtSecretFromDB = ""

	db := newJWTTestDB(t)
	if err := InitJWTSecret(db); err != nil {
		t.Fatalf("InitJWTSecret() error = %v, want nil", err)
	}

	if len(jwtSecretFromDB) < minJWTSecretLength {
		t.Fatalf("generated secret length = %d, want at least %d", len(jwtSecretFromDB), minJWTSecretLength)
	}

	var stored model.SystemSetting
	if err := db.Where("key = ?", jwtSecretKey).First(&stored).Error; err != nil {
		t.Fatalf("failed to load persisted JWT secret: %v", err)
	}
	if stored.Value != jwtSecretFromDB {
		t.Fatalf("persisted secret mismatch: got %q, want %q", stored.Value, jwtSecretFromDB)
	}
}

func TestInitJWTSecretRejectsWeakEnvironmentSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "short-secret")
	jwtSecretFromDB = ""

	db := newJWTTestDB(t)
	err := InitJWTSecret(db)
	if err == nil {
		t.Fatal("InitJWTSecret() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "at least") {
		t.Fatalf("InitJWTSecret() error = %q, want length validation error", err.Error())
	}
}

func TestInitJWTSecretRejectsWeakDatabaseSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "")
	jwtSecretFromDB = ""

	db := newJWTTestDB(t)
	if err := db.Create(&model.SystemSetting{Key: jwtSecretKey, Value: "weak"}).Error; err != nil {
		t.Fatalf("failed to seed weak JWT secret: %v", err)
	}

	err := InitJWTSecret(db)
	if err == nil {
		t.Fatal("InitJWTSecret() error = nil, want error")
	}
	if !strings.Contains(err.Error(), "at least") {
		t.Fatalf("InitJWTSecret() error = %q, want length validation error", err.Error())
	}
}

func TestGenerateTokenIncludesUserAuthTypeAndShortExpiry(t *testing.T) {
	t.Setenv("JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("ACCESS_TOKEN_TTL_MINUTES", "")
	jwtSecretFromDB = ""

	tokenStr, err := GenerateToken(7, "alice", "alice@example.com", "admin")
	if err != nil {
		t.Fatalf("GenerateToken() error = %v, want nil", err)
	}

	parsed, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(t *jwt.Token) (interface{}, error) {
		return GetJWTSecret(), nil
	})
	if err != nil {
		t.Fatalf("ParseWithClaims() error = %v, want nil", err)
	}

	claims, ok := parsed.Claims.(*JWTClaims)
	if !ok || !parsed.Valid {
		t.Fatal("token claims invalid")
	}
	if claims.AuthType != AuthTypeUser {
		t.Fatalf("claims.AuthType = %q, want %q", claims.AuthType, AuthTypeUser)
	}

	ttl := claims.ExpiresAt.Time.Sub(claims.IssuedAt.Time)
	if ttl < 14*time.Minute || ttl > 16*time.Minute {
		t.Fatalf("token ttl = %v, want around 15m", ttl)
	}
}

func TestGenerateRefreshTokenReturnsHashAndExpiry(t *testing.T) {
	t.Setenv("REFRESH_TOKEN_TTL_HOURS", "")

	token, hash, expiresAt, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v, want nil", err)
	}
	if !strings.HasPrefix(token, "sdr_") {
		t.Fatalf("token prefix = %q, want sdr_", token)
	}

	expectedHash := HashRefreshToken(token)
	if hash != expectedHash {
		t.Fatalf("refresh token hash mismatch: got %q, want %q", hash, expectedHash)
	}

	minExpiry := time.Now().Add(defaultRefreshTokenTTL - time.Hour)
	if expiresAt.Before(minExpiry) {
		t.Fatalf("refresh token expiry = %v, want >= %v", expiresAt, minExpiry)
	}
}
