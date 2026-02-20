package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log"
	"os"
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

var jwtSecretFromDB string

func InitJWTSecret(db *gorm.DB) {
	var setting model.SystemSetting
	if err := db.Where("key = ?", jwtSecretKey).First(&setting).Error; err == nil {
		jwtSecretFromDB = setting.Value
		return
	}

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		log.Fatalf("Failed to generate JWT secret: %v", err)
	}
	secret := hex.EncodeToString(secretBytes)

	if err := db.Create(&model.SystemSetting{Key: jwtSecretKey, Value: secret}).Error; err != nil {
		log.Fatalf("Failed to save JWT secret to database: %v", err)
	}

	jwtSecretFromDB = secret
	log.Println("Generated new JWT secret on first run")
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
	return "subdux-default-secret-change-in-production"
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
