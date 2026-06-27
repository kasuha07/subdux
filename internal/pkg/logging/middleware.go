package logging

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

// HeaderRequestID is the canonical header used to receive and emit the request
// correlation identifier.
const HeaderRequestID = "X-Request-ID"

// RequestLogger returns Echo middleware that emits one structured log record
// per request and wires request-scoped logging into the request context.
//
// For every request it:
//   - reuses an inbound X-Request-ID when present, otherwise generates one;
//   - echoes the identifier back on the X-Request-ID response header;
//   - stores a request-scoped logger (tagged with request_id) on the request
//     context so handlers and services can log with automatic correlation;
//   - logs method, sanitized URI, status, client IP, byte count, and latency
//     once the handler returns, choosing the level from the status class.
//
// Sensitive query parameters are masked via SanitizeURI, preserving the
// redaction behavior of the previous logger.
func RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			start := time.Now()

			requestID := ResolveRequestID(req.Header.Get(HeaderRequestID))
			c.Response().Header().Set(HeaderRequestID, requestID)

			reqLogger := L().With(slog.String(requestIDField, requestID))
			ctx := WithRequestID(WithLogger(req.Context(), reqLogger), requestID)
			c.SetRequest(req.WithContext(ctx))

			// Let downstream error handling run so c.Response().Status is final
			// before we log, mirroring the previous middleware's behavior.
			err := next(c)
			if err != nil {
				c.Error(err)
			}

			status := c.Response().Status
			if status == 0 {
				status = http.StatusOK
			}

			reqLogger.LogAttrs(ctx, levelForStatus(status), "http request",
				slog.String("method", req.Method),
				slog.String("uri", SanitizeURI(req.URL.Path, req.URL.Query())),
				slog.Int("status", status),
				slog.String("ip", c.RealIP()),
				slog.Int64("bytes", c.Response().Size),
				slog.Duration("latency", time.Since(start).Round(time.Millisecond)),
			)

			// The error has already been handled via c.Error; returning nil
			// prevents Echo's default logger from emitting a duplicate line.
			return nil
		}
	}
}

// levelForStatus maps an HTTP status code to a log level: 5xx is an error, 4xx
// is a warning, and everything else is informational.
func levelForStatus(status int) slog.Level {
	switch {
	case status >= http.StatusInternalServerError:
		return slog.LevelError
	case status >= http.StatusBadRequest:
		return slog.LevelWarn
	default:
		return slog.LevelInfo
	}
}
