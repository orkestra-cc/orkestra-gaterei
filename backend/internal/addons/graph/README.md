<div align="center">

# Orkestra Graph addon

**Memgraph-backed knowledge graph for the [Orkestra](https://github.com/orkestra-cc/orkestra) modular monolith. Cypher execution over the Bolt protocol (`neo4j-go-driver/v5`), schema introspection, MAGE graph algorithms (PageRank, Louvain, weakly-connected components, …), and vector similarity search — all behind a per-tenant SDK kernel that toggles the underlying `orkestra-memgraph` container on enable/disable.**

[![Go Reference](https://pkg.go.dev/badge/github.com/orkestra-cc/orkestra-addon-graph.svg)](https://pkg.go.dev/github.com/orkestra-cc/orkestra-addon-graph)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)
[![Module](https://img.shields.io/badge/module-github.com%2Forkestra--cc%2Forkestra--addon--graph-blue?style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-graph)
[![Latest tag](https://img.shields.io/github/v/tag/orkestra-cc/orkestra-addon-graph?sort=semver&style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-graph/tags)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)

[SDK](https://github.com/orkestra-cc/orkestra-sdk) · [Monorepo](https://github.com/orkestra-cc/orkestra) · [Module docs](CLAUDE.md)

</div>

---

## What this is

A self-contained Orkestra addon implementing the `Module` interface from [`orkestra-cc/orkestra-sdk`](https://github.com/orkestra-cc/orkestra-sdk). It owns the `orkestra-memgraph` container via the SDK's `InfraContainers` hook (enabling the module at `/admin/modules` starts the database; disabling stops it), exposes `/v1/graph/*` endpoints, and registers a `GraphRepository` in the kernel's `ServiceRegistry` so downstream addons (RAG, agents) can run Cypher without re-implementing connection management.

**Endpoints** (mounted under `/v1/graph` on the operator surface):

| Method | Path | Description |
|---|---|---|
| GET | `/health`, `/databases`, `/schema` | Connectivity + schema introspection |
| POST | `/query` | Execute arbitrary Cypher |
| GET | `/nodes`, `/relationships`, `/nodes/{id}/neighbors` | Browse + neighborhood subgraphs |
| GET / POST | `/algorithms` | List + run MAGE algorithms |
| POST / GET / DELETE | `/vector/search`, `/vector/indexes[/{name}]` | Vector similarity search + index management |

## Why Memgraph

The addon talks Bolt, which means **any Bolt-compatible server works** — both Memgraph (default) and Neo4j. The container manager + the `memgraph-mage` image are the opinionated defaults because MAGE bundles the algorithm library the addon's algorithm service relies on. Point `GRAPH_URI` at a different Bolt endpoint to use Neo4j or a managed Memgraph cluster instead; toggle the addon off in `/admin/modules` to opt out of container management.

## How it ships

The same source tree lives in two places:

- **In-tree** at [`backend/internal/addons/graph/`](https://github.com/orkestra-cc/orkestra/tree/main/backend/internal/addons/graph) inside the [orkestra-cc/orkestra](https://github.com/orkestra-cc/orkestra) monorepo, where cross-module development happens.
- **Standalone** at this repository, tagged from `v0.1.0`, consumed via the Go module proxy by anything outside the monorepo.

The upstream backend's `go.mod` looks roughly like this:

```go
require github.com/orkestra-cc/orkestra-addon-graph v0.1.0
replace github.com/orkestra-cc/orkestra-addon-graph => ./internal/addons/graph
```

The `replace` will retire once cross-cutting addon churn settles.

## Install

```bash
go get github.com/orkestra-cc/orkestra-addon-graph@latest
```

Requires Go 1.25.10 or newer.

```go
import graph "github.com/orkestra-cc/orkestra-addon-graph"
```

## Boot in a host

The addon plugs into any host that implements the Orkestra SDK kernel. Register it alongside the other catalog entries:

```go
import (
    graph "github.com/orkestra-cc/orkestra-addon-graph"
    "github.com/orkestra-cc/orkestra-sdk/module"
)

reg := module.NewModuleRegistry(logger) // your kernel's *slog.Logger
reg.Register(graph.NewModule())
```

The registry takes care of init order, container lifecycle (`InfraContainers` — `orkestra-memgraph` starts/stops with the module toggle), route mounting on the operator surface, RBAC enforcement (`graph.query.read` / `graph.query.write` / `graph.admin`), and module-gate middleware for runtime enable/disable via `/admin/modules`.

Consumer modules pull `GraphRepository` via the standard typed-getter pattern:

```go
import (
    "github.com/orkestra-cc/orkestra-addon-graph/repository"
    "github.com/orkestra-cc/orkestra-sdk/module"
)

repo, ok := module.GetTyped[repository.GraphRepository](deps.Services, module.ServiceGraphRepo)
```

## Configuration

| Env var (seeds first-install) / admin field | Purpose | Default |
|---|---|---|
| `GRAPH_URI` | Bolt endpoint | `bolt://orkestra-memgraph:7687` |
| `GRAPH_USERNAME` | Bolt auth user (leave empty for `NoAuth`) | _(empty)_ |
| `GRAPH_PASSWORD` | Bolt auth password | _(empty)_ |
| `GRAPH_DATABASE` | Logical database name | `memgraph` |
| `GRAPH_MAX_CONN_POOL` | Connection pool size | `50` |
| `GRAPH_IMAGE` | Docker image for the managed container; the `-mage` variant is required for the algorithm service | `memgraph/memgraph-mage:latest` |

Toggling the addon at `/admin/modules` re-reads `GRAPH_IMAGE` live, so upgrading Memgraph only needs a disable → re-enable cycle (no backend restart). Bolt connection settings are read at `Init()` and stay pinned until the backend process restarts.

## Init vs Start split

`Init()` builds the Bolt driver via `NewGraphDriver` **without** dialing — the neo4j-go driver is lazy, so this stays safe when the container isn't up yet. `Start()` calls `VerifyGraphConnection` (with retry up to 30s) after the registry has brought up `orkestra-memgraph`. `Stop()` is a no-op — the driver pool reuses across toggles and only the container stops.

## Memgraph patterns to know

- **Auto-commit transactions** are required for DDL/storage commands (`CREATE INDEX`, `SHOW INDEX INFO`, vector index management). The repository exposes `ExecuteRead`/`ExecuteWrite`/`ExecuteAutoCommit` to make the boundary explicit.
- **Schema introspection** uses Memgraph-specific syntax (`SHOW INDEX INFO`, `MATCH (n) UNWIND labels(n) AS l RETURN DISTINCT l`) rather than Neo4j's `db.labels()` / `SHOW INDEXES`.
- **Vector search** is `CALL vector_search.search($idx, $topK, $vec) YIELD node, similarity`; supported metrics are `cos` / `l2sq` / `ip`.

See [`CLAUDE.md`](CLAUDE.md) for the per-package map, MAGE algorithm catalog, write-query detection rules, and the full result-marshaling pipeline.

## Versioning

Standard Go semver. `v0.x` allows breaking changes (rare); `v1.x` will freeze the public surface alongside the rest of the Orkestra addon ecosystem.

## Contributing

Most development happens in the [upstream monorepo](https://github.com/orkestra-cc/orkestra) where this addon lives alongside the kernel and its consumers — so a change can be implemented, tested, and reviewed against real callers in one diff. PRs against this standalone repo are welcome for addon-only changes (new algorithms, new endpoints scoped to this tree).

See [CONTRIBUTING.md](https://github.com/orkestra-cc/orkestra/blob/main/CONTRIBUTING.md) in the monorepo for the contributor flow.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
