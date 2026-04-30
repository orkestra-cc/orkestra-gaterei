---
name: api-test
description: Generate dev tokens and test protected API endpoints. Use when you need to authenticate with the backend API for testing purposes.
---

# API Test Skill

This skill helps test protected API endpoints by generating dev tokens and making authenticated requests.

## When to Use This Skill

Use this skill when you need to:
- Generate JWT tokens for different roles to test API endpoints
- Make authenticated requests to protected endpoints
- Verify that authentication and authorization are working correctly
- Debug API responses with different role permissions

## Configuration

- **API Base URL**: `http://localhost:3007` (development) or `http://localhost:3000`
- **Dev Token Endpoint**: `POST /dev/token`
- **Roles Endpoint**: `GET /dev/token/roles`

## Available Roles (Hierarchical Access)

| Role            | Access Level | Description                                |
|-----------------|--------------|---------------------------------------------|
| `super_admin`   | Highest      | Wildcard — bypasses all permission checks   |
| `administrator` | High         | All registered permissions                  |
| `developer`     | High         | All registered permissions (technical role) |
| `manager`       | Medium       | Read + create + update (no delete/admin)    |
| `operator`      | Low          | Read-only + self-service actions            |
| `guest`         | Lowest       | Read-only access                            |

## Instructions

### Step 1: Determine What the User Needs

If not already specified, determine:
- Which role to use for the token
- Which endpoint(s) to test
- Any specific request body or query parameters

### Step 2: Generate a Dev Token

Run this command to generate a token:

```bash
curl -s -X POST http://localhost:3007/dev/token \
  -H "Content-Type: application/json" \
  -d '{"role": "ROLE_NAME", "expiry": "15m"}' | jq .
```

Replace `ROLE_NAME` with the appropriate role.

### Step 3: Test the Endpoint

Use the generated token to make authenticated requests:

```bash
TOKEN=$(curl -s -X POST http://localhost:3007/dev/token \
  -H "Content-Type: application/json" \
  -d '{"role": "ROLE_NAME"}' | jq -r .accessToken)

curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:3007/api/v1/ENDPOINT" | jq .
```

### Step 4: Present Results

Format and explain the API response to the user.

## Common Endpoints by Required Role

### Administrator+ Required
- `GET /api/v1/users` - List users
- `POST /api/v1/users` - Create user
- `GET /api/v1/users/{id}` - Get user by ID

### Manager+ Required
- `GET /api/v1/billing/invoices` - List invoices
- `GET /api/v1/billing/customers` - List customers
- `GET /api/v1/documents/templates` - List document templates


### Any Authenticated User
- `GET /api/v1/navigation` - Get navigation menu
- `GET /health` - Health check (no auth required)

## Quick Commands

```bash
# List available roles
curl -s http://localhost:3007/dev/token/roles | jq .

# Generate token with 1 hour expiry
curl -s -X POST http://localhost:3007/dev/token \
  -H "Content-Type: application/json" \
  -d '{"role": "administrator", "expiry": "1h"}' | jq .

# One-liner to test any endpoint
ROLE=administrator ENDPOINT=users && \
curl -s -H "Authorization: Bearer $(curl -s -X POST http://localhost:3007/dev/token -H "Content-Type: application/json" -d "{\"role\":\"$ROLE\"}" | jq -r .accessToken)" \
  "http://localhost:3007/api/v1/$ENDPOINT" | jq .
```

## Error Handling

| Error | Meaning |
|-------|---------|
| 401 Unauthorized | Token missing, expired, or invalid |
| 403 Forbidden | Role lacks permission for this endpoint |
| 404 Not Found | Endpoint doesn't exist |

## Important Notes

- Dev tokens only work in **development** and **staging** environments
- Tokens create synthetic users (no database writes)
- Default expiry is 15 minutes, maximum is 24 hours
- Always use the minimum required role for testing
