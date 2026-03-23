# Agents Module

AI agent system powered by [Hindsight](https://hindsight.vectorize.io/) persistent memory. Each project gets a scoped Hindsight memory bank and queries the existing graphRAG for document retrieval.

## Architecture

```
handlers/
  ├── project_handler.go    ← Project CRUD endpoints
  └── agent_handler.go      ← Agent query + conversation endpoints
services/
  ├── project_service.go    ← Project CRUD + Hindsight bank lifecycle
  ├── agent_service.go      ← Query orchestration: RAG → Retain → Reflect
  ├── hindsight_client.go   ← Thin wrapper around Hindsight Go SDK
  └── rag_bridge.go         ← Consumer interface for scoped RAG queries
models/
  ├── project.go            ← Project MongoDB model
  ├── conversation.go       ← Conversation + Message models
  ├── role_config.go        ← Persona definitions + RBAC validation
  └── dto.go                ← Huma request/response DTOs
repository/
  ├── project_repository.go       ← MongoDB: agent_projects
  └── conversation_repository.go  ← MongoDB: agent_conversations
routes.go                   ← RegisterProjectRoutes, RegisterQueryRoutes, RegisterAdminRoutes
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
2. RAG Phase: QueryWithScope(question, project.documentUUIDs)
3. Retain Phase (async): Store RAG results in Hindsight bank
4. Reflect Phase: Hindsight combines persistent memory + RAG context + persona directives
5. Save conversation to MongoDB
6. Return answer + sources + metadata
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

### Query (`/v1/agents/projects`) — operator role
| Method | Path | Description |
|--------|------|-------------|
| POST | `/{uuid}/query` | Query agent |
| POST | `/{uuid}/conversations` | New conversation |
| GET | `/{uuid}/conversations` | List conversations |
| GET | `/conversations/{uuid}` | Get conversation |
| DELETE | `/conversations/{uuid}` | Delete conversation |

### Admin (`/v1/agents`) — administrator role
| Method | Path | Description |
|--------|------|-------------|
| GET | `/projects/{uuid}/bank` | Hindsight bank info |
| GET | `/health` | Hindsight health check |

## Configuration

```
AGENTS_ENABLED=true
HINDSIGHT_URL=http://hindsight:8888
HINDSIGHT_NAMESPACE=orkestra
```

## Graceful Degradation

If Hindsight is unreachable, queries fall back to RAG-only mode (skip Retain + Reflect, use RAG answer directly).
