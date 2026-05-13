<div align="center">

# Orkestra Dev Token addon

**Mint short-lived dev/staging JWTs (any role, any audience) without a database write — the auth shortcut behind `scripts/devtoken.sh` for the [Orkestra](https://github.com/orkestra-cc/orkestra) modular monolith. Gated behind `module.PlatformInfo.IsProduction()` so the routes never register in prod.**

[![Go Reference](https://pkg.go.dev/badge/github.com/orkestra-cc/orkestra-addon-dev.svg)](https://pkg.go.dev/github.com/orkestra-cc/orkestra-addon-dev)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white&style=flat-square)](https://go.dev)
[![Module](https://img.shields.io/badge/module-github.com%2Forkestra--cc%2Forkestra--addon--dev-blue?style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-dev)
[![Latest tag](https://img.shields.io/github/v/tag/orkestra-cc/orkestra-addon-dev?sort=semver&style=flat-square)](https://github.com/orkestra-cc/orkestra-addon-dev/tags)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache_2.0-blue.svg?style=flat-square)](LICENSE)

[SDK](https://github.com/orkestra-cc/orkestra-sdk) · [Monorepo](https://github.com/orkestra-cc/orkestra) · [Module docs](CLAUDE.md)

</div>

---

## What this is

A tiny Orkestra addon (~600 LOC) implementing the `Module` interface from [`orkestra-cc/orkestra-sdk`](https://github.com/orkestra-cc/orkestra-sdk). Exposes two endpoints on the operator surface that mint synthetic JWTs for any role + audience combination, without writing to the user database.

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/v1/dev/token` | Mint a JWT (role + optional audience + expiry) |
| `GET` | `/v1/dev/roles` | List valid roles + current environment |

Two `iface.JWTProvider`s sit behind the handler — one for the `operator` audience (the default), one for `client` — so the same dev-token call can target either host mux. ADR-0003 PR-D D-10 stamps an explicit `aud` claim on every minted token so the surface's `RequireAudience` gate accepts it.

## Defense in depth

- **First-boot gate**: `Enabled()` returns `!IsProduction()`, so the seeded `module_configs` document keeps the module disabled in production.
- **Runtime gate**: every handler re-checks `PlatformInfo.IsProduction()` before consulting the JWT providers — even if the module gets toggled on by mistake in a prod deployment, the routes refuse to mint.

## How it ships

The same source tree lives in two places:

- **In-tree** at [`backend/internal/addons/dev/`](https://github.com/orkestra-cc/orkestra/tree/main/backend/internal/addons/dev) inside the [orkestra-cc/orkestra](https://github.com/orkestra-cc/orkestra) monorepo, where cross-module development happens.
- **Standalone** at this repository, tagged from `v0.1.0`, consumed via the Go module proxy by anything outside the monorepo.

```go
require github.com/orkestra-cc/orkestra-addon-dev v0.1.0
replace github.com/orkestra-cc/orkestra-addon-dev => ./internal/addons/dev
```

The `replace` will retire once cross-cutting addon churn settles.

## Install

```bash
go get github.com/orkestra-cc/orkestra-addon-dev@latest
```

Requires Go 1.25.10 or newer.

```go
import dev "github.com/orkestra-cc/orkestra-addon-dev"
```

## Boot in a host

```go
import (
    dev "github.com/orkestra-cc/orkestra-addon-dev"
    "github.com/orkestra-cc/orkestra-sdk/module"
)

reg := module.NewModuleRegistry(logger) // your kernel's *slog.Logger
reg.Register(dev.NewModule())
```

The registry resolves `module.ServiceJWTService` + `module.ServiceClientJWTService` from the kernel's `ServiceRegistry` and injects them into the handler. The module is inert in production — both first-boot activation and per-request gates consult `PlatformInfo.IsProduction()`.

## Request body

```json
{
  "role":     "administrator",
  "audience": "operator",
  "expiry":   "15m"
}
```

| Field | Default | Validity |
|---|---|---|
| `role` | (required) | `super_admin`, `administrator`, `developer`, `manager`, `operator`, `guest` |
| `audience` | `operator` | `operator`, `client` |
| `expiry` | `15m` | `1m`–`24h` |

Unknown audiences return 400 *before* either JWT provider is invoked.

See [`CLAUDE.md`](CLAUDE.md) for the per-handler map, the `scripts/devtoken.sh` consumer contract, and the production-gate rationale.

## Versioning

Standard Go semver. `v0.x` allows breaking changes (rare); `v1.x` will freeze the public surface alongside the rest of the Orkestra addon ecosystem.

## Contributing

Most development happens in the [upstream monorepo](https://github.com/orkestra-cc/orkestra) where this addon lives alongside the kernel. PRs against this standalone repo are welcome for addon-only changes.

See [CONTRIBUTING.md](https://github.com/orkestra-cc/orkestra/blob/main/CONTRIBUTING.md) in the monorepo for the contributor flow.

## License

Licensed under the [Apache License, Version 2.0](LICENSE).
