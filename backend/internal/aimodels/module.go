package aimodels

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/aimodels/handlers"
	"github.com/orkestra/backend/internal/aimodels/repository"
	"github.com/orkestra/backend/internal/aimodels/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type AIModelsModule struct {
	handler *handlers.ModelHandler
	service services.AIModelService
}

func NewModule() *AIModelsModule {
	return &AIModelsModule{}
}

func (m *AIModelsModule) Name() string { return "aimodels" }

func (m *AIModelsModule) Enabled(cfg *config.Config) bool {
	return cfg.AIModels.Enabled
}

func (m *AIModelsModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config

	repo := repository.NewModelRepository(deps.DB)
	m.service = services.NewModelService(repo, services.AIModelsConfig{
		OllamaBaseURL: cfg.AIModels.OllamaBaseURL,
		OpenAIAPIKey:  cfg.AIModels.OpenAIAPIKey,
		AnthropicKey:  cfg.AIModels.AnthropicKey,
		GeminiKey:     cfg.AIModels.GeminiKey,
	}, deps.Logger)

	m.handler = handlers.NewModelHandler(m.service)

	// Register AIModelService for RAG and Sales module consumption
	deps.Services.Register(module.ServiceAIModelProvider, m.service)

	deps.Logger.Info("AI Models module initialized")
	return nil
}

func (m *AIModelsModule) RegisterRoutes(ri *module.RouteInfo) {
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("administrator"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(api, m.handler)
	})
}

func (m *AIModelsModule) Start(ctx context.Context) error {
	// Seed default models on startup
	if m.service != nil {
		if err := m.service.SeedDefaults(ctx); err != nil {
			slog.Warn("Failed to seed default AI models", slog.String("error", err.Error()))
		}
	}
	return nil
}

func (m *AIModelsModule) Stop(_ context.Context) error       { return nil }
func (m *AIModelsModule) HealthCheck(_ context.Context) error { return nil }
