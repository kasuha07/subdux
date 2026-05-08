package main

import (
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	subdux "github.com/shiroha/subdux"
	"github.com/shiroha/subdux/internal/api"
	"github.com/shiroha/subdux/internal/model"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
	"gorm.io/gorm"
)

func main() {
	db := pkg.InitDB()
	if err := pkg.InitJWTSecret(db); err != nil {
		log.Fatalf("Failed to initialize JWT secret: %v", err)
	}

	e := echo.New()
	e.HideBanner = true

	e.Use(requestLoggerMiddleware())
	e.Use(api.SecurityHeadersMiddleware)
	e.Use(middleware.Recover())

	allowedOrigins := loadCORSOrigins(db)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	erService, notificationService := api.SetupRoutes(e, db)

	stop := make(chan struct{})
	erService.StartBackgroundRefresh(stop)
	startNotificationChecker(notificationService, stop)

	e.Static("/uploads", filepath.Join(pkg.GetDataPath(), "assets"))

	distFS, err := fs.Sub(subdux.StaticFS, "web/dist")
	if err != nil {
		log.Fatal("Failed to access embedded frontend:", err)
	}
	setupSPA(e, distFS)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Subdux starting on :%s", port)
	e.Logger.Fatal(e.Start(":" + port))
}

var sensitiveQueryParams = map[string]struct{}{
	"access_token":  {},
	"api_key":       {},
	"apikey":        {},
	"code":          {},
	"id_token":      {},
	"key":           {},
	"otp":           {},
	"password":      {},
	"refresh_token": {},
	"secret":        {},
	"token":         {},
	"totp_token":    {},
}

func requestLoggerMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			start := pkg.Now()

			err := next(c)
			if err != nil {
				c.Error(err)
			}

			status := c.Response().Status
			if status == 0 {
				status = http.StatusOK
			}

			log.Printf("%s %s status=%d ip=%s latency=%s",
				req.Method,
				sanitizedRequestURI(req),
				status,
				c.RealIP(),
				time.Since(start).Round(time.Millisecond),
			)

			return nil
		}
	}
}

func sanitizedRequestURI(req *http.Request) string {
	if req == nil || req.URL == nil {
		return ""
	}

	path := req.URL.Path
	if path == "" {
		path = "/"
	}

	rawQuery := sanitizeRawQuery(req.URL.Query())
	if rawQuery == "" {
		return path
	}

	return path + "?" + rawQuery
}

func sanitizeRawQuery(query url.Values) string {
	if len(query) == 0 {
		return ""
	}

	sanitized := make(url.Values, len(query))
	for key, values := range query {
		if isSensitiveQueryParam(key) {
			sanitized[key] = []string{"[REDACTED]"}
			continue
		}

		sanitizedValues := make([]string, len(values))
		copy(sanitizedValues, values)
		sanitized[key] = sanitizedValues
	}

	return sanitized.Encode()
}

func isSensitiveQueryParam(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}

	if _, ok := sensitiveQueryParams[normalized]; ok {
		return true
	}

	return strings.HasSuffix(normalized, "_token") ||
		strings.HasSuffix(normalized, "_secret") ||
		strings.HasSuffix(normalized, "_password") ||
		strings.HasSuffix(normalized, "_key")
}

func loadCORSOrigins(db *gorm.DB) []string {
	if envOrigins := parseCORSOrigins(os.Getenv("CORS_ALLOW_ORIGINS")); len(envOrigins) > 0 {
		return envOrigins
	}

	var setting model.SystemSetting
	if err := db.Where("key = ?", "site_url").First(&setting).Error; err == nil {
		if siteOrigin := normalizeOrigin(setting.Value); siteOrigin != "" {
			return []string{siteOrigin}
		}
	}

	log.Printf("warning: no CORS_ALLOW_ORIGINS or site_url configured; falling back to localhost development origins")

	return []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
	}
}

func parseCORSOrigins(raw string) []string {
	parts := strings.Split(raw, ",")
	seen := make(map[string]struct{}, len(parts))
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := normalizeOrigin(part)
		if origin == "" {
			continue
		}
		if _, exists := seen[origin]; exists {
			continue
		}
		seen[origin] = struct{}{}
		origins = append(origins, origin)
	}
	sort.Strings(origins)
	return origins
}

func normalizeOrigin(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}

	return parsed.Scheme + "://" + parsed.Host
}

func startNotificationChecker(ns *service.NotificationService, stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(3 * time.Hour)
		defer ticker.Stop()

		if err := ns.ProcessPendingNotifications(); err != nil {
			log.Printf("notification check error: %v", err)
		}

		for {
			select {
			case <-ticker.C:
				if err := ns.ProcessPendingNotifications(); err != nil {
					log.Printf("notification check error: %v", err)
				}
			case <-stop:
				return
			}
		}
	}()
}

func setupSPA(e *echo.Echo, fsys fs.FS) {
	indexHTML, _ := fs.ReadFile(fsys, "index.html")
	fileServer := http.FileServer(http.FS(fsys))

	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		if path == "" || path == "index.html" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write(indexHTML)
			return
		}

		// Try serving the actual file
		if f, err := fsys.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(indexHTML)
	})))
}
