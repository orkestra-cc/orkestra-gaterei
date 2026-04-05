package documents

import (
	"context"
	"log/slog"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/documents/config"
	"github.com/orkestra/backend/internal/documents/handlers"
	"github.com/orkestra/backend/internal/documents/repository"
	"github.com/orkestra/backend/internal/documents/services"
	sharedConfig "github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type DocumentsModule struct {
	templateHandler *handlers.TemplateHandler
	documentHandler *handlers.DocumentHandler
	templateSvc     services.TemplateService
}

func NewModule() *DocumentsModule {
	return &DocumentsModule{}
}

func (m *DocumentsModule) Name() string { return "documents" }

func (m *DocumentsModule) Enabled(cfg *sharedConfig.Config) bool {
	return cfg.Documents.GotenbergURL != ""
}

func (m *DocumentsModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config

	docsConfig := &config.Config{
		GotenbergURL:  cfg.Documents.GotenbergURL,
		Timeout:       cfg.Documents.Timeout,
		RetryAttempts: cfg.Documents.RetryAttempts,
		DefaultMargins: config.PDFMargins{
			Top:    cfg.Documents.DefaultMargins.Top,
			Bottom: cfg.Documents.DefaultMargins.Bottom,
			Left:   cfg.Documents.DefaultMargins.Left,
			Right:  cfg.Documents.DefaultMargins.Right,
		},
	}

	templateRepo := repository.NewTemplateRepository(deps.DB)
	documentRepo := repository.NewDocumentRepository(deps.DB)

	gotenbergClient := services.NewGotenbergClient(docsConfig, deps.Logger)
	templateEngine := services.NewTemplateEngine()
	m.templateSvc = services.NewTemplateService(templateRepo, templateEngine, deps.Logger)
	pdfSvc := services.NewPDFService(templateRepo, documentRepo, gotenbergClient, templateEngine, deps.Logger)

	m.templateHandler = handlers.NewTemplateHandler(m.templateSvc)
	m.documentHandler = handlers.NewDocumentHandler(pdfSvc)

	// Register PDFService for billing module consumption
	deps.Services.Register(module.ServicePDFService, pdfSvc)

	deps.Logger.Info("Documents module initialized",
		slog.String("gotenbergURL", cfg.Documents.GotenbergURL),
	)
	return nil
}

func (m *DocumentsModule) RegisterRoutes(ri *module.RouteInfo) {
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("manager"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(api, m.templateHandler, m.documentHandler)
	})
}

func (m *DocumentsModule) Start(ctx context.Context) error {
	// Seed built-in templates on startup
	if m.templateSvc != nil {
		if err := m.templateSvc.SeedBuiltInTemplates(ctx); err != nil {
			// Non-fatal: log warning and continue
			slog.Warn("Failed to seed built-in templates", slog.String("error", err.Error()))
		}
	}
	return nil
}

func (m *DocumentsModule) Stop(_ context.Context) error       { return nil }
func (m *DocumentsModule) HealthCheck(_ context.Context) error { return nil }
