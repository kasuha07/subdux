package reauth

// Operation identifiers scope a ticket to a single sensitive action so a ticket
// minted for one operation cannot authorize another.
const (
	ReauthOperationBackup               = "backup"
	ReauthOperationBackupSchedule       = "backup_schedule"
	ReauthOperationRestore              = "restore"
	ReauthOperationChangeEmail          = "change_email"
	ReauthOperationAddPasskey           = "add_passkey"
	ReauthOperationDeletePasskey        = "delete_passkey"
	ReauthOperationEnableTOTP           = "enable_totp"
	ReauthOperationDisableTOTP          = "disable_totp"
	ReauthOperationConnectOIDC          = "connect_oidc"
	ReauthOperationCreateAPIKey         = "create_api_key"
	ReauthOperationDeleteAPIKey         = "delete_api_key"
	ReauthOperationCreateAdminUser      = "create_admin_user"
	ReauthOperationChangeUserRole       = "change_user_role"
	ReauthOperationAdminDisableTOTP     = "admin_disable_user_totp"
	ReauthOperationAdminDisablePasskeys = "admin_disable_user_passkeys"
	ReauthOperationDeleteUser           = "delete_user"
	ReauthOperationExportRedacted       = "export_redacted"
	ReauthOperationExportSecrets        = "export_secrets"
	ReauthOperationImportSubdux         = "import_subdux"
	ReauthOperationImportWallos         = "import_wallos"
)

// IsValidReauthOperation reports whether operation is a known reauth operation.
// It is the single source of truth for the set of valid operations.
func IsValidReauthOperation(operation string) bool {
	switch operation {
	case ReauthOperationBackup,
		ReauthOperationBackupSchedule,
		ReauthOperationRestore,
		ReauthOperationChangeEmail,
		ReauthOperationAddPasskey,
		ReauthOperationDeletePasskey,
		ReauthOperationEnableTOTP,
		ReauthOperationDisableTOTP,
		ReauthOperationConnectOIDC,
		ReauthOperationCreateAPIKey,
		ReauthOperationDeleteAPIKey,
		ReauthOperationCreateAdminUser,
		ReauthOperationChangeUserRole,
		ReauthOperationAdminDisableTOTP,
		ReauthOperationAdminDisablePasskeys,
		ReauthOperationDeleteUser,
		ReauthOperationExportRedacted,
		ReauthOperationExportSecrets,
		ReauthOperationImportSubdux,
		ReauthOperationImportWallos:
		return true
	default:
		return false
	}
}

type AdminSettingsUpdateInput struct {
	BackupScheduleEnabled    *bool
	BackupTimeOfDay          *string
	BackupIncludeAssets      *bool
	BackupEncryptEnabled     *bool
	BackupEncryptionPassword *string
	BackupLocalDir           *string
	BackupRetentionCount     *int64
}

// OperationForCreateUserRole returns the step-up operation needed by the
// current admin-user creation policy. Creating a regular user remains unchanged
// and does not require a reauth ticket.
func OperationForCreateUserRole(role string) (string, bool) {
	if role == "admin" {
		return ReauthOperationCreateAdminUser, true
	}
	return "", false
}

// OperationForAdminSettingsUpdate returns the step-up operation needed by
// the current admin settings policy. Only backup-schedule settings require a
// reauth ticket today.
func OperationForAdminSettingsUpdate(input AdminSettingsUpdateInput) (string, bool) {
	if input.BackupScheduleEnabled != nil ||
		input.BackupTimeOfDay != nil ||
		input.BackupIncludeAssets != nil ||
		input.BackupEncryptEnabled != nil ||
		input.BackupEncryptionPassword != nil ||
		input.BackupLocalDir != nil ||
		input.BackupRetentionCount != nil {
		return ReauthOperationBackupSchedule, true
	}
	return "", false
}

func OperationForExport(includeSecrets bool) string {
	if includeSecrets {
		return ReauthOperationExportSecrets
	}
	return ReauthOperationExportRedacted
}
