package mcpoauth

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kasuha07/subdux/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestPostgresConcurrentCodeReplayRevokesFamily(t *testing.T) {
	db := openOAuthPostgresTestDB(t)
	if err := db.Create(&model.SystemSetting{Key: "site_url", Value: "https://subdux.example"}).Error; err != nil {
		t.Fatal(err)
	}
	user := model.User{Username: "review", Email: "review@example.com", Password: "hash", Role: "user", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	s := NewService(db)
	client, err := s.Register(ClientMetadata{ClientName: "Test agent", RedirectURIs: []string{"http://127.0.0.1/callback"}})
	if err != nil {
		t.Fatal(err)
	}
	params, _ := authorize(t, s, user, client, true)
	seedOAuthTestGrants(t, db, user.ID, client, 49)

	// Both transactions must observe the unused code before either consumes it.
	// Random concurrent requests can miss this PostgreSQL-specific interleaving.
	reached := make(chan struct{}, 2)
	release := make(chan struct{})
	firstFinished := make(chan struct{})
	type exchangeAttemptKey struct{}
	var releaseOnce sync.Once
	releaseReads := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(releaseReads)
	if err := db.Callback().Query().After("gorm:query").Register("test:unused_code_barrier", func(tx *gorm.DB) {
		if tx.Statement.Table == "mcp_oauth_requests" && strings.Contains(tx.Statement.SQL.String(), "code_hash =") {
			reached <- struct{}{}
			<-release
			// The winner occupies the final connection slot before the loser
			// continues with its stale view of the code. Quota checks must not
			// prevent detection and revocation of that concurrent replay.
			if tx.Statement.Context.Value(exchangeAttemptKey{}) == 1 {
				select {
				case <-firstFinished:
				case <-tx.Statement.Context.Done():
				}
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	type exchangeResult struct {
		token *TokenResponse
		err   error
	}
	results := make(chan exchangeResult, 2)
	for attempt := range 2 {
		go func() {
			if attempt == 0 {
				defer close(firstFinished)
			}
			attemptContext := context.WithValue(ctx, exchangeAttemptKey{}, attempt)
			token, err := s.WithContext(attemptContext).Exchange(params)
			results <- exchangeResult{token, err}
		}()
	}
	for range 2 {
		select {
		case <-reached:
		case <-ctx.Done():
			t.Fatal("code exchanges did not reach the unused-code barrier")
		}
	}
	releaseReads()
	var issued *TokenResponse
	successes, replays := 0, 0
	for range 2 {
		select {
		case result := <-results:
			if result.err == nil {
				successes++
				issued = result.token
				continue
			}
			var protocol *ProtocolError
			if !errors.As(result.err, &protocol) || protocol.Code != "invalid_grant" {
				t.Fatalf("competing code exchange error = %v, want invalid_grant", result.err)
			}
			replays++
		case <-ctx.Done():
			t.Fatal("code exchanges did not finish")
		}
	}
	if successes != 1 || replays != 1 {
		t.Fatalf("exchanges: successes=%d replays=%d, want one each", successes, replays)
	}
	if _, err := s.ValidateToken(issued.AccessToken); err == nil {
		t.Fatal("concurrent authorization code replay left the access token usable")
	}
	refresh := url.Values{"grant_type": {"refresh_token"}, "client_id": {client.ClientID}, "resource": {"https://subdux.example/mcp"}, "refresh_token": {issued.RefreshToken}}
	if _, err := s.Exchange(refresh); err == nil {
		t.Fatal("concurrent authorization code replay left the refresh token usable")
	}
	var grants []model.MCPOAuthGrant
	if err := db.Find(&grants).Error; err != nil {
		t.Fatal(err)
	}
	var revoked int
	for _, grant := range grants {
		if grant.RevokedAt != nil {
			revoked++
		}
	}
	if len(grants) != 50 || revoked != 1 {
		t.Fatalf("concurrent code exchanges: grants=%d revoked=%d, want 50 and 1", len(grants), revoked)
	}
}

func openOAuthPostgresTestDB(t *testing.T) *gorm.DB {
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
		t.Fatal(err)
	}
	adminSQL, err := admin.DB()
	if err != nil {
		t.Fatal(err)
	}
	schema := "subdux_oauth_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if err := admin.Exec(`CREATE SCHEMA "` + schema + `"`).Error; err != nil {
		_ = adminSQL.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := admin.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`).Error; err != nil {
			t.Errorf("drop isolated OAuth schema: %v", err)
		}
		_ = adminSQL.Close()
	})
	query := u.Query()
	query.Set("search_path", schema)
	u.RawQuery = query.Encode()
	db, err := gorm.Open(postgres.Open(u.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(4)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.User{}, &model.SystemSetting{}, &model.MCPOAuthClient{}, &model.MCPOAuthGrant{}, &model.MCPOAuthRequest{}, &model.MCPOAuthToken{}); err != nil {
		t.Fatal(err)
	}
	return db
}
