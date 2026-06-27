package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
)

func TestLevelFromEnv(t *testing.T) {
	tests := []struct {
		raw  string
		want slog.Level
	}{
		{raw: "debug", want: slog.LevelDebug},
		{raw: "DEBUG", want: slog.LevelDebug},
		{raw: "info", want: slog.LevelInfo},
		{raw: "warn", want: slog.LevelWarn},
		{raw: "warning", want: slog.LevelWarn},
		{raw: "error", want: slog.LevelError},
		{raw: "  Error  ", want: slog.LevelError},
		{raw: "", want: slog.LevelInfo},
		{raw: "nonsense", want: slog.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := LevelFromEnv(tt.raw); got != tt.want {
				t.Fatalf("LevelFromEnv(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestFormatFromEnv(t *testing.T) {
	tests := []struct {
		raw  string
		want Format
	}{
		{raw: "json", want: FormatJSON},
		{raw: "JSON", want: FormatJSON},
		{raw: "text", want: FormatText},
		{raw: "", want: FormatAuto},
		{raw: "auto", want: FormatAuto},
		{raw: "weird", want: FormatAuto},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			if got := FormatFromEnv(tt.raw); got != tt.want {
				t.Fatalf("FormatFromEnv(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestConfigureJSONOutputAndKeys(t *testing.T) {
	var buf bytes.Buffer
	logger := Configure(Options{Level: slog.LevelInfo, Format: FormatJSON, Output: &buf})

	logger.Info("hello world", slog.String("scope", "test"))

	var record map[string]any
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v\nraw: %s", err, buf.String())
	}

	if record["msg"] != "hello world" {
		t.Fatalf("msg = %v, want %q", record["msg"], "hello world")
	}
	if record["level"] != "INFO" {
		t.Fatalf("level = %v, want INFO", record["level"])
	}
	if record["scope"] != "test" {
		t.Fatalf("scope = %v, want test", record["scope"])
	}
	if _, ok := record["time"]; !ok {
		t.Fatal("expected a time field in the record")
	}
}

func TestConfigureLevelThreshold(t *testing.T) {
	var buf bytes.Buffer
	logger := Configure(Options{Level: slog.LevelWarn, Format: FormatJSON, Output: &buf})

	logger.Info("suppressed")
	logger.Debug("also suppressed")
	if buf.Len() != 0 {
		t.Fatalf("expected no output below warn level, got: %s", buf.String())
	}

	logger.Warn("surfaced")
	if !strings.Contains(buf.String(), "surfaced") {
		t.Fatalf("expected warn record, got: %s", buf.String())
	}
}

func TestPackageLevelHelpersRouteThroughConfiguredLogger(t *testing.T) {
	var buf bytes.Buffer
	Configure(Options{Level: slog.LevelDebug, Format: FormatJSON, Output: &buf})

	Info("via package helper", slog.Int("n", 1))
	if !strings.Contains(buf.String(), "via package helper") {
		t.Fatalf("package Info did not reach configured logger, got: %s", buf.String())
	}
}

func TestFatalLogsThenExits(t *testing.T) {
	var buf bytes.Buffer
	Configure(Options{Level: slog.LevelInfo, Format: FormatJSON, Output: &buf})

	originalExit := osExit
	t.Cleanup(func() { osExit = originalExit })

	var gotCode int
	var exited bool
	osExit = func(code int) {
		gotCode = code
		exited = true
	}

	Fatal("boom", slog.String("phase", "startup"))

	if !exited {
		t.Fatal("Fatal did not call osExit")
	}
	if gotCode != 1 {
		t.Fatalf("exit code = %d, want 1", gotCode)
	}
	if !strings.Contains(buf.String(), "boom") {
		t.Fatalf("Fatal did not emit a record, got: %s", buf.String())
	}
	if !strings.Contains(buf.String(), "ERROR") {
		t.Fatalf("Fatal record not at error level, got: %s", buf.String())
	}
}

func TestFatalContextUsesContextLogger(t *testing.T) {
	var buf bytes.Buffer
	base := Configure(Options{Level: slog.LevelInfo, Format: FormatJSON, Output: &buf})

	originalExit := osExit
	t.Cleanup(func() { osExit = originalExit })
	osExit = func(int) {}

	ctx := WithLogger(context.Background(), base.With(slog.String("request_id", "abc123")))
	FatalContext(ctx, "context boom")

	if !strings.Contains(buf.String(), "abc123") {
		t.Fatalf("FatalContext did not use context logger, got: %s", buf.String())
	}
}

func TestResolveFormat(t *testing.T) {
	var notTerminal bytes.Buffer

	if got := resolveFormat(FormatJSON, &notTerminal); got != FormatJSON {
		t.Fatalf("resolveFormat(json) = %v, want json", got)
	}
	if got := resolveFormat(FormatText, &notTerminal); got != FormatText {
		t.Fatalf("resolveFormat(text) = %v, want text", got)
	}
	// A non-terminal writer under auto resolves to JSON.
	if got := resolveFormat(FormatAuto, &notTerminal); got != FormatJSON {
		t.Fatalf("resolveFormat(auto, buffer) = %v, want json", got)
	}
}
