package backup

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	backupsettings "github.com/kasuha07/subdux/internal/service/settings"
	"gorm.io/gorm"
)

var (
	ErrBackupDestinationNotFound    = serviceerr.New(serviceerr.KindNotFound, "backup_destination_not_found", "backup destination not found")
	ErrBackupDestinationChanged     = serviceerr.New(serviceerr.KindConflict, "backup_destination_changed", "backup destination has changed; reload it and try again")
	ErrInvalidBackupDestinationType = serviceerr.New(serviceerr.KindInvalid, "invalid_backup_destination_type", "invalid backup destination type")
)

// validDestinationTypes gates what an admin may create. Only types with a
// working target implementation are accepted, so an enabled row can never be
// created that a scheduled run would fail on with "unknown type".
var validDestinationTypes = map[string]struct{}{
	"local":  {},
	"s3":     {},
	"webdav": {},
}

func isValidDestinationType(destinationType string) bool {
	_, ok := validDestinationTypes[strings.ToLower(strings.TrimSpace(destinationType))]
	return ok
}

// CreateDestinationInput and UpdateDestinationInput are the service-layer inputs
// for destination mutations. Config is the raw (plaintext) JSON submitted by the
// admin; secrets within it are encrypted before persistence.
type CreateDestinationInput struct {
	Type      string
	Enabled   bool
	Config    string
	SortOrder int
}

type UpdateDestinationInput struct {
	Revision            uint64
	Enabled             *bool
	Config              *string
	SortOrder           *int
	ClearedSecretFields []string
}

// listEnabledDestinations returns the enabled destinations in delivery order.
func (s *Service) listEnabledDestinations() ([]model.BackupDestination, error) {
	var destinations []model.BackupDestination
	err := s.DB.Where("enabled = ?", true).
		Order("sort_order ASC").Order("id ASC").
		Find(&destinations).Error
	return destinations, err
}

// ListDestinations returns every configured destination with secret fields
// masked, plus the list of secret fields that are actually set, so the admin UI
// can distinguish "configured" from "empty" without receiving the secret.
type DestinationView struct {
	model.BackupDestination
	ConfiguredSecretFields []string `json:"configured_secret_fields"`
}

func destinationView(destination model.BackupDestination) (DestinationView, error) {
	masked, configured, err := sanitizeDestinationConfig(destination.Type, destination.Config)
	if err != nil {
		return DestinationView{}, err
	}
	destination.Config = masked
	return DestinationView{
		BackupDestination:      destination,
		ConfiguredSecretFields: configured,
	}, nil
}

func (s *Service) ListDestinations() ([]DestinationView, error) {
	var destinations []model.BackupDestination
	if err := s.DB.Order("sort_order ASC").Order("id ASC").Find(&destinations).Error; err != nil {
		return nil, err
	}

	views := make([]DestinationView, 0, len(destinations))
	for _, destination := range destinations {
		view, err := destinationView(destination)
		if err != nil {
			return nil, err
		}
		views = append(views, view)
	}
	return views, nil
}

func (s *Service) CreateDestination(input CreateDestinationInput) (*DestinationView, error) {
	destinationType := strings.ToLower(strings.TrimSpace(input.Type))
	if !isValidDestinationType(destinationType) {
		return nil, ErrInvalidBackupDestinationType
	}

	canonicalConfig, err := s.validateAndCanonicalizeConfig(destinationType, input.Config)
	if err != nil {
		return nil, err
	}
	encryptedConfig, err := encryptDestinationConfig(canonicalConfig)
	if err != nil {
		return nil, err
	}

	destination := model.BackupDestination{
		Revision:  1,
		Type:      destinationType,
		Enabled:   input.Enabled,
		Config:    encryptedConfig,
		SortOrder: input.SortOrder,
	}
	var view DestinationView
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&destination).Error; err != nil {
			return err
		}
		createdView, err := destinationView(destination)
		if err != nil {
			return err
		}
		view = createdView
		return nil
	}); err != nil {
		return nil, err
	}
	return &view, nil
}

func (s *Service) UpdateDestination(id uint, input UpdateDestinationInput) (*DestinationView, error) {
	if input.Revision == 0 {
		return nil, ErrBackupDestinationChanged
	}

	var destination model.BackupDestination
	var view DestinationView
	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&destination, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBackupDestinationNotFound
			}
			return err
		}
		if destination.Revision != input.Revision {
			return ErrBackupDestinationChanged
		}

		updates := make(map[string]any)
		if input.Enabled != nil {
			if !*input.Enabled && destination.Enabled {
				scheduleEnabled, err := backupsettings.GetBool(context.Background(), tx, KeyScheduleEnabled, false)
				if err != nil {
					return err
				}
				if scheduleEnabled {
					var enabledCount int64
					if err := tx.Model(&model.BackupDestination{}).Where("enabled = ?", true).Count(&enabledCount).Error; err != nil {
						return err
					}
					if enabledCount <= 1 {
						return ErrNoEnabledBackupDestination
					}
				}
			}
			updates["enabled"] = *input.Enabled
		}
		if input.SortOrder != nil {
			updates["sort_order"] = *input.SortOrder
		}
		if input.Config != nil {
			mergedConfig, err := mergeDestinationConfigWithExistingSecrets(destination.Type, destination.Config, *input.Config, input.ClearedSecretFields)
			if err != nil {
				return err
			}
			canonicalConfig, err := s.validateAndCanonicalizeConfig(destination.Type, mergedConfig)
			if err != nil {
				return err
			}
			encryptedConfig, err := encryptDestinationConfig(canonicalConfig)
			if err != nil {
				return err
			}
			updates["config"] = encryptedConfig
		}

		if len(updates) > 0 {
			updates["revision"] = gorm.Expr("revision + 1")
			result := tx.Model(&model.BackupDestination{}).
				Where("id = ? AND revision = ?", id, input.Revision).
				Updates(updates)
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected == 0 {
				return destinationRevisionConflict(tx, id)
			}
		}
		if err := tx.First(&destination, id).Error; err != nil {
			return err
		}
		updatedView, err := destinationView(destination)
		if err != nil {
			return err
		}
		view = updatedView
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &view, nil
}

func destinationRevisionConflict(db *gorm.DB, id uint) error {
	var destination model.BackupDestination
	if err := db.Select("id").First(&destination, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBackupDestinationNotFound
		}
		return err
	}
	return ErrBackupDestinationChanged
}

func (s *Service) DeleteDestination(id uint, revision uint64) error {
	if revision == 0 {
		return ErrBackupDestinationChanged
	}
	return s.DB.Transaction(func(tx *gorm.DB) error {
		var destination model.BackupDestination
		if err := tx.First(&destination, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrBackupDestinationNotFound
			}
			return err
		}
		if destination.Revision != revision {
			return ErrBackupDestinationChanged
		}

		if destination.Enabled {
			scheduleEnabled, err := backupsettings.GetBool(context.Background(), tx, KeyScheduleEnabled, false)
			if err != nil {
				return err
			}
			if scheduleEnabled {
				var enabledCount int64
				if err := tx.Model(&model.BackupDestination{}).Where("enabled = ?", true).Count(&enabledCount).Error; err != nil {
					return err
				}
				if enabledCount <= 1 {
					return ErrNoEnabledBackupDestination
				}
			}
		}

		result := tx.Where("id = ? AND revision = ?", id, revision).Delete(&model.BackupDestination{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return destinationRevisionConflict(tx, id)
		}
		return nil
	})
}

// validateAndCanonicalizeConfig validates a destination config against its type
// and returns the canonical JSON encoding. It builds the target from the config
// to reuse each target's own validation (e.g. local directory rules) as the
// single source of truth, so the config surface and the runtime agree.
func (s *Service) validateAndCanonicalizeConfig(destinationType, config string) (string, error) {
	parsed, err := parseDestinationConfigMap(config)
	if err != nil {
		return "", err
	}

	if destinationType == "local" {
		if rawDir, ok := parsed["dir"].(string); ok {
			normalized, err := normalizeBackupLocalDir(rawDir)
			if err != nil {
				return "", err
			}
			parsed["dir"] = normalized
		}
	}

	// Constructing the target validates type-specific invariants without
	// persisting anything.
	if _, err := newBackupTargetFromConfig(destinationType, parsed, s.DB); err != nil {
		return "", err
	}

	encoded, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// TestDestination performs a read-only connectivity probe against a saved
// destination by listing its archives. Listing exercises the endpoint,
// credentials, and (for remote targets) the network path without writing
// anything, so an admin can validate a configuration safely. It returns the
// number of existing archives on success.
func (s *Service) TestDestination(ctx context.Context, id uint) (int, error) {
	var destination model.BackupDestination
	if err := s.DB.First(&destination, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, ErrBackupDestinationNotFound
		}
		return 0, err
	}

	target, err := newBackupTarget(destination, s.DB)
	if err != nil {
		return 0, err
	}
	objects, err := target.List(ctx)
	if err != nil {
		return 0, err
	}
	return len(objects), nil
}

// recordDestinationOutcome stores the latest delivery and retention stages on
// the destination summary. A successful write is itself recorded as
// bookkeeping success; callers treat a write error as a partial run rather than
// converting an already delivered archive into a retryable delivery failure.
func (s *Service) recordDestinationOutcome(id uint, deliveryStatus, deliveryError, retentionStatus, retentionError string) error {
	updates := map[string]any{
		"last_status":             deliveryStatus,
		"last_error":              deliveryError,
		"last_retention_status":   retentionStatus,
		"last_retention_error":    retentionError,
		"last_bookkeeping_status": StatusOK,
		"last_bookkeeping_error":  "",
	}
	if deliveryStatus == StatusOK {
		updates["last_run_at"] = pkg.Now()
	}
	result := s.DB.Model(&model.BackupDestination{}).
		Where("id = ?", id).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrBackupDestinationNotFound
	}
	return nil
}
