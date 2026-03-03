# Module: Company Lookup - OpenAPI Company API

_Path: `/backend/internal/company`_
_Parent: [../../CLAUDE.md](../../CLAUDE.md)_

<!-- Navigation -->

[← Backend](../../CLAUDE.md) | [☰ Module Map](../../../../CLAUDE.md#module-map)

<!-- /Navigation -->

## Module Purpose

The company module provides **Italian company data lookup** by tax code (Codice Fiscale) or VAT number (Partita IVA) through integration with the OpenAPI Company API (`company.openapi.com`).

- **Primary Role**: Look up external company data for auto-filling customer/supplier details and company verification
- **External Integration**: OpenAPI Company API for real-time company information
- **Conditional Activation**: Module activates only when `OPENAPI_BEARER_TOKEN` is configured (shared with billing)
- **Scope**: Italy only (`/IT-start/{taxCode}` endpoint)

## Module Structure

```
company/
├── config/
│   └── config.go                  # CompanyAPIConfig (BaseURL, token, timeouts)
├── models/
│   ├── company.go                 # CompanyLookup domain model (MongoDB document)
│   └── dto.go                     # API response types + Huma request/response DTOs
├── repository/
│   └── company_repository.go      # Interface + MongoDB impl (company_lookups collection)
├── services/
│   ├── company_service.go         # Interface + business logic (lookup orchestration)
│   └── openapi_client.go          # External HTTP client (company.openapi.com)
├── handlers/
│   └── company_handler.go         # Huma HTTP handlers
├── routes.go                      # Route registration
└── CLAUDE.md                      # This file
```

## API Endpoints

| Method | Endpoint | Operation ID | Description |
|--------|----------|-------------|-------------|
| GET | `/v1/company/lookup/{taxCode}` | `lookup-company` | Look up company by tax code (calls external API, caches result) |
| GET | `/v1/company/lookups` | `list-company-lookups` | List previously looked-up companies (paginated) |
| GET | `/v1/company/lookups/search` | `search-company-lookups` | Search stored lookups by name/tax code/VAT code |
| GET | `/v1/company/lookups/{id}` | `get-company-lookup` | Get a specific stored lookup by UUID |

## Lookup Flow

```
Request → Redis Cache Check → External API Call → MongoDB Upsert → Redis Cache Set → Response
                ↓ (hit)
            Return cached
```

1. **Redis cache** checked first (key: `company:lookup:{taxCode}`, TTL: 24h default)
2. **External API** called on cache miss (`GET /IT-start/{taxCode}`)
3. **MongoDB** upserted with result (deduplicated by `taxCode`)
4. **Redis** cached for future lookups
5. **Response** returned to client

## External API

- **Production**: `https://company.openapi.com`
- **Sandbox**: `https://test.company.openapi.com`
- **Authentication**: Bearer token (shared with billing module's `OPENAPI_BEARER_TOKEN`)
- **Endpoint used**: `GET /IT-start/{vatCode_taxCode_or_id}`
- **Free tier**: 30 requests/month

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `COMPANY_API_BASE_URL` | Override base URL | Derived from `OPENAPI_SANDBOX_MODE` |
| `COMPANY_API_TIMEOUT` | HTTP request timeout | `15s` |
| `COMPANY_API_RETRY_ATTEMPTS` | Retry count on failure | `3` |
| `COMPANY_API_CACHE_TTL` | Redis cache TTL | `24h` |
| `OPENAPI_BEARER_TOKEN` | Authentication token (shared with billing) | _(required to enable)_ |
| `OPENAPI_SANDBOX_MODE` | Use sandbox environment | `true` |

## MongoDB Collection

| Collection | Description |
|------------|-------------|
| `company_lookups` | Company lookup results |

### Indexes

- `uuid` (unique)
- `taxCode` (unique)
- `vatCode`
- `companyName` (text)
- `createdAt` (descending)

## Resilience

- **Circuit breaker**: 5 failures → 30s open → half-open test
- **Retry with backoff**: Exponential backoff (1s, 2s, 4s...), skips 4xx errors
- **Redis caching**: 24h TTL prevents unnecessary API calls
- **MongoDB persistence**: Previously looked-up companies are always available

## Access Control

- **Role requirement**: `manager` and above (same as billing)
- **Authentication**: JWT Bearer token via auth middleware

---

### Related Guides

- [Billing Module](../billing/CLAUDE.md) - Shares the same OpenAPI.com bearer token
- [Backend Module](../../CLAUDE.md) - Backend architecture and development workflow
- [Project Overview](../../../../CLAUDE.md) - System architecture
