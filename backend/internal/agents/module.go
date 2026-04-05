package agents

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/agents/handlers"
	"github.com/orkestra/backend/internal/agents/repository"
	"github.com/orkestra/backend/internal/agents/services"
	ragSvc "github.com/orkestra/backend/internal/rag/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type AgentsModule struct {
	projectHandler       *handlers.ProjectHandler
	agentHandler         *handlers.AgentHandler
	personalAgentHandler *handlers.PersonalAgentHandler
}

func NewModule() *AgentsModule {
	return &AgentsModule{}
}

func (m *AgentsModule) Name() string { return "agents" }

func (m *AgentsModule) Enabled(cfg *config.Config) bool {
	return cfg.Agents.Enabled
}

func (m *AgentsModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config

	projectRepo := repository.NewProjectRepository(deps.DB)
	conversationRepo := repository.NewConversationRepository(deps.DB)

	hsClient := services.NewHindsightClient(cfg.Agents.HindsightURL, deps.Logger)

	// Create RAG bridge if RAG query service is available
	var ragBridge services.RAGBridge
	if svc := deps.Services.Get(module.ServiceRAGQuery); svc != nil {
		ragQueryService := svc.(ragSvc.QueryService)
		ragBridge = services.NewRAGBridge(ragQueryService, cfg.RAG.DefaultTopK, deps.Logger)
	}

	projectService := services.NewProjectService(projectRepo, hsClient, cfg.Agents.HindsightNamespace, deps.Logger)
	agentService := services.NewAgentService(projectRepo, conversationRepo, hsClient, ragBridge, deps.Logger)

	m.projectHandler = handlers.NewProjectHandler(projectService)
	m.agentHandler = handlers.NewAgentHandler(agentService)

	personalAgentService := services.NewPersonalAgentService(projectRepo, agentService, hsClient, cfg.Agents.HindsightNamespace, deps.Logger)
	m.personalAgentHandler = handlers.NewPersonalAgentHandler(personalAgentService)

	deps.Logger.Info("Agents module initialized",
		slog.String("hindsightURL", cfg.Agents.HindsightURL),
	)
	return nil
}

func (m *AgentsModule) RegisterRoutes(ri *module.RouteInfo) {
	// Personal agent: any authenticated user (guest and above)
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("guest"))
		api := humachi.New(r, ri.APIConfig)
		RegisterPersonalAgentRoutes(api, m.personalAgentHandler)
	})

	// Project management: manager role and above
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("manager"))
		api := humachi.New(r, ri.APIConfig)
		RegisterProjectRoutes(api, m.projectHandler)
	})

	// Agent querying and conversations: operator role and above
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("operator"))
		api := humachi.New(r, ri.APIConfig)
		RegisterQueryRoutes(api, m.agentHandler)
	})

	// Agent admin: administrator role and above
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("administrator"))
		api := humachi.New(r, ri.APIConfig)
		RegisterAdminRoutes(api, m.agentHandler)
	})
}

func (m *AgentsModule) Start(_ context.Context) error      { return nil }
func (m *AgentsModule) Stop(_ context.Context) error       { return nil }
func (m *AgentsModule) HealthCheck(_ context.Context) error { return nil }
