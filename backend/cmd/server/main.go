package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/orkestra/backend/internal/core/auth/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/container"
	"github.com/orkestra/backend/internal/shared/database"
	"github.com/orkestra/backend/internal/shared/errors"
	"github.com/orkestra/backend/internal/shared/iface"
	authMiddleware "github.com/orkestra/backend/internal/shared/middleware"
	"github.com/orkestra/backend/internal/shared/module"
	"github.com/orkestra/backend/internal/shared/remote"
	"github.com/orkestra/backend/internal/shared/setup"
	"github.com/orkestra/backend/internal/shared/utils"
)

func main() {
	logger := utils.SetupLogger()
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect infrastructure. 2-minute budget accommodates the
	// retry-with-backoff loops in NewMongoConnection and NewRedisConnection
	// (up to 20 attempts each) that wait out first-boot auth init races.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	db, err := database.NewMongoConnection(ctx, database.MongoConfig{
		URI:             cfg.Database.MongoURI,
		Database:        cfg.Database.DatabaseName,
		MaxPoolSize:     cfg.Database.MaxPoolSize,
		MinPoolSize:     cfg.Database.MinPoolSize,
		MaxConnIdleTime: cfg.Database.MaxConnIdleTime,
		ConnectTimeout:  cfg.Database.ConnectTimeout,
	})
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	redisClient, err := database.NewRedisConnection(ctx, database.RedisConfig{
		URL:             cfg.Redis.URL,
		MaxRetries:      cfg.Redis.MaxRetries,
		MinIdleConns:    cfg.Redis.MinIdleConns,
		MaxIdleConns:    cfg.Redis.MaxIdleConns,
		ConnMaxLifetime: cfg.Redis.ConnMaxLifetime,
		ReadTimeout:     cfg.Redis.ReadTimeout,
		WriteTimeout:    cfg.Redis.WriteTimeout,
	})
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	redisAdapter := database.NewRedisClientAdapter(redisClient)

	// Module config infrastructure (DB-backed module management)
	configRepo := module.NewModuleConfigRepository(db)
	configService := module.NewModuleConfigService(configRepo, redisAdapter, logger)

	// Initialize module registry
	svcRegistry := module.NewServiceRegistry()
	modRegistry := module.NewModuleRegistry(logger)
	modRegistry.SetConfigService(configService)
	modRegistry.SetContainerManager(container.NewManager(logger))
	modDeps := &module.Dependencies{
		DB:           db,
		RedisAdapter: redisAdapter,
		Config:       cfg,
		Logger:       logger,
		Services:     svcRegistry,
	}

	// Core modules — always loaded (auth, users, navigation)
	for _, factory := range coreModules {
		modRegistry.Register(factory())
	}

	// All optional modules are always instantiated and initialized so they
	// can be enabled/disabled at runtime without restart. Only enabled ones
	// are actually Start()ed after init.
	allOptNames := allOptionalModuleNames(logger)

	var optModules []module.Module
	for _, name := range allOptNames {
		if factory, ok := optionalModules[name]; ok {
			optModules = append(optModules, factory())
		}
	}
	if err := modRegistry.RegisterAll(optModules); err != nil {
		log.Fatalf("Failed to resolve module dependencies: %v", err)
	}

	if err := modRegistry.InitAll(cfg, modDeps); err != nil {
		log.Fatalf("Failed to initialize modules: %v", err)
	}

	// AI service sidecar: register remote providers for AI modules
	// that failed Init locally (e.g. external infra not available).
	if cfg.Server.AIServiceURL != "" {
		remoteAI := remote.NewAIModelProvider(cfg.Server.AIServiceURL)
		remoteRAG := remote.NewRAGQueryProvider(cfg.Server.AIServiceURL)

		failedModules := modRegistry.FailedModules()
		if _, failed := failedModules["aimodels"]; failed {
			svcRegistry.Register(module.ServiceAIModelProvider, remoteAI)
		}
		if _, failed := failedModules["rag"]; failed {
			svcRegistry.Register(module.ServiceRAGQuery, remoteRAG)
		}
		logger.Info("AI service sidecar configured",
			slog.String("url", cfg.Server.AIServiceURL),
		)
	}

	// Retrieve auth infrastructure for middleware setup
	jwtService := svcRegistry.MustGet(module.ServiceJWTService).(services.JWTService)
	authService := svcRegistry.MustGet(module.ServiceAuthService).(services.AuthService)

	// Error management
	errorManager := errors.NewManager(logger, cfg.Server.Environment != "production")
	defer errorManager.Close()

	// Auth middleware. The tenant and authz providers are wired after
	// InitAll so both are guaranteed registered in the ServiceRegistry.
	authMW := authMiddleware.NewAuthMiddlewareWithConfig(jwtService, errorManager, cfg)
	authMW.SetAuthService(authService)
	authMW.SetTenantProvider(module.MustGetTyped[iface.TenantProvider](svcRegistry, module.ServiceTenantProvider))
	authMW.SetAuthzProvider(module.MustGetTyped[iface.AuthzProvider](svcRegistry, module.ServiceAuthzProvider))
	deviceMW := authMiddleware.NewDeviceMiddleware(errorManager)

	// Router + middleware
	router := chi.NewRouter()
	setupMiddleware(router, cfg, errorManager, deviceMW)

	// API config
	apiConfig := huma.DefaultConfig("Orkestra API", "1.0.0")
	apiConfig.DocsPath = ""
	apiConfig.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"bearerAuth": {
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
		},
	}

	publicAPI := humachi.New(router, apiConfig)

	// Protected router
	protectedRouter := chi.NewRouter()
	protectedRouter.Use(authMW.RequireAuth)

	// Module routes
	modRegistry.RegisterAllRoutes(&module.RouteInfo{
		PublicAPI:        publicAPI,
		ProtectedRouter:  protectedRouter,
		Router:           router,
		AuthMW:           authMW,
		APIConfig:        apiConfig,
		ConfigService:    configService,
	})

	// First-install onboarding: public /v1/setup/status and /v1/setup/admin.
	// Reachable without auth — gated by the "no users exist" invariant
	// enforced inside setup.Service.CreateInitialAdmin.
	setupSvc := setup.NewService(
		module.MustGetTyped[iface.UserProvider](svcRegistry, module.ServiceUserService),
		module.MustGetTyped[setup.AdminCreator](svcRegistry, module.ServicePasswordAuthService),
		configService,
		logger,
	)
	setup.NewHandler(setupSvc, cfg.Auth.Cookie).RegisterRoutes(publicAPI)

	// Admin module management routes: platform-level, not per-org.
	protectedRouter.Group(func(r chi.Router) {
		r.Use(authMW.RequireSystemPermission("system.modules.admin"))
		adminAPI := humachi.New(r, apiConfig)
		moduleAdminHandler := module.NewModuleAdminHandler(configService, modRegistry)
		module.RegisterAdminModuleRoutes(adminAPI, moduleAdminHandler)
	})

	router.Mount("/", protectedRouter)

	// Health, readiness, docs
	registerHealthEndpoints(publicAPI, db, redisClient)
	registerDocsEndpoints(router, publicAPI)

	// HTTP server
	srv := &http.Server{
		Addr:           fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:        router,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   5 * time.Minute,
		IdleTimeout:    60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Build the start set: only modules that are currently enabled in the config
	// service (DB/env) get Start()ed. Others are initialized and routed but idle.
	startSet := make(map[string]bool)
	for _, name := range allOptNames {
		if configService.IsEnabled(context.Background(), name) {
			startSet[name] = true
		}
	}
	// Core modules are always started.
	for _, factory := range coreModules {
		m := factory()
		startSet[m.Name()] = true
	}

	// Start module background jobs
	if err := modRegistry.StartAll(context.Background(), startSet); err != nil {
		log.Fatalf("Failed to start modules: %v", err)
	}

	if !cfg.IsProduction() {
		utils.PrintDevelopmentWarning(cfg.Server.Environment)
	}

	// Serve
	go func() {
		logger.Info("Starting server",
			slog.String("port", cfg.Server.Port),
			slog.String("environment", cfg.Server.Environment),
		)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Failed to start server", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")
	modRegistry.StopAll(context.Background())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Failed to shutdown server gracefully", slog.String("error", err.Error()))
	}
	if err := database.DisconnectMongo(shutdownCtx, db); err != nil {
		logger.Error("Failed to disconnect from MongoDB", slog.String("error", err.Error()))
	}
	if err := database.DisconnectRedis(redisClient); err != nil {
		logger.Error("Failed to disconnect from Redis", slog.String("error", err.Error()))
	}

	logger.Info("Server stopped")
}
