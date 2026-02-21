package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	subdux "github.com/shiroha/subdux"
	"github.com/shiroha/subdux/internal/api"
	"github.com/shiroha/subdux/internal/pkg"
	"github.com/shiroha/subdux/internal/service"
)

func main() {
	db := pkg.InitDB()
	pkg.InitJWTSecret(db)

	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
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

func startNotificationChecker(ns *service.NotificationService, stop <-chan struct{}) {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
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
