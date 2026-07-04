package authreauth

import (
	"context"

	"github.com/go-webauthn/webauthn/protocol"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
)

// Adapt exposes auth.Service through the narrow reauth.Authenticator interface.
func Adapt(authSvc *serviceauth.Service) servicereauth.Authenticator {
	return adapter{inner: authSvc}
}

type adapter struct {
	inner *serviceauth.Service
}

func (a adapter) WithContext(ctx context.Context) servicereauth.Authenticator {
	if a.inner == nil {
		return a
	}
	return adapter{inner: a.inner.WithContext(ctx)}
}

func (a adapter) FactorState(userID uint) (servicereauth.FactorState, error) {
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

func (a adapter) VerifyPassword(userID uint, password string, code string, requireTOTP bool) error {
	return a.inner.VerifyPasswordForReauth(userID, password, code, requireTOTP)
}

func (a adapter) BeginPasskeyReauth(userID uint, operation string, origin string, host string, scheme string) (*servicereauth.PasskeyBeginResult, error) {
	result, err := a.inner.BeginPasskeyReauth(userID, operation, origin, host, scheme)
	if err != nil {
		return nil, err
	}
	return &servicereauth.PasskeyBeginResult{SessionID: result.SessionID, Options: result.Options}, nil
}

func (a adapter) FinishPasskeyReauth(userID uint, operation string, sessionID string, parsedResponse *protocol.ParsedCredentialAssertionData, origin string, host string, scheme string) error {
	return a.inner.FinishPasskeyReauth(userID, operation, sessionID, parsedResponse, origin, host, scheme)
}

func (a adapter) BeginOIDCReauth(userID uint, operation string) (*servicereauth.OIDCStartResult, error) {
	result, err := a.inner.BeginOIDCReauth(userID, operation)
	if err != nil {
		return nil, err
	}
	return &servicereauth.OIDCStartResult{AuthorizationURL: result.AuthorizationURL}, nil
}

func (a adapter) ConsumeOIDCReauthResult(sessionID string, userID uint, operation string) (servicereauth.OIDCReauthGrade, error) {
	grade, err := a.inner.ConsumeOIDCReauthResult(sessionID, userID, operation)
	return servicereauth.OIDCReauthGrade(grade), err
}

func (a adapter) HasOIDCConnection(userID uint) (bool, error) {
	return a.inner.HasOIDCConnection(userID)
}
