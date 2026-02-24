package pkg

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type JWTClaims struct {
	UserID   uint     `json:"user_id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Role     string   `json:"role"`
	AuthType string   `json:"auth_type,omitempty"`
	Scopes   []string `json:"scopes,omitempty"`
	jwt.RegisteredClaims
}

const (
	jwtSecretKey       = "jwt_secret"
	minJWTSecretLength = 32

	defaultAccessTokenTTL  = 15 * time.Minute
	minAccessTokenTTL      = 1 * time.Minute
	defaultRefreshTokenTTL = 30 * 24 * time.Hour
	minRefreshTokenTTL     = 1 * time.Hour
)

const (
	AuthTypeUser   = "user"
	AuthTypeAPIKey = "api_key"
)

var jwtSecretFromDB string

func InitJWTSecret(db *gorm.DB) error {
	if envSecret := os.Getenv("JWT_SECRET"); envSecret != "" {
		if err := validateJWTSecret(envSecret, "JWT_SECRET environment variable"); err != nil {
			return err
		}
		jwtSecretFromDB = ""
		return nil
	}

	var setting model.SystemSetting
	if err := db.Where("key = ?", jwtSecretKey).First(&setting).Error; err == nil {
		if err := validateJWTSecret(setting.Value, "database system setting jwt_secret"); err != nil {
			return err
		}
		jwtSecretFromDB = setting.Value
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to load JWT secret from database: %w", err)
	}

	secret, err := generateJWTSecret()
	if err != nil {
		return err
	}

	if err := db.Create(&model.SystemSetting{Key: jwtSecretKey, Value: secret}).Error; err != nil {
		return fmt.Errorf("failed to save JWT secret to database: %w", err)
	}

	jwtSecretFromDB = secret
	log.Println("Generated new JWT secret on first run")
	return nil
}

func GetJWTSecret() []byte {
	return []byte(getJWTSecretValue())
}

func getJWTSecretValue() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	if jwtSecretFromDB != "" {
		return jwtSecretFromDB
	}
	panic("JWT secret is not initialized: call InitJWTSecret during startup")
}

func generateJWTSecret() (string, error) {
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	secret := hex.EncodeToString(secretBytes)
	if err := validateJWTSecret(secret, "generated JWT secret"); err != nil {
		return "", err
	}
	return secret, nil
}

func validateJWTSecret(secret string, source string) error {
	trimmed := strings.TrimSpace(secret)
	if trimmed != secret {
		return fmt.Errorf("%s must not include leading or trailing whitespace", source)
	}
	if len(secret) < minJWTSecretLength {
		return fmt.Errorf("%s must be at least %d characters long", source, minJWTSecretLength)
	}
	return nil
}

// GenerateToken issues the short-lived access token used on normal API requests.
// Kept for compatibility with existing call-sites.
func GenerateToken(userID uint, username string, email string, role string) (string, error) {
	return GenerateAccessToken(userID, username, email, role)
}

func GenerateAccessToken(userID uint, username string, email string, role string) (string, error) {
	now := time.Now().UTC()
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		AuthType: AuthTypeUser,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(getAccessTokenTTL())),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
}

func getAccessTokenTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("ACCESS_TOKEN_TTL_MINUTES"))
	if raw == "" {
		return defaultAccessTokenTTL
	}

	minutes, err := strconv.Atoi(raw)
	if err != nil || minutes <= 0 {
		log.Printf("invalid ACCESS_TOKEN_TTL_MINUTES value %q, using default", raw)
		return defaultAccessTokenTTL
	}

	ttl := time.Duration(minutes) * time.Minute
	if ttl < minAccessTokenTTL {
		return minAccessTokenTTL
	}
	return ttl
}

func getRefreshTokenTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("REFRESH_TOKEN_TTL_HOURS"))
	if raw == "" {
		return defaultRefreshTokenTTL
	}

	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		log.Printf("invalid REFRESH_TOKEN_TTL_HOURS value %q, using default", raw)
		return defaultRefreshTokenTTL
	}

	ttl := time.Duration(hours) * time.Hour
	if ttl < minRefreshTokenTTL {
		return minRefreshTokenTTL
	}
	return ttl
}

func HashRefreshToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func GenerateRefreshToken() (string, string, time.Time, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", time.Time{}, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	rawToken := "sdr_" + hex.EncodeToString(tokenBytes)
	return rawToken, HashRefreshToken(rawToken), time.Now().UTC().Add(getRefreshTokenTTL()), nil
}

// TOTPPendingClaims is a short-lived intermediate token (5 min) issued after
// password auth succeeds when 2FA is enabled. Only accepted by /api/auth/totp/verify-login.
type TOTPPendingClaims struct {
	UserID      uint `json:"user_id"`
	PendingTOTP bool `json:"pending_totp"`
	jwt.RegisteredClaims
}

func GenerateTOTPPendingToken(userID uint) (string, error) {
	claims := &TOTPPendingClaims{
		UserID:      userID,
		PendingTOTP: true,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
}

func ValidateTOTPPendingToken(tokenStr string) (uint, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &TOTPPendingClaims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return GetJWTSecret(), nil
	})
	if err != nil {
		return 0, err
	}
	claims, ok := token.Claims.(*TOTPPendingClaims)
	if !ok || !token.Valid {
		return 0, errors.New("invalid token")
	}
	if !claims.PendingTOTP {
		return 0, errors.New("not a pending TOTP token")
	}
	return claims.UserID, nil
}
