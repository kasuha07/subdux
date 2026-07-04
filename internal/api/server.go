package api

import (
	"context"
	"log/slog"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kasuha07/subdux/internal/api/apimw"
	mcpapi "github.com/kasuha07/subdux/internal/api/mcp"
	"github.com/kasuha07/subdux/internal/pkg"
	"github.com/kasuha07/subdux/internal/pkg/logging"
	adminservice "github.com/kasuha07/subdux/internal/service/admin"
	apikeyservice "github.com/kasuha07/subdux/internal/service/apikey"
	auditservice "github.com/kasuha07/subdux/internal/service/audit"
	serviceauth "github.com/kasuha07/subdux/internal/service/auth"
	authreauth "github.com/kasuha07/subdux/internal/service/authreauth"
	servicebackup "github.com/kasuha07/subdux/internal/service/backup"
	calendarservice "github.com/kasuha07/subdux/internal/service/calendar"
	catalogservice "github.com/kasuha07/subdux/internal/service/catalog"
	exchangerate "github.com/kasuha07/subdux/internal/service/exchangerate"
	exporter "github.com/kasuha07/subdux/internal/service/exporter"
	iconproxy "github.com/kasuha07/subdux/internal/service/iconproxy"
	importer "github.com/kasuha07/subdux/internal/service/importer"
	notificationservice "github.com/kasuha07/subdux/internal/service/notification"
	servicereauth "github.com/kasuha07/subdux/internal/service/reauth"
	"github.com/kasuha07/subdux/internal/service/serviceutil"
	systemsettings "github.com/kasuha07/subdux/internal/service/settings"
	subscriptionservice "github.com/kasuha07/subdux/internal/service/subscription"
	echojwt "github.com/labstack/echo-jwt/v4"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// App is the API composition root. It owns the fully-constructed service and
// handler graph and knows how to mount every route onto an Echo instance. This
// separates dependency wiring (NewApp) from route registration (Mount) so each
// can be reasoned about and tested independently.
type App struct {
	db            *gorm.DB
	apiKeyService *apikeyservice.Service

	// Services surfaced to the caller for background workers.
	exchangeRateService *exchangerate.Service
	notificationService *notificationservice.Service

	handlers    apiHandlers
	registrars  []Registrar
	mcpHandler  *mcpapi.MCPHandler
	settingsSvc *systemsettings.Service
}

// apiHandlers groups the constructed handlers. Handlers hold their own service
// dependencies; grouping them keeps NewApp readable.
type apiHandlers struct {
	auth          *AuthHandler
	subscription  *SubscriptionHandler
	admin         *AdminHandler
	reauth        *ReauthHandler
	siteInfo      *SiteInfoHandler
	iconProxy     *IconProxyHandler
	exchangeRate  *ExchangeRateHandler
	currency      *CurrencyHandler
	category      *CategoryHandler
	paymentMethod *PaymentMethodHandler
	dashboard     *DashboardBootstrapHandler
	notification  *NotificationHandler
	template      *NotificationTemplateHandler
	apiKey        *APIKeyHandler
	audit         *AuditHandler
	calendar      *CalendarHandler
	export        *ExportHandler
	importer      *ImportHandler
	version       *VersionHandler
}

// NewApp constructs every service and handler. It starts the auth session
// cleanup loop and seeds default settings so the returned App is ready to mount.
func NewApp(ctx context.Context, db *gorm.DB, taskMonitor *serviceutil.BackgroundTaskMonitor) *App {
	authService := serviceauth.NewService(db)
	authService.StartSessionCleanupLoop(ctx)
	totpService := serviceauth.NewTOTPService(db)
	subService := subscriptionservice.NewService(db)
	adminService := adminservice.NewService(db)
	reauthService := servicereauth.NewService(db, authreauth.Adapt(authService))
	systemSettingsService := systemsettings.NewService(db)
	iconProxyService := iconproxy.NewService(db)
	erService := exchangerate.NewService(db)
	currencyService := catalogservice.NewCurrencyService(db)
	categoryService := catalogservice.NewCategoryService(db)
	paymentMethodService := catalogservice.NewPaymentMethodService(db)
	validator := notificationservice.NewTemplateValidator()
	renderer := notificationservice.NewTemplateRenderer(validator)
	templateService := notificationservice.NewNotificationTemplateService(db, validator)
	notificationService := notificationservice.NewService(db, templateService, renderer)
	apiKeyService := apikeyservice.NewService(db)
	auditService := auditservice.NewService(db)
	calendarService := calendarservice.NewService(db)
	exportService := exporter.NewService(db)
	importService := importer.NewService(db)
	if err := systemSettingsService.SeedDefaults(); err != nil {
		logging.Error("failed to seed default system settings", slog.Any("error", err))
	}

	h := apiHandlers{
		auth:          NewAuthHandler(authService, totpService, reauthService),
		subscription:  NewSubscriptionHandler(subService, erService),
		admin:         NewAdminHandler(adminService, taskMonitor, reauthService, servicebackup.NewService(db)),
		reauth:        NewReauthHandler(reauthService),
		siteInfo:      NewSiteInfoHandler(systemSettingsService),
		iconProxy:     NewIconProxyHandler(iconProxyService),
		exchangeRate:  NewExchangeRateHandler(erService),
		currency:      NewCurrencyHandler(currencyService, erService),
		category:      NewCategoryHandler(categoryService),
		paymentMethod: NewPaymentMethodHandler(paymentMethodService),
		dashboard:     NewDashboardBootstrapHandler(subService, erService, currencyService, categoryService, paymentMethodService),
		notification:  NewNotificationHandler(notificationService),
		template:      NewNotificationTemplateHandler(templateService),
		apiKey:        NewAPIKeyHandler(apiKeyService, reauthService),
		audit:         NewAuditHandler(auditService),
		calendar:      NewCalendarHandler(calendarService),
		export:        NewExportHandler(exportService, reauthService),
		importer:      NewImportHandler(importService, reauthService),
		version:       NewVersionHandler(db),
	}

	return &App{
		db:                  db,
		apiKeyService:       apiKeyService,
		exchangeRateService: erService,
		notificationService: notificationService,
		handlers:            h,
		mcpHandler: mcpapi.NewMCPHandler(apiKeyService, auditService, subService, erService,
			currencyService, categoryService, paymentMethodService),
		settingsSvc: systemSettingsService,
	}
}

// ExchangeRateService exposes the shared exchange-rate service for background
// refresh workers started by the caller.
func (a *App) ExchangeRateService() *exchangerate.Service { return a.exchangeRateService }

// NotificationService exposes the shared notification service for background
// dispatch workers started by the caller.
func (a *App) NotificationService() *notificationservice.Service { return a.notificationService }

// Mount registers the central error handler, the MCP endpoint, and every API
// route group onto e.
func (a *App) Mount(e *echo.Echo) {
	// Route all handler-returned errors through the single typed-error handler.
	// Handlers signal failures by returning *serviceerr.Error (or a wrapped
	// cause); the handler renders the frozen {"error": message} envelope with a
	// Kind-derived status. Echo's own HTTPErrors (jwt 401, 404, 405) are
	// delegated to the previous default so their behavior is unchanged.
	e.HTTPErrorHandler = APIErrorHandler(e.HTTPErrorHandler)

	a.mountMCP(e)
	groups := a.buildRouteGroups(e)
	for _, r := range a.registrarList() {
		r.RegisterRoutes(groups)
	}
}

func (a *App) mountMCP(e *echo.Echo) {
	requireMCPEnabled := apimw.MCPEnabledMiddleware(a.settingsSvc)
	e.POST("/mcp", a.mcpHandler.HandlePost, requireMCPEnabled, apimw.RequestBodyLimitMiddleware(1<<20, nil))
	e.GET("/mcp", a.mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.PUT("/mcp", a.mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.PATCH("/mcp", a.mcpHandler.MethodNotAllowed, requireMCPEnabled)
	e.DELETE("/mcp", a.mcpHandler.MethodNotAllowed, requireMCPEnabled)
}

func (a *App) buildRouteGroups(e *echo.Echo) RouteGroups {
	jwtConfig := echojwt.Config{
		SigningKey: pkg.GetJWTSecret(),
		NewClaimsFunc: func(c echo.Context) jwt.Claims {
			return new(pkg.JWTClaims)
		},
	}

	api := e.Group("/api")
	api.Use(apimw.RequestBodyLimitMiddleware(1<<20, func(c echo.Context) bool {
		path := c.Path()
		if path == "" {
			path = c.Request().URL.Path
		}
		return path == "/api/admin/restore" || path == "/api/import/wallos" || path == "/api/import/subdux"
	}))

	authGroup := api.Group("/auth")
	authGroup.Use(apimw.RequestBodyLimitMiddleware(apimw.MaxAuthRequestBodyBytes, nil))

	protected := api.Group("")
	protected.Use(apimw.JWTOrAPIKeyMiddleware(jwtConfig, a.apiKeyService))
	protected.Use(apimw.APIKeyScopeMiddleware)

	humanProtected := api.Group("")
	humanProtected.Use(apimw.JWTOrAPIKeyMiddleware(jwtConfig, a.apiKeyService))
	humanProtected.Use(apimw.HumanSessionOnlyMiddleware)
	humanProtected.Use(apimw.APIKeyScopeMiddleware)

	// Reauth ("step-up") is a generic capability, not admin-only: any human
	// session can prove presence and mint an operation-scoped ticket. It lives on
	// a human-session-only group mirroring humanProtected, not under /admin.
	reauth := api.Group("/reauth")
	reauth.Use(apimw.JWTOrAPIKeyMiddleware(jwtConfig, a.apiKeyService))
	reauth.Use(apimw.HumanSessionOnlyMiddleware)
	reauth.Use(apimw.APIKeyScopeMiddleware)

	admin := api.Group("/admin")
	admin.Use(echojwt.WithConfig(jwtConfig))
	admin.Use(apimw.AdminMiddleware)

	return RouteGroups{
		Public:         api,
		Auth:           authGroup,
		Protected:      protected,
		HumanProtected: humanProtected,
		Reauth:         reauth,
		Admin:          admin,
		Limiters:       newRateLimiters(),
	}
}

// registrarList returns every domain registrar in a stable order.
func (a *App) registrarList() []Registrar {
	h := a.handlers
	return []Registrar{
		h.version, h.iconProxy, h.siteInfo,
		h.auth, h.reauth,
		h.subscription, h.dashboard,
		h.exchangeRate, h.currency, h.category, h.paymentMethod,
		h.notification, h.template,
		h.apiKey, h.audit, h.calendar, h.export, h.importer,
		h.admin,
	}
}

// SetupRoutes builds the application and mounts it, returning the services the
// caller needs for background workers. It is a thin shim over NewApp/Mount kept
// for the existing cmd/server entrypoint and tests.
func SetupRoutes(
	ctx context.Context,
	e *echo.Echo,
	db *gorm.DB,
	taskMonitor *serviceutil.BackgroundTaskMonitor,
) (*exchangerate.Service, *notificationservice.Service) {
	app := NewApp(ctx, db, taskMonitor)
	app.Mount(e)
	return app.ExchangeRateService(), app.NotificationService()
}
