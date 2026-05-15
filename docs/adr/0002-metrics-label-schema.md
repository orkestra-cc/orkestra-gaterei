---
title: ADR-0002 — Prometheus metric label schema
status: accepted
public: true
---

# ADR-0002 — Prometheus metric label schema

| Field | Value |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-04-20 |
| **Authors** | @salvatore.balestrino |
| **Supersedes** | — |
| **Related** | [ADR-0001](0001-unified-tenant-model.md), Phase 5.3 of the internal tenancy plan v2 |

## Context

Phase 5.3 of the tenancy plan ships a Prometheus `/metrics` endpoint on the Orkestra backend so the observability stack (Phase 5.2) has something to plot beyond trace data. The first three metric families are:

| Family | Type | Purpose |
|---|---|---|
| `orkestra_cedar_shadow_divergence_total` | Counter | Cedar shadow-mode disagreements with the role-table decision |
| `orkestra_capability_denied_total` | Counter | `402 Payment Required` responses from `RequireCapability` |
| `orkestra_entitlement_projection_lag_seconds` | Gauge | Seconds since the last successful entitlement grant / revoke |

Prometheus charges nothing up front for labels, but **each unique label-value combination creates a new time series.** Unbounded labels like a raw `tenant.id` blow up storage cost, scrape cost, and query latency — and once historical data carries those labels, you cannot retroactively remove them without breaking dashboards and recording rules.

Label schemas also break silently: there is no compile-time check that a dashboard's `sum by (foo)` still matches the producer. A renamed label yields zero rows and a green dashboard.

## Decision

The label schema for Phase 5.3 is **frozen** as follows. Changing it requires a new ADR.

### `orkestra_cedar_shadow_divergence_total`

```
labels: {action_suffix, matched_policy, outcome}
```

- **action_suffix**: the last dotted segment of the permission key (`read`, `create`, `update`, `delete`, `admin`, plus the ~10 domain-specific verbs enumerated in `backend/internal/core/authz/CLAUDE.md#permission-naming-convention`). Bounded cardinality.
- **matched_policy**: the `@id` annotation of the Cedar policy that drove the disagreement. Bounded by the number of policies in `internal/core/authz/cedar/policies/` (~13 today).
- **outcome**: one of `role_only`, `cedar_only`, `both`, `neither`. Fixed enumeration.

Total cardinality upper bound: ~20 suffixes × ~13 policies × 4 outcomes ≈ 1000 series.

### `orkestra_capability_denied_total`

```
labels: {capability_id}
```

- **capability_id**: one of the IDs declared by a module's `Capabilities()`. Bounded by the capability catalog (~7 today, project a ceiling of ~50).

### `orkestra_entitlement_projection_lag_seconds`

```
labels: {tenant_kind}
```

- **tenant_kind**: `internal` | `external`. Fixed 2-value enumeration per ADR-0001.

### Forbidden labels

The following labels are **explicitly banned** from Phase-5-and-later metrics without a superseding ADR:

- **`tenant_id`** — high cardinality, principal-identifying, and available on Tempo spans for per-request drill-down. Do not re-surface it as a metric label.
- **`user_id`** — same reasoning.
- **`route`** / **`path`** — unbounded (Huma registers dozens of routes per module). Use a route-template shim when request-counting becomes necessary; do not label by raw path.
- **`http_status`** below a bucket — prefer `status_class` (`2xx`, `4xx`, `5xx`) if a status dimension is ever needed.

### Principles

1. **Every label must have a bounded domain** known at code-review time.
2. **Prefer a tier-level label** (`tenant_kind`, `environment`) over an instance-level label (`tenant_id`, `user_id`).
3. **If you want high-cardinality visibility, use a trace attribute in Tempo, not a Prometheus label.** The two systems are complementary.
4. **New labels land in a new ADR**, not a PR. Renaming a label is a new ADR. Removing a label is a new ADR. This is load-bearing because recording rules and dashboards depend on the schema.
5. **Zero-observation series are not initialized**. `/metrics` will not emit a family until its first recorded observation. Dashboards must handle `no data` gracefully; do not preload fake observations.

## Consequences

- Operators accept that per-tenant metric drill-down goes through Tempo, not Prometheus.
- Adding a new metric is a two-line code change; adding a new label is an ADR.
- The existing dashboards provisioned by `docker-compose.observability.yml` are safe to treat as code — regenerating them against a new backend build will never silently drop panels.

## Implementation

- Metric definitions live in `backend/pkg/sdk/metrics/metrics.go`.
- Label-schema tests live in `backend/pkg/sdk/metrics/metrics_test.go` — they fail loudly if a label is renamed or a new label is added without updating the tests.
- The `/metrics` handler is registered in `backend/cmd/server/main.go` behind `METRICS_ENABLED=true` (default on).

## Alternatives considered

1. **Use the OpenTelemetry metrics SDK instead of Prometheus client_golang.** Rejected for Phase 5 because OTEL metrics is still maturing and the Prometheus client's `promhttp` handler is the simplest shippable surface. A future ADR may flip this.
2. **Salted HMAC hash of tenant.id as a label.** Rejected: still high cardinality (50k tenants ⇒ 50k series), still principal-identifying under correlation attacks. The cost is real; the benefit is marginal; Tempo covers the use case.
3. **Per-route request counters as the first metric family.** Rejected for Phase 5.3 specifically — the cardinality trap is easy to walk into and the three chosen families deliver more tenancy-plan-relevant data. A future sub-phase can add request counters behind a route-template normalization.
