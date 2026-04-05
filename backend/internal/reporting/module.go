package reporting

import (
	"context"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/reporting/handlers"
	"github.com/orkestra/backend/internal/reporting/repository"
	"github.com/orkestra/backend/internal/reporting/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type ReportingModule struct {
	handler *handlers.DeadlineHandler
}

func NewModule() *ReportingModule {
	return &ReportingModule{}
}

func (m *ReportingModule) Name() string { return "reporting" }

func (m *ReportingModule) Enabled(_ *config.Config) bool { return true }

func (m *ReportingModule) Init(deps *module.Dependencies) error {
	repo := repository.NewDeadlineRepository(deps.DB)
	svc := services.NewDeadlineService(repo)
	m.handler = handlers.NewDeadlineHandler(svc)
	return nil
}

func (m *ReportingModule) RegisterRoutes(ri *module.RouteInfo) {
	// Reporting routes: manager role and above
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("manager"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(api, m.handler)
	})
}

func (m *ReportingModule) Start(_ context.Context) error      { return nil }
func (m *ReportingModule) Stop(_ context.Context) error       { return nil }
func (m *ReportingModule) HealthCheck(_ context.Context) error { return nil }
