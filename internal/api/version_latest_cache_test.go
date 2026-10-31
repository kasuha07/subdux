package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func newCachedVersionTestHandler(fetch func(ctx context.Context) (string, error)) (*VersionHandler, *time.Time) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	h := NewVersionHandler(nil)
	h.now = func() time.Time { return now }
	h.fetch = fetch
	return h, &now
}

func getLatestVersion(t *testing.T, h *VersionHandler) *httptest.ResponseRecorder {
	t.Helper()
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/version/latest", nil)
	rec := httptest.NewRecorder()
	if err := h.GetLatest(e.NewContext(req, rec)); err != nil {
		t.Fatalf("GetLatest returned error: %v", err)
	}
	return rec
}

func TestLatestVersionCoalescesConcurrentRequests(t *testing.T) {
	var calls atomic.Int32
	release := make(chan struct{})
	h, _ := newCachedVersionTestHandler(func(ctx context.Context) (string, error) {
		calls.Add(1)
		<-release
		return "v1.2.3", nil
	})

	var wg sync.WaitGroup
	codes := make([]int, 20)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			codes[i] = getLatestVersion(t, h).Code
		}(i)
	}
	time.Sleep(50 * time.Millisecond)
	close(release)
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls = %d, want 1", got)
	}
	for i, code := range codes {
		if code != http.StatusOK {
			t.Fatalf("request %d status = %d, want 200", i, code)
		}
	}
}

func TestLatestVersionCachesAndServesStaleOnFailure(t *testing.T) {
	var calls atomic.Int32
	fail := false
	h, now := newCachedVersionTestHandler(func(ctx context.Context) (string, error) {
		calls.Add(1)
		if fail {
			return "", &latestReleaseError{status: http.StatusBadGateway, code: "github_api_returned_non_200"}
		}
		return "v1.0.0", nil
	})

	for i := 0; i < 5; i++ {
		if rec := getLatestVersion(t, h); rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls within TTL = %d, want 1", got)
	}

	fail = true
	*now = now.Add(latestReleaseTTL + time.Second)
	rec := getLatestVersion(t, h)
	if rec.Code != http.StatusOK || !containsJSONValue(rec.Body.String(), "v1.0.0") {
		t.Fatalf("stale fallback status = %d body = %s, want cached v1.0.0", rec.Code, rec.Body.String())
	}
	// The failure is remembered, so further requests do not hit upstream.
	getLatestVersion(t, h)
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls after failure = %d, want 2", got)
	}
}

func TestLatestVersionThrottlesUpstreamAfterFailure(t *testing.T) {
	var calls atomic.Int32
	h, now := newCachedVersionTestHandler(func(ctx context.Context) (string, error) {
		calls.Add(1)
		return "", &latestReleaseError{status: http.StatusBadGateway, code: "github_api_returned_non_200"}
	})

	for i := 0; i < 5; i++ {
		rec := getLatestVersion(t, h)
		if rec.Code != http.StatusBadGateway || !containsJSONValue(rec.Body.String(), "github_api_returned_non_200") {
			t.Fatalf("status = %d body = %s, want 502 github_api_returned_non_200", rec.Code, rec.Body.String())
		}
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("upstream calls during failure window = %d, want 1", got)
	}
	*now = now.Add(latestReleaseFailureTTL + time.Second)
	getLatestVersion(t, h)
	if got := calls.Load(); got != 2 {
		t.Fatalf("upstream calls after failure window = %d, want 2", got)
	}
}

func containsJSONValue(body, value string) bool {
	return strings.Contains(body, `"`+value+`"`)
}
