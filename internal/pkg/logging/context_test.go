package logging

import (
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestFromContextFallsBackToProcessLogger(t *testing.T) {
	// A nil context and an empty context both resolve to the process-wide
	// logger rather than returning nil.
	if FromContext(nil) == nil {
		t.Fatal("FromContext(nil) returned nil")
	}
	if FromContext(context.Background()) == nil {
		t.Fatal("FromContext(empty) returned nil")
	}
}

func TestWithLoggerRoundTrip(t *testing.T) {
	stored := slog.Default().With(slog.String("scope", "unit"))
	ctx := WithLogger(context.Background(), stored)

	if got := FromContext(ctx); got != stored {
		t.Fatal("FromContext did not return the stored logger")
	}
}

func TestWithLoggerIgnoresNil(t *testing.T) {
	ctx := WithLogger(context.Background(), nil)
	// Storing nil must not panic and must not poison the context: FromContext
	// should still fall back to the process logger.
	if FromContext(ctx) == nil {
		t.Fatal("FromContext returned nil after WithLogger(nil)")
	}
}

func TestRequestIDRoundTrip(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-123")
	if got := RequestIDFromContext(ctx); got != "req-123" {
		t.Fatalf("RequestIDFromContext = %q, want %q", got, "req-123")
	}
}

func TestRequestIDFromContextEmpty(t *testing.T) {
	if got := RequestIDFromContext(context.Background()); got != "" {
		t.Fatalf("RequestIDFromContext(empty) = %q, want empty", got)
	}
	if got := RequestIDFromContext(nil); got != "" {
		t.Fatalf("RequestIDFromContext(nil) = %q, want empty", got)
	}
}

func TestWithRequestIDIgnoresEmpty(t *testing.T) {
	ctx := WithRequestID(context.Background(), "")
	if got := RequestIDFromContext(ctx); got != "" {
		t.Fatalf("expected empty id to be ignored, got %q", got)
	}
}

func TestNewRequestIDIsUniqueAndHex(t *testing.T) {
	const hexLen = requestIDBytes * 2

	seen := make(map[string]struct{}, 100)
	for range 100 {
		id := NewRequestID()
		if id == "unknown" {
			t.Fatal("NewRequestID returned the failure sentinel")
		}
		if len(id) != hexLen {
			t.Fatalf("NewRequestID length = %d, want %d (%q)", len(id), hexLen, id)
		}
		for _, r := range id {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Fatalf("NewRequestID returned non-hex character %q in %q", r, id)
			}
		}
		if _, dup := seen[id]; dup {
			t.Fatalf("NewRequestID returned a duplicate: %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestResolveRequestID(t *testing.T) {
	longID := strings.Repeat("a", maxRequestIDLen+1)

	tests := []struct {
		name     string
		inbound  string
		wantKept bool // true: inbound is returned verbatim; false: a fresh ID is generated
	}{
		{name: "well-formed inbound kept", inbound: "trace-abc_123.4", wantKept: true},
		{name: "max length kept", inbound: strings.Repeat("a", maxRequestIDLen), wantKept: true},
		{name: "empty generates", inbound: "", wantKept: false},
		{name: "too long generates", inbound: longID, wantKept: false},
		{name: "space rejected", inbound: "has space", wantKept: false},
		{name: "crlf rejected", inbound: "a\r\nb", wantKept: false},
		{name: "slash rejected", inbound: "a/b", wantKept: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ResolveRequestID(tt.inbound)
			if got == "" {
				t.Fatal("ResolveRequestID returned empty")
			}
			if tt.wantKept && got != tt.inbound {
				t.Fatalf("ResolveRequestID(%q) = %q, want it kept verbatim", tt.inbound, got)
			}
			if !tt.wantKept && got == tt.inbound {
				t.Fatalf("ResolveRequestID(%q) returned the rejected value verbatim", tt.inbound)
			}
			if !tt.wantKept && !isAcceptableRequestID(got) {
				t.Fatalf("ResolveRequestID(%q) generated an unacceptable id %q", tt.inbound, got)
			}
		})
	}
}
