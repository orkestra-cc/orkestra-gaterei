package module

import (
	"context"
	"log/slog"

	"github.com/orkestra/backend/internal/shared/database"
	"go.mongodb.org/mongo-driver/mongo"
)

// ModuleConfigService manages module configurations in MongoDB with Redis caching.
// It provides the hot-path IsEnabled() check used by the ModuleGate middleware.
type ModuleConfigService struct {
	db     *mongo.Database
	redis  *database.RedisClientAdapter
	logger *slog.Logger
}

// NewModuleConfigService creates a new config service.
func NewModuleConfigService(db *mongo.Database, redis *database.RedisClientAdapter, logger *slog.Logger) *ModuleConfigService {
	return &ModuleConfigService{db: db, redis: redis, logger: logger}
}

// IsEnabled checks if a module is enabled. Hot path: Redis cache (30s TTL) → MongoDB fallback.
// Core modules always return true without DB check.
func (s *ModuleConfigService) IsEnabled(ctx context.Context, moduleName string) bool {
	// TODO: Phase 2 — implement Redis cache + MongoDB lookup
	// For now, return true (all modules enabled by default)
	return true
}

// SeedFromModules populates the module_configs collection on first boot
// using each module's ConfigSchema() and current env var values.
func (s *ModuleConfigService) SeedFromModules(ctx context.Context, modules []Module) error {
	// TODO: Phase 2 — implement seeding logic
	return nil
}
