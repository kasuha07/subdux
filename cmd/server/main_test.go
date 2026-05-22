package main

import (
	"net/http"
	"testing"
)

func TestNewHTTPServerConfiguresTimeouts(t *testing.T) {
	server := newHTTPServer(":8080", http.NewServeMux())

	if got, want := server.Addr, ":8080"; got != want {
		t.Fatalf("Addr = %q, want %q", got, want)
	}
	if got, want := server.ReadHeaderTimeout, httpReadHeaderTimeout; got != want {
		t.Fatalf("ReadHeaderTimeout = %s, want %s", got, want)
	}
	if got, want := server.ReadTimeout, httpReadTimeout; got != want {
		t.Fatalf("ReadTimeout = %s, want %s", got, want)
	}
	if got, want := server.WriteTimeout, httpWriteTimeout; got != want {
		t.Fatalf("WriteTimeout = %s, want %s", got, want)
	}
	if got, want := server.IdleTimeout, httpIdleTimeout; got != want {
		t.Fatalf("IdleTimeout = %s, want %s", got, want)
	}
}
