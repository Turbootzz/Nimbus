package main

import (
	"context"
	"errors"
	"log"
	"net/mail"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/nimbus/backend/internal/config"
	"github.com/nimbus/backend/internal/configfile"
	"github.com/nimbus/backend/internal/db"
	"github.com/nimbus/backend/internal/handlers"
	"github.com/nimbus/backend/internal/mcpserver"
	"github.com/nimbus/backend/internal/middleware"
	"github.com/nimbus/backend/internal/models"
	"github.com/nimbus/backend/internal/repository"
	"github.com/nimbus/backend/internal/services"
	"github.com/nimbus/backend/internal/utils"
	"github.com/nimbus/backend/internal/workers"
)

func main() {
	// Load environment variables
	config.MustLoadEnv()

	// Integration and notification credentials are encrypted with
	// ENCRYPTION_KEY. Only the server needs it, so it is checked here instead
	// of in LoadEnv.
	credentialCipher, err := utils.NewCipherFromEnv()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}

	// Connect to database
	database, err := db.Connect()
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()
	log.Println("✓ Connected to database")

	// Run database migrations
	log.Println("Running database migrations...")
	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("✓ Database migrations completed")

	// Initialize repositories
	userRepo := repository.NewUserRepository(database)
	serviceRepo := repository.NewServiceRepository(database)
	preferencesRepo := repository.NewPreferencesRepository(database)
	statusLogRepo := repository.NewStatusLogRepository(database)
	groupRepo := repository.NewGroupRepository(database)
	webhookRepo := repository.NewWebhookRepository(database)
	settingsRepo := repository.NewSettingsRepository(database)
	apiTokenRepo := repository.NewAPITokenRepository(database)
	integrationRepo := repository.NewIntegrationRepository(database)
	widgetRepo := repository.NewWidgetRepository(database)
	snapshotRepo := repository.NewSnapshotRepository(database)
	discoveryRepo := repository.NewDiscoveryRepository(database)

	// Initialize services
	authService := services.NewAuthService()
	emailService := services.NewEmailService(settingsRepo)
	passwordResetRepo := repository.NewPasswordResetRepository(database)

	// Auto-create admin from env vars (for cloud/automated deployments)
	if adminEmail := os.Getenv("INITIAL_ADMIN_EMAIL"); adminEmail != "" {
		adminPassword := os.Getenv("INITIAL_ADMIN_PASSWORD")
		if _, parseErr := mail.ParseAddress(adminEmail); parseErr != nil {
			log.Printf("WARNING: INITIAL_ADMIN_EMAIL is not a valid email address, skipping auto-setup")
		} else if adminPassword == "" {
			log.Println("WARNING: INITIAL_ADMIN_EMAIL set but INITIAL_ADMIN_PASSWORD is empty, skipping auto-setup")
		} else if len(adminPassword) < 8 {
			log.Println("WARNING: INITIAL_ADMIN_PASSWORD must be at least 8 characters, skipping auto-setup")
		} else {
			hashedPassword, hashErr := authService.HashPassword(adminPassword)
			if hashErr != nil {
				log.Printf("WARNING: Failed to hash admin password: %v", hashErr)
			} else {
				now := time.Now()
				user := &models.User{
					Email:         adminEmail,
					Name:          "Admin",
					Password:      &hashedPassword,
					Role:          "admin",
					Provider:      "local",
					EmailVerified: true,
					CreatedAt:     now,
					UpdatedAt:     now,
				}
				if createErr := userRepo.CreateAdminIfNone(user); createErr != nil {
					if errors.Is(createErr, repository.ErrUsersAlreadyExist) {
						log.Println("Auto-setup: Users already exist, skipping")
					} else {
						log.Printf("WARNING: Auto-setup failed: %v", createErr)
					}
				} else {
					log.Printf("Auto-setup: Admin user created (%s)", adminEmail)
				}
			}
		}
	}

	// Initialize OAuth service with provider configurations
	googleConfig := config.GetGoogleOAuthConfig()
	githubConfig := config.GetGitHubOAuthConfig()
	discordConfig := config.GetDiscordOAuthConfig()
	oidcConfig := config.GetOIDCConfig()
	oauthStateSecret := os.Getenv("OAUTH_STATE_SECRET")
	if oauthStateSecret == "" {
		oauthStateSecret = os.Getenv("JWT_SECRET") // Fallback to JWT_SECRET
	}
	oauthService := services.NewOAuthService(googleConfig, githubConfig, discordConfig, oidcConfig, oauthStateSecret)

	// Initialize notification service
	notificationService := services.NewNotificationService(webhookRepo, serviceRepo)
	notificationService.SetCipher(credentialCipher)

	// Initialize health check service
	healthCheckTimeout := getEnvDuration("HEALTH_CHECK_TIMEOUT", 10*time.Second)
	healthCheckService := services.NewHealthCheckService(serviceRepo, statusLogRepo, notificationService, healthCheckTimeout)

	// Initialize metrics service
	metricsService := services.NewMetricsService(statusLogRepo, serviceRepo)

	// Initialize integration service; DOCKER_SOCKET lets the Docker integration use that socket
	dockerSocket := config.GetEnvOrDefault("DOCKER_SOCKET", "")
	if dockerSocket != "" {
		log.Printf("Docker integration may use the socket %s", dockerSocket)
	}
	integrationService := services.NewIntegrationService(integrationRepo, credentialCipher, dockerSocket)

	// Docker discovery: offers containers as services, reads only the container list
	discoveryService := services.NewDiscoveryService(integrationService, discoveryRepo, serviceRepo, groupRepo, settingsRepo, handlers.DashboardIconsBase())

	// Initialize widget service
	widgetService := services.NewWidgetService(widgetRepo, groupRepo, integrationRepo)

	// Live widget data: the poller fetches, the hub pushes to open dashboards
	sseHub := services.NewSSEHub()
	liveDataService := services.NewLiveDataService(widgetRepo, integrationRepo, integrationService, snapshotRepo, sseHub)
	liveDataService.SetStripSource(preferencesRepo)
	if err := liveDataService.Warm(context.Background()); err != nil {
		log.Printf("WARNING: Failed to load widget snapshots: %v", err)
	}
	widgetPoller := workers.NewWidgetPoller(liveDataService)
	widgetService.SetPoller(widgetPoller)
	// Boards: who may see which one is decided in boardService only
	boardService := services.NewBoardService(repository.NewBoardRepository(database))
	widgetService.SetBoards(boardService)
	integrationService.SetPoller(widgetPoller)
	healthCheckService.SetStatusPublisher(liveDataService)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userRepo, authService, settingsRepo, preferencesRepo)
	oauthHandler := handlers.NewOAuthHandler(oauthService, authService, userRepo, settingsRepo)
	serviceHandler := handlers.NewServiceHandler(serviceRepo, groupRepo, healthCheckService, integrationRepo)
	serviceHandler.SetPreferences(preferencesRepo)
	serviceHandler.SetPoller(widgetPoller)
	serviceHandler.SetBoards(boardService)
	preferencesHandler := handlers.NewPreferencesHandler(preferencesRepo)
	preferencesHandler.SetPoller(widgetPoller)
	// Wallpapers of users deleted by an admin, or left by a crash
	go handlers.PruneWallpapers(context.Background(), preferencesRepo)
	adminHandler := handlers.NewAdminHandler(userRepo)
	metricsHandler := handlers.NewMetricsHandler(metricsService, serviceRepo)
	metricsHandler.SetBoards(boardService)
	uploadHandler := handlers.NewUploadHandler(userRepo)
	staticHandler := handlers.NewStaticHandler()
	groupHandler := handlers.NewGroupHandler(groupRepo)
	groupHandler.SetBoards(boardService)
	boardHandler := handlers.NewBoardHandler(boardService)
	webhookHandler := handlers.NewWebhookHandler(webhookRepo, notificationService)
	passwordHandler := handlers.NewPasswordHandler(userRepo, authService, emailService, passwordResetRepo)
	settingsHandler := handlers.NewSettingsHandler(settingsRepo, emailService)
	setupHandler := handlers.NewSetupHandler(userRepo, authService)
	apiTokenHandler := handlers.NewAPITokenHandler(apiTokenRepo)
	apiTokenHandler.SetActivityLog(repository.NewActivityLogRepository(database))
	middleware.SetAPITokenRateLimit(getEnvPositiveInt("API_TOKEN_RATE_LIMIT", middleware.DefaultAPITokenRateLimit))
	integrationHandler := handlers.NewIntegrationHandler(integrationService)
	widgetHandler := handlers.NewWidgetHandler(widgetService)
	widgetHandler.SetBoards(boardService)
	discoveryHandler := handlers.NewDiscoveryHandler(discoveryService)
	dashboardHandler := handlers.NewDashboardHandler(liveDataService, sseHub)
	dashboardHandler.SetBoards(boardService)
	statusPageService := services.NewStatusPageService(settingsRepo, groupRepo, serviceRepo, statusLogRepo, userRepo, handlers.UploadDir)
	statusPageHandler := handlers.NewStatusPageHandler(statusPageService)

	// Dashboard export/import
	transferService := services.NewDashboardTransferService(services.DashboardTransferDeps{
		Dashboard:          repository.NewDashboardRepository(database),
		Groups:             groupRepo,
		Services:           serviceRepo,
		Widgets:            widgetRepo,
		Integrations:       integrationRepo,
		Preferences:        preferencesRepo,
		Users:              userRepo,
		IntegrationService: integrationService,
		Images:             handlers.NewUploadStore(preferencesRepo),
	})
	transferService.SetPoller(widgetPoller)
	transferHandler := handlers.NewDashboardTransferHandler(transferService)

	// Config-file mode: one admin's dashboard follows NIMBUS_CONFIG_FILE.
	// Created after the auto-setup above, so its admin can be the owner.
	configFile, err := config.GetConfigFile()
	if err != nil {
		log.Fatalf("Failed to load environment: %v", err)
	}
	var configWatcher *configfile.Watcher
	if configFile.Path != "" {
		configWatcher = configfile.NewWatcher(configfile.Options{
			Path: configFile.Path, Prune: configFile.Prune, ReadOnly: configFile.ReadOnly, Settle: configfile.DefaultSettle,
		}, transferService, userRepo)
		authHandler.SetConfigFile(configWatcher)
		// Auto-add leaves a dashboard alone that the file would undo it on
		discoveryService.SetManaged(configWatcher.OwnsServices)
	}
	// NIMBUS_CONFIG_READONLY: the owner can't edit what the file manages
	lockDashboard := middleware.ConfigReadOnly(func(userID string) bool {
		return configWatcher != nil && configWatcher.Locks(userID)
	})
	lockPreferences := middleware.ConfigReadOnly(func(userID string) bool {
		return configWatcher != nil && configWatcher.LocksPreferences(userID)
	})

	// Create fiber app
	app := fiber.New(fiber.Config{
		AppName:   "Nimbus API",
		BodyLimit: handlers.MaxRequestBodySize,
		// In the Docker image nginx sits in front on the same host and sets
		// X-Real-IP; believe that header from loopback only, so c.IP() is the
		// client and nobody else can fake it
		ProxyHeader:             "X-Real-IP",
		EnableTrustedProxyCheck: true,
		TrustedProxies:          []string{"127.0.0.1", "::1"},
		EnableIPValidation:      true, // no header (local development): the peer address
	})

	// Middleware
	app.Use(logger.New())

	// CORS middleware - only enabled when CORS_ORIGINS is set
	// In unified mode (nginx proxy), same-origin requests don't need CORS
	if corsOrigins := os.Getenv("CORS_ORIGINS"); corsOrigins != "" {
		log.Printf("CORS enabled for origins: %s", corsOrigins)
		app.Use(cors.New(cors.Config{
			AllowOrigins:     corsOrigins,
			AllowCredentials: true,
			AllowHeaders:     "Origin, Content-Type, Accept, Authorization",
		}))
	} else {
		log.Println("CORS disabled (same-origin mode)")
	}

	// Routes
	api := app.Group("/api")
	v1 := api.Group("/v1")

	// Health check
	v1.Get("/health", func(c *fiber.Ctx) error {
		response := fiber.Map{"status": "healthy", "app": "nimbus"}
		if err := database.Ping(); err != nil {
			response["status"] = "unhealthy"
			response["database"] = "disconnected"
			return c.Status(fiber.StatusServiceUnavailable).JSON(response)
		}
		response["database"] = "connected"
		return c.JSON(response)
	})

	// Auth routes (public)
	auth := v1.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/logout", authHandler.Logout)
	auth.Post("/forgot-password", passwordHandler.ForgotPassword)
	auth.Post("/reset-password", passwordHandler.ResetPassword)

	// OAuth routes (public for initiate and callback)
	auth.Get("/oauth/providers", oauthHandler.GetProviderStatus)
	auth.Get("/oauth/:provider", oauthHandler.InitiateOAuth)
	auth.Get("/oauth/:provider/callback", oauthHandler.HandleCallback)

	// Protected auth routes (session only — account management is off-limits
	// for API tokens)
	authProtected := auth.Group("", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo), middleware.RequireSessionAuth())
	authProtected.Get("/me", authHandler.GetMe)
	authProtected.Delete("/me", authHandler.DeleteAccount)
	authProtected.Put("/change-password", passwordHandler.ChangePassword)
	authProtected.Post("/oauth/link/:provider", oauthHandler.LinkProvider)
	authProtected.Delete("/oauth/unlink/:provider", oauthHandler.UnlinkProvider)

	// API tokens reach a route group only through the read scope passed to
	// AuthMiddleware: safe requests need that scope. Tokens can only write
	// in read:services groups, with write:services. Groups without a scope
	// are closed to tokens.
	sessionAuth := middleware.AuthMiddleware(authService, userRepo, apiTokenRepo)
	sessionOnly := middleware.RequireSessionAuth()

	// The favicon lookup makes the server fetch a URL and store the result,
	// so it is not a read a token may do. Registered before the services group.
	v1.Get("/services/favicon", sessionAuth, sessionOnly, serviceHandler.FetchServiceFavicon)

	// Status logs are metrics, so they are registered before the services
	// group to get their own scope
	v1.Get("/services/:id<guid>/status-logs", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadMetrics), metricsHandler.GetRecentStatusLogs)

	// Service routes (all protected)
	services := v1.Group("/services", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadServices), lockDashboard)
	services.Post("/", serviceHandler.CreateService)
	services.Get("/", serviceHandler.GetServices)
	// Sets service positions exactly as given. Services share their position
	// space with widgets, so callers must not reuse a widget's position; the
	// dashboard uses PUT /dashboard/reorder instead.
	services.Put("/reorder", serviceHandler.ReorderServices) // Must be before /:id routes
	// <guid> constraint rejects non-UUID :id at the router level (404)
	// so reserved-word paths like /favicon never crash through to SQL.
	services.Get("/:id<guid>", serviceHandler.GetService)
	services.Put("/:id<guid>", serviceHandler.UpdateService)
	services.Delete("/:id<guid>", serviceHandler.DeleteService)
	services.Post("/:id<guid>/check", serviceHandler.CheckService)

	// Group routes (all protected)
	groups := v1.Group("/groups", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadServices), lockDashboard)
	groups.Post("/", groupHandler.CreateGroup)
	groups.Get("/", groupHandler.GetGroups)
	groups.Put("/reorder", groupHandler.ReorderGroups) // Must be before /:id routes
	groups.Get("/:id<guid>", groupHandler.GetGroup)
	groups.Put("/:id<guid>", groupHandler.UpdateGroup)
	groups.Delete("/:id<guid>", groupHandler.DeleteGroup)

	// Board routes. Listing is a read like the services on them; managing
	// boards is session only (sharing one makes it public to every user).
	boardRoutes := v1.Group("/boards", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadServices), lockDashboard)
	boardRoutes.Get("/", boardHandler.ListBoards)
	boardRoutes.Post("/", sessionOnly, boardHandler.CreateBoard)
	boardRoutes.Put("/reorder", sessionOnly, boardHandler.ReorderBoards) // Must be before /:id routes
	boardRoutes.Put("/:id<guid>", sessionOnly, boardHandler.UpdateBoard)
	boardRoutes.Delete("/:id<guid>", sessionOnly, boardHandler.DeleteBoard)
	boardRoutes.Get("/:id<guid>/integrations", sessionOnly, boardHandler.ListBoardIntegrations)

	// Webhook routes (protected; session only: webhook URLs are secrets and
	// a webhook makes the server send requests)
	webhooks := v1.Group("/webhooks", sessionAuth, sessionOnly)
	webhooks.Post("/", webhookHandler.CreateWebhook)
	webhooks.Get("/", webhookHandler.GetWebhooks)
	webhooks.Get("/:id<guid>", webhookHandler.GetWebhook)
	webhooks.Put("/:id<guid>", webhookHandler.UpdateWebhook)
	webhooks.Delete("/:id<guid>", webhookHandler.DeleteWebhook)
	webhooks.Post("/:id<guid>/test", webhookHandler.TestWebhook)
	webhooks.Get("/:id<guid>/logs", webhookHandler.GetWebhookLogs)

	// Static file serving (public, but files are only accessible if you know the filename)
	// IMPORTANT: This must be registered BEFORE the uploads group to avoid auth middleware
	v1.Get("/uploads/service-icons/:filename", staticHandler.ServeServiceIcon)
	v1.Get("/uploads/avatars/:filename", staticHandler.ServeAvatar)
	v1.Get("/uploads/wallpapers/:filename", staticHandler.ServeWallpaper)

	// Upload routes (protected, session only)
	uploads := v1.Group("/uploads", sessionAuth, sessionOnly)
	uploads.Post("/service-icon", uploadHandler.UploadServiceIcon)

	// Avatar, wallpaper and preferences (protected, session only)
	users := v1.Group("/users/me", sessionAuth, sessionOnly)
	users.Put("/avatar", uploadHandler.UploadAvatar)
	users.Put("/wallpaper", lockPreferences, preferencesHandler.UploadWallpaper)

	// Metrics routes (protected)
	metrics := v1.Group("/metrics", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadMetrics))
	metrics.Get("/:id<guid>", metricsHandler.GetServiceMetrics)

	// Prometheus metrics endpoint (supports both JWT and API key authentication)
	// Middleware is optional - handler checks for both JWT (from middleware) and API key
	prometheus := v1.Group("/prometheus")
	prometheus.Get("/metrics/user/:userID<guid>", middleware.OptionalAuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadMetrics), metricsHandler.GetUserPrometheusMetrics)

	// API token routes (protected; session auth only so a leaked token
	// can never create or revoke tokens)
	apiTokens := v1.Group("/tokens", sessionAuth, sessionOnly)
	apiTokens.Post("/", apiTokenHandler.CreateToken)
	apiTokens.Get("/", apiTokenHandler.ListTokens)
	apiTokens.Delete("/:id<guid>", apiTokenHandler.DeleteToken)

	// Integration routes (protected; session auth only so a leaked API token
	// can't point an integration elsewhere and exfiltrate its credentials)
	integrationRoutes := v1.Group("/integrations", sessionAuth, sessionOnly, lockDashboard)
	integrationRoutes.Get("/kinds", integrationHandler.ListKinds)
	integrationRoutes.Post("/test", integrationHandler.TestUnsavedIntegration)
	integrationRoutes.Post("/", integrationHandler.CreateIntegration)
	integrationRoutes.Get("/", integrationHandler.ListIntegrations)
	integrationRoutes.Get("/:id<guid>", integrationHandler.GetIntegration)
	integrationRoutes.Put("/:id<guid>", integrationHandler.UpdateIntegration)
	integrationRoutes.Delete("/:id<guid>", integrationHandler.DeleteIntegration)
	integrationRoutes.Post("/:id<guid>/test", integrationHandler.TestIntegration)

	// Discovery routes (admin only and session only, like the Docker integration they read)
	discovery := v1.Group("/discovery", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo), middleware.RequireSessionAuth(), middleware.AdminOnly())
	discovery.Get("/docker", discoveryHandler.DiscoverDocker)
	discovery.Post("/docker/services", lockDashboard, discoveryHandler.AddDiscovered)
	discovery.Post("/docker/ignored", discoveryHandler.IgnoreDiscovered)
	discovery.Delete("/docker/ignored", discoveryHandler.UnignoreDiscovered)

	// Widget routes (all protected)
	widgetRoutes := v1.Group("/widgets", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadWidgets), lockDashboard)
	widgetRoutes.Get("/types", widgetHandler.ListTypes) // Must be before /:id routes
	widgetRoutes.Post("/", widgetHandler.CreateWidget)
	widgetRoutes.Get("/", widgetHandler.ListWidgets)
	widgetRoutes.Get("/:id<guid>", widgetHandler.GetWidget)
	widgetRoutes.Put("/:id<guid>", widgetHandler.UpdateWidget)
	widgetRoutes.Delete("/:id<guid>", widgetHandler.DeleteWidget)
	widgetRoutes.Post("/:id<guid>/refresh", widgetHandler.RefreshWidget)

	// Dashboard routes (all protected); services and widgets share one order
	dashboard := v1.Group("/dashboard", middleware.AuthMiddleware(authService, userRepo, apiTokenRepo, models.ScopeReadWidgets), lockDashboard)
	dashboard.Put("/reorder", widgetHandler.ReorderTiles)
	dashboard.Get("/data", dashboardHandler.Data)
	// Session only: an open stream would outlive the token's revocation,
	// expiry and rate limit
	dashboard.Get("/stream", sessionOnly, dashboardHandler.Stream)

	users.Get("/preferences", preferencesHandler.GetPreferences)
	users.Put("/preferences", lockPreferences, preferencesHandler.UpdatePreferences)

	// Dashboard export/import (session only, like the integrations it reads
	// and replaces)
	v1.Get("/export", sessionAuth, sessionOnly, transferHandler.Export)
	v1.Post("/import", sessionAuth, sessionOnly, lockDashboard, transferHandler.Import)
	// Turns another dashboard's config into a document; it writes nothing
	v1.Post("/import/convert", sessionAuth, sessionOnly, transferHandler.Convert)

	// Setup routes (public - for first-time setup)
	setup := v1.Group("/setup")
	setup.Get("/status", setupHandler.GetSetupStatus)
	setup.Post("/admin", setupHandler.CreateInitialAdmin)
	setup.Get("/registration-status", settingsHandler.GetPublicRegistrationStatus)

	// Public status page (no auth; off unless an admin enables it)
	v1.Get("/public/status", statusPageHandler.RateLimit(), statusPageHandler.GetPublic)

	// Admin routes (protected, admin only, session only — API tokens can't
	// reach admin surfaces like SMTP settings)
	admin := v1.Group("/admin", sessionAuth, sessionOnly, middleware.AdminOnly())
	admin.Get("/users", adminHandler.GetAllUsers)
	admin.Get("/users/stats", adminHandler.GetUserStats)
	admin.Put("/users/:id<guid>/role", adminHandler.UpdateUserRole)
	admin.Delete("/users/:id<guid>", adminHandler.DeleteUser)
	admin.Get("/settings", settingsHandler.GetSettings)
	admin.Get("/settings/smtp/status", settingsHandler.GetSMTPStatus)
	admin.Put("/settings/smtp", settingsHandler.UpdateSMTPSettings)
	admin.Post("/settings/smtp/test", settingsHandler.TestSMTPConnection)
	admin.Get("/status-page", statusPageHandler.GetSettings)
	admin.Put("/status-page", statusPageHandler.UpdateSettings)
	admin.Get("/settings/:key", settingsHandler.GetSetting)
	admin.Put("/settings/:key", settingsHandler.UpdateSetting)

	// MCP endpoint for AI assistants. Not under /api: clients are configured
	// with this URL. Needs an API token with the mcp scope.
	app.All("/mcp", mcpserver.NewHandler(mcpserver.Deps{
		Services:     serviceRepo,
		Groups:       groupRepo,
		StatusLogs:   statusLogRepo,
		Widgets:      widgetService,
		Snapshots:    liveDataService,
		Integrations: integrationService,
		Health:       healthCheckService,
		Users:        userRepo,
		Tokens:       apiTokenRepo,
	}))

	// Start health check monitor
	healthCheckInterval := getEnvDuration("HEALTH_CHECK_INTERVAL", 60*time.Second)
	healthMonitor := workers.NewHealthMonitor(healthCheckService, serviceRepo, healthCheckInterval)
	healthMonitor.Start()

	// Start metrics cleanup worker (also cleans up webhook logs)
	metricsCleanup := workers.NewMetricsCleanupWorker(metricsService, webhookRepo, passwordResetRepo)
	metricsCleanup.SetSnapshotPruner(liveDataService)
	metricsCleanup.Start()

	// Start widget poller
	widgetPoller.Start()

	// Start discovery worker (auto-adds labelled containers when the admin setting is on)
	discoveryWorker := workers.NewDiscoveryWorker(discoveryService)
	discoveryWorker.Start()

	// Apply the config file now and whenever it changes
	if configWatcher != nil {
		configWatcher.Start()
	}

	// Start DNS cache cleanup worker
	dnsCleanup := workers.NewDNSCleanupWorker()
	dnsCleanup.Start()

	// Start rate limit cache cleanup worker
	rateLimitCleanup := workers.NewRateLimitCleanupWorker()
	rateLimitCleanup.Start()

	// Catch-all 404 handler — registered last so it only fires for paths no
	// other route matched. Returns JSON so API clients can parse it uniformly;
	// in particular this covers Fiber's <guid> constraint rejections, which
	// would otherwise hit Fiber's default text/plain "Cannot GET …" body.
	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not found"})
	})

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	go func() {
		log.Printf("Server starting on port %s", port)
		log.Printf("Auth endpoints available:")
		log.Printf("  POST /api/v1/auth/register")
		log.Printf("  POST /api/v1/auth/login")
		log.Printf("  POST /api/v1/auth/logout")
		log.Printf("  GET  /api/v1/auth/me (protected)")
		log.Printf("Service endpoints available:")
		log.Printf("  POST   /api/v1/services (protected)")
		log.Printf("  GET    /api/v1/services (protected)")
		log.Printf("  GET    /api/v1/services/:id (protected)")
		log.Printf("  PUT    /api/v1/services/:id (protected)")
		log.Printf("  DELETE /api/v1/services/:id (protected)")
		log.Printf("  POST   /api/v1/services/:id/check (protected) - Manual health check")
		log.Printf("  PUT    /api/v1/services/reorder (protected) - Reorder services")
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	log.Println("\nReceived shutdown signal, shutting down gracefully...")

	// Stop workers; close the dashboard streams first, or they keep
	// connections open and Shutdown waits for them
	sseHub.Close()
	if configWatcher != nil {
		configWatcher.Stop()
	}
	widgetPoller.Stop()
	discoveryWorker.Stop()
	healthMonitor.Stop()
	metricsCleanup.Stop()
	dnsCleanup.Stop()
	rateLimitCleanup.Stop()

	// Shutdown Fiber app
	if err := app.Shutdown(); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Server stopped")
}

// getEnvPositiveInt reads a whole number above zero from an environment variable
func getEnvPositiveInt(key string, defaultValue int) int {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valStr)
	if err != nil || value <= 0 {
		log.Printf("Invalid value for %s: %s, using default %d", key, valStr, defaultValue)
		return defaultValue
	}
	return value
}

// getEnvDuration reads a duration from environment variable in seconds
func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	valStr := os.Getenv(key)
	if valStr == "" {
		return defaultValue
	}

	seconds, err := strconv.Atoi(valStr)
	if err != nil {
		log.Printf("Invalid value for %s: %s, using default %v", key, valStr, defaultValue)
		return defaultValue
	}

	if seconds < 0 {
		log.Printf("Negative value for %s: %d, using default %v", key, seconds, defaultValue)
		return defaultValue
	}

	return time.Duration(seconds) * time.Second
}
