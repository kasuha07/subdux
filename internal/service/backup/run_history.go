package backup

import (
	"time"

	"github.com/kasuha07/subdux/internal/model"
)

const (
	defaultBackupRunHistoryLimit = 20
	maxBackupRunHistoryLimit     = 100
)

// BackupRunDestinationRecord is the compact per-destination outcome shown in
// the run history: enough to tell where the archive went and whether delivery
// and pruning succeeded, without the resume-oriented stage detail.
type BackupRunDestinationRecord struct {
	DestinationID   uint   `json:"destination_id"`
	Type            string `json:"type"`
	DeliveryStatus  string `json:"delivery_status"`
	RetentionStatus string `json:"retention_status"`
}

// BackupRunRecord is one entry of the admin backup history. Every backup —
// destination fan-outs (manual or scheduled) and browser downloads — is one
// run row, so this list is the complete record of backups taken.
type BackupRunRecord struct {
	ID           uint                         `json:"id"`
	Source       string                       `json:"source"` // manual | scheduled | download
	ArchiveName  string                       `json:"archive_name"`
	Status       string                       `json:"status"`
	Error        string                       `json:"error,omitempty"`
	SizeBytes    int64                        `json:"size_bytes"`
	Encrypted    bool                         `json:"encrypted"`
	StartedAt    time.Time                    `json:"started_at"`
	FinishedAt   *time.Time                   `json:"finished_at"`
	Destinations []BackupRunDestinationRecord `json:"destinations"`
}

// ListBackupRuns returns the newest limit backup runs across every source with
// a compact per-destination outcome for each. A non-positive limit selects the
// default page size; oversized limits are clamped.
func (s *Service) ListBackupRuns(limit int) ([]BackupRunRecord, error) {
	if limit <= 0 {
		limit = defaultBackupRunHistoryLimit
	}
	if limit > maxBackupRunHistoryLimit {
		limit = maxBackupRunHistoryLimit
	}

	var runs []model.BackupRun
	if err := s.DB.Order("id DESC").Limit(limit).Find(&runs).Error; err != nil {
		return nil, err
	}
	records := make([]BackupRunRecord, 0, len(runs))
	if len(runs) == 0 {
		return records, nil
	}

	runIDs := make([]uint, 0, len(runs))
	for _, run := range runs {
		runIDs = append(runIDs, run.ID)
	}
	var destinations []model.BackupRunDestination
	if err := s.DB.Where("run_id IN ?", runIDs).Order("id ASC").Find(&destinations).Error; err != nil {
		return nil, err
	}
	destinationsByRun := make(map[uint][]BackupRunDestinationRecord, len(runs))
	for _, destination := range destinations {
		destinationsByRun[destination.RunID] = append(destinationsByRun[destination.RunID], BackupRunDestinationRecord{
			DestinationID:   destination.DestinationID,
			Type:            destination.Type,
			DeliveryStatus:  destination.DeliveryStatus,
			RetentionStatus: destination.RetentionStatus,
		})
	}

	for _, run := range runs {
		runDestinations := destinationsByRun[run.ID]
		if runDestinations == nil {
			// Download runs have no destination rows; keep the JSON an empty
			// array rather than null.
			runDestinations = []BackupRunDestinationRecord{}
		}
		records = append(records, BackupRunRecord{
			ID:           run.ID,
			Source:       run.Source,
			ArchiveName:  run.ArchiveName,
			Status:       run.Status,
			Error:        run.Error,
			SizeBytes:    run.SizeBytes,
			Encrypted:    run.Encrypted,
			StartedAt:    run.StartedAt,
			FinishedAt:   run.FinishedAt,
			Destinations: runDestinations,
		})
	}
	return records, nil
}
