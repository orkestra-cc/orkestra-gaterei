package auth

import (
	"log/slog"
	"os"
	"time"

	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/orkestra/backend/internal/core/auth/handlers"
	"github.com/orkestra/backend/internal/core/auth/models"
	"github.com/orkestra/backend/internal/core/auth/repository"
	"github.com/orkestra/backend/internal/core/auth/services"
	sharederrors "github.com/orkestra/backend/internal/shared/errors"
	"github.com/orkestra/backend/internal/shared/iface"
	"github.com/orkestra/backend/internal/shared/module"
)

type AuthModule struct {
	module.BaseModule
	authHandler     *handlers.AuthHandler
	passwordHandler *handlers.PasswordAuthHandler
	mfaHandler      *handlers.MFAHandler
}

func NewModule() *AuthModule { return &AuthModule{} }

func (m *AuthModule) Name() string        { return "auth" }
func (m *AuthModule) DisplayName() string  { return "Authentication" }
func (m *AuthModule) Description() string  { return "OAuth 2.1, JWT, sessions, RBAC" }

func (m *AuthModule) Dependencies() []string { return []string{"user", "notification", "tenant", "authz"} }
func (m *AuthModule) RequiredServices() []module.ServiceKey {
	return []module.ServiceKey{module.ServiceUserService, module.ServiceTenantProvider}
}
func (m *AuthModule) OptionalServices() []module.ServiceKey {
	return []module.ServiceKey{module.ServiceNotificationSender}
}
func (m *AuthModule) ProvidedServices() []module.ServiceKey {
	return []module.ServiceKey{
		module.ServiceAuthService,
		module.ServiceJWTService,
		module.ServicePasswordService,
		module.ServicePasswordAuthService,
		module.ServiceSessionRevocation,
	}
}

func (m *AuthModule) Permissions() []iface.PermissionSpec {
	return []iface.PermissionSpec{
		{Key: "auth.self", Module: "auth", Description: "Edit your own password and sessions"},
		{Key: "auth.mfa.self", Module: "auth", Description: "Enroll, verify, and remove your own MFA factors"},
		{Key: "system.users.mfa_reset", Module: "auth", Description: "Admin: reset another user's MFA factors"},
	}
}

// ConfigSchema declares every OAuth provider setting as admin-manageable.
// Values are seeded from the listed env vars on first boot, then owned by
// the module_configs document in MongoDB. Secrets are encrypted at rest.
// The Group field drives the admin modal's tab rendering — fields in the
// same group land on the same tab, in declaration order.
func (m *AuthModule) ConfigSchema() []module.ConfigField {
	return []module.ConfigField{
		// Google
		{Key: "googleClientId", Label: "Client ID", Group: "Google", Type: module.FieldString, EnvVar: "OAUTH_GOOGLE_CLIENT_ID"},
		{Key: "googleClientSecret", Label: "Client Secret", Group: "Google", Type: module.FieldSecret, EnvVar: "OAUTH_GOOGLE_CLIENT_SECRET"},
		{Key: "googleRedirectURL", Label: "Redirect URL", Group: "Google", Type: module.FieldString, EnvVar: "OAUTH_GOOGLE_REDIRECT_URL"},
		{Key: "googleAndroidClientId", Label: "Android Client ID", Group: "Google", Type: module.FieldString, EnvVar: "OAUTH_GOOGLE_ANDROID_CLIENT_ID"},
		{Key: "googleIOSClientId", Label: "iOS Client ID", Group: "Google", Type: module.FieldString, EnvVar: "OAUTH_GOOGLE_IOS_CLIENT_ID"},

		// Apple
		{Key: "appleClientId", Label: "Client ID", Group: "Apple", Type: module.FieldString, EnvVar: "OAUTH_APPLE_CLIENT_ID"},
		{Key: "appleTeamId", Label: "Team ID", Group: "Apple", Type: module.FieldString, EnvVar: "OAUTH_APPLE_TEAM_ID"},
		{Key: "appleKeyId", Label: "Key ID", Group: "Apple", Type: module.FieldString, EnvVar: "OAUTH_APPLE_KEY_ID"},
		{Key: "applePrivateKey", Label: ".p8 Key (PEM)", Group: "Apple", Description: "Inline PEM content of your Apple Sign-In .p8 key", Type: module.FieldSecret, EnvVar: "OAUTH_APPLE_PRIVATE_KEY"},
		{Key: "applePrivateKeyPath", Label: ".p8 Key Path", Group: "Apple", Description: "Filesystem path fallback if PEM is not inlined", Type: module.FieldString, EnvVar: "OAUTH_APPLE_PRIVATE_KEY_PATH"},
		{Key: "appleRedirectURL", Label: "Redirect URL", Group: "Apple", Type: module.FieldString, EnvVar: "OAUTH_APPLE_REDIRECT_URL"},
		{Key: "appleIOSClientId", Label: "iOS Client ID", Group: "Apple", Type: module.FieldString, EnvVar: "OAUTH_APPLE_IOS_CLIENT_ID"},
		{Key: "appleAndroidClientId", Label: "Android Client ID", Group: "Apple", Type: module.FieldString, EnvVar: "OAUTH_APPLE_ANDROID_CLIENT_ID"},

		// GitHub
		{Key: "githubClientId", Label: "Client ID", Group: "GitHub", Type: module.FieldString, EnvVar: "OAUTH_GITHUB_CLIENT_ID"},
		{Key: "githubClientSecret", Label: "Client Secret", Group: "GitHub", Type: module.FieldSecret, EnvVar: "OAUTH_GITHUB_CLIENT_SECRET"},
		{Key: "githubRedirectURL", Label: "Redirect URL", Group: "GitHub", Type: module.FieldString, EnvVar: "OAUTH_GITHUB_REDIRECT_URL"},

		// Discord
		{Key: "discordClientId", Label: "Client ID", Group: "Discord", Type: module.FieldString, EnvVar: "OAUTH_DISCORD_CLIENT_ID"},
		{Key: "discordClientSecret", Label: "Client Secret", Group: "Discord", Type: module.FieldSecret, EnvVar: "OAUTH_DISCORD_CLIENT_SECRET"},
		{Key: "discordRedirectURL", Label: "Redirect URL", Group: "Discord", Type: module.FieldString, EnvVar: "OAUTH_DISCORD_REDIRECT_URL"},
	}
}

func (m *AuthModule) Collections() []module.CollectionSpec {
	return []module.CollectionSpec{
		{Name: models.OAuthProvidersCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"userUuid": 1, "provider": 1}, Unique: true},
		}},
		{Name: models.RefreshTokensCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{Keys: map[string]int{"userUuid": 1}},
			// Block C: family lookup for RevokeFamily / replay detection.
			{Keys: map[string]int{"familyId": 1}},
		}},
		{Name: models.AuthSessionsCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
		}},
		{Name: models.SecurityEventsCollection},
		{Name: models.EmailTokensCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{Keys: map[string]int{"tokenHash": 1}, Unique: true},
			{Keys: map[string]int{"userUuid": 1}},
			// TTL: documents are removed 24h after ExpiresAt (max TTL for both verify and reset).
			{Keys: map[string]int{"expiresAt": 1}, TTL: 24 * time.Hour},
		}},
		{Name: models.MFAFactorsCollection, Indexes: []module.IndexSpec{
			{Keys: map[string]int{"uuid": 1}, Unique: true},
			{Keys: map[string]int{"userUuid": 1, "type": 1}, Unique: true},
		}},
	}
}

func (m *AuthModule) Init(deps *module.Dependencies) error {
	cfg := deps.Config
	logger := deps.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(os.Stderr, nil))
	}

	// Repositories
	authRepo, err := repository.NewAuthRepository(deps.DB)
	if err != nil {
		return err
	}
	oauthProviderRepo := repository.NewOAuthProviderRepository(deps.DB)
	refreshTokenRepo := repository.NewRefreshTokenRepository(deps.DB)
	authSessionRepo := repository.NewAuthSessionRepository(deps.DB)
	emailTokenRepo := repository.NewEmailTokenRepository(deps.DB)
	mfaFactorRepo := repository.NewMFAFactorRepository(deps.DB)

	// OAuth provider factory + live config resolver.
	//
	// Provider configs live in the module_configs document and are resolved
	// per-request by OAuthConfigResolver — seeded on first boot from the
	// OAUTH_* env vars declared in ConfigSchema(), then owned by the admin
	// UI at /admin/modules. The factory itself holds an empty map; each
	// handler call passes a freshly resolved config through CreateProvider's
	// override parameter so secret rotations take effect without a restart.
	providerFactory := services.NewOAuthProviderFactory(
		map[models.OAuthProvider]*services.OAuthProviderConfig{},
		deps.RedisAdapter,
	)
	oauthResolver := services.NewOAuthConfigResolver(deps.ConfigService)

	// JWT service. The tenant provider is wired in so that token issuance
	// can embed the user's current memberships in the JWT. The environment
	// is stamped into the iss claim (orkestra.<env>) so a token minted in
	// one deployment is rejected by another even if the signing keys ever
	// overlap. Keys themselves should differ per environment — this claim
	// is defense in depth.
	jwtService := services.NewJWTService(
		cfg.Auth.JWT.PrivateKey,
		cfg.Auth.JWT.PublicKey,
		cfg.Server.Environment,
		cfg.Auth.JWT.AccessTokenExpiry,
		cfg.Auth.JWT.RefreshTokenExpiry,
	)
	tenantProvider := module.MustGetTyped[iface.TenantProvider](deps.Services, module.ServiceTenantProvider)
	jwtService.SetTenantProvider(tenantProvider)

	// OAuth state service
	redisStore := services.NewRedisOAuthStateStore(deps.RedisAdapter)
	oauthStateService := services.NewOAuthStateService(redisStore)

	// Get user service from registry
	userService := module.MustGetTyped[iface.UserProvider](deps.Services, module.ServiceUserService)

	// Auth service
	// First-admin claimer is shared between OAuth and password signup. The
	// lookup is lifted here from its earlier spot in the password section
	// so both services see the same instance.
	var firstAdminClaimer services.FirstAdminClaimer
	if c, ok := module.GetTyped[services.FirstAdminClaimer](deps.Services, module.ServiceFirstAdminClaimer); ok {
		firstAdminClaimer = c
	} else {
		logger.Warn("first-admin claimer not wired — signup flows will fall through to non-atomic first-user heuristic")
	}

	// MFA challenge store is needed by both the auth service (OAuth login
	// partial response) and the password auth service (password login
	// partial response), so build it before either consumer.
	mfaChallengeSvc := services.NewMFAChallengeService(redisStore)

	authService, err := services.NewAuthService(&services.AuthConfig{
		AuthRepo:            authRepo,
		UserService:         userService,
		TenantProvider:      tenantProvider,
		OAuthProviderRepo:   oauthProviderRepo,
		RefreshTokenRepo:    refreshTokenRepo,
		AuthSessionRepo:     authSessionRepo,
		JWTService:          jwtService,
		MFAFactorRepo:       mfaFactorRepo,
		MFAChallengeService: mfaChallengeSvc,
		FirstAdminClaimer:   firstAdminClaimer,
	})
	if err != nil {
		return err
	}

	// Session revocation list (Block D): Redis-backed set of revoked `sid`
	// claims checked on every authenticated request. Populated on logout,
	// password reset, and admin kill-session so a stolen access token stops
	// working instantly instead of staying valid until its TTL expires.
	sessionRevocationSvc := services.NewSessionRevocationService(
		deps.RedisAdapter,
		cfg.Auth.JWT.AccessTokenExpiry,
		logger,
	)

	// Auth handler
	m.authHandler = handlers.NewAuthHandler(
		authService,
		providerFactory,
		oauthResolver,
		oauthStateService,
		oauthProviderRepo,
		jwtService,
		cfg,
	)
	m.authHandler.SetSessionRevocation(sessionRevocationSvc)

	// Password auth service — depends on notification module (optional).
	passwordSvc := services.NewPasswordService(logger, true)
	var notifier iface.NotificationSender
	if n, ok := module.GetTyped[iface.NotificationSender](deps.Services, module.ServiceNotificationSender); ok {
		notifier = n
	}

	rateLimiter := sharederrors.NewRateLimiter()

	passwordAuthSvc := services.NewPasswordAuthService(services.PasswordAuthConfig{
		UserService:              userService,
		TenantProvider:           tenantProvider,
		PasswordService:          passwordSvc,
		JWTService:               jwtService,
		EmailTokenRepo:           emailTokenRepo,
		RefreshTokenRepo:         refreshTokenRepo,
		AuthSessionRepo:          authSessionRepo,
		MFAFactorRepo:            mfaFactorRepo,
		MFAChallengeService:      mfaChallengeSvc,
		FirstAdminClaimer:        firstAdminClaimer,
		Notifier:                 notifier,
		RateLimiter:              rateLimiter,
		FrontendURL:              cfg.Server.FrontendURL,
		RequireEmailVerification: getBoolEnv("AUTH_REQUIRE_EMAIL_VERIFICATION", cfg.IsProductionLike()),
		AppName:                  getEnvOrDefault("APP_NAME", "Orkestra"),
		SupportEmail:             os.Getenv("SUPPORT_EMAIL"),
		Logger:                   logger,
	})

	m.passwordHandler = handlers.NewPasswordAuthHandler(
		passwordAuthSvc,
		cfg.Auth.Cookie.Name,
		cfg.Auth.Cookie.Domain,
		cfg.Auth.Cookie.Secure,
	)
	m.passwordHandler.SetSessionRevocation(sessionRevocationSvc)

	// MFA orchestrator — issuer drives the TOTP provisioning URI label.
	// Borrows the password service's argon2id hasher for backup-code
	// storage so we don't ship a second hasher.
	mfaIssuer := getEnvOrDefault("APP_NAME", "Orkestra")
	mfaSvc := services.NewMFAService(mfaFactorRepo, mfaChallengeSvc, passwordSvc, mfaIssuer, logger)

	m.mfaHandler = handlers.NewMFAHandler(
		mfaSvc,
		mfaChallengeSvc,
		jwtService,
		userService,
		passwordAuthSvc,
		cfg.Auth.Cookie.Name,
		cfg.Auth.Cookie.Domain,
		cfg.Auth.Cookie.Secure,
	)

	// Register services for main.go middleware setup
	deps.Services.Register(module.ServiceAuthService, authService)
	deps.Services.Register(module.ServiceJWTService, jwtService)
	deps.Services.Register(module.ServicePasswordService, passwordSvc)
	deps.Services.Register(module.ServicePasswordAuthService, passwordAuthSvc)
	deps.Services.Register(module.ServiceSessionRevocation, sessionRevocationSvc)

	// Register the auth PII producer with the DSR registry pre-created in
	// main.go. Registers even when the registry is absent so the main
	// path stays uniform — the helper is a no-op when the key is missing.
	if reg, ok := module.GetTyped[*iface.PIIProducerRegistry](deps.Services, module.ServicePIIProducerRegistry); ok {
		securityEvents, err := services.NewSecurityEventService(deps.DB)
		if err != nil {
			// Nothing else in auth currently writes security events, so a
			// failure here is survivable — DSR purge of that collection
			// becomes a no-op until the collection is initialized.
			logger.Warn("auth: security event service init failed; DSR will skip the collection", slog.String("error", err.Error()))
		}
		reg.Register(services.NewPIIProducer(refreshTokenRepo, authSessionRepo, emailTokenRepo, mfaFactorRepo, securityEvents))
	}

	return nil
}

func getBoolEnv(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	return v == "true" || v == "1" || v == "yes"
}

func getEnvOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func (m *AuthModule) RegisterRoutes(ri *module.RouteInfo) {
	// Auth has both public and protected routes
	protectedAPI := humachi.New(ri.ProtectedRouter, ri.APIConfig)
	m.authHandler.RegisterRoutes(ri.PublicAPI, protectedAPI, ri.Router, ri.ProtectedRouter)

	// Password auth endpoints: register/login/verify/reset/forgot live on the
	// public API; change-password is protected and runs without an org
	// context (it's a user self-service flow).
	if m.passwordHandler != nil {
		m.passwordHandler.RegisterPublicRoutes(ri.PublicAPI)
		ri.ProtectedRouter.Group(func(r chi.Router) {
			r.Use(ri.AuthMW.RequireGlobal())
			api := humachi.New(r, ri.APIConfig)
			m.passwordHandler.RegisterProtectedRoutes(api)
		})
	}

	// MFA endpoints split into four halves:
	//   - public: /v1/auth/mfa/login/verify completes an in-flight login
	//     (the caller has a challengeId, not yet a bearer token).
	//   - protected (no step-up): enroll/status/verify — RequireGlobal()
	//     so the caller is already authenticated with a primary factor.
	//   - protected (step-up): /v1/auth/me/mfa/remove — dropping your own
	//     second factor is catastrophic, so demand a <5min OTP proof.
	//   - admin (step-up): /v1/admin/users/{id}/mfa/reset — resetting
	//     another user's MFA lets the admin enroll their own device, so
	//     step-up here gates the same move.
	if m.mfaHandler != nil {
		m.mfaHandler.RegisterPublicRoutes(ri.PublicAPI)
		ri.ProtectedRouter.Group(func(r chi.Router) {
			r.Use(ri.AuthMW.RequireGlobal())
			api := humachi.New(r, ri.APIConfig)
			m.mfaHandler.RegisterProtectedRoutes(api)
		})
		ri.ProtectedRouter.Group(func(r chi.Router) {
			r.Use(ri.AuthMW.RequireGlobal())
			r.Use(ri.AuthMW.RequireStepUp(5 * time.Minute))
			api := humachi.New(r, ri.APIConfig)
			m.mfaHandler.RegisterStepUpRoutes(api)
		})
		ri.ProtectedRouter.Group(func(r chi.Router) {
			r.Use(ri.AuthMW.RequireSystemPermission("system.users.mfa_reset"))
			r.Use(ri.AuthMW.RequireStepUp(5 * time.Minute))
			api := humachi.New(r, ri.APIConfig)
			m.mfaHandler.RegisterAdminRoutes(api)
		})
	}
}

