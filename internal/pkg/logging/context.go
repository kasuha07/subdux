package logging

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
)

// contextKey is an unexported type so logging's context values cannot collide
// with keys set by other packages.
type contextKey int

const (
	loggerKey contextKey = iota
	requestIDKey
)

// requestIDField is the attribute key used to tag log records with the request
// identifier, and also the JSON/text field name callers will see.
const requestIDField = "request_id"

// requestIDBytes is the number of random bytes in a generated request ID. Eight
// bytes (16 hex chars) is ample for correlating lines within a single process
// while staying compact in log output.
const requestIDBytes = 8

// maxRequestIDLen bounds the length of a client-supplied request ID that we are
// willing to echo and log. Inbound identifiers longer than this, or containing
// characters outside the safe set, are rejected in favor of a freshly generated
// ID so a client cannot inflate log volume or inject misleading correlation
// values.
const maxRequestIDLen = 64

// WithLogger returns a copy of ctx carrying logger. FromContext will return it
// in downstream code, allowing request-scoped fields (such as request_id) to
// propagate without threading a logger through every function signature.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	if logger == nil {
		return ctx
	}
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext returns the logger stored on ctx, or the process-wide logger when
// none is present (including for a nil context). It never returns nil.
func FromContext(ctx context.Context) *slog.Logger {
	if ctx != nil {
		if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
			return logger
		}
	}
	return L()
}

// WithRequestID stores a request identifier on ctx so it can be retrieved by
// handlers and echoed in responses.
func WithRequestID(ctx context.Context, id string) context.Context {
	if id == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestIDFromContext returns the request identifier stored on ctx, or "" when
// none is present.
func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if id, ok := ctx.Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}

// NewRequestID returns a new random hex-encoded request identifier. It falls
// back to a fixed sentinel only in the extraordinary case that the system
// random source fails, so a request is never left without an ID.
func NewRequestID() string {
	buf := make([]byte, requestIDBytes)
	if _, err := rand.Read(buf); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(buf)
}

// ResolveRequestID returns a trustworthy request identifier from a possibly
// client-supplied value. The inbound value is accepted only when it is
// non-empty, within maxRequestIDLen, and composed solely of safe characters
// ([A-Za-z0-9._-]); otherwise a fresh random ID is generated. This prevents a
// client from injecting oversized or misleading correlation identifiers while
// still honoring well-formed upstream IDs for distributed tracing.
func ResolveRequestID(inbound string) string {
	if isAcceptableRequestID(inbound) {
		return inbound
	}
	return NewRequestID()
}

func isAcceptableRequestID(id string) bool {
	if id == "" || len(id) > maxRequestIDLen {
		return false
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
		default:
			return false
		}
	}
	return true
}
