package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/kasuha07/subdux/internal/api/httpx"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/version"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

const (
	latestReleaseURL = "https://api.github.com/repos/kasuha07/subdux/releases/latest"
	// latestReleaseTTL bounds how often the public endpoint may reach GitHub.
	latestReleaseTTL = 15 * time.Minute
	// latestReleaseFailureTTL throttles upstream retries after a failed
	// refresh, so a GitHub outage or rate limit is not amplified by callers.
	latestReleaseFailureTTL = time.Minute
	latestReleaseTimeout    = 10 * time.Second
	latestReleaseMaxBytes   = 1 << 20
)

// latestReleaseError is a cached upstream failure, mapped to the same HTTP
// error the uncached handler used to return.
type latestReleaseError struct {
	status int
	code   string
}

func (e *latestReleaseError) Error() string { return e.code }

// VersionHandler serves the running build version and the latest published
// release. The "latest" lookup performs an outbound GitHub request through the
// SSRF-aware outbound client, so it lives in a handler rather than inline in the
// router.
//
// The endpoint is public, so the upstream lookup is cached: at most one
// refresh runs at a time, a successful result is reused for latestReleaseTTL,
// a failure suppresses retries for latestReleaseFailureTTL, and a stale tag is
// served when a refresh fails.
type VersionHandler struct {
	db    *gorm.DB
	now   func() time.Time
	fetch func(ctx context.Context) (string, error)

	// refreshMu serializes upstream refreshes (single-flight); mu guards the
	// cached fields and is never held across network I/O.
	refreshMu sync.Mutex
	mu        sync.Mutex
	tag       string
	hasTag    bool
	fetchedAt time.Time
	lastErr   *latestReleaseError
	failedAt  time.Time
}

func NewVersionHandler(db *gorm.DB) *VersionHandler {
	h := &VersionHandler{db: db, now: time.Now}
	h.fetch = h.fetchLatestRelease
	return h
}

func (h *VersionHandler) Get(c echo.Context) error {
	return c.JSON(http.StatusOK, version.Get())
}

func (h *VersionHandler) GetLatest(c echo.Context) error {
	tag, err := h.latestTag()
	if err != nil {
		var cached *latestReleaseError
		if errors.As(err, &cached) {
			return httpx.WriteError(c, cached.status, cached.code)
		}
		return httpx.WriteError(c, http.StatusBadGateway, "failed_to_fetch_latest_release")
	}
	return c.JSON(http.StatusOK, echo.Map{"tag_name": tag})
}

// cached returns the answer the cache can give without an upstream request.
func (h *VersionHandler) cached(now time.Time) (tag string, ok bool, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.hasTag && now.Sub(h.fetchedAt) < latestReleaseTTL {
		return h.tag, true, nil
	}
	if h.lastErr != nil && now.Sub(h.failedAt) < latestReleaseFailureTTL {
		if h.hasTag {
			return h.tag, true, nil
		}
		return "", true, h.lastErr
	}
	return "", false, nil
}

func (h *VersionHandler) latestTag() (string, error) {
	if tag, ok, err := h.cached(h.now()); ok {
		return tag, err
	}

	h.refreshMu.Lock()
	defer h.refreshMu.Unlock()
	// Another request may have refreshed while this one waited.
	if tag, ok, err := h.cached(h.now()); ok {
		return tag, err
	}

	// Detach from the caller's context: the refresh result is shared, so one
	// client disconnecting must not fail the requests queued behind it.
	ctx, cancel := context.WithTimeout(context.Background(), latestReleaseTimeout)
	defer cancel()
	tag, err := h.fetch(ctx)

	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	if err == nil {
		h.tag, h.hasTag, h.fetchedAt, h.lastErr = tag, true, now, nil
		return tag, nil
	}
	cachedErr, ok := err.(*latestReleaseError)
	if !ok {
		cachedErr = &latestReleaseError{status: http.StatusBadGateway, code: "failed_to_fetch_latest_release"}
	}
	h.lastErr, h.failedAt = cachedErr, now
	if h.hasTag {
		return h.tag, nil
	}
	return "", cachedErr
}

func (h *VersionHandler) fetchLatestRelease(ctx context.Context) (string, error) {
	client := serviceoutbound.NewOutboundHTTPClient(h.db, latestReleaseTimeout)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return "", &latestReleaseError{status: http.StatusInternalServerError, code: "failed_to_create_request"}
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", &latestReleaseError{status: http.StatusBadGateway, code: "failed_to_fetch_latest_release"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &latestReleaseError{status: http.StatusBadGateway, code: "github_api_returned_non_200"}
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, latestReleaseMaxBytes)).Decode(&release); err != nil {
		return "", &latestReleaseError{status: http.StatusInternalServerError, code: "failed_to_parse_response"}
	}
	return release.TagName, nil
}
