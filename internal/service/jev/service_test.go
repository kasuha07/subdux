package jev

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/servicetest"
)

func testService(t *testing.T) (*Service, uint) {
	t.Helper()
	t.Setenv("SETTINGS_ENCRYPTION_KEY", "jev-test-encryption-key")
	db := servicetest.NewDB(t)
	if err := db.AutoMigrate(&model.UserJevSetting{}); err != nil {
		t.Fatal(err)
	}
	return NewService(db), servicetest.CreateUser(t, db).ID
}

func TestSettingsIsolationEncryptionAndRevision(t *testing.T) {
	s, userID := testService(t)
	ctx := context.Background()
	initial, err := s.GetSettings(ctx, userID)
	if err != nil || initial.Enabled || initial.APIKeyConfigured || initial.Revision != 0 {
		t.Fatalf("initial = %+v, %v", initial, err)
	}
	if _, err := s.UpdateSettings(ctx, userID, UpdateInput{Enabled: true}); err == nil {
		t.Fatal("enabled without a key")
	}
	settings, err := s.UpdateSettings(ctx, userID, UpdateInput{Enabled: true, APIKey: "test-secret"})
	if err != nil {
		t.Fatal(err)
	}
	var row model.UserJevSetting
	if err := s.db.First(&row, "user_id = ?", userID).Error; err != nil {
		t.Fatal(err)
	}
	if !pkg.IsSystemSettingEncrypted(row.APIKey) || strings.Contains(row.APIKey, "test-secret") {
		t.Fatal("key was not encrypted")
	}
	key, err := pkg.DecryptSystemSettingValue(row.APIKey)
	if err != nil || key != "test-secret" {
		t.Fatal("key did not round trip")
	}
	encoded, _ := json.Marshal(settings)
	if strings.Contains(string(encoded), "test-secret") || strings.Contains(string(encoded), "enc:v1") {
		t.Fatal("settings exposed a secret")
	}
	other, err := s.GetSettings(ctx, userID+1)
	if err != nil || other.Enabled || other.APIKeyConfigured {
		t.Fatal("settings crossed users")
	}
	if _, err := s.UpdateSettings(ctx, userID, UpdateInput{APIKey: "overwrite"}); !errors.Is(err, errConflict) {
		t.Fatalf("stale update = %v", err)
	}
	settings, err = s.UpdateSettings(ctx, userID, UpdateInput{Revision: settings.Revision, Enabled: false})
	if err != nil || !settings.APIKeyConfigured || settings.Enabled {
		t.Fatalf("blank key not preserved: %+v %v", settings, err)
	}
	settings, err = s.UpdateSettings(ctx, userID, UpdateInput{Revision: settings.Revision, Enabled: true, RemoveAPIKey: true})
	if err != nil || settings.APIKeyConfigured || settings.Enabled {
		t.Fatalf("remove: %+v %v", settings, err)
	}
	for _, input := range []UpdateInput{{APIKey: "bad\nkey"}, {APIKey: strings.Repeat("a", 513)}, {APIKey: "key", RemoveAPIKey: true}} {
		input.Revision = settings.Revision
		if _, err := s.UpdateSettings(ctx, userID, input); err == nil {
			t.Fatal("accepted invalid key")
		}
	}
}

func TestSuggestionsRequireOptInAndOwnedCategories(t *testing.T) {
	s, userID := testService(t)
	ctx := context.Background()
	calls := 0
	s.evaluate = func(_ context.Context, key string, input SuggestInput, categories []model.Category) (*uint, error) {
		calls++
		if key != "test-secret" || len(categories) != 1 || categories[0].UserID != userID {
			t.Fatal("wrong user context")
		}
		id := categories[0].ID
		return &id, nil
	}
	input := SuggestInput{Name: "Spotify", URL: "https://spotify.com"}
	if _, err := s.Suggest(ctx, userID, input); err != nil || calls != 0 {
		t.Fatal("disabled user reached provider")
	}
	settings, err := s.UpdateSettings(ctx, userID, UpdateInput{Enabled: true, APIKey: "test-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Suggest(ctx, userID, input); err != nil || calls != 0 {
		t.Fatal("empty taxonomy reached provider")
	}
	category := model.Category{UserID: userID, Name: "Music"}
	if err := s.db.Create(&category).Error; err != nil {
		t.Fatal(err)
	}
	otherUser := model.User{Username: "other", Email: "other@example.com", Password: "hash", Role: "user", Status: "active"}
	if err := s.db.Create(&otherUser).Error; err != nil {
		t.Fatal(err)
	}
	otherCategory := model.Category{UserID: otherUser.ID, Name: "Private"}
	if err := s.db.Create(&otherCategory).Error; err != nil {
		t.Fatal(err)
	}
	suggestion, err := s.Suggest(ctx, userID, input)
	if err != nil || suggestion.CategoryID == nil || *suggestion.CategoryID != category.ID {
		t.Fatalf("suggestion = %+v, %v", suggestion, err)
	}
	s.evaluate = func(context.Context, string, SuggestInput, []model.Category) (*uint, error) {
		return &otherCategory.ID, nil
	}
	suggestion, err = s.Suggest(ctx, userID, input)
	if err != nil || suggestion.CategoryID != nil {
		t.Fatal("cross-user category accepted")
	}
	s.evaluate = func(context.Context, string, SuggestInput, []model.Category) (*uint, error) {
		return nil, errors.New("upstream included test-secret")
	}
	suggestion, err = s.Suggest(ctx, userID, input)
	if err != nil || suggestion.CategoryID != nil {
		t.Fatal("upstream error escaped")
	}
	s.evaluate = func(context.Context, string, SuggestInput, []model.Category) (*uint, error) {
		_, err := s.UpdateSettings(ctx, userID, UpdateInput{Revision: settings.Revision, Enabled: false})
		if err != nil {
			t.Fatal(err)
		}
		return &category.ID, nil
	}
	suggestion, err = s.Suggest(ctx, userID, input)
	if err != nil || suggestion.CategoryID != nil {
		t.Fatal("disabled in-flight suggestion accepted")
	}
}

// Explicitly opt-in, synthetic-data smoke test. No credential is committed or logged.
func TestLiveClassification(t *testing.T) {
	path := os.Getenv("JEV_TEST_KEY_FILE")
	if path == "" {
		t.Skip("set JEV_TEST_KEY_FILE for a live TypeSafe smoke test")
	}
	secret, err := os.ReadFile(path)
	if err != nil {
		t.Fatal("unable to read test credential")
	}
	s, userID := testService(t)
	ctx := context.Background()
	if _, err := s.UpdateSettings(ctx, userID, UpdateInput{Enabled: true, APIKey: strings.TrimSpace(string(secret))}); err != nil {
		t.Fatal("unable to configure test credential")
	}
	categories := []model.Category{{UserID: userID, Name: "音乐"}, {UserID: userID, Name: "开发工具"}, {UserID: userID, Name: "视频娱乐"}}
	if err := s.db.Create(&categories).Error; err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, url string
		want      uint
	}{{"Spotify", "https://spotify.com", categories[0].ID}, {"GitHub Copilot", "https://github.com", categories[1].ID}, {"Netflix", "https://netflix.com", categories[2].ID}} {
		t.Run(tc.name, func(t *testing.T) {
			// Call the production provider path directly so upstream failure cannot
			// masquerade as the service's intentionally quiet fallback.
			id, err := s.classify(ctx, strings.TrimSpace(string(secret)), SuggestInput{Name: tc.name, URL: tc.url}, categories)
			if err != nil {
				t.Fatal(err)
			}
			if id == nil || *id != tc.want {
				t.Fatalf("unexpected category for %s", tc.name)
			}
		})
	}
}
