package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
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

const (
	httpReadHeaderTimeout = 5 * time.Second
	httpReadTimeout       = 30 * time.Second
	httpWriteTimeout      = 60 * time.Second
	httpIdleTimeout       = 120 * time.Second
	shutdownTimeout       = 30 * time.Second
)

func main() {
	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	appCtx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()

	db := pkg.InitDB()
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to access database handle: %v", err)
	}
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

	taskMonitor := service.NewBackgroundTaskMonitor()
	erService, notificationService := api.SetupRoutes(appCtx, e, db, taskMonitor)

	var backgroundTasks sync.WaitGroup
	erService.StartBackgroundRefresh(appCtx, taskMonitor, &backgroundTasks)
	startNotificationChecker(appCtx, notificationService, taskMonitor, &backgroundTasks)

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

	server := newHTTPServer(":"+port, e)
	serverErrors := make(chan error, 1)

	go func() {
		log.Printf("Subdux starting on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
		close(serverErrors)
	}()

	var serveErr error
	select {
	case serveErr = <-serverErrors:
		if serveErr != nil {
			log.Printf("HTTP server stopped unexpectedly: %v", serveErr)
		}
	case <-signalCtx.Done():
		log.Printf("Received shutdown signal, starting graceful shutdown")
	}

	cancelBackground()

	if serveErr == nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("HTTP server shutdown error: %v", err)
			serveErr = err
		}
	} else if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("HTTP server close error: %v", err)
	}

	backgroundTasks.Wait()

	if err := sqlDB.Close(); err != nil {
		log.Printf("Database close error: %v", err)
		if serveErr == nil {
			serveErr = err
		}
	}

	if serveErr != nil {
		log.Fatalf("Subdux stopped with error: %v", serveErr)
	}

	log.Printf("Subdux stopped")
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

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		ReadTimeout:       httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
	}
}

func startNotificationChecker(
	ctx context.Context,
	ns *service.NotificationService,
	monitor *service.BackgroundTaskMonitor,
	wg *sync.WaitGroup,
) {
	const taskKey = "notification_check"
	const checkInterval = 3 * time.Hour

	if ctx == nil {
		ctx = context.Background()
	}

	if monitor != nil {
		monitor.Register(
			taskKey,
			"Notification check",
			"Checks due subscriptions and dispatches reminder notifications through enabled channels.",
			checkInterval,
		)
	}

	runCheck := func() {
		run := ns.ProcessPendingNotifications
		if monitor != nil {
			if err := monitor.Run(taskKey, run); err != nil {
				log.Printf("notification check error: %v", err)
			}
			return
		}
		if err := run(); err != nil {
			log.Printf("notification check error: %v", err)
		}
	}

	if wg != nil {
		wg.Add(1)
	}

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		if ctx.Err() != nil {
			return
		}

		ticker := time.NewTicker(checkInterval)
		defer ticker.Stop()

		runCheck()

		for {
			select {
			case <-ticker.C:
				runCheck()
			case <-ctx.Done():
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
