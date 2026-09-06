package apimw

import (
	"net/http/httptest"
	"testing"
)

func TestClientIPExtractorTrustBoundary(t *testing.T) {
	for _, tc := range []struct{ cidrs, peer, want string }{
		{"", "192.0.2.1:1234", "192.0.2.1"},
		{"10.0.0.0/24", "192.0.2.1:1234", "192.0.2.1"},
		{"10.0.0.0/24", "10.0.0.2:1234", "203.0.113.1"},
		{"10.0.0.0/24", "127.0.0.1:1234", "127.0.0.1"},
	} {
		extract, err := ClientIPExtractor(tc.cidrs)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", "/", nil)
		req.RemoteAddr = tc.peer
		req.Header.Set("X-Forwarded-For", "198.51.100.99, 203.0.113.1")
		req.Header.Set("X-Real-IP", "198.51.100.99")
		if got := extract(req); got != tc.want {
			t.Fatalf("%+v: got %s", tc, got)
		}
	}
	if _, err := ClientIPExtractor("bad-range"); err == nil {
		t.Fatal("invalid proxy CIDR accepted")
	}
}
