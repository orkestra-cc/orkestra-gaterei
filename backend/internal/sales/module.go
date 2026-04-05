package sales

import (
	"context"
	"log/slog"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	aimodelsSvc "github.com/orkestra/backend/internal/aimodels/services"
	"github.com/orkestra/backend/internal/sales/handlers"
	"github.com/orkestra/backend/internal/sales/repository"
	"github.com/orkestra/backend/internal/sales/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type SalesModule struct {
	skillHandler    *handlers.SkillHandler
	prospectHandler *handlers.ProspectHandler
	jobHandler      *handlers.JobHandler
	reportHandler   *handlers.ReportHandler
	promptHandler   *handlers.PromptHandler
	settingsHandler *handlers.SettingsHandler
	batchPoller     *services.BatchPoller
}

func NewModule() *SalesModule {
	return &SalesModule{}
}

func (m *SalesModule) Name() string { return "sales" }

func (m *SalesModule) Enabled(cfg *config.Config) bool {
	return cfg.Sales.Enabled
}

func (m *SalesModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config

	salesJobRepo := repository.NewJobRepository(deps.DB)

	// Use AI model provider if available
	var salesModelProvider services.AIModelProvider
	if svc := deps.Services.Get(module.ServiceAIModelProvider); svc != nil {
		salesModelProvider = svc.(aimodelsSvc.AIModelService)
	}

	// Optional: company enrichment for Italian business registry
	var salesEnrichment services.CompanyEnrichmentService
	// TODO: wire companyService adapter when ready

	salesPromptRepo := repository.NewPromptRepository(deps.DB)
	promptLoader := services.NewPromptLoader(salesPromptRepo, deps.Logger)

	// Seed default prompts from embedded files on first run
	if err := promptLoader.SeedDefaults(context.Background()); err != nil {
		deps.Logger.Warn("Failed to seed sales prompts", slog.String("error", err.Error()))
	}
	scraper := services.NewScraper(cfg.Sales, deps.Logger)
	agentExecutor := services.NewAgentExecutor(cfg.Sales.MaxConcurrency, deps.Logger)
	scorer := services.NewScorer()

	salesSettingsRepo := repository.NewSettingsRepository(deps.DB)
	salesReportRepo := repository.NewReportRepository(deps.DB)
	reportGen := services.NewReportGenerator(salesReportRepo, deps.Logger)

	salesBatchRepo := repository.NewBatchRepository(deps.DB)

	orchestrator := services.NewOrchestrator(
		salesJobRepo, salesReportRepo, salesSettingsRepo, salesBatchRepo, salesModelProvider, promptLoader,
		scraper, agentExecutor, scorer, salesEnrichment, reportGen,
		cfg.Sales, deps.Logger,
	)

	// Create batch poller for async LLM batch results
	m.batchPoller = services.NewBatchPoller(
		salesBatchRepo, salesJobRepo, salesReportRepo,
		salesModelProvider, scorer, reportGen,
		30*time.Second, deps.Logger,
	)

	skillStore := services.NewSkillStore()
	m.skillHandler = handlers.NewSkillHandler(orchestrator, skillStore, deps.Logger)
	m.prospectHandler = handlers.NewProspectHandler(orchestrator)
	m.jobHandler = handlers.NewJobHandler(orchestrator)
	m.reportHandler = handlers.NewReportHandler(salesReportRepo, salesJobRepo, reportGen)
	m.promptHandler = handlers.NewPromptHandler(salesPromptRepo, promptLoader)
	m.settingsHandler = handlers.NewSettingsHandler(salesSettingsRepo)

	// Mark incomplete jobs from previous runs as failed
	if err := salesJobRepo.MarkStaleJobsFailed(context.Background()); err != nil {
		deps.Logger.Warn("Failed to mark stale sales jobs", slog.String("error", err.Error()))
	}

	deps.Logger.Info("Sales Intelligence module initialized")
	return nil
}

func (m *SalesModule) RegisterRoutes(ri *module.RouteInfo) {
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("manager"))
		api := humachi.New(r, ri.APIConfig)
		RegisterSkillRoutes(api, m.skillHandler)
		RegisterProspectRoutes(api, m.prospectHandler)
		RegisterJobRoutes(api, m.jobHandler)
		RegisterReportRoutes(api, m.reportHandler)
		RegisterPromptRoutes(api, m.promptHandler)
		RegisterReportDownloadRoute(r, m.reportHandler)
		RegisterSettingsRoutes(api, m.settingsHandler)
	})
}

func (m *SalesModule) Start(_ context.Context) error {
	if m.batchPoller != nil {
		m.batchPoller.Start()
	}
	return nil
}

func (m *SalesModule) Stop(_ context.Context) error {
	// BatchPoller doesn't have an explicit Stop — it runs until context cancellation
	return nil
}

func (m *SalesModule) HealthCheck(_ context.Context) error { return nil }
