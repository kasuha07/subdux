package service

import (
	svcbackup "github.com/kasuha07/subdux/internal/service/backup"
	"gorm.io/gorm"
)

const (
	backupScheduleEnabledKey    = svcbackup.KeyScheduleEnabled
	backupTimeOfDayKey          = svcbackup.KeyTimeOfDay
	backupIncludeAssetsKey      = svcbackup.KeyIncludeAssets
	backupEncryptEnabledKey     = svcbackup.KeyEncryptEnabled
	backupEncryptionPasswordKey = svcbackup.KeyEncryptionPassword
	backupLocalDirKey           = svcbackup.KeyLocalDir
	backupRetentionCountKey     = svcbackup.KeyRetentionCount
	backupLastRunAtKey          = svcbackup.KeyLastRunAt
	backupLastStatusKey         = svcbackup.KeyLastStatus
	backupLastErrorKey          = svcbackup.KeyLastError

	backupStatusOK     = svcbackup.StatusOK
	backupStatusFailed = svcbackup.StatusFailed
)

var (
	ErrInvalidBackupTimeOfDay           = svcbackup.ErrInvalidBackupTimeOfDay
	ErrInvalidBackupRetentionCount      = svcbackup.ErrInvalidBackupRetentionCount
	ErrInvalidBackupLocalDir            = svcbackup.ErrInvalidBackupLocalDir
	ErrBackupEncryptionPasswordRequired = svcbackup.ErrBackupEncryptionPasswordRequired
	ErrInvalidBackup                    = svcbackup.ErrInvalidBackup
	ErrBackupPasswordRequired           = svcbackup.ErrBackupPasswordRequired
	ErrBackupInvalidPassword            = svcbackup.ErrBackupInvalidPassword
)

type LocalBackupInfo = svcbackup.LocalBackupInfo
type BackupService = svcbackup.Service

func NewBackupService(db *gorm.DB) *BackupService {
	return svcbackup.NewService(db)
}

func (s *AdminService) CreateLocalBackup() (string, error) {
	return svcbackup.NewService(s.DB).CreateLocalBackup()
}

func (s *AdminService) ListLocalBackups() (string, []LocalBackupInfo, error) {
	return svcbackup.NewService(s.DB).ListLocalBackups()
}

func (s *AdminService) RunScheduledBackup(ownerID string) error {
	return svcbackup.NewService(s.DB).RunScheduledBackup(ownerID)
}

func (s *AdminService) RestoreBackup(uploadedBackupPath string, password string) error {
	return svcbackup.NewService(s.DB).RestoreBackup(uploadedBackupPath, password)
}

func applyBackupSettings(tx *gorm.DB, input UpdateSettingsInput) error {
	return svcbackup.ApplySettings(tx, svcbackup.UpdateSettingsInput{
		ScheduleEnabled:    input.BackupScheduleEnabled,
		TimeOfDay:          input.BackupTimeOfDay,
		IncludeAssets:      input.BackupIncludeAssets,
		EncryptEnabled:     input.BackupEncryptEnabled,
		EncryptionPassword: input.BackupEncryptionPassword,
		LocalDir:           input.BackupLocalDir,
		RetentionCount:     input.BackupRetentionCount,
	})
}
