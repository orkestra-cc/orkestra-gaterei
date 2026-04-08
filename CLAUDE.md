# ORKESTRA

Professional business and service management system — electronic invoicing, AI sales intelligence, RAG pipeline, document generation, and operational reporting.

## Tech Stack

| Layer              | Technology                                                         |
| ------------------ | ------------------------------------------------------------------ |
| **Backend**        | Go 1.25.1, Huma v2 (OpenAPI-first), 13 self-contained modules      |
| **Frontend**       | React 19, TypeScript 5.9, Vite 7, Redux Toolkit, TanStack Table    |
| **Mobile**         | Flutter 3.35+, Dart, Riverpod                                      |
| **Database**       | MongoDB 8.0, Redis 8.2                                             |
| **Infrastructure** | Docker Compose (dev/staging/prod), GitHub Actions CI               |
| **Auth**           | OAuth 2.1 (Google, Apple, GitHub, Discord), RS256 JWT, 6-role RBAC |

## Architecture

Single Go binary with **13 self-contained modules** managed by a module registry. Each module declares its own routes, collections, config schema, nav items, health checks, and service dependencies. Modules communicate through a `ServiceRegistry` using interfaces defined in `shared/iface/`.

**Key components** (`backend/internal/shared/module/`):

- **Module interface** — lifecycle contract every module implements (Init, RegisterRoutes, Start, Stop, HealthCheck)
- **ModuleRegistry** — initializes modules in dependency order, tracks failures, gates routes for disabled modules
- **ServiceRegistry** — typed key-value store for cross-module service sharing (`GetTyped[T]`, `MustGetTyped[T]`)
- **ConfigService** — DB-backed (MongoDB) + Redis-cached (30s TTL) module configuration with AES-256-GCM encrypted secrets
- **shared/iface** — consumer-facing interfaces (UserProvider, PDFProvider, GraphProvider, AIModelProvider, RAGQueryProvider, JWTProvider) that prevent direct cross-module imports

**Admin API**: `GET/PATCH /v1/admin/modules`, `GET /v1/admin/modules/health` — runtime enable/disable, config updates, health checks. Frontend at `/admin/modules`.

## Module Map

### Backend Modules (`backend/internal/`)

| Module         | Purpose                                                                                                      | Category   |
| -------------- | ------------------------------------------------------------------------------------------------------------ | ---------- |
| **auth**       | OAuth 2.1, JWT, sessions, RBAC                                                                               | core       |
| **user**       | User CRUD, role management, document tracking                                                                | core       |
| **navigation** | Dynamic menu from module NavItems                                                                            | core       |
| **reporting**  | Analytics dashboards and aggregations                                                                        | core       |
| **billing**    | Italian electronic invoicing (FatturaPA/SDI) — [docs](backend/internal/billing/CLAUDE.md)                    | external   |
| **documents**  | PDF generation via Gotenberg — [docs](backend/internal/documents/CLAUDE.md)                                  | external   |
| **company**    | Italian business registry lookup (OpenAPI) — [docs](backend/internal/company/CLAUDE.md)                      | external   |
| **graph**      | Memgraph knowledge graph — [docs](backend/internal/graph/CLAUDE.md)                                          | external   |
| **aimodels**   | Multi-provider AI model management (Ollama, OpenAI, Anthropic) — [docs](backend/internal/aimodels/CLAUDE.md) | toggleable |
| **rag**        | Document ingestion + retrieval-augmented generation — [docs](backend/internal/rag/CLAUDE.md)                 | toggleable |
| **agents**     | Hindsight AI agents with RAG context — [docs](backend/internal/agents/CLAUDE.md)                             | toggleable |
| **sales**      | AI-driven prospect analysis and scoring                                                                      | toggleable |
| **dev**        | Dev token generation (disabled in production)                                                                | toggleable |

### Other Modules

- **[`/frontend/`](frontend/CLAUDE.md)** — React 19 admin dashboard (port 8080)
- **[`/mobile/`](mobile/CLAUDE.md)** — Flutter cross-platform app
- **[`/docker/`](docker/CLAUDE.md)** — Docker Compose configs (dev/staging/prod/infra)
- **[`/docs/Authentication_flow.md`](docs/Authentication_flow.md)** — OAuth 2.1 + RBAC details

## Quick Start

```bash
# From project root — interactive deployment
./deploy.sh

# Or manually:
cd docker
docker compose -f docker-compose.infra.yml up -d   # MongoDB, Redis, Gotenberg, Hindsight
docker compose -f docker-compose.dev.yml up -d      # Backend (AIR) + Frontend (Vite)

# Frontend: http://localhost:8080
# Backend API: http://localhost:3000
# API Docs: http://localhost:3000/docs
```

## Assistant Rules

### Do

- **Read the module's CLAUDE.md** before modifying any module — each has specific patterns and constraints
- **Use the module system** when adding new functionality: implement the `Module` interface, register in `main.go`, declare collections/nav/config via the module methods
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
