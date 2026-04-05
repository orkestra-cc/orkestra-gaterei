package module

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/orkestra/backend/internal/shared/config"
)

// ModuleRegistry manages the lifecycle of all application modules.
// Modules are processed in registration order, so producers must be
// registered before their consumers.
type ModuleRegistry struct {
	modules []Module
	enabled []Module // populated by InitAll
	logger  *slog.Logger
}

// NewModuleRegistry creates a new registry.
func NewModuleRegistry(logger *slog.Logger) *ModuleRegistry {
	return &ModuleRegistry{
		logger: logger,
	}
}

// Register adds a module to the registry. Registration order determines
// initialization order — register producers before consumers.
func (r *ModuleRegistry) Register(m Module) {
	r.modules = append(r.modules, m)
}

// InitAll initializes all enabled modules in registration order.
// Disabled modules are skipped and logged as warnings.
func (r *ModuleRegistry) InitAll(cfg *config.Config, deps *Dependencies) error {
	for _, m := range r.modules {
		if !m.Enabled(cfg) {
			r.logger.Warn(fmt.Sprintf("Module %s disabled", m.Name()))
			continue
		}

		if err := m.Init(deps); err != nil {
			return fmt.Errorf("module %s init: %w", m.Name(), err)
		}

		r.enabled = append(r.enabled, m)
		r.logger.Info(fmt.Sprintf("Module %s initialized", m.Name()))
	}
	return nil
}

// RegisterAllRoutes calls RegisterRoutes on every enabled module.
func (r *ModuleRegistry) RegisterAllRoutes(ri *RouteInfo) {
	for _, m := range r.enabled {
		m.RegisterRoutes(ri)
	}
}

// StartAll starts background jobs for every enabled module.
func (r *ModuleRegistry) StartAll(ctx context.Context) error {
	for _, m := range r.enabled {
		if err := m.Start(ctx); err != nil {
			return fmt.Errorf("module %s start: %w", m.Name(), err)
		}
	}
	return nil
}

// StopAll shuts down all enabled modules in reverse registration order.
func (r *ModuleRegistry) StopAll(ctx context.Context) {
	for i := len(r.enabled) - 1; i >= 0; i-- {
		m := r.enabled[i]
		if err := m.Stop(ctx); err != nil {
			r.logger.Error(fmt.Sprintf("Module %s stop failed", m.Name()),
				slog.String("error", err.Error()))
		}
	}
}

// HealthCheckAll runs health checks on all enabled modules.
// Returns the first error encountered.
func (r *ModuleRegistry) HealthCheckAll(ctx context.Context) error {
	for _, m := range r.enabled {
		if err := m.HealthCheck(ctx); err != nil {
			return fmt.Errorf("module %s health: %w", m.Name(), err)
		}
	}
	return nil
}

// EnabledModules returns the names of all enabled modules.
func (r *ModuleRegistry) EnabledModules() []string {
	names := make([]string, len(r.enabled))
	for i, m := range r.enabled {
		names[i] = m.Name()
	}
	return names
}
