package user

import (
	"os"
	"strings"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	authRepo "github.com/orkestra/backend/internal/core/auth/repository"
	"github.com/orkestra/backend/internal/core/user/handlers"
	"github.com/orkestra/backend/internal/core/user/repository"
	"github.com/orkestra/backend/internal/core/user/services"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/module"
)

// userTierSplitEnabled reads the ADR-0003 PR-B cutover flag from the
// process environment. When true the operator-tier provider becomes the
// canonical ServiceUserService — login lookups and user CRUD all read
// and write operator_users. False is the safe default: the legacy
// `users` collection stays authoritative and the script in
// backend/scripts/migrate_user_split has not yet copied data into the
// new collection. Operators flip the flag to true on every host once
// the migration finishes.
//
// Reading from os.Getenv (not deps.Config) is deliberate: the flag is
// process-scoped and must take effect at boot — flipping it at runtime
// via /admin/modules would leave half the binary serving the old
// collection. This is the same pattern AUTH_REQUIRE_EMAIL_VERIFICATION
// uses.
func userTierSplitEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("USER_TIER_SPLIT_ENABLED")))
	return v == "true" || v == "1" || v == "yes"
}

type UserModule struct {
	module.BaseModule
	handler *handlers.UserHandler
}

func NewModule() *UserModule { return &UserModule{} }

func (m *UserModule) Name() string        { return "user" }
func (m *UserModule) DisplayName() string  { return "User Management" }
func (m *UserModule) Description() string  { return "User accounts, profiles, and RBAC" }

func (m *UserModule) ProvidedServices() []module.ServiceKey {
	return []module.ServiceKey{module.ServiceUserService}
}

func (m *UserModule) Collections() []module.CollectionSpec {
	// Email uniqueness on the per-tier collections is scoped to that
	// collection — the same email address may legitimately exist as both
	// an operator user and a client user (the same human running an
	// internal staff account and an external client account). The legacy
	// `users` collection retains its global email uniqueness until the
	// migration script copies its rows into operator_users.
	tierUserIndexes := []module.IndexSpec{
		{Keys: map[string]int{"uuid": 1}, Unique: true},
		{Keys: map[string]int{"email": 1}, Unique: true},
		{Keys: map[string]int{"tier": 1}},
	}
	return []module.CollectionSpec{
		{Name: repository.UsersCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{Keys: map[string]int{"email": 1}, Unique: true},
		}},
		// ADR-0003 PR-B: tier-split user collections. Created on every
		// boot so the migration script + tier-aware providers can rely
		// on them existing; queries against them return zero rows until
		// the migration populates operator_users.
		{Name: repository.OperatorUsersCollection, Indexes: tierUserIndexes},
		{Name: repository.ClientUsersCollection, Indexes: tierUserIndexes},
	}
}

func (m *UserModule) NavItems() []module.NavItemSpec {
	return []module.NavItemSpec{
		{Realm: "personal", Section: "My workspace", Name: "Dashboard", Icon: "chart-pie", Path: "/user/dashboard", Active: true},
		{Realm: "personal", Section: "My workspace", Name: "Profile", Icon: "user", Path: "/user/profile", Active: true},
		{Realm: "personal", Section: "My workspace", Name: "Calendar", Icon: "calendar-alt", Path: "/user/calendar", Active: true},
		{Realm: "platform", Section: "Admin", Tier: "internal", Name: "User Management", Icon: "users-cog", Path: "/admin/users", Active: true},
		{Realm: "platform", Section: "Admin", Tier: "internal", Name: "Module Management", Icon: "puzzle-piece", Path: "/admin/modules", Active: true},
	}
}

func (m *UserModule) Permissions() []iface.PermissionSpec {
	return []iface.PermissionSpec{
		{Key: "user.read", Module: "user", Description: "List users"},
		{Key: "user.update", Module: "user", Description: "Update user profiles"},
		{Key: "user.delete", Module: "user", Description: "Delete users"},
		{Key: "user.self", Module: "user", Description: "Edit your own profile"},
	}
}

func (m *UserModule) Init(deps *module.Dependencies) error {
	oauthProviderRepo := authRepo.NewOAuthProviderRepository(deps.DB)

	legacyRepo := repository.NewUserRepository(deps.DB)
	legacySvc := services.NewUserService(legacyRepo, oauthProviderRepo)

	operatorRepo := repository.NewOperatorUserRepository(deps.DB)
	operatorSvc := services.NewUserService(operatorRepo, oauthProviderRepo)

	clientRepo := repository.NewClientUserRepository(deps.DB)
	clientSvc := services.NewUserService(clientRepo, oauthProviderRepo)

	// ADR-0003 PR-B: tier-aware providers always register under their
	// dedicated keys so a downstream MustGetTyped never panics. The
	// USER_TIER_SPLIT_ENABLED flag picks which one serves as the
	// canonical ServiceUserService — the legacy `users` collection
	// stays the source of truth at PR-B's default (flag=false), and
	// operators flip to true after running migrate_user_split.go to
	// move read+write traffic onto operator_users.
	//
	// PR-D will collapse this into per-audience consumption: auth
	// handlers on the operator host pull ServiceOperatorUserProvider
	// directly, client handlers pull ServiceClientUserProvider, and
	// the legacy ServiceUserService key goes away alongside JWT v1.
	deps.Services.Register(module.ServiceOperatorUserProvider, operatorSvc)
	deps.Services.Register(module.ServiceClientUserProvider, clientSvc)

	canonical := legacySvc
	canonicalRepo := legacyRepo
	if userTierSplitEnabled() {
		canonical = operatorSvc
		canonicalRepo = operatorRepo
		deps.Logger.Info("ADR-0003 PR-B: USER_TIER_SPLIT_ENABLED=true — operator_users is the canonical user collection")
	}
	m.handler = handlers.NewUserHandler(canonical)
	deps.Services.Register(module.ServiceUserService, canonical)

	// Register the user PII producer with the DSR registry pre-created in
	// main.go. Missing registry means the platform was booted without
	// compliance infrastructure — tolerate and skip.
	if reg, ok := module.GetTyped[*iface.PIIProducerRegistry](deps.Services, module.ServicePIIProducerRegistry); ok {
		reg.Register(services.NewPIIProducer(canonicalRepo))
	}
	return nil
}

func (m *UserModule) RegisterRoutes(ri *module.RouteInfo) {
	// User management is a platform-level concern: users are global, so
	// routes live on the system permission gate (administrators) rather
	// than per-org.
	ri.Operator.ProtectedRouter.Group(func(r chi.Router) {
		r.Use(ri.Operator.AuthMW.RequireSystemPermission("system.users.admin"))
		api := humachi.New(r, ri.APIConfig)
		RegisterRoutes(api, m.handler)
	})
}
