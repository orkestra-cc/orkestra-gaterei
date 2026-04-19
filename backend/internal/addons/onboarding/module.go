// Package onboarding is the Phase-3 public-signup module. It owns the
// anonymous endpoint that creates a new external (Tier-2) tenant and its
// owner user in a single request, glueing together PasswordAuthService
// (from the auth core module) and TenantProvider.CreateExternalTenant
// (from the tenant core module).
//
// The module is intentionally thin: no collections of its own, no background
// jobs, and no navigation entries. Richer flows (plan selection, invite
// redemption, SCIM provisioning, BYO IdP) land in later Phase-3 commits
// under this same module or under a sibling `identity` module.
package onboarding

import (
	"log/slog"

	"github.com/orkestra/backend/internal/addons/onboarding/handlers"
	"github.com/orkestra/backend/internal/addons/onboarding/services"
	authServices "github.com/orkestra/backend/internal/core/auth/services"
	"github.com/orkestra/backend/internal/shared/config"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/module"
)

// Module is the onboarding module wiring.
type Module struct {
	module.BaseModule
	handler *handlers.Handler
	logger  *slog.Logger
}

// NewModule constructs an empty module instance. Actual wiring happens in Init.
func NewModule() *Module { return &Module{} }

func (m *Module) Name() string                   { return "onboarding" }
func (m *Module) DisplayName() string             { return "Onboarding" }
func (m *Module) Description() string             { return "Anonymous self-service signup: creates a new external tenant and its owner user in a single public call." }
func (m *Module) Category() module.ModuleCategory { return module.CategoryToggleable }

// Onboarding boots as long as the env var isn't explicitly set to false.
// The module has no external dependencies to verify — it's pure glue —
// so the default is "on when the operator selects it via MODULES".
func (m *Module) Enabled(cfg *config.Config) bool { return true }

// Dependencies: auth owns the password + email-verification flows we reuse;
// tenant owns the external-tenant provisioning method. Both are core so
// they always initialize before optional modules — this declaration is
// defensive, it gates a wiring mistake rather than the normal boot path.
func (m *Module) Dependencies() []string { return []string{"auth", "tenant"} }

func (m *Module) RequiredServices() []module.ServiceKey {
	return []module.ServiceKey{
		module.ServicePasswordAuthService,
		module.ServiceTenantProvider,
	}
}

// Init fetches PasswordAuthService + TenantProvider from the registry and
// wires them into the orchestration service. No collections, no seeds.
//
// The onboarding service also owns the activate-on-verify hook: it's set on
// PasswordAuthService after construction so the auth module stays free of a
// runtime dependency on onboarding. Disabling the onboarding module at
// runtime leaves the callback set (the auth module never re-initializes) —
// that's acceptable because the callback is side-effect-only. Future work
// can clear it on Stop() if a clean toggle matters.
func (m *Module) Init(deps *module.Dependencies) error {
	passwordAuth := module.MustGetTyped[*authServices.PasswordAuthService](deps.Services, module.ServicePasswordAuthService)
	tenantProvider := module.MustGetTyped[iface.TenantProvider](deps.Services, module.ServiceTenantProvider)

	svc := services.New(passwordAuth, tenantProvider, deps.Logger)
	m.handler = handlers.New(svc)
	m.logger = deps.Logger

	passwordAuth.SetOnboardingActivator(svc.ActivateOnVerify)

	deps.Logger.Info("Onboarding module initialized")
	return nil
}

// RegisterRoutes mounts the single public endpoint on the shared public
// Huma API. Mirrors the pattern used by auth.RegisterPublicRoutes so the
// endpoint is visible in the root OpenAPI spec and reachable without an
// access token — signup is the one flow that predates authentication.
// Operators disable the endpoint at runtime via the admin UI; the module
// registry clears the route list when disabled.
func (m *Module) RegisterRoutes(ri *module.RouteInfo) {
	handlers.Register(ri.PublicAPI, m.handler)
}
