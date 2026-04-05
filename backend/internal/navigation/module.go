package navigation

import (
	"context"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/orkestra/backend/internal/navigation/config"
	"github.com/orkestra/backend/internal/navigation/handlers"
	"github.com/orkestra/backend/internal/navigation/services"
	"github.com/orkestra/backend/internal/shared/module"
	sharedConfig "github.com/orkestra/backend/internal/shared/config"
)

type NavigationModule struct {
	handler *handlers.NavigationHandler
}

func NewModule() *NavigationModule {
	return &NavigationModule{}
}

func (m *NavigationModule) Name() string { return "navigation" }

func (m *NavigationModule) Enabled(_ *sharedConfig.Config) bool { return true }

func (m *NavigationModule) Init(_ *module.Dependencies) error {
	menuConfig := config.NewMenuConfig()
	svc := services.NewNavigationService(menuConfig)
	m.handler = handlers.NewNavigationHandler(svc)
	return nil
}

func (m *NavigationModule) RegisterRoutes(ri *module.RouteInfo) {
	// Navigation accessible to all authenticated users
	// Role filtering happens inside the service based on user's role
	api := humachi.New(ri.ProtectedRouter, ri.APIConfig)
	RegisterRoutes(api, m.handler)
}

func (m *NavigationModule) Start(_ context.Context) error   { return nil }
func (m *NavigationModule) Stop(_ context.Context) error    { return nil }
func (m *NavigationModule) HealthCheck(_ context.Context) error { return nil }
