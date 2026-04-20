# Module: Subscriptions

_Path: `/backend/internal/addons/subscriptions`_
_Parent: [../../../CLAUDE.md](../../../CLAUDE.md)_

Recurring-revenue core: a catalog of AI services the operator sells, a registry of external clients (not tenants), and subscriptions linking the two with cycle-based billing and an append-only activity log.

## Responsibility split

- **Subscriptions owns**: what's for sale, who bought it, when the next charge is due, and an audit trail. No gateway code.
- **Payments owns**: Stripe (and later PayPal) API calls + webhook verification. Subscriptions consumes it through `iface.PaymentProvider`.
- **Billing (FatturaPA)** is NOT involved in v1 — subscription invoices are internal receipts (number format `SUB-YYYY-NNNNNN`), separate from fiscal compliance. Future integration with `billing` for Italian B2B clients is deferred.

## Cycle-free wiring with `payments`

Neither module declares the other in `Dependencies()`. `subscriptions` resolves `iface.PaymentProvider` lazily from `ServiceRegistry` on every charge (see `services/renewal_service.go:paymentProvider`). `payments` does the same for `iface.SubscriptionReconciler`. Because both registrations complete during `Init()` and lookups happen during `Start()` / HTTP / ticker, there is no init-time ordering constraint.

If `payments` is disabled, the renewal job still generates invoices but marks them `awaiting_manual_payment` and emits `manual_payment_required` activity — the workflow keeps running so manual bank transfers can be reconciled by editing the invoice status.

## State machine (`models.SubStatus`)

```
active ──charge_failed──▶ past_due ──retry_succeeded──▶ active
                              │
                              └──max_failures (3)──▶ suspended
active | past_due | suspended ──admin_cancel──▶ cancelled (terminal)
active ──cancel_at_period_end─▶ (ends at CurrentPeriodEnd)─▶ cancelled
```

Transitions happen in `services/subscription_service.go` (user actions) and `services/renewal_service.go` (charge outcomes). The reconciler (webhook path) transitions on async events and **is idempotent** — receiving the same Stripe event twice is a no-op.

## Renewal job

`jobs/renewal_job.go` tickles every `SUBSCRIPTIONS_RENEWAL_INTERVAL` (default 1h). Each tick: find subscriptions with `NextBillingAt <= now` AND `status ∈ {active, past_due}`, generate an invoice, attempt a charge, update state. Failed charges push `NextBillingAt` out by 1 day so we retry tomorrow, not every tick.

## Kind gate + dual-write + backfill (ADR-0001 Phase 1–2)

Every operator-admin route — service catalog, client CRUD, subscription admin, invoice/activity reads — now sits behind `RequireInternalTenant()`. The self-service `POST /v1/me/subscriptions` stays kind-agnostic because that path exists specifically for external tenants to subscribe themselves. Rollout respects `TENANT_KIND_ENFORCEMENT=warn|enforce`.

Subscription rows dual-write `ClientUUID` (legacy) and `TenantUUID` (forward-looking). `SubscriptionService.Create` resolves TenantUUID from `Client.OrgUUID` or lazy-provisions via `iface.TenantProvider.FindOrProvisionLegacyClientTenant` (idempotent; keyed on `tenant_orgs.metadata.legacyClientUUID` unique sparse index) and back-stamps `Client.OrgUUID`. A new `CreateForTenantUUID` service method supports operator-admin subscribe directly by TenantUUID.

Cold-tail rows are picked up by the one-shot migration `migrations/0002_backfill_tenant_uuid.go`, triggered by `SUBSCRIPTIONS_RUN_BACKFILL=1` + `SUBSCRIPTIONS_BACKFILL_OWNER_UUID=<uuid>` on deploy. Completion is recorded in `subscriptions_migrations` so reboots skip the scan. The same migration walks `payments_transactions` after pass 1 finishes.

A thin `TenantSubscriptionAdapter` (`services/tenant_provider.go`) implements `iface.TenantSubscriptionProvider` so `core/tenant` can serve `GET /v1/admin/tenants/{id}/subscriptions` without importing this addon.

## Collections

| Collection | Notes |
|---|---|
| `subscriptions_services` | Catalog items with pricing tiers. `code` is unique. |
| `subscriptions_clients` | **Deprecated** (ADR-0001). External buyers. `email` unique. `orgUUID` nullable link to tenant — Phase 1 back-stamps this for every live client. |
| `subscriptions_subscriptions` | Cycle state. `(clientUUID, status)` for legacy lookups, `(tenantUUID, status)` sparse for Phase 2 aggregator, `(nextBillingAt, status)` for the renewal scan. |
| `subscriptions_invoices` | Generated per cycle. `(subscriptionUUID, periodStart)` unique prevents double-billing the same cycle. |
| `subscriptions_activity` | Append-only audit log. `(subscriptionUUID, createdAt desc)` indexed for the detail-page timeline. |
| `subscriptions_migrations` | One-shot migration completion rows. `name` unique. |

## Permissions

```
subscriptions.service.{view,manage}
subscriptions.client.{view,manage}
subscriptions.subscription.{view,manage}
subscriptions.invoice.view
subscriptions.activity.view
```

## Not in v1

Trials, proration, metered usage, multi-currency, FatturaPA issuance, self-service client portal, dunning sequences beyond one failure email, org-scoped RBAC.
