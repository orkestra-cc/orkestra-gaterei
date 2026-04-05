package company

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/company/config"
	"github.com/orkestra/backend/internal/company/handlers"
	"github.com/orkestra/backend/internal/company/repository"
	"github.com/orkestra/backend/internal/company/services"
	sharedConfig "github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type CompanyModule struct {
	handler *handlers.CompanyHandler
}

func NewModule() *CompanyModule {
	return &CompanyModule{}
}

func (m *CompanyModule) Name() string { return "company" }

func (m *CompanyModule) Enabled(cfg *sharedConfig.Config) bool {
	return cfg.Company.BearerToken != ""
}

func (m *CompanyModule) Init(deps *module.Dependencies) error {
	companyCfg := &config.CompanyAPIConfig{
		BaseURL:       deps.Config.Company.BaseURL,
		BearerToken:   deps.Config.Company.BearerToken,
		Timeout:       deps.Config.Company.Timeout,
		RetryAttempts: deps.Config.Company.RetryAttempts,
		CacheTTL:      deps.Config.Company.CacheTTL,
	}

	repo := repository.NewCompanyRepository(deps.DB)
	client := services.NewCompanyAPIClientWithCache(companyCfg, deps.Logger, deps.RedisAdapter)
	svc := services.NewCompanyService(repo, client, deps.Logger)
	m.handler = handlers.NewCompanyHandler(svc)

	deps.Logger.Info("Company lookup module initialized",
		slog.String("baseURL", deps.Config.Company.BaseURL),
	)
	return nil
}

func (m *CompanyModule) RegisterRoutes(ri *module.RouteInfo) {
	// Company routes: manager role and above
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("manager"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(api, m.handler)
	})
}

func (m *CompanyModule) Start(_ context.Context) error      { return nil }
func (m *CompanyModule) Stop(_ context.Context) error       { return nil }
func (m *CompanyModule) HealthCheck(_ context.Context) error { return nil }
