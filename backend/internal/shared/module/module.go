package module

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/database"
	"github.com/orkestra/backend/internal/shared/middleware"
	"go.mongodb.org/mongo-driver/mongo"
)

// Module defines the contract for a self-contained backend module.
// Each module manages its own initialization, route registration,
// background jobs, and graceful shutdown.
type Module interface {
	// Name returns the unique identifier for this module.
	Name() string

	// Enabled returns whether this module should be initialized
	// based on the current configuration.
	Enabled(cfg *config.Config) bool

	// Init initializes the module's repositories, services, and handlers.
	// Cross-module dependencies should be retrieved from deps.Services.
	Init(deps *Dependencies) error

	// RegisterRoutes registers the module's HTTP endpoints.
	RegisterRoutes(ri *RouteInfo)

	// Start launches background goroutines (polling jobs, workers).
	// Called after all modules are initialized and routes are registered.
	// Modules with no background work should return nil.
	Start(ctx context.Context) error

	// Stop performs graceful shutdown of background goroutines.
	// Called in reverse registration order during server shutdown.
	Stop(ctx context.Context) error

	// HealthCheck verifies the module's runtime dependencies are healthy.
	HealthCheck(ctx context.Context) error
}

// Dependencies holds shared infrastructure injected into every module.
type Dependencies struct {
	DB           *mongo.Database
	RedisAdapter *database.RedisClientAdapter
	Config       *config.Config
	Logger       *slog.Logger
	Services     *ServiceRegistry
}

// RouteInfo provides the routing infrastructure for module route registration.
type RouteInfo struct {
	// PublicAPI is for unauthenticated endpoints (health, webhooks, OAuth callbacks).
	PublicAPI huma.API

	// ProtectedRouter is the chi.Router with auth middleware applied.
	// Use this to create role-scoped route groups.
	ProtectedRouter chi.Router

	// Router is the root chi.Router for special cases (dev endpoints, SSE streams).
	Router chi.Router

	// AuthMW provides role-based middleware helpers.
	AuthMW *middleware.AuthMiddleware

	// APIConfig is the shared Huma API configuration (OpenAPI metadata, security schemes).
	APIConfig huma.Config
}
