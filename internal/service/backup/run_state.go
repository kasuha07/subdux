package backup

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kasuha07/subdux/internal/model"
	"github.com/kasuha07/subdux/internal/pkg"
	"gorm.io/gorm"
)

const (
	backupRunSourceManual    = "manual"
	backupRunSourceScheduled = "scheduled"
)

var (
	ErrBackupRunNotFound            = errors.New("backup run not found")
	ErrBackupRunDestinationNotFound = errors.New("backup run destination not found")
)

type backupRunState struct {
	run          model.BackupRun
	destinations []model.BackupRunDestination
}

func (s *Service) beginBackupRun(source, archiveName string, destinations []model.BackupDestination) (backupRunState, error) {
	now := pkg.Now()
	state := backupRunState{
		run: model.BackupRun{
			Source:                  source,
			ArchiveName:             archiveName,
			Status:                  StatusPending,
			DeliveryStatus:          StatusPending,
			RetentionStatus:         StatusNotAttempted,
			BookkeepingStatus:       StatusPending,
			GlobalBookkeepingStatus: StatusPending,
			StartedAt:               now,
		},
		destinations: make([]model.BackupRunDestination, 0, len(destinations)),
	}
	for _, destination := range destinations {
		state.destinations = append(state.destinations, model.BackupRunDestination{
			DestinationID:       destination.ID,
			DestinationRevision: destination.Revision,
			Type:                destination.Type,
			DeliveryStatus:      StatusPending,
			RetentionStatus:     StatusNotAttempted,
			BookkeepingStatus:   StatusPending,
		})
	}

	err := s.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&state.run).Error; err != nil {
			return err
		}
		for i := range state.destinations {
			state.destinations[i].RunID = state.run.ID
			if err := tx.Create(&state.destinations[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return backupRunState{}, err
	}
	return state, nil
}

// findResumableScheduledRuns returns every scheduled run that still has work
// left. Per-destination schedules mean a single pass can leave several runs in
// flight at once (one per archive variant), so resumption is a set operation
// rather than "the latest run".
//
// The SQL filter mirrors the terminal conditions checked below so completed runs
// are excluded by the database rather than scanned and discarded in Go as the
// run history grows.
func (s *Service) findResumableScheduledRuns() ([]backupRunState, error) {
	var runs []model.BackupRun
	err := s.DB.
		Where("source = ?", backupRunSourceScheduled).
		Where("status <> ?", StatusSuperseded).
		Where(
			"status <> ? OR bookkeeping_status <> ? OR global_bookkeeping_status <> ? OR (archive_path IS NOT NULL AND archive_path <> '')",
			StatusOK, StatusOK, StatusOK,
		).
		Order("id ASC").
		Find(&runs).Error
	if err != nil {
		return nil, err
	}

	var states []backupRunState
	for i := range runs {
		state, resumeErr := s.resumableScheduledRun(runs[i])
		if resumeErr != nil {
			return nil, resumeErr
		}
		if state != nil {
			states = append(states, *state)
		}
	}
	return states, nil
}

// resumableScheduledRun decides whether one run may still be resumed, retiring
// it when its archive has gone stale, its spool has vanished, or its
// destinations have moved out from under it.
func (s *Service) resumableScheduledRun(run model.BackupRun) (*backupRunState, error) {
	var destinations []model.BackupRunDestination
	if err := s.DB.Where("run_id = ?", run.ID).Order("id ASC").Find(&destinations).Error; err != nil {
		return nil, err
	}
	if pkg.Now().Sub(run.StartedAt) >= backupRunResumeWindow {
		// The run has held the schedule for longer than its archive stays worth
		// delivering. Supersede it (which also releases the spool) so the next due
		// tick takes a fresh snapshot instead of retrying stale bytes forever.
		reason := fmt.Sprintf(
			"scheduled backup run %d superseded because it exceeded the %s resume window",
			run.ID,
			backupRunResumeWindow,
		)
		if supersedeErr := s.supersedeScheduledRun(&run, reason); supersedeErr != nil {
			return nil, supersedeErr
		}
		return nil, nil
	}
	if archivePath := strings.TrimSpace(run.ArchivePath); archivePath != "" {
		if _, statErr := os.Stat(archivePath); errors.Is(statErr, os.ErrNotExist) {
			reason := fmt.Sprintf("scheduled backup run %d superseded because its archive spool is missing", run.ID)
			if supersedeErr := s.supersedeScheduledRun(&run, reason); supersedeErr != nil {
				return nil, supersedeErr
			}
			return nil, nil
		} else if statErr != nil {
			return nil, statErr
		}
	}
	for _, runDestination := range destinations {
		var destination model.BackupDestination
		err := s.DB.Select("id, revision").First(&destination, runDestination.DestinationID).Error
		if errors.Is(err, gorm.ErrRecordNotFound) ||
			(err == nil && destination.Revision != runDestination.DestinationRevision) {
			reason := fmt.Sprintf(
				"scheduled backup run %d superseded because destination %d changed or was deleted",
				run.ID,
				runDestination.DestinationID,
			)
			if supersedeErr := s.supersedeScheduledRun(&run, reason); supersedeErr != nil {
				return nil, supersedeErr
			}
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
	}
	return &backupRunState{run: run, destinations: destinations}, nil
}

// scheduledRunResumeDue reports whether an incomplete run may be retried yet.
//
// A run with no FinishedAt never reached finalization, which means the process
// died mid-flight rather than completing an attempt with a bad outcome; that is
// resumed immediately because no work was wasted and none will be repeated. A
// finalized-but-incomplete run did burn a full attempt, so the next one waits
// backupRunRetryInterval from the moment that attempt ended.
func scheduledRunResumeDue(run model.BackupRun, now time.Time) bool {
	if run.FinishedAt == nil {
		return true
	}
	return !now.Before(run.FinishedAt.Add(backupRunRetryInterval))
}

func (s *Service) loadBackupDestinationForRun(runDestination model.BackupRunDestination) (model.BackupDestination, error) {
	var destination model.BackupDestination
	if err := s.DB.First(&destination, runDestination.DestinationID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return model.BackupDestination{}, ErrBackupDestinationNotFound
		}
		return model.BackupDestination{}, err
	}
	if destination.Revision != runDestination.DestinationRevision {
		return model.BackupDestination{}, ErrBackupDestinationChanged
	}
	return destination, nil
}

func (s *Service) persistRunDestination(id uint, updates map[string]any) error {
	result := s.DB.Model(&model.BackupRunDestination{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrBackupRunDestinationNotFound
	}
	return nil
}

func destinationRunResult(runDestination model.BackupRunDestination) DestinationRunResult {
	retentionStatus := runDestination.RetentionStatus
	if retentionStatus == "" {
		retentionStatus = StatusNotAttempted
	}
	bookkeepingStatus := runDestination.BookkeepingStatus
	if bookkeepingStatus == "" {
		bookkeepingStatus = StatusPending
	}
	return DestinationRunResult{
		DestinationID:     runDestination.DestinationID,
		Type:              runDestination.Type,
		DeliveryStatus:    runDestination.DeliveryStatus,
		Success:           runDestination.DeliveryStatus == StatusOK,
		Error:             runDestination.DeliveryError,
		RetentionStatus:   retentionStatus,
		RetentionError:    runDestination.RetentionError,
		BookkeepingStatus: bookkeepingStatus,
		BookkeepingError:  runDestination.BookkeepingError,
	}
}

func aggregateBackupRunResult(run model.BackupRun, destinations []model.BackupRunDestination) BackupRunResult {
	result := BackupRunResult{
		RunID:                   run.ID,
		ArchiveName:             run.ArchiveName,
		Results:                 make([]DestinationRunResult, 0, len(destinations)),
		BookkeepingStatus:       StatusOK,
		GlobalBookkeepingStatus: run.GlobalBookkeepingStatus,
	}
	if result.GlobalBookkeepingStatus == "" {
		result.GlobalBookkeepingStatus = StatusPending
	}

	deliverySuccesses := 0
	deliveryFailures := 0
	retentionFailures := 0
	retentionAttempts := 0
	bookkeepingFailures := 0
	bookkeepingPending := 0
	var messages []string
	for _, destination := range destinations {
		item := destinationRunResult(destination)
		result.Results = append(result.Results, item)
		if item.Success {
			deliverySuccesses++
			if item.RetentionStatus != StatusNotAttempted {
				retentionAttempts++
			}
			if item.RetentionStatus != StatusOK {
				retentionFailures++
			}
		} else {
			deliveryFailures++
		}
		if item.BookkeepingStatus == StatusFailed {
			bookkeepingFailures++
		} else if item.BookkeepingStatus != StatusOK {
			bookkeepingPending++
		}
		if item.Error != "" {
			messages = append(messages, fmt.Sprintf("destination %d delivery: %s", item.DestinationID, item.Error))
		}
		if item.RetentionError != "" {
			messages = append(messages, fmt.Sprintf("destination %d retention: %s", item.DestinationID, item.RetentionError))
		}
		if item.BookkeepingError != "" {
			messages = append(messages, fmt.Sprintf("destination %d bookkeeping: %s", item.DestinationID, item.BookkeepingError))
		}
	}

	switch {
	case deliverySuccesses == 0:
		result.DeliveryStatus = StatusFailed
	case deliveryFailures == 0:
		result.DeliveryStatus = StatusOK
	default:
		result.DeliveryStatus = StatusPartial
	}
	if deliverySuccesses == 0 {
		result.RetentionStatus = StatusNotAttempted
	} else if retentionFailures > 0 || retentionAttempts != deliverySuccesses {
		result.RetentionStatus = StatusPartial
	} else {
		result.RetentionStatus = StatusOK
	}
	if bookkeepingFailures > 0 {
		result.BookkeepingStatus = StatusFailed
	} else if bookkeepingPending > 0 {
		result.BookkeepingStatus = StatusPending
	}

	if deliverySuccesses == 0 {
		result.Status = StatusFailed
	} else if result.DeliveryStatus != StatusOK || result.RetentionStatus != StatusOK || result.BookkeepingStatus != StatusOK {
		result.Status = StatusPartial
	} else {
		result.Status = StatusOK
	}
	result.Error = strings.Join(messages, "; ")
	return result
}

func (s *Service) finalizeBackupRun(runID uint, result BackupRunResult, scheduled bool) error {
	now := pkg.Now()
	bookkeepingStatus := result.BookkeepingStatus
	globalBookkeepingStatus := StatusOK
	if scheduled {
		globalBookkeepingStatus = StatusPending
	}
	updates := map[string]any{
		"status":                    result.Status,
		"delivery_status":           result.DeliveryStatus,
		"retention_status":          result.RetentionStatus,
		"bookkeeping_status":        bookkeepingStatus,
		"global_bookkeeping_status": globalBookkeepingStatus,
		"error":                     result.Error,
		"finished_at":               now,
	}
	dbResult := s.DB.Model(&model.BackupRun{}).Where("id = ?", runID).Updates(updates)
	if dbResult.Error != nil {
		return dbResult.Error
	}
	if dbResult.RowsAffected != 1 {
		return ErrBackupRunNotFound
	}
	return nil
}

// supersedeScheduledRun closes a run whose destination snapshot no longer
// matches the configured destination rows. Retrying it would either target a
// different resource under the old run ID or remain permanently blocked by a
// deleted destination, so the next due tick must start a new run.
func (s *Service) supersedeScheduledRun(run *model.BackupRun, reason string) error {
	archivePath := strings.TrimSpace(run.ArchivePath)
	cleanupError := ""
	if archivePath != "" {
		if !isPrivateStagedArchivePath(archivePath) {
			cleanupError = ErrBackupArchiveUnavailable.Error()
		} else if err := os.Remove(archivePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			cleanupError = err.Error()
		}
	}
	globalBookkeepingStatus := StatusOK
	globalBookkeepingError := ""
	if cleanupError != "" {
		globalBookkeepingStatus = StatusFailed
		globalBookkeepingError = cleanupError
		reason = fmt.Sprintf("%s; archive cleanup: %s", reason, cleanupError)
	}
	status := StatusSuperseded
	if cleanupError != "" {
		// Keep the run resumable while the spool is still present. A terminal
		// superseded state would make a failed unlink lose its retry path even
		// though archive_path is intentionally retained below.
		status = StatusPartial
	}
	updates := map[string]any{
		"status":                    status,
		"global_bookkeeping_status": globalBookkeepingStatus,
		"global_bookkeeping_error":  globalBookkeepingError,
		"error":                     reason,
		"finished_at":               pkg.Now(),
	}
	if cleanupError == "" {
		updates["archive_path"] = ""
	}
	dbResult := s.DB.Model(&model.BackupRun{}).Where("id = ?", run.ID).Updates(updates)
	if dbResult.Error != nil {
		return dbResult.Error
	}
	if dbResult.RowsAffected != 1 {
		return ErrBackupRunNotFound
	}
	if cleanupError != "" {
		return errors.New(cleanupError)
	}
	return nil
}

func (s *Service) markGlobalBookkeepingSuccess(runID uint) error {
	result := s.DB.Model(&model.BackupRun{}).Where("id = ?", runID).Updates(map[string]any{
		"global_bookkeeping_status": StatusOK,
		"global_bookkeeping_error":  "",
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrBackupRunNotFound
	}
	return nil
}

func (s *Service) markGlobalBookkeepingFailure(runID uint, err error) error {
	message := ""
	if err != nil {
		message = err.Error()
	}
	result := s.DB.Model(&model.BackupRun{}).Where("id = ?", runID).Updates(map[string]any{
		"global_bookkeeping_status": StatusFailed,
		"global_bookkeeping_error":  message,
		"status":                    gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", StatusOK, StatusPartial),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return ErrBackupRunNotFound
	}
	return nil
}

func appendBackupBookkeepingError(existing []error, destinationID uint, err error) []error {
	if err == nil {
		return existing
	}
	return append(existing, fmt.Errorf("backup destination %d bookkeeping: %w", destinationID, err))
}
