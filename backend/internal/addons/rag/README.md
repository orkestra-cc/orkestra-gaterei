<div align="center">

# Orkestra RAG addon

**Retrieval-Augmented Generation pipeline for ISO norms / legal documents in the [Orkestra](https://github.com/orkestra-cc/orkestra) modular monolith. 20-step ingestion (extract → markdown/structural parse → chunk → embed → graph) persists each document as a Memgraph knowledge graph; question-answering combines vector search with section/definition/similarity graph traversal. Plugs into the [`aimodels`](https://github.com/orkestra-cc/orkestra-addon-aimodels) addon for embedding + LLM providers and the [`graph`](https://github.com/orkestra-cc/orkestra-addon-graph) addon for the Memgraph backend.**

[![Go Reference](https://pkg.go.dev/badge/github.com/orkestra-cc/orkestra-addon-rag.svg)](https://pkg.go.dev/github.com/orkestra-cc/orkestra-addon-rag)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)
[![Module](https://img.shields.io/badge/module-github.com%2Forkestra--cc%2Forkestra--addon--rag-blue?style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-rag)
[![Latest tag](https://img.shields.io/github/v/tag/orkestra-cc/orkestra-addon-rag?sort=semver&style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-rag/tags)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)

[SDK](https://github.com/orkestra-cc/orkestra-sdk) · [AI Models](https://github.com/orkestra-cc/orkestra-addon-aimodels) · [Graph](https://github.com/orkestra-cc/orkestra-addon-graph) · [Monorepo](https://github.com/orkestra-cc/orkestra) · [Module docs](CLAUDE.md)

</div>

---

## What this is

A self-contained Orkestra addon implementing the `Module` interface from [`orkestra-cc/orkestra-sdk`](https://github.com/orkestra-cc/orkestra-sdk). Owns three MongoDB collections — `rag_documents`, `rag_models`, `rag_relationship_types` — and a Memgraph subgraph rooted at `:RagDocument` / `:RagSection` / `:RagChunk` / `:RagDefinition` nodes connected by `HAS_SECTION` / `CONTAINS` / `NEXT` / `REFERENCES` / `SIMILAR_TO` edges.

**Endpoints** (mounted under `/v1/rag` on the operator surface):

| Method | Path | Purpose |
|---|---|---|
| `POST/GET/PATCH/DELETE` | `/models[/{uuid}]` | AI model config CRUD + connectivity test |
| `POST/GET/PATCH/DELETE` | `/documents[/{uuid}]` | Document upload, ingest, metadata edit |
| `GET` | `/documents/{uuid}/{chunks,sections,relations}` | Per-document introspection |
| `POST` | `/documents/{uuid}/reprocess` | Clear graph nodes for re-ingestion |
| `POST` | `/query` | Ask a question; returns grounded answer + source citations |
| (streaming) | `/query/stream` | Server-Sent-Events streaming variant |

## How it ships

The same source tree lives in two places:

- **In-tree** at [`backend/internal/addons/rag/`](https://github.com/orkestra-cc/orkestra/tree/main/backend/internal/addons/rag) inside the [orkestra-cc/orkestra](https://github.com/orkestra-cc/orkestra) monorepo, where cross-module development happens.
- **Standalone** at this repository, tagged from `v0.1.0`, consumed via the Go module proxy.

```go
require github.com/orkestra-cc/orkestra-addon-rag v0.1.0
replace github.com/orkestra-cc/orkestra-addon-rag => ./internal/addons/rag
```

The `replace` will retire once cross-cutting addon churn settles.

## Install

```bash
go get github.com/orkestra-cc/orkestra-addon-rag@latest
```

Requires Go 1.25.10 or newer and `orkestra-cc/orkestra-sdk v0.4.0+`.

```go
import rag "github.com/orkestra-cc/orkestra-addon-rag"
```

## Boot in a host

rag declares `Dependencies()` of `["graph", "aimodels"]` — the topo sort runs both before this module's `Init()`. Register all three (graph publishes `iface.GraphRepository`; aimodels publishes `iface.AIModelProvider`; rag consumes both via the kernel's `ServiceRegistry`):

```go
import (
    aimodels "github.com/orkestra-cc/orkestra-addon-aimodels"
    graph "github.com/orkestra-cc/orkestra-addon-graph"
    rag "github.com/orkestra-cc/orkestra-addon-rag"
    "github.com/orkestra-cc/orkestra-sdk/module"
)

reg := module.NewModuleRegistry(logger) // your kernel's *slog.Logger
reg.Register(graph.NewModule())     // → GraphRepository
reg.Register(aimodels.NewModule())  // → AIModelProvider
reg.Register(rag.NewModule())       // consumes both at Init
```

## Document ingestion (20 steps)

```
1. Create MongoDB record (status: pending)
2. Update status → processing
3. Extract text (PDF via Gotenberg, plain text/markdown as-is)
4. Parse structure:
    - .md  → goldmark CommonMark AST → StructuralNode tree
    - PDF/TXT → regex parser (ISO clauses, Italian legal docs)
5. Chunk respecting structural boundaries
6. Embed each chunk (EmbeddingProvider.Embed)
7–16. Build the :RagDocument / :RagSection / :RagChunk graph with
       HAS_SECTION / CONTAINS / NEXT_SECTION / NEXT / DEFINES /
       REFERENCES edges
17. Compute intra-document SIMILAR_TO edges (cosine > 0.85)
18. Compute cross-document SIMILAR_TO edges via Memgraph vector search
19. Create vector + property indexes
20. Update status → completed
```

Both parsers produce the same `StructuralNode` tree shape so chunking, embedding, and graph creation are identical downstream — only the input parsing differs.

## Query pipeline

```
1. Embed the question
2. Vector search (top-K, optional filter: requirementLevel, nodeType)
3. Context expansion ("vector" or "graph" mode):
    - prev/next chunks via NEXT
    - parent section hierarchy + cross-refs + definitions
    - cross-document SIMILAR_TO hits
4. Fetch doc metadata (:RagDocument title, isoStandard)
5. Build a structurally-rich prompt with paths + requirement levels
6. LLM completion (5-minute timeout for local inference)
7. Return answer + source citations + timing metadata
```

## Configuration

| Env var (seeds first-install) / admin field | Purpose | Default |
|---|---|---|
| `OLLAMA_BASE_URL` | Default Ollama endpoint | `http://localhost:11434` |
| `OPENAI_API_KEY` | OpenAI fallback (only when aimodels is disabled) | _(empty)_ |
| `RAG_CHUNK_SIZE` | Default chunk size in characters | `512` |
| `RAG_CHUNK_OVERLAP` | Overlap between chunks | `50` |
| `RAG_DEFAULT_TOP_K` | Default vector search result count | `10` |

All five are editable at `/admin/modules/rag` once a config document exists; defaults match the legacy `RAG_*` env-var defaults so existing deployments keep their behavior.

## Supported AI providers

| Provider | Auth | Notes |
|---|---|---|
| `openai` | API key or none | Works with OpenAI cloud, llama.cpp, vLLM, LocalAI — the BaseURL is what matters |
| `ollama` | None | Ollama REST API (`/api/embeddings`, `/api/generate`) |

Default seed (when `rag_models` is empty): Qwen3-Embedding-8B + Qwen3.5-9B at `192.168.88.52:{8082,8081}/v1` via the OpenAI-compatible llama.cpp server. Override at `/admin` or wipe the collection to reseed.

See [`CLAUDE.md`](CLAUDE.md) for the full Memgraph schema, the chunker + structural parser internals, the cross-document similarity flow, and the boundary-lift rationale that closed the SDK-split series at Phase 5l.

## Versioning

Standard Go semver. `v0.x` allows breaking changes (rare); `v1.x` will freeze the public surface alongside the rest of the Orkestra addon ecosystem.

## Contributing

Most development happens in the [upstream monorepo](https://github.com/orkestra-cc/orkestra) where this addon lives alongside the kernel + aimodels + graph + documents. PRs against this standalone repo are welcome for addon-only changes (new structural parsers, new embedding providers, alternative chunking strategies).

See [CONTRIBUTING.md](https://github.com/orkestra-cc/orkestra/blob/main/CONTRIBUTING.md) in the monorepo for the contributor flow.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
