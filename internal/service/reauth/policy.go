package reauth

// Methods reports which factors a user can use to re-authenticate, after
// applying the account's step-up policy. Password is offered only when the
// knowledge factor is an accepted method for the account — it is withheld from
// passkey accounts that have no TOTP, which must step up with the passkey
// itself. PasswordRequiresTOTP is set when the password path is offered but must
// be accompanied by a current TOTP code. Passkey is offered when the user has
// one registered. OIDC is offered when the provider is enabled and the user has
// a linked OIDC identity; the grade the provider must prove is enforced at
// verification time (VerifyOIDC), not advertised here.
type Methods struct {
	Password             bool `json:"password"`
	PasswordRequiresTOTP bool `json:"password_requires_totp"`
	Passkey              bool `json:"passkey"`
	OIDC                 bool `json:"oidc"`
}

// Policy captures the accepted step-up methods for an account and
// operation given its enrolled factors. It is the single source of truth for the
// default reauth matrix:
//
//	enrolled factors            password        passkey   OIDC min grade
//	password                    yes             -         OIDC-1
//	password+TOTP               yes (+TOTP)     -         OIDC-2
//	password+passkey            no              yes       OIDC-3
//	password+TOTP+passkey       yes (+TOTP)     yes       OIDC-3
//
// The knowledge (password) factor is disabled exactly when a passkey is enrolled
// without TOTP: such an account must use its passkey (a stronger, phishing-
// resistant factor) rather than a bare password. When TOTP is also enrolled the
// password path returns as password+TOTP, matching the passkey's strength.
//
// Disabling TOTP has one operation-specific exception: when both TOTP and a
// passkey are enrolled, OIDC-2 remains an accepted fallback even though OIDC-3 is
// preferred. This lets a user remove the TOTP factor after proving either a
// passkey/OIDC-3 path or the still-current MFA path (password+TOTP or OIDC-2).
//
// Deleting a passkey deliberately uses the normal passkey-account policy and is
// scoped only to the user and delete_passkey operation, not to the credential
// being deleted. The user may prove presence with any registered passkey; the
// target passkey ID is authorized by the deletion endpoint's ownership check.
type Policy struct {
	PasswordAllowed      bool
	PasswordRequiresTOTP bool
	PasskeyAllowed       bool
	OIDCAllowed          bool
	RequiredOIDCGrade    OIDCReauthGrade
}

type reauthFactorAvailability struct {
	hasTOTP    bool
	hasPasskey bool
	hasOIDC    bool
}

func reauthPolicyFor(operation string, factors reauthFactorAvailability) Policy {
	requiredGrade := OIDCGradeFresh
	if factors.hasPasskey {
		requiredGrade = OIDCGradePhishingResistant
	} else if factors.hasTOTP {
		requiredGrade = OIDCGradeMFA
	}
	if operation == ReauthOperationDisableTOTP && factors.hasTOTP {
		requiredGrade = OIDCGradeMFA
	}
	passwordAllowed := !(factors.hasPasskey && !factors.hasTOTP)
	return Policy{
		PasswordAllowed:      passwordAllowed,
		PasswordRequiresTOTP: factors.hasTOTP,
		PasskeyAllowed:       factors.hasPasskey,
		OIDCAllowed:          factors.hasOIDC,
		RequiredOIDCGrade:    requiredGrade,
	}
}

// PolicyFor resolves what the given user's current factor enrollment requires
// for operation. It decides what should be accepted; the factor verifier methods
// still own how password, TOTP, passkey, and OIDC proofs are checked.
func (s *Service) PolicyFor(userID uint, operation string) (Policy, error) {
	if !IsValidReauthOperation(operation) {
		return Policy{}, ErrInvalidReauthOperation
	}

	factors, err := s.factorAvailability(userID)
	if err != nil {
		return Policy{}, err
	}
	return reauthPolicyFor(operation, factors), nil
}

func (s *Service) factorAvailability(userID uint) (reauthFactorAvailability, error) {
	state, err := s.auth.FactorState(userID)
	if err != nil {
		return reauthFactorAvailability{}, err
	}
	return reauthFactorAvailability{
		hasTOTP:    state.HasTOTP,
		hasPasskey: state.HasPasskey,
		hasOIDC:    state.HasOIDC,
	}, nil
}

// AvailableMethods reports the factors the user can present for reauth, after
// applying the account's step-up policy.
func (s *Service) AvailableMethods(userID uint, operation string) (Methods, error) {
	policy, err := s.PolicyFor(userID, operation)
	if err != nil {
		return Methods{}, err
	}
	return Methods{
		Password:             policy.PasswordAllowed,
		PasswordRequiresTOTP: policy.PasswordAllowed && policy.PasswordRequiresTOTP,
		Passkey:              policy.PasskeyAllowed,
		OIDC:                 policy.OIDCAllowed,
	}, nil
}

// ConsumeOIDCConnect consumes a connect_oidc ticket only when the current policy
// requires it. Existing behavior is preserved: first-time OIDC linking requires
// reauth, while already-linked users can reconnect without an extra ticket.
func (s *Service) ConsumeOIDCConnect(userID uint, ticket string) error {
	hasConnection, err := s.auth.HasOIDCConnection(userID)
	if err != nil {
		return err
	}
	if hasConnection {
		return nil
	}
	return s.Consume(userID, ReauthOperationConnectOIDC, ticket)
}
