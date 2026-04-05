package billing

import (
	"context"
	"log/slog"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	billingConfig "github.com/orkestra/backend/internal/billing/config"
	"github.com/orkestra/backend/internal/billing/handlers"
	"github.com/orkestra/backend/internal/billing/jobs"
	"github.com/orkestra/backend/internal/billing/repository"
	"github.com/orkestra/backend/internal/billing/services"
	documentsSvc "github.com/orkestra/backend/internal/documents/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/module"
)

type BillingModule struct {
	module.BaseModule
	invoiceHandler          *handlers.InvoiceHandler
	customerHandler         *handlers.CustomerHandler
	supplierHandler         *handlers.SupplierHandler
	companyHandler          *handlers.CompanyHandler
	notificationHandler     *handlers.NotificationHandler
	businessRegistryHandler *handlers.BusinessRegistryHandler
	syncHandler             *handlers.SyncHandler
	webhookHandler          *handlers.WebhookHandler

	pollingJob    *jobs.PollingJob
	openAPIClient services.OpenAPIClient
	cfg           *config.Config
	logger        *slog.Logger
}

func NewModule() *BillingModule { return &BillingModule{} }

func (m *BillingModule) Name() string                   { return "billing" }
func (m *BillingModule) DisplayName() string             { return "Fatturazione Elettronica" }
func (m *BillingModule) Description() string             { return "Italian electronic invoicing via FatturaPA/SDI" }
func (m *BillingModule) Category() module.ModuleCategory { return module.CategoryExternal }
func (m *BillingModule) Enabled(cfg *config.Config) bool { return cfg.Billing.OpenAPIBearerToken != "" }
func (m *BillingModule) Dependencies() []string          { return []string{"documents"} }
func (m *BillingModule) OptionalServices() []module.ServiceKey {
	return []module.ServiceKey{module.ServicePDFService}
}

func (m *BillingModule) ConfigSchema() []module.ConfigField {
	return []module.ConfigField{
		{Key: "bearerToken", Label: "OpenAPI Bearer Token", Type: module.FieldSecret, Required: true, EnvVar: "OPENAPI_BILLING_BEARER_TOKEN"},
		{Key: "baseURL", Label: "API Base URL", Type: module.FieldString, Default: "https://test.sdi.openapi.it", EnvVar: "OPENAPI_BILLING_BASE_URL"},
		{Key: "fiscalID", Label: "Codice Fiscale", Type: module.FieldString, Required: true, EnvVar: "OPENAPI_BILLING_FISCAL_ID"},
		{Key: "recipientCode", Label: "Codice Destinatario", Type: module.FieldString, EnvVar: "OPENAPI_BILLING_RECIPIENT_CODE"},
		{Key: "sandboxMode", Label: "Sandbox Mode", Type: module.FieldBool, Default: "true", EnvVar: "OPENAPI_SANDBOX_MODE"},
		{Key: "pollingEnabled", Label: "Enable SDI Polling", Type: module.FieldBool, Default: "true"},
		{Key: "pollingInterval", Label: "Polling Interval", Type: module.FieldDuration, Default: "12h"},
		{Key: "webhookURL", Label: "Webhook URL", Type: module.FieldString, EnvVar: "BILLING_WEBHOOK_URL"},
		{Key: "webhookSecret", Label: "Webhook Secret", Type: module.FieldSecret, EnvVar: "BILLING_WEBHOOK_SECRET"},
	}
}

func (m *BillingModule) Collections() []module.CollectionSpec {
	return []module.CollectionSpec{
		{Name: "invoices", Indexes: []module.IndexSpec{{Keys: map[string]int{"uuid": 1}, Unique: true}}},
		{Name: "billing_customers", Indexes: []module.IndexSpec{{Keys: map[string]int{"uuid": 1}, Unique: true}}},
		{Name: "billing_suppliers", Indexes: []module.IndexSpec{{Keys: map[string]int{"uuid": 1}, Unique: true}}},
		{Name: "billing_companies", Indexes: []module.IndexSpec{{Keys: map[string]int{"uuid": 1}, Unique: true}}},
		{Name: "billing_notifications"},
		{Name: "billing_polling_state"},
	}
}

func (m *BillingModule) NavItems() []module.NavItemSpec {
	return []module.NavItemSpec{{
		Group: "Administration", Name: "Fatturazione", Icon: "file-invoice", Path: "/billing",
		MinRole: "manager", Active: true,
		Children: []module.NavItemSpec{
			{Name: "Dashboard", Icon: "chart-pie", Path: "/billing/dashboard", Active: true},
			{Name: "Fatture Emesse", Icon: "paper-plane", Path: "/billing/invoices/issued", Active: true},
			{Name: "Fatture Ricevute", Icon: "inbox", Path: "/billing/invoices/received", Active: true},
			{Name: "Clienti", Icon: "users", Path: "/billing/customers", Active: true},
			{Name: "Fornitori", Icon: "truck", Path: "/billing/suppliers", Active: true},
			{Name: "Notifiche SDI", Icon: "bell", Path: "/billing/notifications", Active: true},
		},
	}}
}

func (m *BillingModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config
	m.cfg = cfg
	m.logger = deps.Logger

	openAPIConfig := &billingConfig.OpenAPIConfig{
		BaseURL:        cfg.Billing.OpenAPIBaseURL,
		BearerToken:    cfg.Billing.OpenAPIBearerToken,
		FiscalID:       cfg.Billing.OpenAPIFiscalID,
		RecipientCode:  cfg.Billing.OpenAPIRecipientCode,
		ApplySignature: cfg.Billing.ApplySignature,
		ApplyStorage:   cfg.Billing.ApplyStorage,
		Timeout:        cfg.Billing.Timeout,
		RetryAttempts:  cfg.Billing.RetryAttempts,
		SandboxMode:    cfg.Billing.SandboxMode,
	}

	invoiceRepo := repository.NewInvoiceRepository(deps.DB)
	customerRepo := repository.NewCustomerRepository(deps.DB)
	supplierRepo := repository.NewSupplierRepository(deps.DB)
	companyRepo := repository.NewCompanyRepository(deps.DB)
	notificationRepo := repository.NewNotificationRepository(deps.DB)

	m.openAPIClient = services.NewOpenAPIClientWithCache(openAPIConfig, deps.Logger, deps.RedisAdapter)
	xmlBuilder := services.NewXMLBuilder(openAPIConfig)

	// Retrieve PDFService from ServiceRegistry (can be nil if documents module is disabled)
	var pdfSvc documentsSvc.PDFService
	if svc := deps.Services.Get(module.ServicePDFService); svc != nil {
		pdfSvc = svc.(documentsSvc.PDFService)
	}

	invoiceSvc := services.NewInvoiceService(invoiceRepo, customerRepo, supplierRepo, companyRepo, m.openAPIClient, xmlBuilder, nil, pdfSvc, deps.Logger)
	customerSvc := services.NewCustomerService(customerRepo, deps.Logger)
	supplierSvc := services.NewSupplierService(supplierRepo, deps.Logger)
	companySvc := services.NewCompanyService(companyRepo, m.openAPIClient, deps.Logger)
	notificationSvc := services.NewNotificationService(notificationRepo, deps.Logger)

	m.invoiceHandler = handlers.NewInvoiceHandler(invoiceSvc)
	m.customerHandler = handlers.NewCustomerHandler(customerSvc)
	m.supplierHandler = handlers.NewSupplierHandler(supplierSvc)
	m.companyHandler = handlers.NewCompanyHandler(companySvc)
	m.notificationHandler = handlers.NewNotificationHandler(notificationSvc)
	m.businessRegistryHandler = handlers.NewBusinessRegistryHandler(m.openAPIClient)

	m.pollingJob = jobs.NewPollingJob(
		m.openAPIClient,
		invoiceRepo,
		notificationRepo,
		deps.Logger,
		cfg.Billing.PollingInterval,
	)

	m.syncHandler = handlers.NewSyncHandler(m.pollingJob)

	if cfg.Billing.WebhookURL != "" {
		m.webhookHandler = handlers.NewWebhookHandler(
			m.pollingJob,
			cfg.Billing.WebhookSecret,
			deps.Logger,
		)
	}

	deps.Logger.Info("Billing module initialized",
		slog.String("baseURL", cfg.Billing.OpenAPIBaseURL),
		slog.Bool("sandbox", cfg.Billing.SandboxMode),
	)
	return nil
}

func (m *BillingModule) RegisterRoutes(ri *module.RouteInfo) {
	// Protected billing routes: manager role and above
	ri.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.AuthMW.RequireHierarchicalRole("manager"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(
			api,
			m.invoiceHandler,
			m.customerHandler,
			m.supplierHandler,
			m.companyHandler,
			m.notificationHandler,
			m.businessRegistryHandler,
			m.syncHandler,
		)
	})

	// Public webhook routes (no JWT, authenticated via webhook secret)
	if m.webhookHandler != nil {
		RegisterWebhookRoutes(ri.PublicAPI, m.webhookHandler)
		m.logger.Info("Billing webhook routes registered",
			slog.String("webhookURL", m.cfg.Billing.WebhookURL),
		)
	}
}

func (m *BillingModule) Start(_ context.Context) error {
	cfg := m.cfg

	// Start SDI notification polling job
	if m.pollingJob != nil && cfg.Billing.PollingEnabled {
		pollingCtx, pollingCancel := context.WithCancel(context.Background())
		_ = pollingCancel // Will be cancelled via Stop()

		go func() {
			m.logger.Info("Starting SDI notification polling job",
				slog.Duration("interval", cfg.Billing.PollingInterval),
			)
			m.pollingJob.Start(pollingCtx)
		}()
	} else if !cfg.Billing.PollingEnabled {
		m.logger.Info("SDI polling job disabled - use manual sync endpoints instead (POST /v1/billing/sync)")
	}

	// Auto-configure API callbacks on OpenAPI.it for webhook reception
	if cfg.Billing.WebhookURL != "" && m.openAPIClient != nil {
		go func() {
			configCtx, configCancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer configCancel()

			authHeader := "Bearer " + cfg.Billing.WebhookSecret
			err := m.openAPIClient.ConfigureAPICallbacks(configCtx, services.APICallbackConfig{
				FiscalID: cfg.Billing.OpenAPIFiscalID,
				Callbacks: []services.CallbackConfig{
					{Event: "supplier-invoice", URL: cfg.Billing.WebhookURL, AuthHeader: authHeader},
					{Event: "customer-notification", URL: cfg.Billing.WebhookURL, AuthHeader: authHeader},
					{Event: "legal-storage-receipt", URL: cfg.Billing.WebhookURL, AuthHeader: authHeader},
				},
			})
			if err != nil {
				m.logger.Error("Failed to configure API callbacks on OpenAPI.it",
					slog.String("error", err.Error()),
					slog.String("webhookURL", cfg.Billing.WebhookURL),
				)
			} else {
				m.logger.Info("API callbacks configured on OpenAPI.it",
					slog.String("webhookURL", cfg.Billing.WebhookURL),
					slog.String("fiscalID", cfg.Billing.OpenAPIFiscalID),
				)
			}
		}()
	}

	return nil
}

func (m *BillingModule) Stop(_ context.Context) error {
	if m.pollingJob != nil {
		m.logger.Info("Stopping SDI notification polling job")
		m.pollingJob.Stop()
	}
	return nil
}

func (m *BillingModule) HealthCheck(_ context.Context) error { return nil }
