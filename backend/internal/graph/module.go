package graph

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/graph/handlers"
	"github.com/orkestra/backend/internal/graph/repository"
	"github.com/orkestra/backend/internal/graph/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/database"
	"github.com/orkestra/backend/internal/shared/module"
)

type GraphModule struct {
	handler    *handlers.GraphHandler
	graphRepo  repository.GraphRepository
	disconnect func(ctx context.Context) error
}

func NewModule() *GraphModule {
	return &GraphModule{}
}

func (m *GraphModule) Name() string { return "graph" }

func (m *GraphModule) Enabled(cfg *config.Config) bool {
	return cfg.Graph.Enabled
}

func (m *GraphModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config

	graphDriver, err := database.NewGraphConnection(context.Background(), database.GraphDBConfig{
		URI:         cfg.Graph.URI,
		Username:    cfg.Graph.Username,
		Password:    cfg.Graph.Password,
		Database:    cfg.Graph.Database,
		MaxConnPool: cfg.Graph.MaxConnPool,
	})
	if err != nil {
		return err
	}
	m.disconnect = func(ctx context.Context) error {
		database.DisconnectGraph(ctx, graphDriver)
		return nil
	}

	m.graphRepo = repository.NewGraphRepository(graphDriver, cfg.Graph.Database)
	graphService := services.NewGraphService(m.graphRepo, deps.Logger)
	algorithmService := services.NewAlgorithmService(m.graphRepo, deps.Logger)
	vectorService := services.NewVectorService(m.graphRepo, deps.Logger)
	m.handler = handlers.NewGraphHandler(graphService, algorithmService, vectorService, cfg.Graph.URI)

	// Register GraphRepository for RAG module consumption
	deps.Services.Register(module.ServiceGraphRepo, m.graphRepo)

	deps.Logger.Info("Graph database module initialized",
		slog.String("uri", cfg.Graph.URI),
		slog.String("database", cfg.Graph.Database),
	)
	return nil
}

func (m *GraphModule) RegisterRoutes(ri *module.RouteInfo) {
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("administrator"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(api, m.handler)
	})
}

func (m *GraphModule) Start(_ context.Context) error      { return nil }
func (m *GraphModule) Stop(ctx context.Context) error {
	if m.disconnect != nil {
		return m.disconnect(ctx)
	}
	return nil
}
func (m *GraphModule) HealthCheck(_ context.Context) error { return nil }
