package api

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/shiroha/subdux/internal/service"
)

type IconProxyHandler struct {
	Service *service.IconProxyService
}

func NewIconProxyHandler(s *service.IconProxyService) *IconProxyHandler {
	return &IconProxyHandler{Service: s}
}

func (h *IconProxyHandler) Get(c echo.Context) error {
	resolution, err := h.Service.Resolve(c.Param("provider"), c.QueryParam("domain"))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidIconProxyProvider), errors.Is(err, service.ErrInvalidIconProxyTargetDomain):
			return c.JSON(http.StatusBadRequest, echo.Map{"error": err.Error()})
		case errors.Is(err, service.ErrIconProxyDomainNotAllowed):
			return c.JSON(http.StatusForbidden, echo.Map{"error": err.Error()})
		default:
			return writeInternalServerError(c, err)
		}
	}

	if !resolution.Proxy {
		return c.Redirect(http.StatusTemporaryRedirect, resolution.UpstreamURL)
	}

	resp, err := h.Service.Fetch(c.Request().Context(), resolution)
	if err != nil {
		if errors.Is(err, service.ErrIconProxyDomainNotAllowed) ||
			errors.Is(err, service.ErrInvalidIconProxyTargetDomain) {
			return c.JSON(http.StatusForbidden, echo.Map{"error": err.Error()})
		}
		return c.JSON(http.StatusBadGateway, echo.Map{"error": "failed to fetch icon"})
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
