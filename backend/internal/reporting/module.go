package reporting

import (
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/reporting/handlers"
	"github.com/orkestra/backend/internal/reporting/repository"
	"github.com/orkestra/backend/internal/reporting/services"
	"github.com/orkestra/backend/internal/shared/module"
)

type ReportingModule struct {
	module.BaseModule
	handler *handlers.DeadlineHandler
}

func NewModule() *ReportingModule { return &ReportingModule{} }

func (m *ReportingModule) Name() string        { return "reporting" }
func (m *ReportingModule) DisplayName() string  { return "Reports & Deadlines" }
func (m *ReportingModule) Description() string  { return "Deadline tracking and analytics" }

func (m *ReportingModule) NavItems() []module.NavItemSpec {
	return []module.NavItemSpec{{
		Group: "Administration", Name: "Deadlines", Icon: "calendar-exclamation",
		Path: "/admin/reports/deadlines", MinRole: "manager", Active: true,
	}}
}

func (m *ReportingModule) Init(deps *module.Dependencies) error {
	repo := repository.NewDeadlineRepository(deps.DB)
	svc := services.NewDeadlineService(repo)
	m.handler = handlers.NewDeadlineHandler(svc)
	return nil
}

func (m *ReportingModule) RegisterRoutes(ri *module.RouteInfo) {
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("manager"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(api, m.handler)
	})
}
