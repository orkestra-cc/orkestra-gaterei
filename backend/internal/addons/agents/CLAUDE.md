# Agents Module

AI agent system powered by [Hindsight](https://hindsight.vectorize.io/) persistent memory. Each project gets a scoped Hindsight memory bank and queries the existing graphRAG for document retrieval.

## Architecture

```
handlers/
  ├── project_handler.go         ← Project CRUD endpoints
  ├── agent_handler.go           ← Agent query + conversation endpoints
  └── personal_agent_handler.go  ← Per-user personal agent endpoints
services/
  ├── project_service.go         ← Project CRUD + Hindsight bank lifecycle
  ├── agent_service.go           ← Query orchestration: RAG → Retain → Reflect
  ├── personal_agent_service.go  ← Per-user auto-provisioning + delegation
  ├── hindsight_client.go        ← Thin wrapper around Hindsight Go SDK
  └── rag_bridge.go              ← Consumer interface for scoped RAG queries
models/
  ├── project.go                 ← Project MongoDB model (incl. IsPersonal fields)
  ├── conversation.go            ← Conversation + Message models
  ├── role_config.go             ← Persona definitions + RBAC validation
  ├── dto.go                     ← Huma request/response DTOs
  └── personal_dto.go            ← Personal agent DTOs
repository/
  ├── project_repository.go      ← MongoDB: agent_projects (incl. GetPersonalByUserUUID)
  └── conversation_repository.go ← MongoDB: agent_conversations
routes.go                        ← RegisterProjectRoutes, RegisterQueryRoutes, RegisterAdminRoutes, RegisterPersonalAgentRoutes
```

## Data Storage

| Data | Storage | Collection |
|------|---------|------------|
| Project metadata | MongoDB | `agent_projects` |
| Conversations | MongoDB | `agent_conversations` |
| Agent memory | Hindsight | Bank per project |
| Document chunks | Memgraph | Via RAG module |

## Query Orchestration Flow

```
1. Validate project + resolve persona
2. Merge project AgentSettings with persona defaults (system prompt, directives, disposition)
3. RAG Phase: QueryWithScope(question, project.documentUUIDs)
4. Retain Phase (async): Store RAG results in Hindsight bank
5. Reflect Phase: Hindsight combines persistent memory + RAG context + persona directives
6. Capture token usage from Hindsight response (inputTokens, outputTokens, totalTokens)
7. Save conversation + metadata to MongoDB
8. Return answer + sources + metadata + token counts
```

## Personas

Query-time behavior profiles (not RBAC roles). Users select any persona at or below their system role level.

| Persona | Focus | Min RBAC |
|---------|-------|----------|
| developer | Technical details, raw data | developer |
| administrator | Comprehensive, compliance | administrator |
| auditor | Evidence-based, compliance | administrator |
| manager | Summaries, business impact | manager |
| guest | General overviews | guest |

## Endpoints

### Projects (`/v1/agents/projects`) — manager role
| Method | Path | Description |
|--------|------|-------------|
| POST | `/` | Create project + Hindsight bank |
| GET | `/` | List projects |
| GET | `/{uuid}` | Get project |
| PATCH | `/{uuid}` | Update project |
| DELETE | `/{uuid}` | Delete project + bank |
| POST | `/{uuid}/documents` | Add documents |
| DELETE | `/{uuid}/documents` | Remove documents |
| PATCH | `/{uuid}/filters` | Update filters |
| GET | `/{uuid}/settings` | Get agent settings |
| PATCH | `/{uuid}/settings` | Update agent settings |

### Query (`/v1/agents/projects`) — operator role
| Method | Path | Description |
|--------|------|-------------|
| POST | `/{uuid}/query` | Query agent |
| POST | `/{uuid}/conversations` | New conversation |
| GET | `/{uuid}/conversations` | List conversations |
| GET | `/conversations/{uuid}` | Get conversation |
| DELETE | `/conversations/{uuid}` | Delete conversation |

### Personal Agent (`/v1/agents/personal`) — any authenticated user (guest+)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Get or auto-create personal agent |
| POST | `/query` | Query personal agent |
| POST | `/documents` | Add documents to scope |
| DELETE | `/documents` | Remove documents from scope |
| GET | `/settings` | Get personal agent settings |
| PATCH | `/settings` | Update personal agent settings |
| GET | `/conversations` | List personal conversations |
| GET | `/conversations/{uuid}` | Get conversation |
| DELETE | `/conversations/{uuid}` | Delete conversation |

### Admin (`/v1/agents`) — administrator role
| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{uuid}/bank` | Hindsight bank info |
| GET | `/health` | Hindsight health check |

## Personal Agent

Every user gets a personal AI agent backed by an auto-provisioned project. The first `GET /v1/agents/personal` call creates a project with `isPersonal: true` and `personalUserUuid: <userUUID>`, plus a Hindsight bank (`{namespace}-personal-{userUUID}`).

Key design decisions:
- Personal projects are **excluded** from `ListProjects` (filtered by `isPersonal != true`)
- Personal projects **cannot be deleted** via project management endpoints
- Conversations are **ownership-verified** (user can only access their own)
- **Race condition safe**: unique sparse index on `personalUserUuid` prevents double-creation
- Uses the same query orchestration flow (RAG → Retain → Reflect) as project agents

## Configuration

Module config (live-editable at `/admin/modules/agents`; seeded from the env vars below on first boot):

| Key | Env var | Default | Notes |
|---|---|---|---|
| `hindsightURL` | `HINDSIGHT_URL` | `http://orkestra-hindsight:8888` | Internal URL the module uses to reach Hindsight |
| `hindsightNamespace` | `HINDSIGHT_NAMESPACE` | `orkestra` | Bank namespace prefix |
| `hindsightImage` | `HINDSIGHT_IMAGE` | `ghcr.io/vectorize-io/hindsight:latest` | Image used when the backend manages the container |

### LLM credentials come from `aimodels`

The Hindsight container's LLM provider/model/API key are **not** configured here. They are read from the **AI Models** module's default LLM (set via `/admin/aimodels` → mark a model as default). The agents module declares `aimodels` as a hard dependency, and its `Preflight()` refuses to start Hindsight unless:

- the `aimodels` module is enabled, and
- at least one model is marked `isDefault: true` with `modelType: "llm"`, and
- the model has a valid API key (or is a local provider).

Translation rules fed into the Hindsight container:

| aimodels provider | `HINDSIGHT_API_LLM_PROVIDER` | `HINDSIGHT_API_LLM_BASE_URL` | `HINDSIGHT_API_LLM_API_KEY` |
|---|---|---|---|
| `openai` | `openai` | from model (optional) | from model |
| `anthropic` | `anthropic` | from model (optional) | from model |
| `gemini` | `gemini` | from model (optional) | from model |
| `ollama` | `openai` (reached via OpenAI-compat `/v1`) | model's baseURL, or `http://orkestra-ollama:11434/v1` | `ollama` (placeholder — Ollama ignores it) |

### Container Lifecycle

The agents module owns the `orkestra-hindsight` container via `InfraContainers()`. When you enable the module in the admin UI, the shared container manager (see `shared/container/`) resolves the default LLM from `aimodels`, creates and starts the container with those credentials, waits for `/health` to come up, and only then marks the module as started. Disabling the module stops the container. Hindsight is no longer declared in the auto-start set of `docker-compose.infra.yml` — it lives behind the `manual-only` profile there so the compose file still documents the image/ports/volumes for reference.

If the admin changes the default LLM while agents is running, the change takes effect on the next agents toggle-off → toggle-on (the Hindsight container's env vars are only written at container create time).

## Per-Project Agent Settings

Stored in `AgentSettings` on the Project model. All optional — unset fields fall back to persona defaults.

| Setting | Effect |
|---------|--------|
| `systemPrompt` | Overrides persona's default system context |
| `directives` | Extra rules merged on top of persona directives |
| `skepticism` | 1=trusting, 5=strict to docs (0=persona default) |
| `literalism` | 1=creative, 5=literal (0=persona default) |
| `empathy` | 1=detached, 5=helpful/warm (0=persona default) |
| `temperature` | "precise", "balanced", "creative" |
| `language` | Force response language (e.g. "en", "it") |
| `maxTokens` | Response length budget |

## Token Usage Tracking

Each assistant message records Hindsight Reflect token usage:
- `inputTokens` — question + RAG context + directives sent to LLM
- `outputTokens` — generated response tokens
- `totalTokens` — sum

Token data flows: Hindsight API response → `MsgMeta` → MongoDB → frontend display.

## Graceful Degradation

If Hindsight is unreachable, queries fall back to RAG-only mode (skip Retain + Reflect, use RAG answer directly).
