package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/shiroha/subdux/internal/model"
	"gorm.io/gorm"
)

type JWTClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

const jwtSecretKey = "jwt_secret"
const minJWTSecretLength = 32

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

func GenerateToken(userID uint, username string, email string, role string) (string, error) {
	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		Email:    email,
		Role:     role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(72 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(GetJWTSecret())
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
