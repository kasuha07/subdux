package api

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	backupservice "github.com/kasuha07/subdux/internal/service/backup"
)

func TestRestoreRefreshesMountedJWTMiddleware(t *testing.T) {
	for _, fixed := range []bool{false, true} {
		name := "database_secret"
		if fixed {
			name = "fixed_secret"
		}
		t.Run(name, func(t *testing.T) {
			t.Setenv("SETTINGS_ENCRYPTION_KEY", "restore-jwt-test-settings-key")
			t.Setenv("JWT_SECRET", "")
			if fixed {
				t.Setenv("JWT_SECRET", strings.Repeat("f", 32))
			}
			replacementDir := t.TempDir()
			t.Setenv("DATA_PATH", replacementDir)
			replacement := pkg.InitDB()
			user := model.User{Username: "restore-admin", Email: "restore@example.com", Password: "unused", Role: "admin", Status: "active"}
			if err := replacement.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			if err := replacement.Create(&model.SystemSetting{Key: "jwt_secret", Value: strings.Repeat("b", 32)}).Error; err != nil {
				t.Fatal(err)
			}
			replacementSQL, err := replacement.DB()
			if err != nil {
				t.Fatal(err)
			}
			if err := replacementSQL.Close(); err != nil {
				t.Fatal(err)
			}

			t.Setenv("DATA_PATH", t.TempDir())
			db := pkg.InitDB()
			t.Cleanup(func() { sqlDB, _ := db.DB(); _ = sqlDB.Close() })
			if err := db.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			if err := db.Create(&model.SystemSetting{Key: "jwt_secret", Value: strings.Repeat("a", 32)}).Error; err != nil {
				t.Fatal(err)
			}
			if err := pkg.InitJWTSecret(db); err != nil {
				t.Fatal(err)
			}
			e := newHumanOnlyRouteTestServer(t, db)
			token := func() string {
				value, err := pkg.GenerateAccessToken(user.ID, user.Username, user.Email, user.Role)
				if err != nil {
					t.Fatal(err)
				}
				return value
			}
			check := func(value string, want int) {
				for _, path := range []string{"/api/auth/me", "/api/subscriptions", "/api/admin/users"} {
					req := httptest.NewRequest(http.MethodGet, path, nil)
					req.Header.Set("Authorization", "Bearer "+value)
					rec := httptest.NewRecorder()
					e.ServeHTTP(rec, req)
					if rec.Code != want {
						t.Fatalf("%s: got %d want %d: %s", path, rec.Code, want, rec.Body.String())
					}
				}
			}
			old := token()
			check(old, 200)
			result, err := backupservice.NewService(db).RestoreBackup(filepath.Join(replacementDir, "subdux.db"), "")
			if err != nil || !result.Reopened {
				t.Fatalf("restore: %+v, %v", result, err)
			}
			check(token(), 200)
			wantOld := 401
			if fixed {
				wantOld = 200
			}
			check(old, wantOld)
		})
	}
}
