# Technical Controls — Implementation Reference

## Audit Logging Architecture

### Design Principles

1. **Append-only, immutable** — Logs must not be editable after write. Use write-once storage or cryptographic chaining.
2. **Structured and searchable** — JSON with consistent schema. No free-text dumping.
3. **Tamper-evident** — Hash chains or WORM storage so deletions/modifications are detectable.
4. **Centralized** — Ship to a dedicated log store that application owners cannot access to delete.
5. **Retained per policy** — 1 year minimum for most events, 3 years for admin/privileged actions.

### What to Log

| Category | Events | Retention | TSC Mapping |
|----------|--------|-----------|-------------|
| Authentication | Login success/failure, MFA challenge, password reset, session creation/destruction | 1 year | CC6.1, CC6.2 |
| Authorization | Permission grant/deny, role assignment, privilege escalation | 1 year | CC6.1, CC6.4 |
| Data access | Read/export/download of sensitive data, bulk queries | 1 year | CC6.1, C1.1 |
| Data mutation | Create, update, delete on business entities | 1 year | PI1.1, PI1.2 |
| System config | Infrastructure changes, feature flags, env var changes | 3 years | CC8.1 |
| Admin actions | User CRUD, policy changes, key rotation, permission changes | 3 years | CC6.1, CC6.3 |
| Security events | Blocked requests, rate limiting, WAF triggers, anomaly detections | 1 year | CC7.1, CC7.2 |

### What NOT to Log

- Passwords, tokens, secrets, API keys (even hashed — use a reference ID instead)
- Full PII in plaintext (mask to last 4, or log a pseudonymous identifier)
- Request/response bodies containing sensitive data (log metadata only)
- Health check pings (noise that obscures real events)

### Audit Log Schema (Go)

```go
package audit

import (
    "context"
    "crypto/sha256"
    "encoding/json"
    "time"

    "github.com/google/uuid"
)

type Action string

const (
    ActionAuthLogin       Action = "auth.login"
    ActionAuthLogout      Action = "auth.logout"
    ActionAuthMFA         Action = "auth.mfa"
    ActionDataRead        Action = "data.read"
    ActionDataWrite       Action = "data.write"
    ActionDataDelete      Action = "data.delete"
    ActionDataExport      Action = "data.export"
    ActionAdminUserCreate Action = "admin.user.create"
    ActionAdminRoleAssign Action = "admin.role.assign"
    ActionConfigChange    Action = "config.change"
    ActionSecurityBlock   Action = "security.block"
)

type Result string

const (
    ResultSuccess Result = "success"
    ResultFailure Result = "failure"
    ResultDenied  Result = "denied"
    ResultError   Result = "error"
)

type Entry struct {
    ID            string            `json:"id" bson:"_id"`
    Timestamp     time.Time         `json:"timestamp" bson:"timestamp"`
    Action        Action            `json:"action" bson:"action"`
    Result        Result            `json:"result" bson:"result"`
    Actor         Actor             `json:"actor" bson:"actor"`
    Resource      Resource          `json:"resource" bson:"resource"`
    Details       map[string]any    `json:"details,omitempty" bson:"details,omitempty"`
    CorrelationID string            `json:"correlation_id" bson:"correlation_id"`
    PreviousHash  string            `json:"previous_hash" bson:"previous_hash"`
    Hash          string            `json:"hash" bson:"hash"`
}

type Actor struct {
    ID        string `json:"id" bson:"id"`
    Email     string `json:"email" bson:"email"`
    IP        string `json:"ip" bson:"ip"`
    UserAgent string `json:"user_agent" bson:"user_agent"`
    Role      string `json:"role" bson:"role"`
}

type Resource struct {
    Type string `json:"type" bson:"type"`
    ID   string `json:"id" bson:"id"`
    Name string `json:"name,omitempty" bson:"name,omitempty"`
}

// NewEntry creates a hash-chained audit entry.
func NewEntry(action Action, result Result, actor Actor, resource Resource, prevHash string) Entry {
    e := Entry{
        ID:            uuid.NewString(),
        Timestamp:     time.Now().UTC(),
        Action:        action,
        Result:        result,
        Actor:         actor,
        Resource:      resource,
        CorrelationID: "", // set from context middleware
        PreviousHash:  prevHash,
    }
    e.Hash = e.computeHash()
    return e
}

func (e Entry) computeHash() string {
    data, _ := json.Marshal(struct {
        ID        string    `json:"id"`
        Timestamp time.Time `json:"ts"`
        Action    Action    `json:"action"`
        PrevHash  string    `json:"prev"`
    }{e.ID, e.Timestamp, e.Action, e.PreviousHash})
    h := sha256.Sum256(data)
    return fmt.Sprintf("%x", h)
}
```

### Middleware Pattern (Go + Chi/Echo)

```go
func AuditMiddleware(logger *audit.Logger) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            correlationID := r.Header.Get("X-Correlation-ID")
            if correlationID == "" {
                correlationID = uuid.NewString()
            }
            ctx := context.WithValue(r.Context(), ctxKeyCorrelation, correlationID)
            
            wrapped := &responseWriter{ResponseWriter: w, statusCode: 200}
            next.ServeHTTP(wrapped, r.WithContext(ctx))

            // Log after response — don't block the request
            go logger.Write(ctx, audit.Entry{
                Action:        actionFromRoute(r),
                Result:        resultFromStatus(wrapped.statusCode),
                Actor:         actorFromContext(ctx),
                Resource:      resourceFromRequest(r),
                CorrelationID: correlationID,
            })
        })
    }
}
```

### Storage Backends

| Backend | Immutability | Search | Cost | Best For |
|---------|-------------|--------|------|----------|
| MongoDB (capped + no delete perms) | Application-level | Good (indexes) | Low | Self-hosted, your existing stack |
| S3 + Object Lock (WORM) | Storage-level | Via Athena/OpenSearch | Medium | Tamper-proof archival |
| CloudWatch Logs / Loki | Managed | Native | Medium | Cloud-native ops |
| TimescaleDB (hypertables) | Application-level | Excellent (SQL) | Low | Analytics on audit data |
| Dedicated SIEM (Wazuh, Elastic) | Managed | Excellent | High | Large scale, correlation |

**Recommendation for self-hosted Go/MongoDB stacks:** Write audit logs to a dedicated MongoDB collection with no application-level delete permissions. Replicate to S3-compatible storage (MinIO with object locking) nightly for tamper-proof archival. Use TTL indexes for retention enforcement.

## Access Control

### RBAC Implementation (Go)

```go
package rbac

type Permission struct {
    Resource string `json:"resource" bson:"resource"` // "projects", "users", "billing"
    Action   string `json:"action" bson:"action"`     // "read", "write", "delete", "admin"
}

type Role struct {
    Name        string       `json:"name" bson:"name"`
    Permissions []Permission `json:"permissions" bson:"permissions"`
    Inherits    []string     `json:"inherits,omitempty" bson:"inherits,omitempty"`
}

// Predefined roles — customize per system
var DefaultRoles = map[string]Role{
    "viewer": {
        Name: "viewer",
        Permissions: []Permission{
            {Resource: "*", Action: "read"},
        },
    },
    "editor": {
        Name: "editor",
        Permissions: []Permission{
            {Resource: "*", Action: "read"},
            {Resource: "*", Action: "write"},
        },
    },
    "admin": {
        Name: "admin",
        Permissions: []Permission{
            {Resource: "*", Action: "*"},
        },
    },
}

type Enforcer struct {
    roles map[string]Role
}

func (e *Enforcer) Check(userRole, resource, action string) bool {
    role, ok := e.roles[userRole]
    if !ok {
        return false
    }
    for _, p := range role.Permissions {
        if (p.Resource == "*" || p.Resource == resource) &&
           (p.Action == "*" || p.Action == action) {
            return true
        }
    }
    // Check inherited roles
    for _, inherited := range role.Inherits {
        if e.Check(inherited, resource, action) {
            return true
        }
    }
    return false
}
```

### MFA Enforcement

Auditors will check:
- MFA enrollment rate (target: 100% for in-scope users)
- MFA enforced on privileged actions (not just login)
- MFA bypass process documented and requires approval

### Access Review Process

| Frequency | Scope | Reviewer | Evidence |
|-----------|-------|----------|----------|
| Quarterly | All user access to in-scope systems | Manager + system owner | Exported user list, review sign-off |
| On termination | All systems | IT/HR (automated) | Deprovisioning ticket, timestamp |
| On role change | Affected systems | New manager | Access modification ticket |
| Annually | Service accounts, API keys | System owner | Key inventory, rotation records |

**Key metric:** Time-to-revoke on termination. Auditors sample terminated employees and check when access was actually removed. Target: < 24 hours. Automate via HR system → IdP webhook.

### API Key & Secret Management

- Rotate production secrets every 90 days maximum
- Store in a secrets manager (Vault, AWS SSM, or env-injected — never in code/config files)
- Log every secret access (who read which secret, when)
- Separate secrets per environment (dev/staging/prod never share)
- Service accounts must have individual credentials, not shared

## Encryption

### At Rest

| Layer | Mechanism | Verification |
|-------|-----------|-------------|
| Database | MongoDB encrypted storage engine / LUKS | Config screenshot, encryption status query |
| Object storage | MinIO server-side encryption (SSE-S3 or SSE-KMS) | Bucket policy export |
| Backups | GPG/age encryption before upload | Backup script showing encryption step |
| Disk | LUKS full-disk encryption | `lsblk` + `cryptsetup status` output |

### In Transit

| Component | Mechanism | Verification |
|-----------|-----------|-------------|
| External API | TLS 1.2+ (prefer 1.3) | SSL Labs scan, `nmap --script ssl-enum-ciphers` |
| Internal services | mTLS or WireGuard | Cert configs, WireGuard peer list |
| Database connections | TLS required (reject plaintext) | MongoDB `--tls` flag, connection string |

### Key Management

- Document key hierarchy (master key → data encryption keys)
- Key rotation schedule (master: annual, DEKs: quarterly or on-demand)
- Key revocation procedure with timeline
- Backup of key material (encrypted, separate from data)

## Vulnerability Management

### Pipeline Integration

```yaml
# CI/CD security gates — integrate into existing pipeline
security-scan:
  steps:
    # Dependency vulnerabilities
    - name: SCA scan
      run: govulncheck ./... # Go-native; or trivy, snyk
      fail_on: critical,high

    # Static analysis
    - name: SAST
      run: semgrep --config=auto --error
      fail_on: error

    # Secret detection
    - name: Secret scan
      run: gitleaks detect --source=. --verbose
      fail_on: any

    # Container scanning (if applicable)
    - name: Container scan
      run: trivy image $IMAGE_TAG
      fail_on: critical,high
```

### Vulnerability SLA

| Severity | Patch Deadline | Escalation |
|----------|---------------|------------|
| Critical (CVSS 9.0+) | 72 hours | Immediate to security lead |
| High (CVSS 7.0–8.9) | 14 days | Weekly security review |
| Medium (CVSS 4.0–6.9) | 30 days | Monthly review |
| Low (CVSS < 4.0) | 90 days | Quarterly review |

Track all vulnerabilities in a register with: discovery date, severity, affected system, remediation date, owner.

## Network Security

- Document network architecture diagram (required for audit)
- Segment management, application, and data networks (VLANs)
- Firewall rules follow deny-by-default
- VPN/WireGuard for administrative access
- No direct internet exposure for databases or internal services
- Regular port scans to verify exposure matches intent
