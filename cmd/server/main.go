package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path"
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
	"github.com/shiroha/subdux/internal/pkg/logging"
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
	logging.Init()

	signalCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	appCtx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()

	db := pkg.InitDB()
	sqlDB, err := db.DB()
	if err != nil {
		logging.Fatal("failed to access database handle", slog.Any("error", err))
	}
	if err := pkg.InitJWTSecret(db); err != nil {
		logging.Fatal("failed to initialize JWT secret", slog.Any("error", err))
	}
	bootstrapInitialAdmin(db)

	e := echo.New()
	e.HideBanner = true

	e.Use(logging.RequestLogger())
	e.Use(api.SecurityHeadersMiddleware)
	e.Use(middleware.RecoverWithConfig(middleware.RecoverConfig{
		LogErrorFunc: func(c echo.Context, err error, stack []byte) error {
			logging.FromContext(c.Request().Context()).Error("recovered from panic",
				slog.Any("error", err),
				slog.String("stack", string(stack)),
			)
			return err
		},
	}))

	allowedOrigins := loadCORSOrigins(db)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: allowedOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "X-API-Key", "MCP-Protocol-Version"},
	}))

	taskMonitor := service.NewBackgroundTaskMonitor()
	erService, notificationService := api.SetupRoutes(appCtx, e, db, taskMonitor)

	var backgroundTasks sync.WaitGroup
	erService.StartBackgroundRefresh(appCtx, taskMonitor, &backgroundTasks)
	startNotificationWorkers(appCtx, notificationService, taskMonitor, &backgroundTasks)
	startSubscriptionLifecycleSweep(appCtx, service.NewSubscriptionService(db), taskMonitor, &backgroundTasks)

	setupUploads(e, filepath.Join(pkg.GetDataPath(), "assets"))

	distFS, err := fs.Sub(subdux.StaticFS, "web/dist")
	if err != nil {
		logging.Fatal("failed to access embedded frontend", slog.Any("error", err))
	}
	setupSPA(e, distFS)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	server := newHTTPServer(":"+port, e)
	serverErrors := make(chan error, 1)

	go func() {
		logging.Info("subdux starting", slog.String("port", port))
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		}
		close(serverErrors)
	}()

	var serveErr error
	select {
	case serveErr = <-serverErrors:
		if serveErr != nil {
			logging.Error("HTTP server stopped unexpectedly", slog.Any("error", serveErr))
		}
	case <-signalCtx.Done():
		logging.Info("received shutdown signal, starting graceful shutdown")
	}

	cancelBackground()

	if serveErr == nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			logging.Error("HTTP server shutdown error", slog.Any("error", err))
			serveErr = err
		}
	} else if err := server.Close(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logging.Error("HTTP server close error", slog.Any("error", err))
	}

	backgroundTasks.Wait()

	if err := sqlDB.Close(); err != nil {
		logging.Error("database close error", slog.Any("error", err))
		if serveErr == nil {
			serveErr = err
		}
	}

	if serveErr != nil {
		logging.Fatal("subdux stopped with error", slog.Any("error", serveErr))
	}

	logging.Info("subdux stopped")
}

func bootstrapInitialAdmin(db *gorm.DB) {
	input := service.InitialAdminInput{
		Username: envOrDefault("SUBDUX_INITIAL_ADMIN_USERNAME", "admin"),
		Email:    envOrDefault("SUBDUX_INITIAL_ADMIN_EMAIL", "admin@subdux.local"),
		Password: strings.TrimSpace(os.Getenv("SUBDUX_INITIAL_ADMIN_PASSWORD")),
	}

	result, err := service.NewAuthService(db).EnsureInitialAdmin(input)
	if err != nil {
		logging.Fatal("failed to initialize admin user", slog.Any("error", err))
	}
	if result == nil || !result.Created {
		return
	}

	logging.Info("initial admin user created",
		slog.String("username", result.Username),
		slog.String("email", result.Email),
	)
	if input.Password == "" {
		// Intentional first-boot bootstrap behavior: print the generated password once
		// when no SUBDUX_INITIAL_ADMIN_PASSWORD is provided.
		//
		// The password is written directly to standard output (not through the
		// structured logger) so it is always visible regardless of LOG_LEVEL and
		// is never captured by the secret-redacting log handler.
		writeInitialAdminPassword(result.Password)
		return
	}
	logging.Info("initial admin password loaded from SUBDUX_INITIAL_ADMIN_PASSWORD and was not printed")
}

// writeInitialAdminPassword prints the one-time generated admin password to
// standard output. It deliberately bypasses the structured logger: the value
// must be shown exactly once during first-time initialization and must not be
// redacted or filtered by log level.
func writeInitialAdminPassword(password string) {
	fmt.Println("================================================================")
	fmt.Println("Subdux initial admin password (shown only once, store it now):")
	fmt.Printf("  %s\n", password)
	fmt.Println("================================================================")
}

func envOrDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
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

	logging.Warn("no CORS_ALLOW_ORIGINS or site_url configured; falling back to localhost development origins")

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

func setupUploads(e *echo.Echo, assetsRoot string) {
	e.GET("/uploads/*", func(c echo.Context) error {
		return serveUploadedAsset(c, assetsRoot)
	})
}

func serveUploadedAsset(c echo.Context, assetsRoot string) error {
	relativePath, ok := cleanUploadedAssetPath(c.Param("*"))
	if !ok {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	root, err := os.OpenRoot(assetsRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		return err
	}
	defer func() {
		_ = root.Close()
	}()

	linkInfo, err := root.Lstat(relativePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		return err
	}
	if !linkInfo.Mode().IsRegular() {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	file, err := root.Open(relativePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return echo.NewHTTPError(http.StatusNotFound)
		}
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return echo.NewHTTPError(http.StatusNotFound)
	}

	filename := path.Base(relativePath)
	header := c.Response().Header()
	header.Set(echo.HeaderContentType, uploadedAssetContentType(relativePath))
	header.Set(echo.HeaderXContentTypeOptions, "nosniff")
	header.Set(echo.HeaderContentSecurityPolicy, "default-src 'none'; base-uri 'none'; form-action 'none'; sandbox")
	header.Set("Content-Disposition", mime.FormatMediaType("inline", map[string]string{"filename": filename}))
	header.Set(echo.HeaderCacheControl, "public, max-age=86400")

	http.ServeContent(c.Response().Writer, c.Request(), filename, info.ModTime(), file)
	return nil
}

func cleanUploadedAssetPath(rawPath string) (string, bool) {
	unescaped, err := url.PathUnescape(strings.TrimPrefix(rawPath, "/"))
	if err != nil {
		return "", false
	}
	cleanPath := path.Clean(strings.TrimPrefix(unescaped, "/"))
	if cleanPath == "." || cleanPath == ".." || strings.HasPrefix(cleanPath, "../") {
		return "", false
	}
	if !isServableUploadedAssetPath(cleanPath) {
		return "", false
	}
	return cleanPath, true
}

func isServableUploadedAssetPath(relativePath string) bool {
	parts := strings.Split(relativePath, "/")
	if len(parts) != 2 || parts[0] != "icons" {
		return false
	}
	filename := parts[1]
	if filename == "" || path.Base(filename) != filename {
		return false
	}

	switch strings.ToLower(path.Ext(filename)) {
	case ".png", ".jpg", ".jpeg", ".ico":
		return true
	default:
		return false
	}
}

func uploadedAssetContentType(relativePath string) string {
	switch strings.ToLower(path.Ext(relativePath)) {
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".ico":
		return "image/x-icon"
	default:
		return "application/octet-stream"
	}
}

func startNotificationWorkers(
	ctx context.Context,
	ns *service.NotificationService,
	monitor *service.BackgroundTaskMonitor,
	wg *sync.WaitGroup,
) {
	const (
		scanTaskKey      = "notification_scan"
		dispatchTaskKey  = "notification_dispatch"
		scanInterval     = 3 * time.Hour
		dispatchInterval = time.Minute
	)

	if ctx == nil {
		ctx = context.Background()
	}

	if monitor != nil {
		monitor.Register(
			scanTaskKey,
			"Notification scan",
			"Checks due subscriptions and enqueues reminder notification jobs.",
			scanInterval,
		)
		monitor.Register(
			dispatchTaskKey,
			"Notification dispatch",
			"Claims queued reminder notification jobs and delivers them through enabled channels.",
			dispatchInterval,
		)
	}

	runScan := func() {
		run := ns.EnqueuePendingNotifications
		if monitor != nil {
			if err := monitor.Run(scanTaskKey, run); err != nil {
				logging.Error("notification scan failed", slog.Any("error", err))
			}
			return
		}
		if err := run(); err != nil {
			logging.Error("notification scan failed", slog.Any("error", err))
		}
	}

	runDispatch := func() {
		run := func() error {
			_, err := ns.DispatchDueNotificationOutbox(ctx)
			return err
		}
		if monitor != nil {
			if err := monitor.Run(dispatchTaskKey, run); err != nil {
				logging.Error("notification dispatch failed", slog.Any("error", err))
			}
			return
		}
		if err := run(); err != nil {
			logging.Error("notification dispatch failed", slog.Any("error", err))
		}
	}

	if wg != nil {
		wg.Add(2)
	}

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		if ctx.Err() != nil {
			return
		}

		ticker := time.NewTicker(scanInterval)
		defer ticker.Stop()

		runScan()
		runDispatch()

		for {
			select {
			case <-ticker.C:
				runScan()
				runDispatch()
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		if wg != nil {
			defer wg.Done()
		}

		if ctx.Err() != nil {
			return
		}

		ticker := time.NewTicker(dispatchInterval)
		defer ticker.Stop()

		runDispatch()

		for {
			select {
			case <-ticker.C:
				runDispatch()
			case <-ctx.Done():
				return
			}
		}
	}()
}

// startSubscriptionLifecycleSweep periodically advances subscription lifecycle
// for all users in the background. It is the primary driver of lifecycle
// transitions so that read requests no longer have to write: the sweep handles
// the boundary-crossing case ahead of time, and the read path's own reconcile
// remains only as a backstop for the window between sweeps.
func startSubscriptionLifecycleSweep(
	ctx context.Context,
	subscriptions *service.SubscriptionService,
	monitor *service.BackgroundTaskMonitor,
	wg *sync.WaitGroup,
) {
	const (
		taskKey       = "subscription_lifecycle_sweep"
		sweepInterval = time.Hour
	)

	if ctx == nil {
		ctx = context.Background()
	}

	ownerID := service.NewBackgroundTaskOwnerID()

	if monitor != nil {
		monitor.Register(
			taskKey,
			"Subscription lifecycle sweep",
			"Rolls renewals forward and ends overdue subscriptions so read requests stay write-free.",
			sweepInterval,
		)
	}

	runSweep := func() {
		run := func() error { return subscriptions.ReconcileDueLifecycles(ownerID) }
		if monitor != nil {
			if err := monitor.Run(taskKey, run); err != nil {
				logging.Error("subscription lifecycle sweep failed", slog.Any("error", err))
			}
			return
		}
		if err := run(); err != nil {
			logging.Error("subscription lifecycle sweep failed", slog.Any("error", err))
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

		runSweep()

		ticker := time.NewTicker(sweepInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				runSweep()
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
			serveSPAIndex(w, indexHTML)
			return
		}

		// Try serving the actual file
		if f, err := fsys.Open(path); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		serveSPAIndex(w, indexHTML)
	})))
}

func serveSPAIndex(w http.ResponseWriter, indexHTML []byte) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, max-age=0, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	_, _ = w.Write(indexHTML)
}
