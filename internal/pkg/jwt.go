package pkg

import (
	"crypto/rand"
	"encoding/hex"
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
