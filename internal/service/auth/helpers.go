package auth

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	"github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

func withContext(db *gorm.DB, ctx context.Context) *gorm.DB {
	if db == nil {
		return nil
	}
	return db.WithContext(ctx)
}

func generateSecureToken(byteLen int) (string, error) {
	return serviceutil.GenerateSecureToken(byteLen)
}

func seedUserDefaults(tx *gorm.DB, userID uint) error {
	return serviceutil.SeedUserDefaults(tx, userID)
}

func getSystemSettingString(ctx context.Context, db *gorm.DB, key string, defaultValue string) (string, error) {
	return settings.GetString(ctx, db, key, defaultValue)
}

func getSystemSettingBool(ctx context.Context, db *gorm.DB, key string, defaultValue bool) (bool, error) {
	return settings.GetBool(ctx, db, key, defaultValue)
}

func buildOIDCOutboundHTTPClient(ctx context.Context, db *gorm.DB, timeout time.Duration) (*http.Client, error) {
	return outbound.BuildHTTPClientWithTimeout(ctx, db, outbound.PurposeOIDC, timeout)
}

func validateHTTPURL(rawURL string, fieldLabel string, requireHTTPS bool) (*url.URL, error) {
	return outbound.ValidateHTTPURL(rawURL, fieldLabel, requireHTTPS)
}
