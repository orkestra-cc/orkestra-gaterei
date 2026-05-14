# Module: Dev Token Generator

_Path: `/backend/internal/addons/dev`_
_Parent: [../../../CLAUDE.md](../../../CLAUDE.md)_

[← Backend](../../../CLAUDE.md) | [☰ Module Map](../../../../../CLAUDE.md#module-map)

## Module home

This directory is a **separate Go module**
(`github.com/orkestra-cc/orkestra-addon-dev`) since Phase 5i of the
SDK split — first of the "hard tier" extractions. Source lives
in-tree at this path for monorepo development; the same tree is
mirrored to
[github.com/orkestra-cc/orkestra-addon-dev](https://github.com/orkestra-cc/orkestra-addon-dev)
and tagged starting from `v0.1.0`. Backend's `go.mod` carries a
`replace` directive pointing at this path so changes here take effect
without a tag bump during cross-cutting work; CI and external
consumers fetch the published version through the Go module proxy.

Two boundary tightenings landed in the same commit:

- `handlers/dev_token_handler.go` dropped its
  `github.com/orkestra/backend/internal/core/user/models` import in
  favor of the SDK-canonical `iface.User`. The `core/user/models`
  package was already a re-export shim around `iface.User`, so the
  swap was a one-symbol edit with no behavior change.
- `handlers/dev_token_audience_test.go` dropped its
  `github.com/orkestra/backend/internal/shared/config` import. The
  test only needed `cfg.Server.Environment = "development"` to drive
  the handler's production-mode gate; replaced with a local
  `stubPlatform` that implements `module.PlatformInfo` directly. Same
  testkit-replacement pattern used by the Phase 5f/5g extractions
  (subscriptions, payments).

## What it does

Mints short-lived JWTs (any role, any audience) without writing to
the database, so integration tests, dev-loop `curl` probes, and the
preview-environment smoke runs can authenticate against the operator
or client surfaces without going through OAuth / email-password flows.

The handler holds **one `iface.JWTProvider` per audience**
(`operator`, `client`) and dispatches by the request body's
`audience` field — ADR-0003 PR-D D-10. Each minted token carries an
explicit `aud` claim so the host muxes' `RequireAudience` gates
accept it on the matching surface.

## Production gate

`DevTokenHandler` consults `module.PlatformInfo.IsProduction()` on
every request. When the runtime environment is `production`, the
handler returns a 4xx error and refuses to call into the JWT
providers — defense in depth in case the module accidentally gets
toggled on in a prod deployment. The module's own `Enabled()` already
gates first-boot activation on `!IsProduction()` so the routes
typically don't even register in prod.

## Endpoints (`/v1/dev`)

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/token` | Mint a JWT (role + optional audience + expiry) |
| `GET` | `/roles` | List the valid roles + current environment |

The HTTP wrappers `GenerateTokenHTTP` / `ListRolesHTTP` are registered
directly on the operator chi router so `scripts/devtoken.sh` can
shell out via plain HTTP without going through Huma's router.

## Request body

```json
{
  "role":     "administrator",
  "audience": "operator",       // optional; default "operator"
  "expiry":   "15m"             // optional; default 15m, max 24h
}
```

Valid roles: `super_admin`, `administrator`, `developer`, `manager`,
`operator`, `guest`. Valid audiences: `operator`, `client`. Anything
else returns 400 before either provider is invoked.

## Module surface

| Field | Value |
|---|---|
| `Name()` | `dev` |
| `Category()` | `CategoryToggleable` |
| `Enabled()` | `!IsProduction()` — gates first-boot activation |
| `RequiredServices()` | `module.ServiceJWTService`, `module.ServiceClientJWTService` |
| `NavItems()` | none |
| `Collections()` | none |
| `Permissions()` | none — the routes themselves are dev-only |

## Not in this module

Persistent users (the synthetic user lives in memory only — no DB
write), refresh-token rotation (the minted token is access-only), or
any production-mode bypass. Use the real auth flows when not in dev /
staging.
