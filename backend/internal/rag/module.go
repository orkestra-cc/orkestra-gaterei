package rag

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	aimodelsSvc "github.com/orkestra/backend/internal/aimodels/services"
	graphRepo "github.com/orkestra/backend/internal/graph/repository"
	"github.com/orkestra/backend/internal/rag/handlers"
	"github.com/orkestra/backend/internal/rag/repository"
	"github.com/orkestra/backend/internal/rag/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type RAGModule struct {
	documentHandler     *handlers.DocumentHandler
	queryHandler        *handlers.QueryHandler
	streamHandler       *handlers.StreamHandler
	relationshipHandler *handlers.RelationshipHandler
}

func NewModule() *RAGModule {
	return &RAGModule{}
}

func (m *RAGModule) Name() string { return "rag" }

func (m *RAGModule) Enabled(cfg *config.Config) bool {
	return cfg.RAG.Enabled
}

func (m *RAGModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config

	documentRepository := repository.NewDocumentRepository(deps.DB)
	relTypeRepo := repository.NewRelationshipTypeRepository(deps.DB)
	relTypeService := services.NewRelationshipTypeService(relTypeRepo, deps.Logger)
	m.relationshipHandler = handlers.NewRelationshipHandler(relTypeService)

	// Use aimodels service if available, otherwise create a standalone model service
	var ragModelProvider services.AIModelProvider
	if svc := deps.Services.Get(module.ServiceAIModelProvider); svc != nil {
		ragModelProvider = svc.(aimodelsSvc.AIModelService)
	} else {
		modelRepository := repository.NewModelRepository(deps.DB)
		localModelService := services.NewModelService(modelRepository, cfg.RAG, deps.Logger)
		if err := localModelService.SeedDefaults(context.Background()); err != nil {
			deps.Logger.Warn("Failed to seed default RAG models", slog.String("error", err.Error()))
		}
		ragModelProvider = localModelService
	}

	// Text extractor using Gotenberg
	textExtractor := services.NewTextExtractor(cfg.Documents.GotenbergURL)

	// Ingestion + query services require graph repository
	if graphSvc := deps.Services.Get(module.ServiceGraphRepo); graphSvc != nil {
		graphRepository := graphSvc.(graphRepo.GraphRepository)

		ingestionService := services.NewIngestionService(
			documentRepository, relTypeRepo, graphRepository, ragModelProvider, textExtractor,
			cfg.RAG.ChunkSize, cfg.RAG.ChunkOverlap, deps.Logger,
		)
		m.documentHandler = handlers.NewDocumentHandler(ingestionService)

		ragQueryService := services.NewQueryService(graphRepository, ragModelProvider, cfg.RAG.DefaultTopK, deps.Logger)
		m.queryHandler = handlers.NewQueryHandler(ragQueryService)
		m.streamHandler = handlers.NewStreamHandler(ragQueryService, deps.Logger)

		// Register RAGQueryService for agents module consumption
		deps.Services.Register(module.ServiceRAGQuery, ragQueryService)
	}

	deps.Logger.Info("RAG module initialized")
	return nil
}

func (m *RAGModule) RegisterRoutes(ri *module.RouteInfo) {
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("administrator"))
		api := humachi.New(r, ri.APIConfig)
		if m.documentHandler != nil {
			RegisterDocumentRoutes(api, m.documentHandler)
		}
		if m.queryHandler != nil {
			RegisterQueryRoutes(api, m.queryHandler)
		}
		if m.streamHandler != nil {
			RegisterStreamRoute(r, m.streamHandler)
		}
		if m.relationshipHandler != nil {
			RegisterRelationshipTypeRoutes(api, m.relationshipHandler)
		}
	})
}

func (m *RAGModule) Start(_ context.Context) error      { return nil }
func (m *RAGModule) Stop(_ context.Context) error       { return nil }
func (m *RAGModule) HealthCheck(_ context.Context) error { return nil }
