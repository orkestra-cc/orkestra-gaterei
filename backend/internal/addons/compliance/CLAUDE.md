# Module: Compliance (Audit + DSR + SOC2)

_Path: `/backend/internal/addons/compliance`_
_Parent: [../../../CLAUDE.md](../../../CLAUDE.md)_

[← Backend](../../../CLAUDE.md) | [☰ Module Map](../../../../../CLAUDE.md#module-map)

## Module home

This directory is a **separate Go module**
(`github.com/orkestra-cc/orkestra-addon-compliance`) since Phase 5j of
the SDK split — the second of the "hard tier" extractions (after dev,
Phase 5i). Source lives in-tree at this path for monorepo
development; the same tree is mirrored to
[github.com/orkestra-cc/orkestra-addon-compliance](https://github.com/orkestra-cc/orkestra-addon-compliance)
and tagged starting from `v0.1.0`. Backend's `go.mod` carries a
`replace` directive pointing at this path so changes here take effect
without a tag bump during cross-cutting work; CI and external
consumers fetch the published version through the Go module proxy.

### Boundary lifts that unblocked this extraction

Compliance was the most-entangled remaining addon: its post-init wiring
fished concrete pointer types out of the `ServiceRegistry` for six
sibling services (PasswordAuthService, AuthPolicyService,
tenantServices.Service, identityServices.Service,
identityHandlers.{AdminHandler, ScimAdminHandler},
subscriptionServices.SubscriptionService) to push the audit sink into
each one. Three small interfaces were lifted into the SDK to break
those concrete-type imports:

- **`iface.AuditSinkSetter`** — anything that exposes
  `SetAuditSink(iface.AuditSink)`. All seven receivers already had
  this exact method signature, so the satisfaction is automatic and
  the compliance probe collapses to a single loop over service keys.
- **`iface.KMSProviderSetter`** — `tenantServices.Service` exposes
  `SetKMSProvider(iface.KMSProvider)`. Compliance probes for it from
  its own Init after constructing the KMS provider.
- **`iface.ClientSelfDeletionGate`** — the single method on
  `AuthPolicyService` that compliance reads
  (`SelfServiceAccountDeletionClient(ctx) bool`). The whole concrete
  policy service stays in auth; compliance holds only this slice.

The SOC2 evidence service queries `operator_users` and
`operator_mfa_factors` collections directly. Those names are inlined
in `services/soc2.go` as private constants (the cross-module
collection-naming contract is documented in the same file) so the
addon doesn't have to import `core/user/repository` or
`core/auth/models` for the literals.

### testkit replacement

`handlers/me_handler_test.go` was the last serial consumer of
`backend/internal/testkit`. Phase 5j replaced the testkit calls with
the same inline `authedCtx` helper used by the subscriptions /
payments / dev extractions — stamps `ctxauth.Key*` constants onto
the context directly. `internal/testkit` is now backend-internal
only, and its drift-catching identity_test.go still pins the
context-key shape against the SDK's `ctxauth` getters.

## What it does

The platform compliance plane:

1. **Audit log** — owns the append-only `audit_events` collection
   and publishes `iface.AuditSink` so every other module can emit
   events without importing compliance.
2. **DSR pipeline** — fans out across every registered
   `iface.PIIProducer` to satisfy GDPR right-of-access (export) and
   right-to-erasure (purge) requests. Tier-2 self-service via
   `/v1/me/dsr/*`, gated by the `selfServiceAccountDeletionClient`
   policy on erase.
3. **KMS lifecycle** — per-tenant envelope encryption + crypto-shred
   on purge. Pushes the KMS provider into the tenant service so
   tenant deletion can shred the key alongside the data.
4. **SOC2 evidence** — read-only aggregation snapshot at
   `/v1/admin/compliance/soc2/evidence` covering CC6.1 (privileged
   users), CC6.6 (MFA coverage), CC6.8 (data protection), CC7.2
   (monitoring + audit coverage).

## Cross-module wiring

Compliance pushes its sink into every `AuditSinkSetter` it can
resolve from the ServiceRegistry at boot. Receivers (today: 7
services) declare nothing about compliance — they only expose
`SetAuditSink(iface.AuditSink)` as a regular method, and compliance
finds them by service-key + interface-typed `GetTyped`.

| Compliance probes from | Resolved-as interface | Service key |
|---|---|---|
| Tenant service | `iface.AuditSinkSetter` + `iface.KMSProviderSetter` | `ServiceTenantService` |
| Operator password auth | `iface.AuditSinkSetter` | `ServicePasswordAuthService` |
| Client password auth | `iface.AuditSinkSetter` | `ServiceClientPasswordAuthService` |
| Identity OIDC service | `iface.AuditSinkSetter` | `ServiceIdentityOIDCService` |
| Identity admin handler | `iface.AuditSinkSetter` | `ServiceIdentityAdminHandler` |
| Identity SCIM handler | `iface.AuditSinkSetter` | `ServiceIdentityScimAdminHandler` |
| Subscription service | `iface.AuditSinkSetter` | `ServiceSubscriptionService` |
| Operator user provider | `iface.AuditSinkSetter` | `ServiceOperatorUserProvider` |
| Client user provider | `iface.AuditSinkSetter` | `ServiceClientUserProvider` |
| Auth policy service | `iface.ClientSelfDeletionGate` | `ServiceAuthPolicy` |

Missing services (module disabled, out of init order) are silently
skipped — compliance boots cleanly regardless of which optional
modules are active.

## Endpoints

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/v1/admin/audit-events` | Operator-tier event reader |
| `GET` | `/v1/admin/compliance/soc2/evidence` | SOC2 snapshot |
| `POST` | `/v1/me/dsr/export` | Tier-1 (operator-tier user) export |
| `POST` | `/v1/me/dsr/erase` | Tier-1 erase |
| `POST` | `/v1/me/dsr/export` (client surface) | Tier-2 client export (always available) |
| `POST` | `/v1/me/dsr/erase` (client surface) | Tier-2 client erase — only mounted when `selfServiceAccountDeletionClient` is true |

## Collections

| Collection | Notes |
|---|---|
| `audit_events` | Append-only. TTL = 2 years (covers SOC2 1-year + GDPR legitimate-interest retention). Indexes on tenantId+timestamp, actorUserId+timestamp, action+timestamp, resourceType+resourceId. |
| `kms_keys` | One row per tenant. `tenantUuid` unique. |
