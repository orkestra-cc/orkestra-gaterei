package navigation

import (
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/orkestra/backend/internal/navigation/config"
	"github.com/orkestra/backend/internal/navigation/handlers"
	"github.com/orkestra/backend/internal/navigation/services"
	"github.com/orkestra/backend/internal/shared/module"
)

type NavigationModule struct {
	module.BaseModule
	handler *handlers.NavigationHandler
}

func NewModule() *NavigationModule { return &NavigationModule{} }

func (m *NavigationModule) Name() string        { return "navigation" }
func (m *NavigationModule) DisplayName() string  { return "Navigation" }
func (m *NavigationModule) Description() string  { return "Dynamic navigation menu aggregator" }

func (m *NavigationModule) Init(_ *module.Dependencies) error {
	menuConfig := config.NewMenuConfig()
	svc := services.NewNavigationService(menuConfig)
	m.handler = handlers.NewNavigationHandler(svc)
	return nil
}

func (m *NavigationModule) RegisterRoutes(ri *module.RouteInfo) {
	api := humachi.New(ri.ProtectedRouter, ri.APIConfig)
	RegisterRoutes(api, m.handler)
}
