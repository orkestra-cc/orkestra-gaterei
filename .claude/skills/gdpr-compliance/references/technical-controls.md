# Technical Controls — Privacy Implementation Reference

## PII Detection and Masking

### PII Field Registry

Maintain a central registry of all PII fields across your data stores. This is the foundation for masking, retention, access logging, and DSR fulfillment.

```go
package pii

// Sensitivity levels determine handling requirements.
type Sensitivity int

const (
    SensitivityPublic      Sensitivity = 0 // Company name, public URL
    SensitivityInternal    Sensitivity = 1 // Internal user ID
    SensitivityPersonal    Sensitivity = 2 // Name, email, phone
    SensitivitySensitive   Sensitivity = 3 // Health, religion, biometric
    SensitivityRestricted  Sensitivity = 4 // Passwords, financial, government ID
)

type FieldDefinition struct {
    Name        string      `json:"name"`
    Sensitivity Sensitivity `json:"sensitivity"`
    MaskFunc    string      `json:"mask_func"` // Masking strategy name
    Searchable  bool        `json:"searchable"` // Can be used in queries
    Exportable  bool        `json:"exportable"` // Included in Art. 15 access export
    Portable    bool        `json:"portable"`   // Included in Art. 20 portability
    Erasable    bool        `json:"erasable"`   // Deleted on Art. 17 erasure
}

// Registry of known PII fields — customize for your data model.
var PIIRegistry = map[string]FieldDefinition{
    "email":          {Sensitivity: SensitivityPersonal, MaskFunc: "mask_email", Searchable: true, Exportable: true, Portable: true, Erasable: true},
    "name":           {Sensitivity: SensitivityPersonal, MaskFunc: "mask_name", Searchable: true, Exportable: true, Portable: true, Erasable: true},
    "phone":          {Sensitivity: SensitivityPersonal, MaskFunc: "mask_phone", Searchable: false, Exportable: true, Portable: true, Erasable: true},
    "address":        {Sensitivity: SensitivityPersonal, MaskFunc: "mask_full", Searchable: false, Exportable: true, Portable: true, Erasable: true},
    "date_of_birth":  {Sensitivity: SensitivityPersonal, MaskFunc: "mask_full", Searchable: false, Exportable: true, Portable: true, Erasable: true},
    "ip_address":     {Sensitivity: SensitivityPersonal, MaskFunc: "mask_ip", Searchable: false, Exportable: true, Portable: false, Erasable: true},
    "codice_fiscale": {Sensitivity: SensitivityRestricted, MaskFunc: "mask_cf", Searchable: false, Exportable: true, Portable: true, Erasable: true},
    "partita_iva":    {Sensitivity: SensitivityInternal, MaskFunc: "mask_partial", Searchable: true, Exportable: true, Portable: true, Erasable: false},  // Tax obligation
    "iban":           {Sensitivity: SensitivityRestricted, MaskFunc: "mask_iban", Searchable: false, Exportable: true, Portable: true, Erasable: true},
    "health_data":    {Sensitivity: SensitivitySensitive, MaskFunc: "mask_full", Searchable: false, Exportable: true, Portable: false, Erasable: true},
}
```

### Masking Functions

```go
package pii

import (
    "fmt"
    "strings"
)

// MaskEmail: "user@example.com" → "us***@example.com"
func MaskEmail(email string) string {
    parts := strings.SplitN(email, "@", 2)
    if len(parts) != 2 {
        return "***"
    }
    local := parts[0]
    if len(local) <= 2 {
        return "***@" + parts[1]
    }
    return local[:2] + "***@" + parts[1]
}

// MaskPhone: "+39 333 1234567" → "+39 333 ***4567"
func MaskPhone(phone string) string {
    if len(phone) < 4 {
        return "***"
    }
    return strings.Repeat("*", len(phone)-4) + phone[len(phone)-4:]
}

// MaskIP: "192.168.1.42" → "192.168.1.xxx"
func MaskIP(ip string) string {
    parts := strings.Split(ip, ".")
    if len(parts) == 4 {
        parts[3] = "xxx"
        return strings.Join(parts, ".")
    }
    return "xxx.xxx.xxx.xxx"
}

// MaskCodiceFiscale: "RSSMRA85T10A562S" → "RSS***********62S"
func MaskCodiceFiscale(cf string) string {
    if len(cf) < 6 {
        return "***"
    }
    return cf[:3] + strings.Repeat("*", len(cf)-6) + cf[len(cf)-3:]
}

// MaskIBAN: "IT60X0542811101000000123456" → "IT60************3456"
func MaskIBAN(iban string) string {
    if len(iban) < 8 {
        return "***"
    }
    return iban[:4] + strings.Repeat("*", len(iban)-8) + iban[len(iban)-4:]
}

// MaskFull: replace entirely with "[REDACTED]"
func MaskFull(_ string) string {
    return "[REDACTED]"
}

// MaskForLog applies the appropriate masking based on field name.
func MaskForLog(field string, value string) string {
    def, ok := PIIRegistry[field]
    if !ok || def.Sensitivity < SensitivityPersonal {
        return value // Non-PII, pass through
    }
    switch def.MaskFunc {
    case "mask_email":
        return MaskEmail(value)
    case "mask_phone":
        return MaskPhone(value)
    case "mask_ip":
        return MaskIP(value)
    case "mask_cf":
        return MaskCodiceFiscale(value)
    case "mask_iban":
        return MaskIBAN(value)
    case "mask_full":
        return MaskFull(value)
    default:
        return MaskFull(value)
    }
}
```

### Log Sanitization Middleware

```go
// SanitizeLogFields strips or masks PII before writing to logs.
// Apply this to any structured logger (zerolog, zap, slog).
func SanitizeLogFields(fields map[string]any) map[string]any {
    sanitized := make(map[string]any, len(fields))
    for k, v := range fields {
        if str, ok := v.(string); ok {
            sanitized[k] = MaskForLog(k, str)
        } else {
            sanitized[k] = v
        }
    }
    return sanitized
}
```

## Consent Management

### Consent Model

```go
package consent

import "time"

type Purpose string

const (
    PurposeMarketing   Purpose = "marketing_email"
    PurposeAnalytics   Purpose = "analytics"
    PurposeProfiling   Purpose = "profiling"
    PurposeNewsletter  Purpose = "newsletter"
    PurposeThirdParty  Purpose = "third_party_sharing"
)

type ConsentRecord struct {
    ID         string    `json:"id" bson:"_id"`
    SubjectID  string    `json:"subject_id" bson:"subject_id"`
    Purpose    Purpose   `json:"purpose" bson:"purpose"`
    Granted    bool      `json:"granted" bson:"granted"`
    GrantedAt  *time.Time `json:"granted_at,omitempty" bson:"granted_at,omitempty"`
    RevokedAt  *time.Time `json:"revoked_at,omitempty" bson:"revoked_at,omitempty"`
    Method     string    `json:"method" bson:"method"`           // "web_form", "api", "in_app"
    PolicyVer  string    `json:"policy_version" bson:"policy_version"` // Privacy policy version at time of consent
    ConsentText string  `json:"consent_text" bson:"consent_text"`   // Exact text shown to user
    IP         string    `json:"ip,omitempty" bson:"ip,omitempty"`
    UserAgent  string    `json:"user_agent,omitempty" bson:"user_agent,omitempty"`
}
```

### GDPR Consent Requirements

1. **Freely given** — no pre-ticked boxes, no bundled consent, no service conditional on consent for unrelated processing
2. **Specific** — separate consent per purpose (marketing ≠ analytics ≠ profiling)
3. **Informed** — clear language explaining what they're consenting to
4. **Unambiguous** — affirmative action (click, check, type)
5. **Withdrawable** — as easy to withdraw as to give (Art. 7(3))
6. **Evidenced** — you must prove consent was given (store the record)
7. **Children** — under 16 requires parental consent (Italy: 14, per D.Lgs. 101/2018 Art. 2-quinquies)

### Cookie Consent (Italian Garante Guidelines)

The Garante issued specific cookie guidelines (Provvedimento 10 giugno 2021, n. 231):

- **Technical cookies** — no consent needed, just inform in privacy notice
- **Analytics cookies** — no consent if anonymized and first-party (or third-party with data sharing contractually excluded)
- **Profiling/marketing cookies** — explicit prior consent required
- **Cookie banner requirements:**
  - First layer: brief info + accept/reject/customize buttons
  - Reject must be as prominent as accept (no dark patterns)
  - Scrolling ≠ consent
  - "Accept all" cannot be pre-selected
  - Must re-ask consent after 6 months maximum
  - Record consent choices

## Encryption

### Field-Level Encryption for PII

```go
package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"
)

type FieldEncryptor struct {
    key []byte // 32 bytes for AES-256
}

func NewFieldEncryptor(key []byte) (*FieldEncryptor, error) {
    if len(key) != 32 {
        return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d", len(key))
    }
    return &FieldEncryptor{key: key}, nil
}

func (fe *FieldEncryptor) Encrypt(plaintext string) (string, error) {
    block, err := aes.NewCipher(fe.key)
    if err != nil {
        return "", err
    }
    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonce := make([]byte, aesGCM.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return "", err
    }
    ciphertext := aesGCM.Seal(nonce, nonce, []byte(plaintext), nil)
    return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (fe *FieldEncryptor) Decrypt(encoded string) (string, error) {
    ciphertext, err := base64.StdEncoding.DecodeString(encoded)
    if err != nil {
        return "", err
    }
    block, err := aes.NewCipher(fe.key)
    if err != nil {
        return "", err
    }
    aesGCM, err := cipher.NewGCM(block)
    if err != nil {
        return "", err
    }
    nonceSize := aesGCM.NonceSize()
    if len(ciphertext) < nonceSize {
        return "", fmt.Errorf("ciphertext too short")
    }
    plaintext, err := aesGCM.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
    if err != nil {
        return "", err
    }
    return string(plaintext), nil
}
```

### When to Use Field-Level vs Full-Disk Encryption

| Approach | Use When | Trade-off |
|----------|----------|-----------|
| Full-disk (LUKS) | All data at rest — baseline protection | Protects against disk theft, not application-level access |
| Database encryption (MongoDB encrypted storage) | All DB data — baseline | Same as above but DB-specific |
| Field-level encryption | Specific PII fields need extra protection | Performance cost on encrypt/decrypt, no querying encrypted fields |
| Application-level envelope encryption | Key hierarchy needed, per-tenant keys | Most flexible, most complex |

**Recommendation:** Use full-disk + database encryption as baseline. Add field-level encryption for Restricted-sensitivity fields (codice fiscale, IBAN, health data). Keep encryption keys separate from data (different infrastructure, different access controls).

## Retention Automation

```go
package retention

import (
    "context"
    "fmt"
    "time"
)

type Policy struct {
    DataCategory string        `json:"data_category"`
    Retention    time.Duration `json:"retention"`
    LegalBasis   string        `json:"legal_basis"`
    Action       string        `json:"action"` // "delete" or "anonymize"
}

// Italian-aware retention policies
var Policies = []Policy{
    // Active user data — keep while account active
    {DataCategory: "user_profile", Retention: 0, LegalBasis: "contract", Action: "anonymize"},
    // Inactive users — 2 years after last activity
    {DataCategory: "inactive_user", Retention: 2 * 365 * 24 * time.Hour, LegalBasis: "legitimate_interest", Action: "delete"},
    // Invoices — 10 years (Italian civil code + tax law)
    {DataCategory: "invoices", Retention: 10 * 365 * 24 * time.Hour, LegalBasis: "legal_obligation", Action: "archive"},
    // Access logs — 1 year
    {DataCategory: "access_logs", Retention: 365 * 24 * time.Hour, LegalBasis: "legitimate_interest", Action: "delete"},
    // Support tickets — 3 years after resolution
    {DataCategory: "support_tickets", Retention: 3 * 365 * 24 * time.Hour, LegalBasis: "legitimate_interest", Action: "anonymize"},
    // Marketing consent — until revoked (no auto-expiry, but re-confirm periodically)
    {DataCategory: "marketing_consent", Retention: 0, LegalBasis: "consent", Action: "delete"},
    // Analytics (aggregated, no PII) — indefinite
    {DataCategory: "analytics_aggregated", Retention: 0, LegalBasis: "legitimate_interest", Action: "none"},
    // Cookie consent records — 6 months (Garante guideline: re-consent every 6 months)
    {DataCategory: "cookie_consent", Retention: 6 * 30 * 24 * time.Hour, LegalBasis: "legal_obligation", Action: "delete"},
}

type RetentionReport struct {
    RunAt         time.Time           `json:"run_at"`
    Results       []CategoryResult    `json:"results"`
    Errors        []string            `json:"errors,omitempty"`
}

type CategoryResult struct {
    Category     string `json:"category"`
    Action       string `json:"action"`
    RecordsFound int    `json:"records_found"`
    RecordsActed int    `json:"records_acted"`
}

func EnforceRetention(ctx context.Context, store DataStore) (*RetentionReport, error) {
    report := &RetentionReport{RunAt: time.Now().UTC()}

    for _, policy := range Policies {
        if policy.Retention == 0 || policy.Action == "none" {
            continue // Managed by other triggers (account deletion, consent revocation)
        }

        cutoff := time.Now().UTC().Add(-policy.Retention)
        result := CategoryResult{Category: policy.DataCategory, Action: policy.Action}

        records, err := store.FindExpired(ctx, policy.DataCategory, cutoff)
        if err != nil {
            report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", policy.DataCategory, err))
            continue
        }
        result.RecordsFound = len(records)

        for _, record := range records {
            switch policy.Action {
            case "delete":
                err = store.Delete(ctx, policy.DataCategory, record.ID)
            case "anonymize":
                err = store.Anonymize(ctx, policy.DataCategory, record.ID)
            case "archive":
                err = store.Archive(ctx, policy.DataCategory, record.ID)
            }
            if err != nil {
                report.Errors = append(report.Errors, fmt.Sprintf("%s/%s: %v", policy.DataCategory, record.ID, err))
                continue
            }
            result.RecordsActed++
        }

        report.Results = append(report.Results, result)
    }

    return report, nil
}
```

### Cron Schedule

```cron
# Retention enforcement — run daily during off-peak
0 3 * * * /usr/local/bin/retention-enforcer --config=/etc/retention/policies.json

# Inactive user notification (warn 30 days before deletion)
0 4 1 * * /usr/local/bin/inactive-user-notifier --days-before-deletion=30

# Cookie consent expiry check
0 5 * * * /usr/local/bin/cookie-consent-check --max-age-months=6
```

## Pseudonymization

Pseudonymization (Art. 4(5)) replaces identifying data with tokens while keeping a separate mapping. Unlike anonymization, pseudonymous data is still personal data — but it reduces risk and can satisfy DPIA requirements.

```go
package pseudonymize

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
)

// Pseudonymize creates a deterministic token from an identifier.
// The key MUST be stored separately from the pseudonymized data.
func Pseudonymize(identifier string, key []byte) string {
    h := hmac.New(sha256.New, key)
    h.Write([]byte(identifier))
    return "PSE-" + hex.EncodeToString(h.Sum(nil))[:16]
}

// Use cases:
// - Analytics: replace user_id with pseudonym, analyze without PII access
// - Logs: store pseudonymized actor ID, keep mapping in separate secured store
// - Testing: pseudonymize production data for dev/staging environments
// - Research: share datasets without exposing identities
```

**Key separation is critical.** If the mapping table is stored alongside the pseudonymized data, it offers no meaningful protection. Store the key/mapping in a different system with stricter access controls.

## Privacy by Design Patterns

### Data Minimization in APIs

```go
// BAD: Return all user fields
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, _ := h.store.FindUser(r.Context(), userID)
    json.NewEncoder(w).Encode(user) // Includes SSN, DOB, internal fields
}

// GOOD: Return only fields needed for the API consumer
type UserPublicView struct {
    ID    string `json:"id"`
    Name  string `json:"name"`
    Email string `json:"email"`
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, _ := h.store.FindUser(r.Context(), userID)
    json.NewEncoder(w).Encode(UserPublicView{
        ID:    user.ID,
        Name:  user.Name,
        Email: user.Email,
    })
}
```

### Access Logging for PII

Every access to personal data should be logged (who accessed what, when, why). This is both a GDPR accountability requirement and essential for DSR fulfillment.

```go
func (h *Handler) GetUserProfile(w http.ResponseWriter, r *http.Request) {
    actor := actorFromContext(r.Context())
    targetID := chi.URLParam(r, "userID")

    // Log the access BEFORE returning data
    h.auditLogger.Log(audit.Entry{
        Action:   audit.ActionDataRead,
        Actor:    actor,
        Resource: audit.Resource{Type: "user_profile", ID: targetID},
        Details:  map[string]any{"fields": "name,email,phone"},
    })

    user, err := h.store.FindUser(r.Context(), targetID)
    // ... return response
}
```

## Data Flow Mapping

Document all personal data flows — this is required for RoPA and essential for DSRs and breach response.

For each flow, document:
1. **Source** — where data comes from (web form, API, third party)
2. **Processing** — what happens to it (storage, analysis, enrichment)
3. **Storage** — where it lives (database, object storage, cache)
4. **Sharing** — who it's shared with (processors, partners)
5. **Deletion** — when and how it's removed
6. **Transfers** — does it leave the EU/EEA?

Maintain this as a living document — update when adding new features, integrations, or data categories.
