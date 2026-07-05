// Package serviceerr defines a small typed-error vocabulary shared across the
// service layer. Each Error carries a Kind that the transport layer maps to an
// HTTP status in exactly one place, replacing brittle message-substring
// classification. The transport emits stable error_code/error_params, and
// callers must provide those public codes explicitly.
package serviceerr

import (
	"errors"
)

// Kind classifies a service error for transport-layer status mapping. It is an
// abstract category, deliberately decoupled from HTTP so the service layer does
// not import net/http.
type Kind int

const (
	// KindInternal is the zero value: an unclassified/internal failure. It maps
	// to 500 and its message is not surfaced to clients.
	KindInternal Kind = iota
	// KindInvalid is a client-correctable bad request (validation, malformed
	// input). Maps to 400.
	KindInvalid
	// KindNotFound is a missing resource. Maps to 404.
	KindNotFound
	// KindConflict is a state/uniqueness conflict. Maps to 409.
	KindConflict
	// KindForbidden is an authorization/policy denial. Maps to 403.
	KindForbidden
	// KindUnauthorized is an authentication failure (bad/expired credential).
	// Maps to 401. Kept distinct from KindForbidden so callers that must avoid
	// tripping a client's token-refresh/logout flow can choose 403/400 instead.
	KindUnauthorized
	// KindTooMany is a rate-limit rejection. Maps to 429.
	KindTooMany
	// KindUnavailable is a transient dependency failure. Maps to 503.
	KindUnavailable
)

// Error is a typed service error. Msg is the canonical source message backing
// Error(); Code is the explicit public contract identifier; Err is an optional
// wrapped cause for errors.Is/As chains.
type Error struct {
	Kind Kind
	Msg  string
	Code string
	// Params carries interpolation values for a localized client message when
	// the contract text is parameterized.
	Params map[string]any
	Err    error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return e.Msg
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New builds a typed error with the given kind, explicit public code, and
// canonical source message.
func New(kind Kind, code, msg string) *Error {
	return &Error{Kind: kind, Msg: msg, Code: code}
}

// NewCode builds a typed error with an explicit stable code and interpolation
// params for parameterized contracts.
func NewCode(kind Kind, code, msg string, params map[string]any) *Error {
	return &Error{Kind: kind, Msg: msg, Code: code, Params: cloneParams(params)}
}

// Wrap builds a typed error that carries msg as the canonical source text while
// preserving cause for errors.Is/As inspection.
func Wrap(kind Kind, code, msg string, cause error) *Error {
	return &Error{Kind: kind, Msg: msg, Code: code, Err: cause}
}

// WrapCode builds a typed error with an explicit stable code, interpolation
// params, and a wrapped cause.
func WrapCode(kind Kind, code, msg string, params map[string]any, cause error) *Error {
	return &Error{Kind: kind, Msg: msg, Code: code, Params: cloneParams(params), Err: cause}
}

// KindOf reports the Kind of err if it (or anything it wraps) is a *Error, and
// whether such an Error was found. Callers that need a default should treat
// ok==false as KindInternal.
func KindOf(err error) (Kind, bool) {
	var typed *Error
	if errors.As(err, &typed) && typed != nil {
		return typed.Kind, true
	}
	return KindInternal, false
}

func cloneParams(params map[string]any) map[string]any {
	if len(params) == 0 {
		return nil
	}

	cloned := make(map[string]any, len(params))
	for key, value := range params {
		cloned[key] = value
	}
	return cloned
}
