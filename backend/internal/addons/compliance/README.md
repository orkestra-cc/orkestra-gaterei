<div align="center">

# Orkestra Compliance addon

**The platform compliance plane for the [Orkestra](https://github.com/orkestra-cc/orkestra) modular monolith. Owns the append-only `audit_events` log, publishes `iface.AuditSink` so every module emits without depending on this addon, drives the GDPR DSR pipeline (export + erasure across every registered `iface.PIIProducer`), runs per-tenant KMS envelope encryption with crypto-shred on purge, and computes the SOC2 evidence snapshot at `/v1/admin/compliance/soc2/evidence`.**

[![Go Reference](https://pkg.go.dev/badge/github.com/orkestra-cc/orkestra-addon-compliance.svg)](https://pkg.go.dev/github.com/orkestra-cc/orkestra-addon-compliance)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)
[![Module](https://img.shields.io/badge/module-github.com%2Forkestra--cc%2Forkestra--addon--compliance-blue?style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-compliance)
[![Latest tag](https://img.shields.io/github/v/tag/orkestra-cc/orkestra-addon-compliance?sort=semver&style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-compliance/tags)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)

[SDK](https://github.com/orkestra-cc/orkestra-sdk) · [Monorepo](https://github.com/orkestra-cc/orkestra) · [Module docs](CLAUDE.md)

</div>

---

## What this is

A self-contained Orkestra addon implementing the `Module` interface from [`orkestra-cc/orkestra-sdk`](https://github.com/orkestra-cc/orkestra-sdk). Compliance is a write-once-publish pattern: every consumer module exposes a `SetAuditSink(iface.AuditSink)` method, and this addon's `Init()` pushes the canonical sink into each one via the SDK's `iface.AuditSinkSetter` contract — no concrete-type imports cross the boundary.

**Four workloads:**

1. **Audit log** — `audit_events` collection with 2-year TTL. Indexed for the operator reader UI (tenantId+timestamp, actorUserId+timestamp, action+timestamp, resourceType+resourceId).
2. **DSR (GDPR Articles 15 + 17)** — `iface.PIIProducerRegistry` is the fan-out hub; every module that holds personal data registers a producer at its own `Init()`. Compliance enumerates them on `/v1/me/dsr/export` (right of access) and `/v1/me/dsr/erase` (right to erasure). The client-tier erase route is gated by the `selfServiceAccountDeletionClient` auth policy — closed by default, opens live on admin toggle.
3. **KMS lifecycle** — per-tenant envelope encryption keys (`kms_keys` collection). The local provider boots when `ORKESTRA_KMS_MASTER_KEY` is set; a future AWS KMS provider swaps in at the same SDK contract.
4. **SOC2 evidence** — read-only aggregation served at `/v1/admin/compliance/soc2/evidence`. Maps to CC6.1 (privileged users), CC6.6 (MFA coverage), CC6.8 (KMS lifecycle), CC7.2 (failed-login trends + audit coverage).

## How it ships

The same source tree lives in two places:

- **In-tree** at [`backend/internal/addons/compliance/`](https://github.com/orkestra-cc/orkestra/tree/main/backend/internal/addons/compliance) inside the [orkestra-cc/orkestra](https://github.com/orkestra-cc/orkestra) monorepo, where cross-module development happens.
- **Standalone** at this repository, tagged from `v0.1.0`, consumed via the Go module proxy by anything outside the monorepo.

```go
require github.com/orkestra-cc/orkestra-addon-compliance v0.1.0
replace github.com/orkestra-cc/orkestra-addon-compliance => ./internal/addons/compliance
```

The `replace` will retire once cross-cutting addon churn settles.

## Install

```bash
go get github.com/orkestra-cc/orkestra-addon-compliance@latest
```

Requires Go 1.25.10 or newer.

```go
import compliance "github.com/orkestra-cc/orkestra-addon-compliance"
```

## Boot in a host

```go
import (
    compliance "github.com/orkestra-cc/orkestra-addon-compliance"
    "github.com/orkestra-cc/orkestra-sdk/module"
)

reg := module.NewModuleRegistry(logger) // your kernel's *slog.Logger
reg.Register(compliance.NewModule())
```

Compliance declares `Dependencies()` of `["auth", "tenant", "identity", "subscriptions"]` so the topo sort runs their `Init()` first — that way every `iface.AuditSinkSetter` receiver is in the `ServiceRegistry` before compliance probes for it. Missing addons are silently skipped (the probe returns `ok=false`).

## What the audit-sink push looks like from a consumer

Every module that wants its events on the audit trail exposes a single setter:

```go
type Service struct {
    auditSink iface.AuditSink
    // ...
}

func (s *Service) SetAuditSink(sink iface.AuditSink) { s.auditSink = sink }
```

That's it — no compliance import. The kernel's `ModuleRegistry` registers the service under a `ServiceKey`; compliance's `Init()` walks a fixed key list, probes `module.GetTyped[iface.AuditSinkSetter]`, and calls `SetAuditSink` on each hit.

## DSR producer contract

PII-holding modules implement `iface.PIIProducer`:

```go
type PIIProducer interface {
    Subject() string  // stable identifier (e.g. "user", "auth", "subscriptions")
    ExportPersonalData(ctx, userUUID) (any, error)
    PurgePersonalData(ctx, userUUID) (PurgeResult, error)
}
```

…and register with `iface.PIIProducerRegistry` during their own `Init()`. Compliance discovers them and fans out on every DSR request — the registered producer set is the union of every consenting module, so adding compliance to an existing host immediately picks up every PII holder without configuration.

## Configuration

| Env var | Purpose | Default |
|---|---|---|
| `ORKESTRA_KMS_MASTER_KEY` | Hex-encoded 32-byte AES-256 master key for the local KMS provider | _(empty — KMS disabled, no crypto-shred)_ |

When `ORKESTRA_KMS_MASTER_KEY` is missing the addon still boots, the audit log + DSR pipeline + SOC2 evidence keep working — only crypto-shred on tenant purge is unavailable. Dev deployments commonly opt out.

See [`CLAUDE.md`](CLAUDE.md) for the per-package map, the cross-module `AuditSinkSetter` probe table, the SOC2 control mapping, and the testkit-replacement note that closed the last in-tree drift point.

## Versioning

Standard Go semver. `v0.x` allows breaking changes (rare); `v1.x` will freeze the public surface alongside the rest of the Orkestra addon ecosystem.

## Contributing

Most development happens in the [upstream monorepo](https://github.com/orkestra-cc/orkestra) where this addon lives alongside the kernel and its consumers. PRs against this standalone repo are welcome for addon-only changes (new PIIProducers, new SOC2 controls).

See [CONTRIBUTING.md](https://github.com/orkestra-cc/orkestra/blob/main/CONTRIBUTING.md) in the monorepo for the contributor flow.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
