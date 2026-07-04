package api

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/kasuha07/subdux/internal/api/httpx"
	"github.com/kasuha07/subdux/internal/pkg/logging"
	"github.com/kasuha07/subdux/internal/service/serviceerr"
	"github.com/labstack/echo/v4"
)

// statusForServiceError maps a typed service error's Kind to an HTTP status.
// This is the single place the abstract Kind vocabulary crosses into HTTP, so
// the service layer never imports net/http and status assignment lives in one
// auditable table.
func statusForServiceError(kind serviceerr.Kind) int {
	switch kind {
	case serviceerr.KindInvalid:
		return http.StatusBadRequest
	case serviceerr.KindNotFound:
		return http.StatusNotFound
	case serviceerr.KindConflict:
		return http.StatusConflict
	case serviceerr.KindForbidden:
		return http.StatusForbidden
	case serviceerr.KindUnauthorized:
		return http.StatusUnauthorized
	case serviceerr.KindTooMany:
		return http.StatusTooManyRequests
	case serviceerr.KindUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// APIErrorHandler is the single Echo error handler for the API. Handlers signal
// failures by returning an error; this renders the frozen {"error": message}
// envelope with the appropriate status:
//
//   - *serviceerr.Error   → Kind-derived status, client-facing message
//   - transient SQLite busy → 503 with Retry-After, generic message
//   - echo.HTTPError       → delegated to Echo's default (jwt 401, 404, 405…)
//   - anything else        → 500, message hidden, cause logged
//
// It never double-writes: if the response was already committed by a handler
// that wrote directly, it delegates to the default handler which no-ops.
func APIErrorHandler(defaultHandler echo.HTTPErrorHandler) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		if err == nil || c.Response().Committed {
			return
		}

		var typed *serviceerr.Error
		if errors.As(err, &typed) && typed != nil {
			_ = httpx.WriteError(c, statusForServiceError(typed.Kind), typed.Msg)
			return
		}

		if httpx.IsTransientSQLiteBusyError(err) {
			logging.FromContext(c.Request().Context()).Warn("transient database busy error", slog.Any("error", err))
			c.Response().Header().Set("Retry-After", "1")
			_ = httpx.WriteError(c, http.StatusServiceUnavailable, "database is busy, retry later")
			return
		}

		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			defaultHandler(err, c)
			return
		}

		logging.FromContext(c.Request().Context()).Error("internal server error", slog.Any("error", err))
		_ = httpx.WriteError(c, http.StatusInternalServerError, "internal server error")
	}
}
