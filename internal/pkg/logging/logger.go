// Package logging provides Subdux's structured logging system.
//
// It is a thin, dependency-free layer over the standard library's log/slog.
// A single process-wide logger is configured once at startup from environment
// variables and then shared everywhere through the package-level helpers
// (Info, Warn, Error, ...) or, for request-scoped work, through a logger
// carried on a context.Context.
//
// Design goals:
//   - Zero third-party dependencies (slog only), matching Subdux's
//     single-binary, minimal-footprint philosophy.
//   - One consistent output format and one place to configure it.
//   - Secret-safe by default: attributes whose key is sensitive (and the
//     corresponding query parameters) are redacted before they reach a
//     handler. Redaction is key-based; it does not scan the contents of
//     error or struct values, so avoid placing secrets inside an error
//     message or a logged struct field.
//   - Easy correlation of log lines belonging to the same HTTP request.
//
// This package intentionally does not import internal/pkg to avoid an import
// cycle (internal/pkg packages may want to log).
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/mattn/go-isatty"
)

// Environment variables that configure the logger.
const (
	// EnvLevel selects the minimum level that is emitted: debug, info, warn,
	// or error. Defaults to info.
	EnvLevel = "LOG_LEVEL"
	// EnvFormat selects the output encoding: json, text, or auto. Defaults to
	// auto, which emits human-readable text on a TTY and JSON otherwise.
	EnvFormat = "LOG_FORMAT"
)

// Format identifies how log records are encoded.
type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
	// FormatAuto picks text when the output is an interactive terminal and
	// JSON otherwise.
	FormatAuto Format = "auto"
)

// Options fully describes a logger configuration. The zero value is valid and
// resolves to the info level with auto format on standard error.
type Options struct {
	Level     slog.Level
	Format    Format
	Output    io.Writer
	AddSource bool
}

var (
	mu      sync.RWMutex
	current = slog.New(newHandler(Options{Output: os.Stderr}))
)

// Init configures the process-wide logger from the environment and installs it
// as the default for both this package and the standard slog package. It is
// safe to call once during startup, before any goroutines log. The resolved
// logger is returned for convenience.
func Init() *slog.Logger {
	return Configure(Options{
		Level:  LevelFromEnv(os.Getenv(EnvLevel)),
		Format: FormatFromEnv(os.Getenv(EnvFormat)),
		Output: os.Stderr,
	})
}

// Configure installs an explicit configuration. It is primarily useful for
// tests; production code should prefer Init.
func Configure(opts Options) *slog.Logger {
	logger := slog.New(newHandler(opts))

	mu.Lock()
	current = logger
	mu.Unlock()

	// Mirror onto the standard slog default so any incidental slog.Info calls
	// (including from dependencies) share the same handler and formatting.
	slog.SetDefault(logger)
	return logger
}

// L returns the process-wide logger. It never returns nil.
func L() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

func newHandler(opts Options) slog.Handler {
	out := opts.Output
	if out == nil {
		out = os.Stderr
	}

	handlerOpts := &slog.HandlerOptions{
		Level:       opts.Level,
		AddSource:   opts.AddSource,
		ReplaceAttr: redactAttr,
	}

	switch resolveFormat(opts.Format, out) {
	case FormatJSON:
		return slog.NewJSONHandler(out, handlerOpts)
	default:
		return slog.NewTextHandler(out, handlerOpts)
	}
}

// resolveFormat collapses FormatAuto (and any unknown value) into a concrete
// encoding based on whether out is an interactive terminal.
func resolveFormat(format Format, out io.Writer) Format {
	switch format {
	case FormatJSON:
		return FormatJSON
	case FormatText:
		return FormatText
	default:
		if isTerminal(out) {
			return FormatText
		}
		return FormatJSON
	}
}

func isTerminal(out io.Writer) bool {
	f, ok := out.(interface{ Fd() uintptr })
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// LevelFromEnv parses a level string. Unknown or empty values resolve to
// slog.LevelInfo so misconfiguration never silences logging.
func LevelFromEnv(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// FormatFromEnv parses a format string. Unknown or empty values resolve to
// FormatAuto.
func FormatFromEnv(raw string) Format {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "json":
		return FormatJSON
	case "text":
		return FormatText
	default:
		return FormatAuto
	}
}

// Package-level convenience wrappers around the process-wide logger. They keep
// call sites terse while still routing through the configured handler.

func Debug(msg string, args ...any) { L().Debug(msg, args...) }
func Info(msg string, args ...any)  { L().Info(msg, args...) }
func Warn(msg string, args ...any)  { L().Warn(msg, args...) }
func Error(msg string, args ...any) { L().Error(msg, args...) }

// With returns a child logger that includes the given attributes on every
// record.
func With(args ...any) *slog.Logger { return L().With(args...) }

// Fatal logs an error-level record and then terminates the process with a
// non-zero status. It replaces ad-hoc log.Fatal calls so fatal startup
// failures are emitted through the structured handler before exit.
//
// Like log.Fatal, deferred functions do not run; reserve it for unrecoverable
// startup errors in main and process initialization.
func Fatal(msg string, args ...any) {
	L().Error(msg, args...)
	osExit(1)
}

// FatalContext behaves like Fatal but uses the logger carried on ctx (falling
// back to the process-wide logger).
func FatalContext(ctx context.Context, msg string, args ...any) {
	FromContext(ctx).Error(msg, args...)
	osExit(1)
}

// osExit is a seam so tests can assert Fatal behavior without killing the test
// process.
var osExit = os.Exit
