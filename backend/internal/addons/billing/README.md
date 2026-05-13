<div align="center">

# Orkestra Billing addon

**Italian electronic invoicing (Fatturazione Elettronica / FatturaPA) for the [Orkestra](https://github.com/orkestra-cc/orkestra) modular monolith. Generates FatturaPA v1.2.3 XML, submits invoices to Sistema di Interscambio (SDI) via the [OpenAPI.it SDI](https://www.openapi.it/documentazione-sdi/) gateway, reconciles status from webhook callbacks + a polling safety net, and handles every notification type (RC / NS / MC / NE / DT / AT). Optional digital signature and legal storage (conservazione sostitutiva).**

[![Go Reference](https://pkg.go.dev/badge/github.com/orkestra-cc/orkestra-addon-billing.svg)](https://pkg.go.dev/github.com/orkestra-cc/orkestra-addon-billing)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)
[![Module](https://img.shields.io/badge/module-github.com%2Forkestra--cc%2Forkestra--addon--billing-blue?style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-billing)
[![Latest tag](https://img.shields.io/github/v/tag/orkestra-cc/orkestra-addon-billing?sort=semver&style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-billing/tags)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)

[SDK](https://github.com/orkestra-cc/orkestra-sdk) · [OpenAPI auth](https://github.com/orkestra-cc/orkestra-openapi-auth) · [Monorepo](https://github.com/orkestra-cc/orkestra) · [Module docs](CLAUDE.md)

</div>

---

## What this is

A self-contained Orkestra addon implementing the `Module` interface from [`orkestra-cc/orkestra-sdk`](https://github.com/orkestra-cc/orkestra-sdk). Owns four MongoDB collections (`billing_invoices`, `billing_suppliers`, `billing_notifications`, `billing_polling_state`) and exposes `/v1/billing/*` for invoice lifecycle, supplier management, SDI notifications, and webhook ingestion.

**Internal-tenant only** (ADR-0001 Phase 2): every protected route sits behind `RequireInternalTenant()`. FatturaPA/SDI is an operator-side concern; external-tenant tokens cannot hit these endpoints. The Stripe-style webhook endpoint stays public (HMAC-style auth via the configured webhook secret).

**Invoice lifecycle:**

```
Draft → Sent → [SDI Processing] → Delivered / Rejected
                     │
                     └──▶ SDI Notifications (RC, NS, MC, NE, DT, AT)
```

**Notification types**:

| Code | Name | Meaning |
|---|---|---|
| `RC` | Ricevuta di Consegna | Invoice delivered to the recipient |
| `NS` | Notifica di Scarto | SDI rejected the invoice (XML / business-rule errors) |
| `MC` | Mancata Consegna | Recipient unreachable; SDI will keep retrying |
| `NE` | Notifica Esito | Public-sector recipient accepted/rejected |
| `DT` | Decorrenza Termini | Silent acceptance (15 days elapsed) |
| `AT` | Attestazione | Transmission attestation |

## How it ships

The same source tree lives in two places:

- **In-tree** at [`backend/internal/addons/billing/`](https://github.com/orkestra-cc/orkestra/tree/main/backend/internal/addons/billing) inside the [orkestra-cc/orkestra](https://github.com/orkestra-cc/orkestra) monorepo, where cross-module development happens.
- **Standalone** at this repository, tagged from `v0.1.0`, consumed via the Go module proxy by anything outside the monorepo.

```go
require github.com/orkestra-cc/orkestra-addon-billing v0.1.0
replace github.com/orkestra-cc/orkestra-addon-billing => ./internal/addons/billing
```

The `replace` will retire once cross-cutting addon churn settles.

## Install

```bash
go get github.com/orkestra-cc/orkestra-addon-billing@latest
```

Requires Go 1.25.10 or newer.

```go
import billing "github.com/orkestra-cc/orkestra-addon-billing"
```

## Boot in a host

```go
import (
    billing "github.com/orkestra-cc/orkestra-addon-billing"
    "github.com/orkestra-cc/orkestra-sdk/module"
)

reg := module.NewModuleRegistry(logger) // your kernel's *slog.Logger
reg.Register(billing.NewModule())
```

The registry takes care of init order, Mongo collection creation, route mounting on the operator surface (every protected route gated by `RequireInternalTenant()`), the `RequireCapability("billing.access")` capability check, RBAC enforcement, and module-gate middleware for runtime enable/disable via `/admin/modules`. Startup also kicks off the SDI polling job and registers the OpenAPI.it webhook callback (when `OPENAPI_BILLING_WEBHOOK_URL` is set).

The addon publishes `iface.BillingProvider` into the kernel's `ServiceRegistry` so the unified-clients aggregate (tenant module) can resolve a tenant's billing identity (`LegalName`, `VAT`, `CodiceDestinatario`, etc.) for non-billing flows that need to display it.

## Endpoints

Mounted under `/v1/billing` on the operator surface; the webhook callback sits at `/v1/billing/webhooks/sdi` (public, HMAC-verified).

| Method | Path | Purpose |
|---|---|---|
| `POST/GET/PATCH/DELETE` | `/invoices[/{id}]` | Invoice lifecycle (draft, send to SDI, list, get, soft-delete) |
| `POST` | `/invoices/{id}/send`, `/invoices/{id}/cancel` | Submit + cancel via SDI |
| `GET` | `/invoices/{id}/{xml,html,pdf}` | Rendered artefacts (PDF requires the documents addon) |
| `POST` | `/invoices/import` | Import a supplier-side XML invoice as a draft |
| `GET / POST / PATCH / DELETE` | `/suppliers[/{id}]` | Supplier registry |
| `GET / POST` | `/notifications[/{id}/process]` | Read + mark-as-processed |
| `GET` | `/notifications/summary`, `/stats` | Dashboard aggregates |
| `POST` | `/sync`, `/sync/invoices` | Manual safety-net sync against SDI |
| `POST` | `/webhooks/sdi` | Public SDI webhook (HMAC) |

## Configuration

| Env var (seeds first-install) / admin field | Purpose | Default |
|---|---|---|
| `OPENAPI_BILLING_BASE_URL` | SDI gateway URL | `https://test.sdi.openapi.it` (prod: `https://sdi.openapi.it`) |
| `OPENAPI_BILLING_ACCOUNT_EMAIL` | OpenAPI.it account (paired with API key for OAuth `/token`) | _(empty)_ |
| `OPENAPI_BILLING_API_KEY` | Long-lived API key from console.openapi.com; module mints + caches JWTs from this | _(empty)_ |
| `OPENAPI_OAUTH_BASE_URL` | OAuth host (shared with company addon) | `https://oauth.openapi.it` (sandbox: `https://test.oauth.openapi.it`) |
| `OPENAPI_BILLING_BEARER_TOKEN` | Legacy static JWT fallback (used only when API key is empty) | _(empty)_ |
| `OPENAPI_BILLING_FISCAL_ID` | Company P.IVA / fiscal code | _(required to send)_ |
| `OPENAPI_BILLING_RECIPIENT_CODE` | SDI recipient code for incoming invoices | `JKKZDGR` |
| `OPENAPI_BILLING_APPLY_SIGNATURE` | Send signed via `/invoices_signature` | `true` |
| `OPENAPI_BILLING_APPLY_STORAGE` | Send to legal storage via `/invoices_legal_storage` | `true` |
| `OPENAPI_BILLING_POLLING_INTERVAL` | Renewal-style poll interval (safety net for missed webhooks) | `12h` |
| `OPENAPI_BILLING_POLLING_ENABLED` | Toggle the polling job | `true` |
| `OPENAPI_BILLING_WEBHOOK_URL` | Public URL for SDI webhook callbacks (registered with OpenAPI.it at boot) | _(optional)_ |
| `OPENAPI_BILLING_WEBHOOK_SECRET` | Bearer token verifying incoming webhook requests | _(optional)_ |
| `OPENAPI_SANDBOX_MODE` | Use sandbox environment (shared toggle with company addon) | `true` |

The OAuth credentials are exchanged for a short-lived JWT via [`orkestra-openapi-auth`](https://github.com/orkestra-cc/orkestra-openapi-auth) — the same minter the [`company`](https://github.com/orkestra-cc/orkestra-addon-company) addon uses, cached two-tier (in-process + Redis) and invalidated on upstream `401/403`.

See [`CLAUDE.md`](CLAUDE.md) for the per-package map, [`FATTURAPA_SPEC.md`](FATTURAPA_SPEC.md) (when checked in) for the bilingual XML field reference, the SDI integration patterns (autofatture TD16-TD20, recipient codes, ditte individuali CF-vs-PIVA gotcha, error-code catalogue), and the polling-job's API-budget math (~120 calls/month at the default 12h interval).

## Versioning

Standard Go semver. `v0.x` allows breaking changes (rare); `v1.x` will freeze the public surface alongside the rest of the Orkestra addon ecosystem.

## Contributing

Most development happens in the [upstream monorepo](https://github.com/orkestra-cc/orkestra) where this addon lives alongside the kernel and its consumers — so a change can be implemented, tested, and reviewed against real callers in one diff. PRs against this standalone repo are welcome for addon-only changes (new XML helpers, new error mappings scoped to this tree).

See [CONTRIBUTING.md](https://github.com/orkestra-cc/orkestra/blob/main/CONTRIBUTING.md) in the monorepo for the contributor flow.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
