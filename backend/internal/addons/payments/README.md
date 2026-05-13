<div align="center">

# Orkestra Payments addon

**Gateway-agnostic payment processing for the [Orkestra](https://github.com/orkestra-cc/orkestra) modular monolith. Stripe-backed in v1 (PayPal reserved for v2), implementing `iface.PaymentProvider` so the [`subscriptions`](https://github.com/orkestra-cc/orkestra-addon-subscriptions) addon can charge cards, issue refunds, and react to webhook events without importing any gateway-specific code.**

[![Go Reference](https://pkg.go.dev/badge/github.com/orkestra-cc/orkestra-addon-payments.svg)](https://pkg.go.dev/github.com/orkestra-cc/orkestra-addon-payments)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)
[![Module](https://img.shields.io/badge/module-github.com%2Forkestra--cc%2Forkestra--addon--payments-blue?style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-payments)
[![Latest tag](https://img.shields.io/github/v/tag/orkestra-cc/orkestra-addon-payments?sort=semver&style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-payments/tags)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)

[SDK](https://github.com/orkestra-cc/orkestra-sdk) · [Subscriptions](https://github.com/orkestra-cc/orkestra-addon-subscriptions) · [Monorepo](https://github.com/orkestra-cc/orkestra) · [Module docs](CLAUDE.md)

</div>

---

## What this is

A self-contained Orkestra addon implementing the `Module` interface from [`orkestra-cc/orkestra-sdk`](https://github.com/orkestra-cc/orkestra-sdk). Owns three MongoDB collections — `payments_transactions`, `payments_payment_methods`, `payments_webhook_events` — and routes every charge / refund / webhook through a per-tenant `iface.PaymentProvider` façade.

**Webhook pipeline:**

```
POST /v1/payments/webhooks/stripe  (public, no JWT)
          │
   raw body + Stripe-Signature
          │
  stripeHandler → provider.VerifyWebhook (HMAC via stripe-go)
          │                                 │
          │                            invalid → 401
          ▼
  services.Dispatcher.Handle
      1. Insert into payments_webhook_events
         (unique (provider, providerEventID) → dedupe guard)
         duplicate → 200 {"status":"duplicate"}
      2. Lookup iface.SubscriptionReconciler from ServiceRegistry
      3. Call MarkInvoicePaid | MarkInvoiceFailed | RecordRefund
      4. MarkProcessed on the webhook event
```

All reconciler methods are idempotent, so a replayed webhook is double-safe: the dedupe insert fails first, but the reconciler's own short-circuits cover it either way.

## Cycle-free wiring with `subscriptions`

Neither module declares the other in `Dependencies()`. `payments` publishes `iface.PaymentProvider` (the façade) into the kernel's `ServiceRegistry`; `subscriptions` reads `iface.SubscriptionReconciler` from the same registry on every charge outcome. Because both registrations complete during `Init()` and lookups happen at runtime, there is no init-time ordering constraint.

If `payments` is disabled, the renewal job still generates invoices but marks them `awaiting_manual_payment` — the workflow keeps running so manual bank transfers can be reconciled by editing the invoice status.

## Metadata contract with subscriptions

`PaymentIntent.metadata` must carry `subscriptionUUID` + `invoiceUUID` — those are how webhook handlers find the right subscription/invoice to update. The subscriptions renewal service sets these on `chargeInvoice`, and Stripe echoes them back on every related event (succeeded, failed, refunded).

## How it ships

The same source tree lives in two places:

- **In-tree** at [`backend/internal/addons/payments/`](https://github.com/orkestra-cc/orkestra/tree/main/backend/internal/addons/payments) inside the [orkestra-cc/orkestra](https://github.com/orkestra-cc/orkestra) monorepo.
- **Standalone** at this repository, tagged from `v0.1.0`, consumed via the Go module proxy by anything outside the monorepo.

```go
require github.com/orkestra-cc/orkestra-addon-payments v0.1.0
replace github.com/orkestra-cc/orkestra-addon-payments => ./internal/addons/payments
```

The `replace` will retire once cross-cutting addon churn settles.

## Install

```bash
go get github.com/orkestra-cc/orkestra-addon-payments@latest
```

Requires Go 1.25.10 or newer.

```go
import payments "github.com/orkestra-cc/orkestra-addon-payments"
```

## Boot in a host

```go
import (
    payments "github.com/orkestra-cc/orkestra-addon-payments"
    subscriptions "github.com/orkestra-cc/orkestra-addon-subscriptions"
    "github.com/orkestra-cc/orkestra-sdk/module"
)

reg := module.NewModuleRegistry(logger) // your kernel's *slog.Logger
reg.Register(payments.NewModule())
reg.Register(subscriptions.NewModule())  // payments + subscriptions resolve each other via ServiceRegistry — no Dependencies() edge
```

The registry takes care of init order, Mongo collection creation, route mounting (operator-side admin + Tier-2 client surfaces + the public `/v1/payments/webhooks/stripe` endpoint), RBAC enforcement, and module-gate middleware for runtime enable/disable via `/admin/modules`.

## Tier-2 self-service routes (ADR-0003)

Mounted on `ri.Client.ProtectedRouter` behind `RequireGlobal()`; only mount when `TenantProvider` is wired.

| Method | Path | Body |
|---|---|---|
| `GET` | `/v1/me/transactions` | `?tenantUuid&subscriptionUuid&status` |
| `GET` | `/v1/me/payment-methods` | `?tenantUuid` |
| `POST` | `/v1/me/payments/checkout-session` | `{ subscriptionUuid, successUrl, cancelUrl }` |
| `POST` | `/v1/me/payments/setup-checkout-session` | `{ tenantUuid?, successUrl, cancelUrl }` |

`checkout-session` opens a Stripe Checkout in `mode=payment` for the subscription's most recent pending invoice; `setup-checkout-session` opens it in `mode=setup` so a user can save a card without a charge.

## Configuration (encrypted secrets)

| Field / env var | Purpose |
|---|---|
| `STRIPE_API_KEY` | `sk_test_…` / `sk_live_…` |
| `STRIPE_WEBHOOK_SECRET` | `whsec_…` from Stripe CLI / Dashboard |
| `STRIPE_API_VERSION` | Pinned to `2024-12-18.acacia` |
| `defaultProvider` | `"stripe"` in v1 |

Secret-typed fields are AES-256-GCM encrypted at rest via the SDK's `ConfigService`.

See [`CLAUDE.md`](CLAUDE.md) for the per-package map, MongoDB index layout, the tenant-scope rules (operator routes sit behind `RequireInternalTenant()`; the Stripe webhook stays public + HMAC-verified), and the not-in-v1 list.

## Versioning

Standard Go semver. `v0.x` allows breaking changes (rare); `v1.x` will freeze the public surface alongside the rest of the Orkestra addon ecosystem.

## Contributing

Most development happens in the [upstream monorepo](https://github.com/orkestra-cc/orkestra) where this addon lives alongside the kernel and its consumers. PRs against this standalone repo are welcome for addon-only changes.

See [CONTRIBUTING.md](https://github.com/orkestra-cc/orkestra/blob/main/CONTRIBUTING.md) in the monorepo for the contributor flow.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
