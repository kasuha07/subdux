package service

import (
	"context"

	"github.com/go-webauthn/webauthn/protocol"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"gorm.io/gorm"
)

type AuthService = serviceauth.Service
type TOTPService = serviceauth.TOTPService

type RegisterInput = serviceauth.RegisterInput
type LoginInput = serviceauth.LoginInput
type AuthResponse = serviceauth.AuthResponse
type InitialAdminInput = serviceauth.InitialAdminInput
type InitialAdminResult = serviceauth.InitialAdminResult
type LoginResponse = serviceauth.LoginResponse
type ChangePasswordInput = serviceauth.ChangePasswordInput
type RegistrationConfig = serviceauth.RegistrationConfig
type PasskeyCredentialInfo = serviceauth.PasskeyCredentialInfo
type PasskeyBeginResult = serviceauth.PasskeyBeginResult
type TotpSetupResult = serviceauth.TotpSetupResult
type OIDCPublicConfig = serviceauth.OIDCPublicConfig
type OIDCStartResult = serviceauth.OIDCStartResult
type OIDCConnectionInfo = serviceauth.OIDCConnectionInfo
type OIDCCallbackResult = serviceauth.OIDCCallbackResult
type OIDCSessionResult = serviceauth.OIDCSessionResult
type OIDCReauthGrade = serviceauth.OIDCReauthGrade

const (
	OIDCGradeFresh             = serviceauth.OIDCGradeFresh
	OIDCGradeMFA               = serviceauth.OIDCGradeMFA
	OIDCGradePhishingResistant = serviceauth.OIDCGradePhishingResistant
)

var (
	ErrRegistrationDisabled                  = serviceauth.ErrRegistrationDisabled
	ErrRegistrationEmailVerificationDisabled = serviceauth.ErrRegistrationEmailVerificationDisabled
	ErrVerificationCodeRequired              = serviceauth.ErrVerificationCodeRequired
	ErrVerificationCodeInvalid               = serviceauth.ErrVerificationCodeInvalid
	ErrVerificationCodeTooManyAttempts       = serviceauth.ErrVerificationCodeTooManyAttempts
	ErrVerificationCodeTooFrequent           = serviceauth.ErrVerificationCodeTooFrequent
	ErrInvalidEmail                          = serviceauth.ErrInvalidEmail
	ErrSMTPUnavailable                       = serviceauth.ErrSMTPUnavailable
	ErrEmailDomainNotAllowed                 = serviceauth.ErrEmailDomainNotAllowed
	ErrEmailAlreadyRegistered                = serviceauth.ErrEmailAlreadyRegistered
	ErrUsernameAlreadyTaken                  = serviceauth.ErrUsernameAlreadyTaken
	ErrUserNotFound                          = serviceauth.ErrUserNotFound
	ErrCurrentPasswordIncorrect              = serviceauth.ErrCurrentPasswordIncorrect
	ErrNewEmailSameAsCurrent                 = serviceauth.ErrNewEmailSameAsCurrent
	ErrPasswordTooLong                       = serviceauth.ErrPasswordTooLong
	ErrInvalidRefreshToken                   = serviceauth.ErrInvalidRefreshToken
	ErrTOTPAlreadyEnabled                    = serviceauth.ErrTOTPAlreadyEnabled
	ErrTOTPSetupExpired                      = serviceauth.ErrTOTPSetupExpired
	ErrTOTPInvalidCode                       = serviceauth.ErrTOTPInvalidCode
	ErrTOTPInvalidPassword                   = serviceauth.ErrTOTPInvalidPassword
	ErrTOTPInvalidAuthCode                   = serviceauth.ErrTOTPInvalidAuthCode
	ErrNoPasskeyRegistered                   = serviceauth.ErrNoPasskeyRegistered
)

func NewAuthService(db *gorm.DB) *AuthService {
	return serviceauth.NewService(db)
}

func NewTOTPService(db *gorm.DB) *TOTPService {
	return serviceauth.NewTOTPService(db)
}

type ReauthService = servicereauth.Service
type ReauthPolicy = servicereauth.Policy
type ReauthMethods = servicereauth.Methods

const (
	ReauthOperationBackup          = servicereauth.ReauthOperationBackup
	ReauthOperationBackupSchedule  = servicereauth.ReauthOperationBackupSchedule
	ReauthOperationRestore         = servicereauth.ReauthOperationRestore
	ReauthOperationChangeEmail     = servicereauth.ReauthOperationChangeEmail
	ReauthOperationAddPasskey      = servicereauth.ReauthOperationAddPasskey
	ReauthOperationDeletePasskey   = servicereauth.ReauthOperationDeletePasskey
	ReauthOperationEnableTOTP      = servicereauth.ReauthOperationEnableTOTP
	ReauthOperationDisableTOTP     = servicereauth.ReauthOperationDisableTOTP
	ReauthOperationConnectOIDC     = servicereauth.ReauthOperationConnectOIDC
	ReauthOperationCreateAPIKey    = servicereauth.ReauthOperationCreateAPIKey
	ReauthOperationDeleteAPIKey    = servicereauth.ReauthOperationDeleteAPIKey
	ReauthOperationCreateAdminUser = servicereauth.ReauthOperationCreateAdminUser
	ReauthOperationChangeUserRole  = servicereauth.ReauthOperationChangeUserRole
	ReauthOperationDeleteUser      = servicereauth.ReauthOperationDeleteUser
	ReauthOperationExportRedacted  = servicereauth.ReauthOperationExportRedacted
	ReauthOperationExportSecrets   = servicereauth.ReauthOperationExportSecrets
	ReauthOperationImportSubdux    = servicereauth.ReauthOperationImportSubdux
	ReauthOperationImportWallos    = servicereauth.ReauthOperationImportWallos
)

var (
	ErrInvalidReauthOperation = servicereauth.ErrInvalidReauthOperation
	ErrReauthRequired         = servicereauth.ErrReauthRequired
	ErrPasswordReauthDisabled = servicereauth.ErrPasswordReauthDisabled
	ErrOIDCReauthInsufficient = servicereauth.ErrOIDCReauthInsufficient
)

func NewReauthService(db *gorm.DB, authSvc *AuthService) *ReauthService {
	return servicereauth.NewService(db, authReauthAdapter{inner: authSvc})
}

func IsValidReauthOperation(operation string) bool {
	return servicereauth.IsValidReauthOperation(operation)
}

func ReauthOperationForCreateUser(input CreateUserInput) (string, bool) {
	return servicereauth.OperationForCreateUserRole(input.Role)
}

func ReauthOperationForAdminSettingsUpdate(input UpdateSettingsInput) (string, bool) {
	return servicereauth.OperationForAdminSettingsUpdate(servicereauth.AdminSettingsUpdateInput{
		BackupScheduleEnabled:    input.BackupScheduleEnabled,
		BackupTimeOfDay:          input.BackupTimeOfDay,
		BackupIncludeAssets:      input.BackupIncludeAssets,
		BackupEncryptEnabled:     input.BackupEncryptEnabled,
		BackupEncryptionPassword: input.BackupEncryptionPassword,
		BackupLocalDir:           input.BackupLocalDir,
		BackupRetentionCount:     input.BackupRetentionCount,
	})
}

func ReauthOperationForExport(includeSecrets bool) string {
	return servicereauth.OperationForExport(includeSecrets)
}

type authReauthAdapter struct {
	inner *AuthService
}

func (a authReauthAdapter) WithContext(ctx context.Context) servicereauth.Authenticator {
	if a.inner == nil {
		return a
	}
	return authReauthAdapter{inner: a.inner.WithContext(ctx)}
}

func (a authReauthAdapter) FactorState(userID uint) (servicereauth.FactorState, error) {
	state, err := a.inner.FactorState(userID)
	if err != nil {
		return servicereauth.FactorState{}, err
	}
	return servicereauth.FactorState{
		HasPassword: state.HasPassword,
		HasTOTP:     state.HasTOTP,
		HasPasskey:  state.HasPasskey,
		HasOIDC:     state.HasOIDC,
	}, nil
}

func (a authReauthAdapter) VerifyPassword(userID uint, password string, code string, requireTOTP bool) error {
	return a.inner.VerifyPasswordForReauth(userID, password, code, requireTOTP)
}

func (a authReauthAdapter) BeginPasskeyReauth(userID uint, operation string, origin string, host string, scheme string) (*servicereauth.PasskeyBeginResult, error) {
	result, err := a.inner.BeginPasskeyReauth(userID, operation, origin, host, scheme)
	if err != nil {
		return nil, err
	}
	return &servicereauth.PasskeyBeginResult{SessionID: result.SessionID, Options: result.Options}, nil
}

func (a authReauthAdapter) FinishPasskeyReauth(userID uint, operation string, sessionID string, parsedResponse *protocol.ParsedCredentialAssertionData, origin string, host string, scheme string) error {
	return a.inner.FinishPasskeyReauth(userID, operation, sessionID, parsedResponse, origin, host, scheme)
}

func (a authReauthAdapter) BeginOIDCReauth(userID uint, operation string) (*servicereauth.OIDCStartResult, error) {
	result, err := a.inner.BeginOIDCReauth(userID, operation)
	if err != nil {
		return nil, err
	}
	return &servicereauth.OIDCStartResult{AuthorizationURL: result.AuthorizationURL}, nil
}

func (a authReauthAdapter) ConsumeOIDCReauthResult(sessionID string, userID uint, operation string) (servicereauth.OIDCReauthGrade, error) {
	grade, err := a.inner.ConsumeOIDCReauthResult(sessionID, userID, operation)
	return servicereauth.OIDCReauthGrade(grade), err
}

func (a authReauthAdapter) HasOIDCConnection(userID uint) (bool, error) {
	return a.inner.HasOIDCConnection(userID)
}
