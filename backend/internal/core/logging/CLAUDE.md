# Module: Logging — Runtime log-level admin

_Path: `/backend/internal/core/logging`_
_Parent: [../CLAUDE.md](../CLAUDE.md)_

[← Core](../CLAUDE.md) | [☰ Backend](../../../CLAUDE.md) | [Root](../../../../CLAUDE.md)

## Purpose

Owns the runtime log-level configuration for the slog pipeline (ADR-0005 Phase F). Operators flip per-module thresholds at `/admin/observability/log-levels` and every existing module logger picks up the change instantly — no restart, no env-var edit, no re-pull of the GHCR image.

Sits at the end of the core init order so by the time it runs, every other module has already taken its `deps.Logger` clone. The shared `resolverBox` behind `PerModuleLevelHandler` means those pre-existing clones still observe the swap.

## What it owns

| File | Purpose |
|---|---|
| `module.go` | Module registration; reads `ServiceLogLevelModuleNames` to render admin rows; publishes `ServiceLogLevelResolver` |
| `routes.go` | Huma route registration (5 endpoints under `/v1/admin/observability/log-levels`) |
| `handlers/log_level_handler.go` | HTTP ↔ service translation; pulls actor UUID via `ctxauth` |
| `services/log_level_service.go` | atomic.Pointer snapshot of `(global, perModule)`, mutex-serialized writes |
| `services/log_level_service_test.go` | unit tests including `-race` concurrent reads/writes |
| `repository/log_level_repository.go` | Mongo single-document upsert (`_id="default"`) |
| `models/log_level.go` | `LogLevel` value-type, `Parse`, `Slog`, `LogLevelDoc`, `AdminView` |

## MongoDB collections

| Collection | Shape | Notes |
|---|---|---|
| `log_levels` | single document with `_id="default"`, `global: LogLevel`, `perModule: map[string]LogLevel`, `updatedAt/updatedBy/updateNote` | Replaced wholesale on every admin write so concurrent writers can't produce two documents |

No declared indexes — `_id` is the primary key and that's all we filter on.

## Dependencies

- **Modules**: none declared. Logging is intentionally a leaf in the DAG so it can be the last to init.
- **Required services**: none — it reads `ServiceLogLevelModuleNames` from the registry but tolerates its absence (the admin view just renders zero module rows).
- **Provides**:
  - `ServiceLogLevelResolver` → `*services.LogLevelService` (satisfies `utils.LevelResolver` structurally).
- **Permissions contributed**: none. Admin endpoints are gated by the operator-mux `administrator` role at the API security scope, not by Cedar permissions.

## Lifecycle

`Init` builds the repository, captures the boot env defaults (`utils.GlobalLevelFromEnv` + `utils.LoadPerModuleLevels`), looks up the module-names catalog via `ServiceLogLevelModuleNames`, constructs the `LogLevelService` seeded with the env snapshot, and calls `svc.Load(ctx)` to pull the persisted document (if any) into the atomic snapshot. The service is then registered under `ServiceLogLevelResolver`.

`main.go` reads that key **after** `InitAll` returns and calls `utils.SwapLevelResolver(svc)` to replace the boot-time `StaticLevelResolver` in `PerModuleLevelHandler`. Every existing module logger picks up the swap through the shared `resolverBox` atomic pointer.

`Start` / `Stop` / `HealthCheck` are no-ops inherited from `BaseModule`.

## HTTP endpoints

All five are mounted on the operator-protected router and require the `administrator` system role (`Security: bearerAuth.administrator` on every operation).

| Method | Path | Purpose |
|---|---|---|
| GET    | `/v1/admin/observability/log-levels` | Returns `AdminView`: global level + one row per registered module |
| PUT    | `/v1/admin/observability/log-levels/global` | Sets the global threshold |
| PUT    | `/v1/admin/observability/log-levels/{module}` | Sets a per-module override |
| DELETE | `/v1/admin/observability/log-levels/{module}` | Removes a per-module override (falls back to global) |
| POST   | `/v1/admin/observability/log-levels/reset` | Reverts global + every override to boot env defaults |

Mutations return the fresh `AdminView` so the UI re-renders without a separate refetch.

## Service contract

The service implements both `utils.LevelResolver` (consumed by `PerModuleLevelHandler.Enabled` on the hot read path) and `services.LevelResolver` (the local mirror so dependent code doesn't have to import `shared/utils`). The two interfaces have the same shape; structural typing satisfies both.

## Key invariants

- **Single document.** The repository filters by `_id="default"` so concurrent writers can never produce more than one row. The service serializes writes under a mutex; reads consult the atomic snapshot lock-free.
- **Atomic snapshot for the hot path.** `*snapshot` lives behind `atomic.Pointer[snapshot]`; readers (every log call) get a consistent view without locks. Mutations build a new snapshot and `Store` it after persisting — readers either see the old snapshot or the new one, never partial state.
- **Persist before publish.** If the Mongo upsert fails, the snapshot is **not** updated. The in-memory view always reflects what's on disk.
- **Env defaults are remembered separately.** `ResetToEnv` reverts to the values captured at `NewLogLevelService` time, not to "whatever the env says right now" — restarts re-resolve from env, but a Mongo doc takes precedence.
- **No retroactive seeding.** When the document is missing (fresh deployment), the service stays on the env snapshot. The first admin write creates the document; subsequent restarts load from Mongo.

## What this module does NOT do

- Loki retention overrides — out of scope for Phase F; reserved as a future amendment.
- Per-tenant log levels — the threshold is global per module; tenant-scoped filtering happens at query time in Loki via the `tenant_id` field already stamped on every log line.
- Streaming live logs to the admin UI — the page sets thresholds; live tailing happens in Grafana.
- Audit-log integration — runtime log-level changes are persisted with `updatedBy`/`updatedAt` but are not pushed through the compliance `AuditSink`. Future work.

## Rules

- **Never mutate the snapshot in place.** Always build a new `*snapshot` and `Store` it — readers depend on the immutability invariant.
- **Never expose the resolver as `services.LogLevelService` to consumers.** The interface boundary is `utils.LevelResolver`; the concrete type can rename without breaking consumers.
- **Env vars are seed-only.** After first boot, the Mongo doc is authoritative. Setting `LOG_LEVEL_<MODULE>` after the document exists is silently shadowed — surface this in operator-facing docs whenever it changes.

## Related

- [`../../shared/utils/per_module_level_handler.go`](../../shared/utils/per_module_level_handler.go) — the slog handler that consumes the resolver
- [`../../shared/utils/logger.go`](../../shared/utils/logger.go) — `SwapLevelResolver` global called by `main.go`
- [`../../../pkg/sdk/module/services.go`](../../../pkg/sdk/module/services.go) — `ServiceLogLevelResolver` / `ServiceLogLevelModuleNames` keys
- [`../../../../docs/adr/0005-observability-logging-tracing-metrics.md`](../../../../docs/adr/0005-observability-logging-tracing-metrics.md) — full Phase F design
- [`../navigation/CLAUDE.md`](../navigation/CLAUDE.md) — neighbour core module; same module-interface shape
