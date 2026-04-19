// Package identity is the Phase-3 BYO-IdP module.
//
// Each external tenant can plug in its own OpenID Connect provider by
// storing a single `identity_idp_configs` document (issuer URL, client
// credentials, claim mapping, enabled flag). Orkestra stands up the OIDC
// login flow on top of `coreos/go-oidc`, verifies ID tokens against the
// IdP's published keys, and mints an Orkestra session via the existing
// PasswordAuthService token-issuance path.
//
// The module is toggleable — operators can disable it entirely via the
// admin UI, which gates the public /start + /callback routes behind a
// 503. Per-tenant control is via `IdPConfig.Enabled`.
package identity

import (
	"log/slog"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"

	"github.com/orkestra/backend/internal/addons/identity/handlers"
	"github.com/orkestra/backend/internal/addons/identity/models"
	"github.com/orkestra/backend/internal/addons/identity/repository"
	"github.com/orkestra/backend/internal/addons/identity/services"
	authServices "github.com/orkestra/backend/internal/core/auth/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/module"
)

// Module wires the identity module's handlers + service.
type Module struct {
	module.BaseModule
	admin  *handlers.AdminHandler
	public *handlers.PublicHandler
	logger *slog.Logger
}

// NewModule constructs an empty module; Init wires collaborators.
func NewModule() *Module { return &Module{} }

func (m *Module) Name() string                     { return "identity" }
func (m *Module) DisplayName() string              { return "Identity (BYO IdP)" }
func (m *Module) Description() string              { return "Per-tenant BYO OpenID Connect login. Tenants configure their own IdP (Okta, Entra, Auth0, …) and users sign in against it; Orkestra mints its own session after verifying the ID token." }
func (m *Module) Category() module.ModuleCategory  { return module.CategoryToggleable }
func (m *Module) Enabled(_ *config.Config) bool    { return true }

// Dependencies: auth owns the token-issuance service we delegate to at
// the end of callback; tenant owns slug/UUID resolution. Both are core
// so this is defensive.
func (m *Module) Dependencies() []string { return []string{"auth", "tenant"} }

func (m *Module) RequiredServices() []module.ServiceKey {
	return []module.ServiceKey{
		module.ServicePasswordAuthService,
		module.ServiceTenantProvider,
		module.ServiceUserService,
	}
}

// Collections declares the one collection the module owns. Composite
// unique on (tenantId, protocol) so v1's one-config-per-tenant-per-
// protocol invariant is enforced at the DB layer.
func (m *Module) Collections() []module.CollectionSpec {
	return []module.CollectionSpec{
		{Name: models.IdPConfigsCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{OrderedKeys: []module.IndexKey{
				{Field: "tenantId", Direction: 1},
				{Field: "protocol", Direction: 1},
			}, Unique: true},
		}},
	}
}

// Init wires the repository, OIDC service, and handlers.
func (m *Module) Init(deps *module.Dependencies) error {
	passwordAuth := module.MustGetTyped[*authServices.PasswordAuthService](deps.Services, module.ServicePasswordAuthService)
	tenantProvider := module.MustGetTyped[iface.TenantProvider](deps.Services, module.ServiceTenantProvider)
	userProvider := module.MustGetTyped[iface.UserProvider](deps.Services, module.ServiceUserService)

	repo := repository.New(deps.DB)
	svc := services.New(services.Config{
		Repo:         repo,
		Users:        userProvider,
		Tenant:       tenantProvider,
		PasswordAuth: passwordAuth,
		State:        deps.RedisAdapter,
		Logger:       deps.Logger,
	})

	m.admin = handlers.NewAdminHandler(repo)
	m.public = handlers.NewPublicHandler(svc)
	m.logger = deps.Logger

	deps.Logger.Info("Identity module initialized")
	return nil
}

// RegisterRoutes mounts public (start/callback) and admin (CRUD) routes.
// Admin routes live under the protected router gated by RequirePermission
// on the tenant admin surface — the same gate the tenant module uses for
// its own mutations.
func (m *Module) RegisterRoutes(ri *module.RouteInfo) {
	if m.public != nil {
		handlers.RegisterPublicRoutes(ri.PublicAPI, m.public)
	}
	if m.admin != nil {
		ri.ProtectedRouter.Group(func(r chi.Router) {
			r.Use(ri.AuthMW.RequirePermission("tenant.org.update"))
			api := humachi.New(r, ri.APIConfig)
			handlers.RegisterAdminRoutes(api, m.admin)
		})
	}
}
