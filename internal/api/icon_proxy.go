package api

import (
	"net/http"

	iconproxy "github.com/kasuha07/subdux/internal/service/iconproxy"
	"github.com/labstack/echo/v4"
)

type IconProxyHandler struct {
	Service *iconproxy.Service
}

func NewIconProxyHandler(s *iconproxy.Service) *IconProxyHandler {
	return &IconProxyHandler{Service: s}
}

func (h *IconProxyHandler) Get(c echo.Context) error {
	svc := h.Service.WithContext(c.Request().Context())
	resolution, err := svc.Resolve(c.Param("provider"), c.QueryParam("domain"))
	if err != nil {
		return writeServiceError(c, err,
			serviceError(http.StatusBadRequest, iconproxy.ErrInvalidIconProxyProvider, iconproxy.ErrInvalidIconProxyTargetDomain),
			serviceError(http.StatusForbidden, iconproxy.ErrIconProxyDomainNotAllowed),
		)
	}

	if !resolution.Proxy {
		return c.Redirect(http.StatusTemporaryRedirect, resolution.UpstreamURL)
	}

	resp, err := svc.Fetch(c.Request().Context(), resolution)
	if err != nil {
		forbiddenError := serviceError(http.StatusForbidden, iconproxy.ErrIconProxyDomainNotAllowed, iconproxy.ErrInvalidIconProxyTargetDomain)
		if forbiddenError.matches(err) {
			return writeServiceError(c, err, forbiddenError)
		}
		return writeError(c, http.StatusBadGateway, "failed to fetch icon")
	}
	defer resp.Body.Close()

	if cacheControl := resp.Header.Get("Cache-Control"); cacheControl != "" {
		c.Response().Header().Set("Cache-Control", cacheControl)
	} else {
		c.Response().Header().Set("Cache-Control", "public, max-age=3600")
	}
	if etag := resp.Header.Get("ETag"); etag != "" {
		c.Response().Header().Set("ETag", etag)
	}
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		c.Response().Header().Set("Last-Modified", lastModified)
	}
	if expires := resp.Header.Get("Expires"); expires != "" {
		c.Response().Header().Set("Expires", expires)
	}
	c.Response().Header().Set("X-Content-Type-Options", "nosniff")

	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return c.Stream(resp.StatusCode, contentType, resp.Body)
}
