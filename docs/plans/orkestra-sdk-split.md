# Orkestra SDK Split — Implementation Plan

**Status:** Draft · branch `orkestra-sdk-dev` · 2026-05-13
**Owner:** Salvatore Balestrino
**Tracking:** _will be linked once first PR opens_

## 1. Why this exists

Every optional addon today (`backend/internal/addons/*`) is welded to the monolith because it imports `github.com/orkestra/backend/internal/...`. Go's `internal/` rule means an addon cannot be moved to its own Go module *and* keep those imports — the boundary is structural, not just organizational.

Goal: turn the monorepo's de-facto module contract (`shared/iface` + `shared/module` + a handful of helpers) into a **versioned, publishable Go SDK**, then make each addon a consumer of that SDK on the same terms as a hypothetical third-party plugin author. This unlocks:

- closed-source / per-customer addons that ship as their own repo + `go get`
- third-party addon ecosystem with a real semver contract
- independent release cadence per addon
- the AI sidecar (`cmd/ai-service`) depends on the SDK directly instead of on backend internals

Non-goals (explicitly out of scope for this branch):

- splitting `auth`, `user`, `tenant`, `authz`, `notification`, `navigation` — these are core and stay inside the backend repo
- replacing the runtime enable/disable mechanism, the build-tag profile system, or the ConfigService schema
- changing the on-disk MongoDB layout
- moving frontend or mobile code

## 2. Architecture target

```
github.com/orkestra-cc/orkestra-sdk            ← NEW public Go module, semver'd
  /module          Module iface, ModuleRegistry, ServiceRegistry, ConfigService, admin routes
  /iface           cross-module interfaces + DTOs (UserProvider, NotificationSender, ...)
  /capability      Capability type
  /modulegate      ModuleGate middleware (route gating only; not auth)
  /testkit         helpers for module authors (in-memory ServiceRegistry, fake ConfigService)

github.com/orkestra-cc/addon-billing           ← per-addon repo (one per extracted addon)
  go.mod requires orkestra-sdk vX.Y.Z
  module.go: billing.NewModule() implements sdk/module.Module
  handlers/ services/ repository/ models/ jobs/ config/ docs/

github.com/orkestra-cc/orkestra                ← main monorepo (this one)
  backend/
    go.mod requires orkestra-sdk + each extracted addon
    cmd/server/catalog_<addon>.go imports remote addon module
    internal/core/   — still uses sdk; stays in backend
    internal/shared/ — shrinks to backend-private helpers only
```

Both binaries (`cmd/server/`, `cmd/ai-service/`) become **SDK consumers**. Core modules implement `sdk.Module` on the same terms as external addons. There is no second-class API for in-repo modules.

## 3. SDK package surface (sized from current codebase)

Measured 2026-05-13 against `dev`:

| Current path                     | LOC  | Reach (files) | SDK destination       | Notes                                                    |
| -------------------------------- | ---- | ------------- | --------------------- | -------------------------------------------------------- |
| `internal/shared/iface`          | 1308 | 99            | `sdk/iface`           | Pure interfaces + DTOs — moves wholesale                 |
| `internal/shared/module`         | 3341 | 43            | `sdk/module`          | Registry, ConfigService, admin routes — biggest piece    |
| `internal/shared/capability`     |  122 |  7            | `sdk/capability`      | Tiny, moves as-is                                        |
| `internal/shared/middleware`     | 2561 | 32            | `sdk/modulegate` (subset) | **Split.** `ModuleGate` → SDK; `AuthMiddleware`/`JWTValidator` stay in backend |
| `internal/shared/openapiauth`    |   ?  |  ?            | **into `addon-billing`** | Billing-specific OpenAPI.com gateway helper              |
| `internal/shared/tenantrepo`     |  186 |  2            | `sdk/tenantrepo`      | orgId scope helpers — load-bearing for every addon       |
| `internal/shared/errors`         | 2263 |  6            | _stays in backend_    | Backend-internal helpers; addons should use stdlib errors + huma |
| `internal/shared/utils`          | 1447 | 15            | _audit + split_       | Most likely backend-private; SDK gets only what addons need |
| `internal/shared/types`          |   27 |  1            | _stays_               | Trivially scoped                                          |
| `internal/shared/database`       |  294 |  5            | _stays_               | Backend owns connections; addons receive handles via `Init(deps)` |

**Hard SDK surface (~5.0K LOC):** `iface` + `module` + `capability` + `modulegate` + `tenantrepo` + minimal slices of `utils`.

Addons that need the rest (errors helpers, utils) get them injected through `Init(deps)` instead of importing the backend.

## 4. The `Module` interface needs to be reshaped before publishing

Today's `Module` interface is wide (14+ methods). Once externalized it's frozen — adding a method is a breaking change for every published addon. Before v1, split it like Huma did with optional capabilities:

```go
// sdk/module/module.go — required surface, kept minimal
type Module interface {
    Name() string
    DisplayName() string
    Description() string
    Category() Category
    Dependencies() []string
    Enabled() bool
    Init(deps Deps) error
    RegisterRoutes(api huma.API)
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    HealthCheck(ctx context.Context) HealthStatus
}

// Optional sub-interfaces, queried with type assertions in the registry.
// Any future capability is added as a new sub-interface — never by widening Module.
type HasConfigSchema    interface { ConfigSchema() ConfigSchema }
type HasCollections     interface { Collections() []CollectionSpec }
type HasNavItems        interface { NavItems() []NavItem }
type HasProvidedSvcs    interface { ProvidedServices() []ServiceKey }
type HasRequiredSvcs    interface { RequiredServices() []ServiceKey }
type HasInfraContainers interface { InfraContainers() []InfraContainerSpec }
```

Phase 1 work item: refactor the registry to use type assertions for optional capabilities **before** anything moves out of the backend. This is an internal-only change with zero behavior diff and is fully reversible.

## 5. Phased migration

Phases are designed so each one ends on a green `make ci-all` and is independently revertible. The branch `orkestra-sdk-dev` lives until Phase 5 completes; after that, work continues on `dev` per addon.

### Phase 0 — Plan + dependency cleanup _(this PR)_
- Write this plan, commit on `orkestra-sdk-dev`.
- Identify and inventory every `internal/shared/*` symbol used by addons. Output: an updated table in this doc with confirmed SDK / not-SDK classification per symbol.
- Decide: rename `internal/shared/openapiauth` → `internal/addons/billing/openapi/` (it's only used by billing). Do NOT execute yet.
- **Exit criteria:** plan reviewed, no code changes.

### Phase 1 — Reshape `Module` interface inside the monorepo
- Split `module.Module` into the minimal + sub-interface form (section 4 above).
- Update `registry.go` to query optional capabilities via type assertion.
- Update every existing module (core + addons) to keep working — most lose nothing, some shed methods they were stubbing.
- Add a `module_test.go` that compiles a fake addon implementing only the required surface, to lock the minimum contract.
- **Exit criteria:** `make ci-all` green; no change in admin UI or runtime behavior. Reversible: revert the PR.

### Phase 2 — Move SDK candidates from `internal/shared/` to `backend/pkg/sdk/` _(still monorepo)_
- Move `iface`, `module`, `capability`, `tenantrepo`, the `ModuleGate` subset of `middleware`, and the addon-relevant pieces of `utils` to `backend/pkg/sdk/<name>/`.
- `pkg/` is outside `internal/`, so the same code is now importable by future external modules — but everything is still one Go module, one repo.
- Rewrite all imports across `backend/`. This is a mechanical change, scriptable with `goimports -r` or a small Go program.
- Verify `tools/tenantscope` still works against new paths.
- **Exit criteria:** `make ci-all` green; no behavior change; `internal/shared/` is now strictly backend-private. Reversible: one `git revert`.

### Phase 3 — Carve `backend/pkg/sdk/` into its own go.mod inside the monorepo
- Add `backend/pkg/sdk/go.mod` declaring `module github.com/orkestra-cc/orkestra-sdk`.
- Add `go.work` at repo root listing `./backend` and `./backend/pkg/sdk`.
- Backend now requires the SDK as a separate module; `go.work` resolves it locally.
- Pin the SDK at `v0.0.0-<commit>` for reproducibility (or use `replace` until Phase 5).
- **Exit criteria:** `make ci-all` green; `cd backend && go build ./...` works; `cd backend/pkg/sdk && go test ./...` works in isolation. Reversible but harder: would need to merge the go.mod back.

### Phase 4 — Extract the SDK to its own GitHub repo
- Create `orkestra-cc/orkestra-sdk` empty repo.
- Use `git filter-repo --subdirectory-filter backend/pkg/sdk/` on a clone of the main repo to preserve history.
- Tag `v0.1.0` (intentionally pre-v1 to allow shaping changes).
- In the main repo: remove `backend/pkg/sdk/`, replace `go.work` use directive with a normal `require` against the published SDK. Keep a `replace` directive available for local-checkout dev.
- Update CI: backend CI now does `go mod download` against the public SDK; nothing else changes.
- **Exit criteria:** `make ci-all` green when fetching SDK from GHCR-equivalent (GitHub module proxy); local dev with `replace github.com/orkestra-cc/orkestra-sdk => ../orkestra-sdk` works.

### Phase 5 — Extract first addon (`billing`) as proof-of-concept
**Why billing first:** largest addon by LOC (11.7K), most external dependencies (OpenAPI.com gateway), exercises every capability (config schema, collections, jobs, nav items, RBAC). If billing extracts cleanly, every smaller addon will.
- Move `internal/shared/openapiauth/` → `internal/addons/billing/openapi/` (Phase 0 decision).
- Create `orkestra-cc/addon-billing` repo via `git filter-repo --subdirectory-filter backend/internal/addons/billing`.
- Add `go.mod` declaring `module github.com/orkestra-cc/addon-billing`, requiring `orkestra-sdk vX.Y.Z`.
- In main repo: remove `backend/internal/addons/billing/`. Update `cmd/server/catalog_billing.go` to import the remote module. `go.work` lists both repos for local dev.
- Spin up `addon-billing` CI: `go test ./...` + a smoke build that links against a fake monolith stub.
- **Exit criteria:** `make build-billing` from main repo works; admin UI can enable/disable billing; FatturaPA polling still runs; integration tests still pass. Branch `orkestra-sdk-dev` merges to `dev` and is retired.

### Phases 6+ — Roll forward addon-by-addon
After Phase 5, each remaining addon is its own PR on `dev`. Suggested order (small/decoupled first):

1. `dev` — 442 LOC, no MongoDB
2. `compliance` — 1.7K, narrow
3. `documents` — 4.4K, depended on by billing (might want first)
4. `company` — 2.5K
5. `graph` — 1.8K
6. `aimodels` — 3.5K, depended on by rag/agents/sales
7. `rag` — 6.3K
8. `agents` — 3.0K
9. `sales` — 4.2K
10. `payments` — 2.2K
11. `subscriptions` — 3.3K
12. `identity` — 2.3K

Each extraction is one PR, one repo create, one commit in main bumping the dependency. Cumulative cost is roughly one day per addon once Phase 5 is done.

## 6. Versioning policy

- **SDK v0.x** until at least three addons are extracted and surviving real changes (`v0.x` allows breaking changes; we use this to shake out the surface).
- **SDK v1.0** declared when: (a) the `Module` minimal interface has not changed in 30 days, (b) at least three external addons are tracking it, (c) we have an integration test that compiles a stub addon and catches accidental breakage.
- **Post-v1:** breaking changes go through deprecation cycles — `Deprecated:` comments for one minor version, then removal in next major. Optional sub-interfaces (`HasInfraContainers`, etc.) are the *only* approved way to add new capabilities.
- **DTO compatibility:** all `iface.*` types follow "additive only" — new fields are optional or pointer-typed. Required-field additions are a major bump.

Addons declare SDK compatibility in their `module.go` via a sentinel:

```go
var _ sdkmodule.Module = (*Module)(nil)
const SDKMinVersion = "v1.2.0"   // queried by registry at boot, logged + emitted in /health
```

## 7. CI and developer workflow

**Local dev (touching all three repos):**
```
~/work/
  orkestra/             # main monorepo
  orkestra-sdk/         # SDK repo
  addon-billing/        # extracted addon

# go.work in ~/work/ links all three
go 1.25.10
use (
    ./orkestra/backend
    ./orkestra-sdk
    ./addon-billing
)
```
Optionally commit a `go.work` at `~/work/orkestra/go.work` with relative `../orkestra-sdk` paths — contributors `git clone` siblings and it just works.

**Backend CI:** `make ci-backend` unchanged externally; internally pulls SDK + addons via `go mod download` on a cache miss. Add `go.sum` to the cache key.

**SDK CI:** `go test ./...` + a compile-only smoke test (`testkit/stub-addon-test`) that imports the SDK as an external module would.

**Addon CI (per repo):** `go test ./...` against pinned SDK version. No backend dependency. A nightly cron in each addon repo tests against `orkestra-sdk@main` to catch breakage early.

**Cross-cutting change workflow** (the real ergonomic cost):
1. PR A → `orkestra-sdk` adding the new SDK feature
2. Tag `orkestra-sdk vX.Y.0`
3. PR B → `addon-billing` consuming it, bumping `go.mod`
4. Tag `addon-billing vA.B.0`
5. PR C → `orkestra` bumping both deps in `backend/go.mod`

Five commits across three repos for one logical change. Mitigate with a `scripts/cross-repo-pr.sh` helper that opens the three PRs from a single staging branch.

## 8. Risks & open questions

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| `module.ConfigService` API leaks too much MongoDB / Redis detail and locks us in | High | Phase 1 audit — narrow `ConfigService` to an interface; backend keeps concrete `MongoConfigService` implementing it |
| `iface.UserProvider` DTOs balloon over time, every field addition risks a major bump | Med | Stable subset under `iface/core.go`; experimental fields under `iface/experimental/` with explicit unstable marker |
| Addon → backend `internal/errors` deps are deeper than the grep shows | Med | Phase 1 task: compile each addon against an empty `internal/shared/errors` to find true reach |
| `go.work` not picked up by IDE tooling consistently (VSCode + gopls bugs) | Low | Document IDE setup; provide a `make dev-setup` that prints the relevant settings |
| Closed-source addons need a private Go proxy | Low (later) | Out of scope until first paying customer needs it; GitHub provides `GOPROXY` with token auth when needed |
| `cmd/ai-service/` consumes `internal/` packages too | High (known) | Same path — split happens for free since AI service is in same backend module today |
| `tenantscope` analyzer assumes monorepo paths | Med | Update analyzer to follow `go.work` and recognize SDK paths |
| Frontend admin module catalog (`/admin/modules`) shows hardcoded module list | Low | Already dynamic — pulls from registry. No change needed. |
| Profile builds (`make build-billing` etc.) reference internal paths in build tags | Low | Build tags are on `catalog_<addon>.go` which stays in monorepo; tag logic unchanged |

**Open questions to resolve before Phase 1 starts:**

1. Should the SDK module path be `github.com/orkestra-cc/orkestra-sdk` (matches org) or `orkestra.cc/sdk` (vanity, requires hosting)? **Recommendation:** GitHub path for v0, optional vanity later.
2. Does `ConfigService` belong in the SDK or behind an interface in the SDK with the impl in backend? **Recommendation:** interface in SDK (`module.ConfigService`), impl stays in backend. Addons consume the interface only.
3. How does an extracted addon ship its frontend? Mobile? **Recommendation:** out of scope for this plan — frontend stays in main monorepo for now; cross-repo frontend extraction is a separate plan.
4. Where do shared E2E tests live? **Recommendation:** main repo only — extracted addons keep unit + integration tests; cross-module E2E exercises the assembled monolith.

## 9. Acceptance criteria (top-level)

The SDK split is "done" when all of the following are true:

- `github.com/orkestra-cc/orkestra-sdk` exists, tagged ≥ v1.0.0, with public docs
- At least 3 addons are extracted to their own repos and consumed by the monolith
- `make ci-all` from the main repo green, including the full profile matrix
- A documented "write your own Orkestra addon" guide in the SDK README compiles and runs a hello-world addon outside the monorepo
- The AI sidecar (`cmd/ai-service`) builds against the SDK with no `internal/` imports
- No regression in `/admin/modules` runtime enable/disable

Sub-acceptance per phase is listed in section 5.

## 10. Phase 0 — Symbol-level inventory (delivered 2026-05-13)

Method: grep'd every selector-expression `<pkg>.Symbol` reference across `internal/addons/` + `internal/core/`, deduped per consumer × package × symbol. 647 distinct (consumer, package, symbol) triples found. False positives from local package-alias collisions filtered by re-verifying actual `import` statements.

### 10.1 Consumer reach per shared package

| Shared package      | Consumers (of 19) | Distinct symbols | Classification                  |
| ------------------- | ----------------- | ---------------- | ------------------------------- |
| `shared/iface`      | 19                | 80               | **SDK — wholesale**             |
| `shared/module`     | 19                | 73               | **SDK — wholesale**             |
| `shared/middleware` | 16                | 18               | **SDK — split**                 |
| `shared/config`     | 14                | (Config struct)  | **BACKEND — remove coupling**   |
| `shared/capability` | 7                 | 1                | **SDK**                         |
| `shared/utils`      | 3                 | 18               | **Split**                       |
| `shared/database`   | 2                 | 5                | **BACKEND — inject via Init**   |
| `shared/openapiauth`| 2                 | 5                | **Move to addon-billing**       |
| `shared/metrics`    | 2                 | 1                | **SDK**                         |
| `shared/tenantrepo` | 1                 | 3                | **SDK** _(usage anomaly — see 10.5)_ |
| `shared/errors`     | 1                 | 4                | **BACKEND**                     |
| `shared/geoip`      | 1                 | 6                | **BACKEND**                     |
| `shared/types`      | 1                 | 1                | **SDK — merge into iface**      |

### 10.2 SDK package layout (concrete)

```
orkestra-cc/orkestra-sdk/
├── iface/                  ← shared/iface verbatim (80 symbols, 1308 LOC)
├── module/                 ← shared/module verbatim (73 symbols, 3341 LOC)
├── capability/             ← shared/capability verbatim (1 symbol, 122 LOC)
├── tenantrepo/             ← shared/tenantrepo verbatim (3 symbols, 186 LOC)
├── modulegate/             ← subset of shared/middleware — see 10.3
├── ctxauth/                ← context getters from middleware — see 10.3
├── metrics/                ← shared/metrics (1 symbol)
└── testkit/                ← NEW — fake ServiceRegistry, in-memory ConfigService
```

Estimated total SDK surface: **~5,500 LOC** (1,308 + 3,341 + 122 + 186 + ~400 from middleware subset + ~50 small).

### 10.3 `shared/middleware` split

The 18 distinct symbols used by addons divide cleanly:

| Symbol                  | Goes to            | Reason                                           |
| ----------------------- | ------------------ | ------------------------------------------------ |
| `GetUserUUID`           | `sdk/ctxauth`      | Addons read auth identity from request context   |
| `GetUserEmail`          | `sdk/ctxauth`      | ”                                                |
| `GetTenantID`           | `sdk/ctxauth`      | ”                                                |
| `GetTenantRoles`        | `sdk/ctxauth`      | ”                                                |
| `GetSystemRole`         | `sdk/ctxauth`      | ”                                                |
| `GetSessionID`          | `sdk/ctxauth`      | ”                                                |
| `GetAMR`                | `sdk/ctxauth`      | ”                                                |
| `GetClientIP`           | `sdk/ctxauth`      | ”                                                |
| `GetDeviceInfo`         | `sdk/ctxauth`      | ”                                                |
| `IsImpersonating`       | `sdk/ctxauth`      | ”                                                |
| `IsMFAEnrolled`         | `sdk/ctxauth`      | ”                                                |
| `TenantKindFromContext` | `sdk/ctxauth`      | ”                                                |
| `WithAMR`               | `sdk/ctxauth`      | Test helper for setting context                  |
| `WithSessionID`         | `sdk/ctxauth`      | ”                                                |
| `WithTenantKind`        | `sdk/ctxauth`      | ”                                                |
| `RequireAuth`           | `sdk/modulegate`   | Route gate — every addon endpoint uses it        |
| `ModuleGate`            | `sdk/modulegate`   | Registry-driven enable/disable gate              |
| `ModuleEnabledChecker`  | `sdk/modulegate`   | Interface implemented by registry                |

Everything else in `shared/middleware` (`AuthMiddleware` impl, OAuth flow, JWT validation, CSRF, rate limit, audience checks, etc.) stays backend-private. Addons interact with auth state only through `ctxauth` getters.

### 10.4 `shared/utils` split

| Symbol(s)                                                                 | Goes to              |
| ------------------------------------------------------------------------- | -------------------- |
| `EncryptMFASecret`, `DecryptMFASecret`, `EncryptOAuthToken`, `DecryptOAuthToken` | _backend-private_    |
| `HashRefreshToken`, `SetRefreshTokenCookie`, `ClearRefreshTokenCookie`, `GetAllRefreshTokensFromCookies`, `GetRefreshTokenFromCookieByName` | _backend-private_    |
| `AuthDebug*` (4 symbols)                                                   | _backend-private_    |
| `GeneratePKCEChallengeFromVerifier`, `GenerateState`, `GenerateNonce`     | _backend-private_ (auth-flow specific) |
| `SecureRandomString`                                                       | `sdk/cryptoutil` (small new pkg) |
| `EscapeRegex`                                                              | `sdk/cryptoutil`     |
| `GetClientIP`                                                              | (duplicate — exists in middleware too; consolidate at `sdk/ctxauth.GetClientIP`) |

Net SDK additions from `utils`: ~50 LOC in a new `sdk/cryptoutil` package; rest stays backend.

### 10.5 Anomaly: `shared/tenantrepo` usage

`backend/CLAUDE.md` states: _"every addon repo must use these"_ — but only **1 of 19** consumers actually imports `shared/tenantrepo` (`identity`). The static analyzer at `backend/tools/tenantscope` enforces this rule but evidently 18 modules satisfy it by other means (raw `bson.M{"tenantUUID": ...}` queries that `tenantscope` recognizes, or scope helpers re-implemented locally).

**Action item — pre-Phase 1:** read `tools/tenantscope` to confirm whether modules are out-of-compliance or whether the rule is enforced differently than CLAUDE.md describes. If the rule is real but unenforced, file a separate tracking issue; this plan does not undertake the cleanup.

### 10.6 The `*config.Config` coupling (the big one)

**Every single one of 13 addons + auth** imports `github.com/orkestra/backend/internal/shared/config` and takes a `*config.Config` argument in its constructor (the global app config struct). This is the **largest hidden coupling** in the codebase relative to the SDK boundary, and it is not addressable by simply moving packages.

After the SDK split, an addon CANNOT import the backend's global config struct — it lives in a different module and the addon must not depend on backend internals. The fix is structural:

1. Each addon already declares a `ConfigSchema()` returning typed fields (string, int, secret, etc.).
2. The registry already stores values in MongoDB via `ConfigService` and caches in Redis.
3. **The addon's `Init(deps)` must receive a typed-config value derived from its own `ConfigSchema()` — not a pointer to the global struct.**

Concretely, where today an addon does:

```go
func NewModule(cfg *config.Config, ...) *Module {
    if cfg.Billing.OpenAPIToken == "" { ... }
}
```

post-split it does:

```go
func (m *Module) Init(deps sdkmodule.Deps) error {
    var cfg billingconfig.Settings
    if err := deps.Config.UnmarshalModule("billing", &cfg); err != nil { return err }
    if cfg.OpenAPIToken == "" { ... }
}
```

This is a **Phase 1 prerequisite** that must land before any package moves. It touches every addon's `module.go` and is the biggest single source of risk in the migration.

### 10.7 Updated phase plan

Phase 1 is now two sub-phases:

- **Phase 1a — Decouple from `*config.Config`.** For each addon, replace constructor arg with `Init(deps)` consuming a per-module typed-config via `ConfigService`. Land as 14 individual PRs (one per consumer). Each is independently revertible and test-covered. _Estimated effort: 5–10 working days._
- **Phase 1b — Reshape `Module` interface** (the original Phase 1: split into minimal + optional sub-interfaces). Single PR.

Phases 2–5 are unchanged from section 5.

### 10.8 Acceptance for Phase 0

- ✅ Per-package consumer count + distinct-symbol count established
- ✅ SDK / BACKEND / SPLIT / MOVE-TO-ADDON classification documented per package
- ✅ `shared/middleware` split into `sdk/modulegate` + `sdk/ctxauth` defined at symbol granularity
- ✅ `shared/utils` partitioned by symbol
- ✅ `*config.Config` coupling surfaced as Phase 1a — material rescope of next phase
- ✅ `tenantrepo` enforcement anomaly flagged for separate investigation

Phase 0 complete on `orkestra-sdk-dev` 2026-05-13.

## 11. Pre-Phase-1a checks (delivered 2026-05-13)

### 11.1 ConfigService — does `UnmarshalModule` exist? **No, but groundwork is solid.**

What exists today (`internal/shared/module/config_service.go` + `module.go:204-260`):

- `Dependencies` struct already passed to every module's `Init(deps *Dependencies)`. Contains `DB`, `RedisAdapter`, `Config *config.Config`, `Logger`, `Services *ServiceRegistry`, `ConfigService *ModuleConfigService`.
- Per-key typed getters: `deps.GetConfig(module, key) string`, `GetSecret`, `GetConfigBool`, `GetConfigInt`, `GetConfigDuration` — already in use by every addon's `module.go`.
- Backing store: `ModuleConfigService.GetValue(ctx, name, key)` and `GetSecret(...)` traverse active environment → schema fallback → env var.
- Schema is declarative: `ConfigSchema() []ConfigField` with field types `FieldString | FieldBool | FieldInt | FieldDuration | FieldSecret | FieldEnum | FieldStringList`.

**What's missing for the SDK split:** a typed-struct unmarshal helper. Today every addon writes:

```go
deps.GetSecret("billing", "apiKey"),
deps.GetConfig("billing", "fiscalID"),
deps.GetConfigBool("billing", "applySignature", false),
// ... × 15 keys
```

That's repetitive and easy to drift from the schema. Phase 1a should add **one new method** to `ConfigService`:

```go
// UnmarshalModule decodes the active-environment config of moduleName into v.
// Field tags drive the mapping; `secret:"true"` triggers decryption.
// Type mismatches fail loudly. Missing keys fall back to schema defaults / env vars.
func (s *ModuleConfigService) UnmarshalModule(ctx context.Context, moduleName string, v any) error
```

with usage:

```go
type billingSettings struct {
    APIKey         string        `module:"apiKey" secret:"true"`
    OAuthBaseURL   string        `module:"oauthBaseURL"`
    BearerToken    string        `module:"bearerToken" secret:"true"`
    FiscalID       string        `module:"fiscalID"`
    RecipientCode  string        `module:"recipientCode"`
    ApplySignature bool          `module:"applySignature"`
    PollInterval   time.Duration `module:"pollInterval"`
}

func (m *Module) Init(deps *module.Dependencies) error {
    var cfg billingSettings
    if err := deps.ConfigService.UnmarshalModule(deps.Context, "billing", &cfg); err != nil {
        return err
    }
    // use cfg.APIKey, cfg.PollInterval, ...
}
```

Implementation: ~80 LOC using `reflect` over the existing `ConfigField` schema. No external dependency. Encrypted fields are auto-detected from `FieldSecret` type — the addon doesn't need to declare `secret:"true"` separately if we read the schema; the struct tag is only needed when the field name differs.

**Decision: add `UnmarshalModule` in its own PR before any addon refactor.** Single self-contained change, ~150 LOC including tests. This is the **first PR of Phase 1a.**

### 11.2 `tenantscope` audit — anomaly resolved, no drift

`backend/tools/tenantscope/analyzer.go:328` matches scope helpers by **local identifier `tenantrepo`**, not by full import path:

```go
if pkg.Name != "tenantrepo" {
    return false
}
```

So when `shared/tenantrepo` moves to `sdk/tenantrepo`, the analyzer keeps working as long as addons import it under the same local name `tenantrepo` (idiomatic Go default). The CLAUDE.md statement _"every addon must use these"_ is enforced at call-site granularity — every `Find/Update/Delete` argument must derive from `tenantrepo.Scope/MustScope/ScopeAggregate`, OR have an `//tenantscope:allow <reason>` comment, OR be in `baseline.txt`. The "only 1 of 19 imports tenantrepo" finding in section 10.1 reflects that 18 modules satisfy invariant #1 via call-site allow-comments or baseline entries, not via the helper.

**SDK split implications:**

- `sdk/tenantrepo` package keeps the same exported names (`Scope`, `MustScope`, `ScopeAggregate`, `StampInsert`, `StampInsertM`). Addons importing it under `tenantrepo` still pass the analyzer unchanged.
- `baseline.txt` records file paths like `internal/addons/billing/repository/foo.go:42`. After billing extracts, these entries no longer match any source file in the monorepo and the analyzer ignores them (silent dead entries). Per-addon repos can run a slim `tenantscope` invocation with their own per-addon baseline if needed.
- **Action for Phase 5 (billing extract):** prune billing's lines from the main repo's `baseline.txt`. If the analyzer flags new drift inside the extracted billing repo, set up a baseline file there too.
- **No analyzer source change needed.** The `pkg.Name == "tenantrepo"` predicate is path-agnostic by design.

The "drift" theory in section 10.5 is wrong. Removed as a concern.

### 11.3 `Deps` shape — locked

The current `module.Dependencies` (line 204–212 in `shared/module/module.go`) is close to right but has three problems for the SDK boundary:

1. `Config *config.Config` is a pointer to a backend-private struct — extracted addons cannot import it.
2. No `context.Context` field — every helper does `context.Background()`, which is fine for boot but wrong for shutdown.
3. Optional capabilities (container manager) aren't represented; they leak through other channels.

**Locked-in SDK shape (`sdk/module.Deps`):**

```go
package module

// Deps is the dependency bundle the registry passes to every module's Init.
// It is the single boundary across which the backend exposes infrastructure
// to addons. Anything an addon needs that isn't here goes through Services
// or Platform.
type Deps struct {
    // Lifecycle
    Ctx    context.Context  // honored for shutdown; cancelled when registry stops
    Logger *slog.Logger

    // Infrastructure handles (concrete; addons depend on driver types only)
    Mongo *mongo.Database
    Redis RedisClient        // SDK interface wrapping the subset addons use

    // SDK services
    Services *ServiceRegistry
    Config   ConfigService    // SDK interface, see 11.1
    Platform PlatformInfo     // SDK interface: IsProduction(), BaseURL(), Environment(), etc.

    // Optional — populated only for modules implementing HasInfraContainers
    Container ContainerManager  // nil when CONTAINER_CONTROL_ENABLED=false
}

// ConfigService is the SDK-visible subset of the backend's ModuleConfigService.
// Backend's *ModuleConfigService implements this interface.
type ConfigService interface {
    GetString(ctx context.Context, module, key string) string
    GetBool(ctx context.Context, module, key string, fallback bool) bool
    GetInt(ctx context.Context, module, key string, fallback int) int
    GetDuration(ctx context.Context, module, key string, fallback time.Duration) time.Duration
    GetSecret(ctx context.Context, module, key string) string
    UnmarshalModule(ctx context.Context, module string, v any) error  // new in 11.1
    IsEnabled(ctx context.Context, module string) bool
}

// PlatformInfo is the SDK-visible subset of *config.Config — only fields
// addons legitimately need. Backend implements this on *config.Config.
type PlatformInfo interface {
    IsProduction() bool
    Environment() string  // "dev" | "staging" | "production"
    BaseURL() string
}

// RedisClient is the subset of redis operations used by addons.
// Keeps the SDK off the upstream redis-go module's full surface.
type RedisClient interface {
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    Del(ctx context.Context, keys ...string) error
    Exists(ctx context.Context, key string) (bool, error)
    // ... add more as needed; expand cautiously
}
```

**Rationale for each decision:**

- `Config *config.Config` → `Platform PlatformInfo`: only 1 addon reads `cfg.IsProduction()` directly; the rest read per-module nested structs (`cfg.Billing`, `cfg.RAG`, etc.), which become typed addon configs via `UnmarshalModule`. The 13 nested-struct reads are the actual coupling — `PlatformInfo` retires the last legitimate need for `*config.Config`.
- `Redis RedisClient` interface (not `*RedisClientAdapter`): keeps the upstream redis SDK version out of the contract. Backend can swap clients without breaking addons.
- `Mongo *mongo.Database` stays concrete: the official Go driver's `*Database` is already an interface-rich type (it has internal interfaces for cursors, etc.), and abstracting it would force every addon to re-derive bson handling. Pragmatic carve-out.
- `Container ContainerManager` is optional and nil unless the module implements `HasInfraContainers`: keeps the basic Deps shape narrow.
- `Ctx context.Context` enables clean shutdown — modules can pass it into long-running jobs.

**What this leaves behind in `*config.Config`:** OAuth client IDs/secrets, JWT signing keys, CORS allowlists, session config, IP allowlists, BaseURL, environment flags, and the (incorrectly named) per-addon nested structs that should never have lived there. The per-addon nested structs are removed in Phase 1a; everything else is backend-private and accessed only by core modules.

### 11.4 Updated Phase 1a sequencing

Now concrete with the three checks resolved:

1. **PR-1a-0 — add `UnmarshalModule`.** Implement on `*ModuleConfigService` with struct-tag + schema-driven reflection. Tests cover all field types incl. `FieldSecret` decryption and `FieldEnum` validation. ~150 LOC. **No addon changes yet.**
2. **PR-1a-1 — convert `dev` addon** (442 LOC, smallest). Template PR — defines the conversion recipe other PRs copy. Removes `cfg *config.Config` from constructor, uses `deps.Config.UnmarshalModule("dev", &cfg)`.
3. **PR-1a-2 .. PR-1a-13** — same conversion for each remaining addon in increasing complexity:
   - `compliance` → `company` → `payments` → `subscriptions` → `identity` → `documents` → `graph` → `aimodels` → `sales` → `agents` → `rag` → `billing`
   - `auth` (core, but uses `*config.Config` heavily) is the last and largest — defer to a separate Phase 1c if needed.
4. **PR-1a-14 — drop `Config *config.Config` from `Dependencies`.** Replace with `Platform PlatformInfo`. By this point no addon needs the global struct; only core modules do, and they can keep importing `internal/shared/config` directly until they're SDK-fied.

After PR-1a-14 lands, the SDK boundary on configuration is clean: addons consume `ConfigService` + `PlatformInfo` interfaces only. Phases 1b (Module interface reshape) and beyond can proceed.

## 12. Next concrete step

Open **PR-1a-0** on `orkestra-sdk-dev`: implement `(*ModuleConfigService).UnmarshalModule(ctx, name, &v)` with reflection over `ConfigField` schemas + struct tags. Tests must cover:

- Every `ConfigFieldType` (string, bool, int, duration, secret, enum, stringList)
- Decryption of `FieldSecret` values
- `FieldEnum` rejection of out-of-options values
- Schema-default fallback when key absent
- Env-var fallback when key absent and no schema default
- Struct-tag override (`module:"customKey"`)
- Unexported struct fields ignored gracefully
- Error on type mismatch (e.g. schema `FieldString`, target `int`)

Estimated effort: half a day. After merge, PR-1a-1 (dev addon conversion) can follow same day.
