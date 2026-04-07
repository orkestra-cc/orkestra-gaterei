package module

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/database"
	"github.com/orkestra/backend/internal/shared/utils"
)

// ModuleConfigService manages module configurations in MongoDB with Redis caching.
// It provides the hot-path IsEnabled() check used by the ModuleGate middleware.
type ModuleConfigService struct {
	repo        *ModuleConfigRepository
	redis       *database.RedisClientAdapter
	logger      *slog.Logger
	coreModules map[string]bool // precomputed set — never hits DB/Redis
}

const (
	enabledCachePrefix = "module:enabled:"
	enabledCacheTTL    = 30 * time.Second
)

// NewModuleConfigService creates a new config service.
func NewModuleConfigService(repo *ModuleConfigRepository, redis *database.RedisClientAdapter, logger *slog.Logger) *ModuleConfigService {
	return &ModuleConfigService{
		repo:        repo,
		redis:       redis,
		logger:      logger,
		coreModules: make(map[string]bool),
	}
}

// SetCoreModules builds the set of core module names from registered modules.
// Core modules always return true from IsEnabled without any DB/Redis check.
func (s *ModuleConfigService) SetCoreModules(modules []Module) {
	for _, m := range modules {
		if m.Category() == CategoryCore {
			s.coreModules[m.Name()] = true
		}
	}
}

// IsEnabled checks if a module is enabled. This is the hot path called on every
// HTTP request to gated modules.
//
// Lookup order: core map → Redis cache (30s TTL) → MongoDB → fail-open (true).
func (s *ModuleConfigService) IsEnabled(ctx context.Context, moduleName string) bool {
	// Core modules are always enabled — zero I/O.
	if s.coreModules[moduleName] {
		return true
	}

	// Try Redis cache first.
	cacheKey := enabledCachePrefix + moduleName
	cached, err := s.redis.Get(ctx, cacheKey)
	if err == nil && cached != "" {
		return cached == "true"
	}

	// Cache miss — fall through to MongoDB.
	doc, err := s.repo.FindByName(ctx, moduleName)
	if err != nil {
		// Fail-open: if DB is unreachable, don't break running modules.
		s.logger.Warn("IsEnabled: MongoDB lookup failed, assuming enabled",
			slog.String("module", moduleName),
			slog.String("error", err.Error()),
		)
		return true
	}

	// No document yet (first boot, before seeding) — assume enabled.
	if doc == nil {
		return true
	}

	// Cache the result in Redis.
	enabledStr := "false"
	if doc.Enabled {
		enabledStr = "true"
	}
	if err := s.redis.Set(ctx, cacheKey, enabledStr, enabledCacheTTL); err != nil {
		s.logger.Warn("IsEnabled: failed to cache result",
			slog.String("module", moduleName),
			slog.String("error", err.Error()),
		)
	}

	return doc.Enabled
}

// SeedFromModules populates the module_configs collection on first boot
// using each module's ConfigSchema() and current env var values.
// On subsequent boots, it only updates the schema (preserving admin-changed values).
func (s *ModuleConfigService) SeedFromModules(ctx context.Context, modules []Module, cfg *config.Config) error {
	// Build core modules map as a side effect of seeding.
	s.SetCoreModules(modules)

	for _, m := range modules {
		existing, err := s.repo.FindByName(ctx, m.Name())
		if err != nil {
			s.logger.Error("SeedFromModules: failed to check existing config",
				slog.String("module", m.Name()),
				slog.String("error", err.Error()),
			)
			continue
		}

		if existing != nil {
			// Module already has a config document — only update the schema
			// in case fields were added/removed between versions.
			if err := s.repo.UpdateSchema(ctx, m.Name(), m.ConfigSchema()); err != nil {
				s.logger.Error("SeedFromModules: failed to update schema",
					slog.String("module", m.Name()),
					slog.String("error", err.Error()),
				)
			}
			continue
		}

		// First boot for this module — create the config document.
		doc := s.buildInitialConfig(m, cfg)
		if err := s.repo.Upsert(ctx, &doc); err != nil {
			s.logger.Error("SeedFromModules: failed to seed module config",
				slog.String("module", m.Name()),
				slog.String("error", err.Error()),
			)
			continue
		}

		s.logger.Info("Module config seeded",
			slog.String("module", m.Name()),
			slog.String("category", string(m.Category())),
		)
	}

	return nil
}

// buildInitialConfig constructs a ModuleConfig from a module's declarations and current env vars.
func (s *ModuleConfigService) buildInitialConfig(m Module, cfg *config.Config) ModuleConfig {
	configValues := make(map[string]string)
	encryptedValues := make(map[string]string)

	for _, field := range m.ConfigSchema() {
		value := ""
		if field.EnvVar != "" {
			value = os.Getenv(field.EnvVar)
		}
		if value == "" {
			value = field.Default
		}
		if value == "" {
			continue
		}

		if field.Type == FieldSecret {
			encrypted, err := utils.EncryptOAuthToken(value)
			if err != nil {
				s.logger.Warn("SeedFromModules: failed to encrypt secret, storing empty",
					slog.String("module", m.Name()),
					slog.String("field", field.Key),
					slog.String("error", err.Error()),
				)
				continue
			}
			encryptedValues[field.Key] = encrypted
		} else {
			configValues[field.Key] = value
		}
	}

	return ModuleConfig{
		ModuleName:      m.Name(),
		DisplayName:     m.DisplayName(),
		Description:     m.Description(),
		Category:        m.Category(),
		Enabled:         m.Enabled(cfg),
		ConfigValues:    configValues,
		EncryptedValues: encryptedValues,
		ConfigSchema:    m.ConfigSchema(),
		DependsOn:       m.Dependencies(),
	}
}

// GetConfig retrieves the full config document for a module. Used by admin API.
func (s *ModuleConfigService) GetConfig(ctx context.Context, name string) (*ModuleConfig, error) {
	return s.repo.FindByName(ctx, name)
}

// GetAllConfigs retrieves all module config documents. Used by admin API list endpoint.
func (s *ModuleConfigService) GetAllConfigs(ctx context.Context) ([]ModuleConfig, error) {
	return s.repo.FindAll(ctx)
}

// UpdateConfig updates a module's config values and encrypted secrets,
// then invalidates the Redis cache for immediate propagation.
func (s *ModuleConfigService) UpdateConfig(ctx context.Context, name string, values map[string]string, secrets map[string]string) error {
	encrypted := make(map[string]string, len(secrets))
	for k, v := range secrets {
		enc, err := utils.EncryptOAuthToken(v)
		if err != nil {
			return fmt.Errorf("encrypt secret %q: %w", k, err)
		}
		encrypted[k] = enc
	}

	if err := s.repo.UpdateConfigValues(ctx, name, values, encrypted); err != nil {
		return err
	}
	return s.InvalidateCache(ctx, name)
}

// UpdateEnabled toggles a module's enabled state and invalidates the cache.
func (s *ModuleConfigService) UpdateEnabled(ctx context.Context, name string, enabled bool) error {
	if s.coreModules[name] {
		return fmt.Errorf("cannot disable core module %q", name)
	}
	if err := s.repo.UpdateEnabled(ctx, name, enabled); err != nil {
		return err
	}
	return s.InvalidateCache(ctx, name)
}

// InvalidateCache removes the Redis cached enabled state for a module,
// forcing the next IsEnabled call to re-fetch from MongoDB.
func (s *ModuleConfigService) InvalidateCache(ctx context.Context, name string) error {
	return s.redis.Del(ctx, enabledCachePrefix+name)
}

// ModuleStatusInfo provides a summary for the admin API list endpoint.
type ModuleStatusInfo struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName"`
	Description string         `json:"description"`
	Category    ModuleCategory `json:"category"`
	Enabled     bool           `json:"enabled"`
	DependsOn   []string       `json:"dependsOn,omitempty"`
	HasConfig   bool           `json:"hasConfig"`
	FieldCount  int            `json:"fieldCount"`
}

// ModuleStatusJSON returns a JSON-serializable summary of all module configs.
// Used by health/status endpoints.
func (s *ModuleConfigService) ModuleStatusJSON(ctx context.Context) ([]byte, error) {
	configs, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}

	infos := make([]ModuleStatusInfo, len(configs))
	for i, c := range configs {
		infos[i] = ModuleStatusInfo{
			Name:        c.ModuleName,
			DisplayName: c.DisplayName,
			Description: c.Description,
			Category:    c.Category,
			Enabled:     c.Enabled,
			DependsOn:   c.DependsOn,
			HasConfig:   len(c.ConfigValues) > 0 || len(c.EncryptedValues) > 0,
			FieldCount:  len(c.ConfigSchema),
		}
	}
	return json.Marshal(infos)
}
