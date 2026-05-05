# ORKESTRA

**Orkestra is a modular, multi-tenant orchestrator platform.** It is not a single-purpose business app — it is a host that exposes business capabilities (invoicing, billing, AI sales, RAG, documents, payments, etc.) as pluggable modules/addons, and manages *who* can consume those capabilities across a two-tier tenancy model.

## Tenancy Model

Orkestra operates on **two distinct tiers of tenants**. Understanding this distinction is load-bearing for every design decision — data isolation, RBAC scope, billing, and module activation all depend on which tier a request is acting in.

### Tier 1 — Internal tenants (operator side)

The companies that **run Orkestra** (one or more of "our" organizations). For each internal tenant the platform manages:

- Internal users, roles, and RBAC
- The internal company's own electronic invoicing (FatturaPA/SDI), billing, documents
- Which modules/addons are enabled for that tenant
- Operational admin (module config, audit logs, compliance evidence)

### Tier 2 — External client tenants (customer side)

**External clients register on the platform**, and each external client can itself be a multi-tenant organization (multiple sub-tenants / workspaces under one client). For each external client the platform manages:

- Client registration and onboarding
- The client's own users, roles, sub-tenants
- **Subscriptions to the services Orkestra exposes** (via the `subscriptions` + `payments` modules — Stripe-backed recurring billing)
- Usage of the subscribed services (AI agents, RAG, document generation, etc.) scoped to the client's data

The `subscriptions` and `payments` modules are **not ordinary feature addons** — they are the mechanism by which Tier-2 clients consume Tier-1-hosted services. Treat them as architecturally load-bearing.

### Implications for contributors

- Every new endpoint must declare **which tier it serves** (internal operator, external client, or both) and enforce org-scoped RBAC accordingly.
- Every new collection/table must carry a tenant scope (internal org ID *or* external client org ID) and be indexed/queried with that scope — never cross-tenant by default.
- Module enable/disable is **per internal tenant**, but service consumption is gated **per external client subscription**. Do not conflate the two.
- When in doubt about which tier owns a resource, ask before implementing.

## Tech Stack

| Layer              | Technology                                                         |
| ------------------ | ------------------------------------------------------------------ |
| **Backend**        | Go 1.25.1, Huma v2 (OpenAPI-first), 12 self-contained modules      |
| **Frontend**       | React 19, TypeScript 5.9, Vite 7, Redux Toolkit, TanStack Table    |
| **Mobile**         | Flutter 3.35+, Dart, Riverpod                                      |
| **Database**       | MongoDB 8.0, Redis 8.2                                             |
| **Infrastructure** | Docker Compose (dev/staging/prod), GitHub Actions CI               |
| **Auth**           | Email/password (argon2id) + OAuth 2.1 (Google, Apple, GitHub, Discord), RS256 JWT, 6-role RBAC |

## Architecture

**Plugin architecture**: 4 core modules (user, notification, auth, navigation) always load. All other modules are **optional** — every optional module is instantiated and initialized at boot, and operators flip individual modules on/off at `/admin/modules` (state persisted in the `module_configs` MongoDB collection). The registry resolves initialization order automatically from each module's `Dependencies()` declaration.

**Key components** (`backend/internal/shared/module/`):

- **Module interface** — lifecycle contract every module implements (Init, RegisterRoutes, Start, Stop, HealthCheck)
- **ModuleRegistry** — `RegisterAll()` with topological sort from `Dependencies()`; tracks failures, gates routes for disabled modules
- **ServiceRegistry** — typed key-value store for cross-module service sharing (`GetTyped[T]`, `MustGetTyped[T]`)
- **ConfigService** — DB-backed (MongoDB) + Redis-cached (30s TTL) module configuration with AES-256-GCM encrypted secrets
- **shared/iface** — consumer-facing interfaces (UserProvider, NotificationSender, PDFProvider, GraphProvider, AIModelProvider, RAGQueryProvider, JWTProvider) that prevent direct cross-module imports
- **RoleMiddleware** — interface (`module.go`) for RBAC route protection, satisfied by both `AuthMiddleware` (monolith) and `JWTValidator` (AI service)
- **Module catalog** (`cmd/server/catalog.go`) — maps module names to factory functions; all optional modules are always instantiated and initialized at boot for runtime enable/disable without restart

**Admin API**: `GET/PATCH /v1/admin/modules`, `GET /v1/admin/modules/health`, `GET/PATCH /v1/admin/modules/{name}/environments/{env}`, `PUT /v1/admin/modules/{name}/active-environment` — runtime enable/disable (hot-reload), config updates, per-environment config profiles (sandbox/production), health checks. Frontend at `/admin/modules` (list) and `/admin/modules/:name` (detail).

### Module Loading

All optional modules are always **instantiated, initialized, and routed** at boot — regardless of enabled state. Routes for disabled modules are gated by `ModuleGate` middleware (returns 503). Only enabled modules have their `Start()` method called (background jobs, polling, etc.).

**Enabling/disabling at runtime:** The admin API (`PATCH /v1/admin/modules/{name}`) calls `StartModule()`/`StopModule()` on the registry. The module starts or stops immediately — no restart required. Dependency constraints are enforced: you cannot disable a module that another running module depends on (returns 409).

**Which modules start at boot** is determined by the `module_configs` collection in MongoDB (set via admin UI). On first boot of a brand-new install the document is seeded from each module's `ConfigSchema().EnvVar` — see `docker/CLAUDE.md` for the per-bucket split.

The registry topologically sorts modules by `Dependencies()` so initialization order is always correct.

### AI Service Sidecar (Optional Split)

The AI module chain (graph, aimodels, rag, agents) can optionally run as a **standalone sidecar service** (`cmd/ai-service/`) separate from the monolith. Controlled by the `AI_SERVICE_URL` env var on the monolith:

- **Empty (default)**: All 12 modules run in-process in the monolith. No change from baseline.
- **Set** (e.g., `http://orkestra-ai:3100`): The monolith skips registering graph/aimodels/rag/agents modules and instead registers `RemoteAIModelProvider` + `RemoteRAGQueryProvider` (HTTP clients in `shared/remote/`) under the same `ServiceRegistry` keys. Consumer modules like `sales` use the same `GetTyped` pattern — zero code changes.

**How the split works:**

```
┌─ Monolith (port 3000) ──────────────┐   ┌─ AI Service (port 3100) ──────────┐
│ auth, user, navigation, billing,     │   │ graph, aimodels, rag, agents      │
│ documents, company, sales, dev       │   │                                    │
│                                      │   │ Same Go module (backend/),         │
│ sales → RemoteAIModelProvider ───────┼──→│ separate binary (cmd/ai-service/) │
│         (HTTP to AI service)         │   │                                    │
│                                      │   │ JWT validated with public key only │
│                                      │   │ (JWTValidator, no auth module dep) │
└──────────────────────────────────────┘   └────────────────────────────────────┘
```

**Key files:**

| File | Purpose |
|------|---------|
| `backend/cmd/ai-service/main.go` | AI service entry point — boots 4 modules with `JWTValidator` |
| `backend/internal/shared/middleware/jwt_validator.go` | Lightweight RS256 JWT validation (public key only) |
| `backend/internal/shared/remote/` | `RemoteAIModelProvider`, `RemoteRAGQueryProvider`, remote `EmbeddingProvider`/`LLMProvider` |
| `backend/internal/addons/aimodels/internal_routes.go` | Internal API: `/v1/internal/ai/embed`, `/complete`, `/embedding-info`, `/llm-info` |
| `backend/internal/addons/rag/internal_routes.go` | Internal API: `/v1/internal/rag/query` (with `documentUUIDs` scoping) |
| `backend/Dockerfile.ai-service` | Multi-stage build for AI service binary |
| `docker/docker-compose.ai.yml` | Dev container for AI service |

**Running split mode (dev):**

```bash
cd docker
docker compose -f docker-compose.infra.yml up -d
AI_SERVICE_URL=http://orkestra-ai-dev:3100 docker compose -f docker-compose.dev.yml --env-file .env up -d
docker compose -f docker-compose.ai.yml --env-file .env up -d
```

**Design constraints for the split:**

- Both binaries live in the **same Go module** (`backend/`) — no code duplication, shared `internal/` packages
- The AI service uses `JWTValidator` (public key only) instead of `AuthMiddleware` (which depends on the auth module). Both satisfy `module.RoleMiddleware`
- Internal API endpoints (`/v1/internal/*`) are the service-to-service contract. They serialize the `iface.AIModelProvider` and `iface.RAGQueryProvider` method calls as HTTP request/response
- Streaming (`/v1/rag/query/stream`) goes **directly** from frontend → AI service, never proxied through the monolith
- The feature flag is fully backward compatible and K8s-ready (service DNS, Ingress routing, separate `Deployment` with independent scaling)

## Module Map

### Backend Modules (`backend/internal/`)

**Core (always loaded):**

| Module           | Purpose                                                                                                           |
| ---------------- | ----------------------------------------------------------------------------------------------------------------- |
| **user**         | User CRUD, role management, document tracking                                                                     |
| **notification** | Email delivery, templates, user preferences, unsubscribe tokens — [docs](backend/internal/core/notification/CLAUDE.md) |
| **auth**         | Email/password (argon2id) + OAuth 2.1, JWT, sessions, RBAC                                                        |
| **navigation**   | Dynamic menu from module NavItems                                                                                 |

Load order (topologically sorted by `Dependencies()`): `user` → `notification` → `auth` → `navigation`. Auth depends on notification (optional at runtime) so it can deliver verification and password-reset emails.

**Optional (toggled at `/admin/modules`; all instantiated at boot):**

| Module         | Purpose                                                                                                      | Depends on       |
| -------------- | ------------------------------------------------------------------------------------------------------------ | ---------------- |
| **billing**    | Italian electronic invoicing (FatturaPA/SDI) — [docs](backend/internal/addons/billing/CLAUDE.md)                    | documents        |
| **documents**  | PDF generation via Gotenberg — [docs](backend/internal/addons/documents/CLAUDE.md)                                  | —                |
| **company**    | Italian business registry lookup (OpenAPI) — [docs](backend/internal/addons/company/CLAUDE.md)                      | —                |
| **graph**      | Memgraph knowledge graph — [docs](backend/internal/addons/graph/CLAUDE.md)                                          | —                |
| **aimodels**   | Multi-provider AI model management (Ollama, OpenAI, Anthropic) — [docs](backend/internal/addons/aimodels/CLAUDE.md) | —                |
| **rag**        | Document ingestion + retrieval-augmented generation — [docs](backend/internal/addons/rag/CLAUDE.md)                 | graph, aimodels  |
| **agents**     | Hindsight AI agents with RAG context — [docs](backend/internal/addons/agents/CLAUDE.md)                             | auth, aimodels   |
| **sales**      | AI-driven prospect analysis and scoring                                                                      | aimodels         |
| **subscriptions** | Recurring AI-services catalog, clients, subscriptions, activity log — [docs](backend/internal/addons/subscriptions/CLAUDE.md) | —                |
| **payments**   | Stripe gateway — charges, refunds, webhooks — [docs](backend/internal/addons/payments/CLAUDE.md)                    | —                |
| **onboarding** | Anonymous self-service signup — creates external tenant + owner user in one public call                      | auth, tenant     |
| **dev**        | Dev token generation (disabled in production)                                                                | auth             |

### Other Modules

- **[`/frontend/`](frontend/CLAUDE.md)** — React 19 operator console / Tier-1 admin dashboard (port 8080, host `console.localhost`)
- **[`/frontend-client/`](frontend-client/CLAUDE.md)** — React 19 Tier-2 client demo SPA — consumes the ADR-0003 client API surface (port 8081, host `client.localhost`)
- **[`/mobile/`](mobile/CLAUDE.md)** — Flutter cross-platform app
- **[`/docker/`](docker/CLAUDE.md)** — Docker Compose configs (dev/staging/prod/infra)
- **[`/docs/Authentication_flow.md`](docs/Authentication_flow.md)** — Email/password + OAuth 2.1 + RBAC details

## Quick Start

### Minimal profile (recommended for first boot)

Boots only the core modules — user, notification, auth, navigation, plus the `dev` token generator — with MongoDB and Redis. Four containers total, no Gotenberg/Memgraph/Hindsight/Ollama, uses only publicly available Docker images so it runs on any VM without registry authentication. Ports are non-standard (3050/8050/27050/6350) so it can run alongside the full dev stack or other Docker projects without conflicts.

The notification module boots in `noop` mode by default — verification and password-reset emails are logged to the backend stdout rather than delivered. To send real mail, set `NOTIFICATION_EMAIL_PROVIDER=smtp` plus `SMTP_HOST/PORT/USERNAME/PASSWORD` and `NOTIFICATION_EMAIL_FROM` in the env file, or configure them at `/admin/modules` once you are logged in.

```bash
cd docker
docker network create orkestra-network   # first time only
docker compose -f docker-compose.minimal.yml --env-file .env.minimal up -d

# Frontend: http://localhost:8050
# Backend API: http://localhost:3050
# API Docs: http://localhost:3050/docs

# Generate an administrator token for first login (run from project root):
ORKESTRA_API_URL=http://localhost:3050 ./scripts/devtoken.sh administrator
```

Every optional module is loaded on boot regardless of profile. To enable additional modules, log in and toggle them at `/admin/modules` — the registry hot-reloads without a restart and auto-resolves dependencies.

### Full development stack

Adds Gotenberg (PDF), optionally Memgraph/Hindsight, and uses the Chainguard hardened images with AIR hot reload for Go development. Requires access to the `dhi.io` registry.

```bash
# From project root — interactive TUI (pick "Full stack" from the profile menu)
./orkestra.sh

# Or manually:
cd docker
docker compose -f docker-compose.infra.yml up -d   # MongoDB, Redis, Gotenberg, Hindsight
docker compose -f docker-compose.dev.yml up -d      # Backend (AIR) + Frontend (Vite)

# Optional: run AI modules as a separate service
docker compose -f docker-compose.ai.yml up -d       # AI Service (port 3100)
# Set AI_SERVICE_URL=http://orkestra-ai-dev:3100 on the backend to enable split mode
```

## Assistant Rules

### Do

- **Read the module's CLAUDE.md** before modifying any module — each has specific patterns and constraints
- **Use the module system** when adding new functionality: implement the `Module` interface, add to the `optionalModules` catalog in `cmd/server/catalog.go`, declare collections/nav/config via the module methods
- **Use `shared/iface`** for cross-module dependencies — never import another module's `services/` or `repository/` package from a `module.go` wiring file
- **Validate and sanitize** all user inputs; implement RBAC on every endpoint (ask for required permissions)
- **Follow the auth patterns** in [Authentication_flow.md](docs/Authentication_flow.md) for any auth-related changes

### Do Not

- **Never start servers manually** — backend and frontend run in Docker with hot reload (AIR + Vite)
- **Never expose secrets** in logs, API responses, or Git — module secrets use AES-256-GCM encryption via ConfigService
- **Never import cross-module** service/repository packages in `module.go` — use `shared/iface` interfaces + `ServiceRegistry` typed getters instead
- **Never bypass RBAC** — all admin endpoints require `administrator` role; all protected endpoints require auth middleware

### WSL2 Development Caveat

AIR file watcher does not reliably detect changes on the Windows filesystem mounted in WSL2. If backend changes don't take effect after saving, manually rebuild inside the container:

```bash
docker exec orkestra-backend-dev go build -o /app/tmp/main ./cmd/server/
docker restart orkestra-backend-dev
```

### CI/CD

GitHub Actions workflows (`.github/workflows/`) run on PR and push to `dev`/`main`:

- `backend.yml` — lint (golangci-lint), test (80% coverage gate), build, Docker push to GHCR
- `frontend.yml` — TypeScript check, ESLint, build, Docker push
- `mobile.yml` — Flutter analyze, test, APK build
