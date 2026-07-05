package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/kasuha07/subdux/internal/api/httpx"
	serviceoutbound "github.com/kasuha07/subdux/internal/service/outbound"
	"github.com/kasuha07/subdux/internal/version"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// VersionHandler serves the running build version and the latest published
// release. The "latest" lookup performs an outbound GitHub request through the
// SSRF-aware outbound client, so it lives in a handler rather than inline in the
// router.
type VersionHandler struct {
	db *gorm.DB
}

func NewVersionHandler(db *gorm.DB) *VersionHandler {
	return &VersionHandler{db: db}
}

func (h *VersionHandler) Get(c echo.Context) error {
	return c.JSON(http.StatusOK, version.Get())
}

func (h *VersionHandler) GetLatest(c echo.Context) error {
	client := serviceoutbound.NewOutboundHTTPClient(h.db, 10*time.Second)
	req, err := http.NewRequestWithContext(c.Request().Context(), http.MethodGet,
		"https://api.github.com/repos/kasuha07/subdux/releases/latest", nil)
	if err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_create_request")
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return httpx.WriteError(c, http.StatusBadGateway, "failed_to_fetch_latest_release")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return httpx.WriteError(c, http.StatusBadGateway, "github_api_returned_non_200")
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return httpx.WriteError(c, http.StatusInternalServerError, "failed_to_parse_response")
	}

	return c.JSON(http.StatusOK, echo.Map{"tag_name": release.TagName})
}
