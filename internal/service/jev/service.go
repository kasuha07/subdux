// Package jev provides optional, user-scoped subscription classification.
package jev

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	db *gorm.DB
	// An internal transport seam; production always uses the fixed TypeSafe endpoint.
	evaluate func(context.Context, string, SuggestInput, []model.Category) (*uint, error)
	check    func(context.Context, string) ConnectionStatus
}

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

type Settings struct {
	Revision                  uint64           `json:"revision"`
	Enabled                   bool             `json:"enabled"`
	APIKeyConfigured          bool             `json:"api_key_configured"`
	ConnectionStatus          ConnectionStatus `json:"connection_status"`
	LastCheckedAt             *time.Time       `json:"last_checked_at"`
	LastSuccessAt             *time.Time       `json:"last_success_at"`
	ClassificationRequests    uint64           `json:"classification_requests"`
	ClassificationSuggestions uint64           `json:"classification_suggestions"`
}

type UpdateInput struct {
	Revision     uint64 `json:"revision"`
	Enabled      bool   `json:"enabled"`
	APIKey       string `json:"api_key"`
	RemoveAPIKey bool   `json:"remove_api_key"`
}

var errConflict = serviceerr.New(serviceerr.KindConflict, "jev_settings_conflict", "Jev settings changed; reload and try again")

func settingsResponse(row model.UserJevSetting) Settings {
	configured := row.APIKey != ""
	status := ConnectionStatus(row.ConnectionStatus)
	if !configured {
		status = ConnectionNotConfigured
	} else if status == "" {
		status = ConnectionNotTested
	}
	return Settings{
		Revision:                  row.Revision,
		Enabled:                   row.Enabled,
		APIKeyConfigured:          configured,
		ConnectionStatus:          status,
		LastCheckedAt:             row.LastCheckedAt,
		LastSuccessAt:             row.LastSuccessAt,
		ClassificationRequests:    row.ClassificationRequests,
		ClassificationSuggestions: row.ClassificationSuggestions,
	}
}

func (s *Service) load(ctx context.Context, userID uint) (model.UserJevSetting, error) {
	var row model.UserJevSetting
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.UserJevSetting{UserID: userID}, nil
	}
	return row, err
}

func (s *Service) GetSettings(ctx context.Context, userID uint) (Settings, error) {
	row, err := s.load(ctx, userID)
	return settingsResponse(row), err
}

func (s *Service) UpdateSettings(ctx context.Context, userID uint, input UpdateInput) (Settings, error) {
	key := strings.TrimSpace(input.APIKey)
	if len(key) > 512 || strings.ContainsAny(key, "\r\n\t ") || (input.RemoveAPIKey && key != "") {
		return Settings{}, serviceerr.New(serviceerr.KindInvalid, "jev_invalid_api_key", "Invalid Jev API key")
	}
	row, err := s.load(ctx, userID)
	if err != nil {
		return Settings{}, err
	}
	if input.Revision != row.Revision {
		return Settings{}, errConflict
	}
	if input.RemoveAPIKey {
		row.APIKey = ""
	}
	keyChanged := input.RemoveAPIKey || key != ""
	if key != "" {
		row.APIKey, err = pkg.EncryptSystemSettingValue(key)
		if err != nil {
			return Settings{}, err
		}
	}
	row.Enabled = input.Enabled && !input.RemoveAPIKey
	if row.Enabled && row.APIKey == "" {
		return Settings{}, serviceerr.New(serviceerr.KindInvalid, "jev_api_key_required", "Configure a Jev API key before enabling classification")
	}
	row.Revision++
	if keyChanged {
		row.ConnectionStatus = string(ConnectionNotTested)
		row.LastCheckedAt = nil
		row.LastSuccessAt = nil
		row.ClassificationRequests = 0
		row.ClassificationSuggestions = 0
	}
	db := s.db.WithContext(ctx)
	var result *gorm.DB
	if input.Revision == 0 {
		result = db.Clauses(clause.OnConflict{DoNothing: true}).Create(&row)
	} else {
		updates := map[string]any{"enabled": row.Enabled, "api_key": row.APIKey, "revision": row.Revision}
		if keyChanged {
			updates["connection_status"] = row.ConnectionStatus
			updates["last_checked_at"] = nil
			updates["last_success_at"] = nil
			updates["classification_requests"] = 0
			updates["classification_suggestions"] = 0
		}
		result = db.Model(&model.UserJevSetting{}).Where("user_id = ? AND revision = ?", userID, input.Revision).Updates(updates)
	}
	if result.Error != nil {
		return Settings{}, result.Error
	}
	if result.RowsAffected != 1 {
		return Settings{}, errConflict
	}
	return settingsResponse(row), nil
}

func (s *Service) TestConnection(ctx context.Context, userID uint) (Settings, error) {
	row, err := s.load(ctx, userID)
	if err != nil || row.APIKey == "" {
		return settingsResponse(row), err
	}
	key, err := pkg.DecryptSystemSettingValue(row.APIKey)
	if err != nil {
		if recordErr := s.recordDiagnostic(ctx, userID, row.Revision, ConnectionUnavailable, false); recordErr != nil {
			return Settings{}, recordErr
		}
	} else {
		check := s.check
		if check == nil {
			check = s.checkConnection
		}
		if err := s.recordDiagnostic(ctx, userID, row.Revision, check(ctx, key), false); err != nil {
			return Settings{}, err
		}
	}
	row, err = s.load(ctx, userID)
	return settingsResponse(row), err
}

func (s *Service) recordDiagnostic(ctx context.Context, userID uint, revision uint64, status ConnectionStatus, classificationRequest bool) error {
	now := time.Now().UTC()
	updates := map[string]any{"connection_status": status, "last_checked_at": now}
	if status == ConnectionAvailable {
		updates["last_success_at"] = now
	}
	if classificationRequest {
		updates["classification_requests"] = gorm.Expr("classification_requests + 1")
	}
	return s.db.WithContext(ctx).Model(&model.UserJevSetting{}).
		Where("user_id = ? AND revision = ?", userID, revision).Updates(updates).Error
}

type SuggestInput struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Suggestion struct {
	CategoryID *uint `json:"category_id"`
}

func (s *Service) Suggest(ctx context.Context, userID uint, input SuggestInput) (Suggestion, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.URL = strings.TrimSpace(input.URL)
	if len(input.Name) > 1020 || len(input.URL) > 2048 {
		return Suggestion{}, serviceerr.New(serviceerr.KindInvalid, "jev_input_too_long", "Subscription name or URL is too long")
	}
	row, err := s.load(ctx, userID)
	if err != nil {
		return Suggestion{}, err
	}
	if !row.Enabled || row.APIKey == "" || input.Name == "" {
		return Suggestion{}, nil
	}
	var categories []model.Category
	// One of the provider's 255 choices is reserved for abstention. Do not
	// truncate a larger taxonomy and then silently classify against a subset.
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).Order("id").Limit(255).Find(&categories).Error; err != nil {
		return Suggestion{}, err
	}
	if len(categories) == 0 || len(categories) > 254 {
		return Suggestion{}, nil
	}
	key, err := pkg.DecryptSystemSettingValue(row.APIKey)
	if err != nil {
		_ = s.recordDiagnostic(ctx, userID, row.Revision, ConnectionUnavailable, false)
		return Suggestion{}, nil
	}
	evaluate := s.evaluate
	if evaluate == nil {
		evaluate = s.classify
	}
	categoryID, err := evaluate(ctx, key, input, categories)
	status := ConnectionAvailable
	if err != nil {
		status = connectionStatusForError(err)
	}
	// Diagnostics are best-effort and must not make subscription entry fail.
	_ = s.recordDiagnostic(ctx, userID, row.Revision, status, true)
	// Suggestions are best-effort; neither upstream errors nor response bodies
	// should leak credentials or interfere with subscription entry.
	if err != nil || categoryID == nil {
		return Suggestion{}, nil
	}
	// Recheck ownership and opt-in after the network call (settings/categories
	// may have changed while it was in flight).
	current, err := s.load(ctx, userID)
	if err != nil {
		return Suggestion{}, err
	}
	if !current.Enabled || current.Revision != row.Revision {
		return Suggestion{}, nil
	}
	var count int64
	err = s.db.WithContext(ctx).Model(&model.Category{}).Where("id = ? AND user_id = ?", *categoryID, userID).Count(&count).Error
	if err != nil {
		return Suggestion{}, err
	}
	if count != 1 {
		return Suggestion{}, nil
	}
	_ = s.db.WithContext(ctx).Model(&model.UserJevSetting{}).Where("user_id = ? AND revision = ?", userID, row.Revision).
		UpdateColumn("classification_suggestions", gorm.Expr("classification_suggestions + 1")).Error
	return Suggestion{CategoryID: categoryID}, nil
}
