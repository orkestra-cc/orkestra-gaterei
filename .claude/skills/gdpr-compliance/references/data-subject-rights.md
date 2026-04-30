# Data Subject Rights — Implementation Reference

## Rights Overview

All rights apply to identified or identifiable natural persons ("data subjects"). Response deadline is **30 calendar days** from receipt, extendable by **60 days** for complex/numerous requests (but you must inform the subject within the first 30 days of the extension and reasons).

| Right | Article | Conditions | Can Be Refused |
|-------|---------|-----------|----------------|
| Access | Art. 15 | Any data subject | Only if manifestly unfounded/excessive |
| Rectification | Art. 16 | Inaccurate data | No (if data is indeed inaccurate) |
| Erasure ("right to be forgotten") | Art. 17 | Various grounds | Yes — legal obligation, public interest, legal claims |
| Restriction of processing | Art. 18 | Accuracy contested, unlawful processing, etc. | Limited |
| Data portability | Art. 20 | Consent/contract basis + automated processing | Only applies to data *provided by* the subject |
| Object to processing | Art. 21 | Legitimate interest / public interest basis | Only with compelling legitimate grounds |
| Not be subject to automated decisions | Art. 22 | Decisions with legal/significant effects | Yes — if necessary for contract, authorized by law, or explicit consent |

## Identity Verification

**You MUST verify identity before fulfilling any DSR.** Fulfilling a request for the wrong person is itself a data breach.

```go
package dsr

import (
    "errors"
    "time"
)

type VerificationMethod string

const (
    VerifyEmail     VerificationMethod = "email_confirmation"
    VerifyLogin     VerificationMethod = "authenticated_session"
    VerifyDocument  VerificationMethod = "identity_document"
    VerifyKBA       VerificationMethod = "knowledge_based"
)

type VerificationResult struct {
    Verified   bool               `json:"verified"`
    Method     VerificationMethod `json:"method"`
    VerifiedAt time.Time          `json:"verified_at"`
    VerifiedBy string             `json:"verified_by"` // system or operator ID
}

// VerifyIdentity determines the appropriate verification method.
// Authenticated users → session is sufficient.
// Email requests → send confirmation link.
// Unverifiable → request identity document (proportionate to risk).
func VerifyIdentity(subjectID string, isAuthenticated bool, contactEmail string) (*VerificationResult, error) {
    if isAuthenticated {
        return &VerificationResult{
            Verified: true,
            Method:   VerifyLogin,
            VerifiedAt: time.Now().UTC(),
            VerifiedBy: "system",
        }, nil
    }

    if contactEmail != "" {
        // Send verification link, return pending
        return nil, ErrVerificationPending
    }

    // Require manual ID verification
    return nil, ErrIdentityDocumentRequired
}

var (
    ErrVerificationPending    = errors.New("email verification sent, awaiting confirmation")
    ErrIdentityDocumentRequired = errors.New("identity document required for verification")
)
```

**Do NOT** request excessive identification — the verification must be proportionate. If the user is logged in, that's sufficient. Don't ask for a passport scan from someone making a request through their own account.

## DSR Request Lifecycle

```go
package dsr

import (
    "context"
    "fmt"
    "time"
)

type RequestType string

const (
    TypeAccess      RequestType = "access"
    TypeRectify     RequestType = "rectification"
    TypeErase       RequestType = "erasure"
    TypeRestrict    RequestType = "restriction"
    TypePortability RequestType = "portability"
    TypeObjection   RequestType = "objection"
)

type RequestStatus string

const (
    StatusReceived   RequestStatus = "received"
    StatusVerifying  RequestStatus = "verifying"
    StatusProcessing RequestStatus = "processing"
    StatusCompleted  RequestStatus = "completed"
    StatusExtended   RequestStatus = "extended"     // 60-day extension
    StatusRefused    RequestStatus = "refused"
)

type Request struct {
    ID            string        `json:"id" bson:"_id"`
    Type          RequestType   `json:"type" bson:"type"`
    SubjectID     string        `json:"subject_id" bson:"subject_id"`
    SubjectEmail  string        `json:"subject_email" bson:"subject_email"`
    Status        RequestStatus `json:"status" bson:"status"`
    ReceivedAt    time.Time     `json:"received_at" bson:"received_at"`
    Deadline      time.Time     `json:"deadline" bson:"deadline"`          // +30 days
    ExtendedUntil *time.Time    `json:"extended_until,omitempty" bson:"extended_until,omitempty"` // +90 days max
    CompletedAt   *time.Time    `json:"completed_at,omitempty" bson:"completed_at,omitempty"`
    RefusalReason string        `json:"refusal_reason,omitempty" bson:"refusal_reason,omitempty"`
    AuditTrail    []AuditEntry  `json:"audit_trail" bson:"audit_trail"`
}

type AuditEntry struct {
    Timestamp time.Time `json:"timestamp" bson:"timestamp"`
    Action    string    `json:"action" bson:"action"`
    Actor     string    `json:"actor" bson:"actor"`
    Details   string    `json:"details,omitempty" bson:"details,omitempty"`
}

func NewRequest(reqType RequestType, subjectID, subjectEmail string) *Request {
    now := time.Now().UTC()
    return &Request{
        ID:         generateID(),
        Type:       reqType,
        SubjectID:  subjectID,
        SubjectEmail: subjectEmail,
        Status:     StatusReceived,
        ReceivedAt: now,
        Deadline:   now.AddDate(0, 0, 30),
        AuditTrail: []AuditEntry{{
            Timestamp: now,
            Action:    "request_received",
            Actor:     "system",
        }},
    }
}

// CheckDeadline returns days remaining and alerts if approaching.
func (r *Request) CheckDeadline() (daysRemaining int, urgent bool) {
    deadline := r.Deadline
    if r.ExtendedUntil != nil {
        deadline = *r.ExtendedUntil
    }
    remaining := int(time.Until(deadline).Hours() / 24)
    return remaining, remaining <= 5
}
```

## Right of Access (Art. 15) — Implementation

The data subject has the right to obtain:
1. Confirmation of whether their data is being processed
2. A copy of all their personal data
3. Information about: purposes, categories, recipients, retention period, their rights, source of data, automated decision-making details

```go
// GenerateAccessReport compiles all personal data for a subject.
func GenerateAccessReport(ctx context.Context, subjectID string) (*AccessReport, error) {
    report := &AccessReport{
        GeneratedAt: time.Now().UTC(),
        SubjectID:   subjectID,
    }

    // Collect from all data stores
    // IMPORTANT: enumerate ALL systems that hold personal data
    sources := []DataSource{
        &UserProfileSource{},     // Main user record
        &OrderHistorySource{},    // Transactions
        &SupportTicketSource{},   // Support interactions
        &CommunicationSource{},   // Emails, notifications sent
        &ConsentRecordSource{},   // Consent history
        &AuditLogSource{},        // Activity logs (their own actions)
        &AnalyticsSource{},       // Behavioral data, if any
        &ThirdPartySource{},      // Data shared with/from processors
    }

    for _, src := range sources {
        data, err := src.CollectForSubject(ctx, subjectID)
        if err != nil {
            report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", src.Name(), err))
            continue
        }
        report.Categories = append(report.Categories, DataCategory{
            Source:     src.Name(),
            Data:       data,
            Purpose:    src.Purpose(),
            Recipients: src.Recipients(),
            Retention:  src.RetentionPeriod(),
        })
    }

    // Include processing information (Art. 15(1))
    report.ProcessingInfo = ProcessingInfo{
        Purposes:           listProcessingPurposes(),
        Categories:         listDataCategories(),
        Recipients:         listRecipients(),
        RetentionCriteria:  describeRetentionCriteria(),
        Rights:             describeSubjectRights(),
        Source:             describeDataSources(),
        AutomatedDecisions: describeAutomatedDecisions(),
        TransferSafeguards: describeTransferSafeguards(),
    }

    return report, nil
}

type AccessReport struct {
    GeneratedAt    time.Time        `json:"generated_at"`
    SubjectID      string           `json:"subject_id"`
    Categories     []DataCategory   `json:"categories"`
    ProcessingInfo ProcessingInfo   `json:"processing_info"`
    Errors         []string         `json:"errors,omitempty"`
}

type DataCategory struct {
    Source     string   `json:"source"`
    Data       any      `json:"data"`
    Purpose    string   `json:"purpose"`
    Recipients []string `json:"recipients"`
    Retention  string   `json:"retention"`
}
```

**Output format:** Provide data in a commonly used, machine-readable format (JSON preferred). For non-technical subjects, also offer a human-readable summary (PDF/HTML).

## Right to Erasure (Art. 17) — Implementation

Erasure is NOT unconditional. You CAN refuse if data is needed for:
- Exercising freedom of expression (Art. 17(3)(a))
- Legal obligation compliance — e.g., Italian tax records must be kept 10 years (Art. 17(3)(b))
- Public health (Art. 17(3)(c))
- Archiving in public interest (Art. 17(3)(d))
- Legal claims (Art. 17(3)(e))

```go
// ExecuteErasure performs cascading erasure/anonymization across all systems.
func ExecuteErasure(ctx context.Context, subjectID string) (*ErasureReport, error) {
    report := &ErasureReport{
        SubjectHash: hashSubjectID(subjectID),
        StartedAt:   time.Now().UTC(),
    }

    // Check for legal holds — REFUSE erasure if data is under legal hold
    if held, reason := checkLegalHold(ctx, subjectID); held {
        return nil, fmt.Errorf("erasure blocked: legal hold — %s", reason)
    }

    // Determine what CAN be erased vs what must be retained
    plan := buildErasurePlan(ctx, subjectID)

    err := withTransaction(ctx, func(tx Transaction) error {
        // 1. Anonymize user profile (keep pseudonymous record for referential integrity)
        if err := tx.AnonymizeUser(subjectID); err != nil {
            return fmt.Errorf("anonymize user: %w", err)
        }
        report.addStep("users", "anonymized", "Profile fields nulled, email replaced with hash")

        // 2. Delete personal content
        for _, table := range plan.DeleteTables {
            count, err := tx.DeleteBySubject(table, subjectID)
            if err != nil {
                return fmt.Errorf("delete %s: %w", table, err)
            }
            report.addStep(table, "deleted", fmt.Sprintf("%d records", count))
        }

        // 3. Anonymize records that must be retained (e.g., invoices for tax)
        for _, table := range plan.AnonymizeTables {
            count, err := tx.AnonymizeBySubject(table, subjectID)
            if err != nil {
                return fmt.Errorf("anonymize %s: %w", table, err)
            }
            report.addStep(table, "anonymized", fmt.Sprintf("%d records — retained for %s", count, plan.RetentionReason[table]))
        }

        // 4. Anonymize audit logs (the log entry stays, PII goes)
        if err := tx.AnonymizeAuditLogs(subjectID); err != nil {
            return fmt.Errorf("anonymize audit logs: %w", err)
        }
        report.addStep("audit_logs", "anonymized", "Actor PII removed, events preserved")

        // 5. Revoke all sessions and tokens
        if err := tx.RevokeAllSessions(subjectID); err != nil {
            return fmt.Errorf("revoke sessions: %w", err)
        }
        report.addStep("sessions", "revoked", "All active sessions terminated")

        return nil
    })
    if err != nil {
        return report, err
    }

    // 6. Notify processors (Art. 17(2)) — you must inform all recipients
    processors := listProcessors(ctx, subjectID)
    for _, p := range processors {
        if err := notifyProcessorErasure(ctx, p, subjectID); err != nil {
            report.addStep("processor_notification", "failed", fmt.Sprintf("%s: %v", p.Name, err))
        } else {
            report.addStep("processor_notification", "sent", p.Name)
        }
    }

    // 7. Record erasure event (keep proof of compliance, no PII)
    report.CompletedAt = timePtr(time.Now().UTC())
    if err := storeErasureRecord(ctx, report); err != nil {
        return report, fmt.Errorf("store erasure record: %w", err)
    }

    return report, nil
}

type ErasureReport struct {
    SubjectHash string       `json:"subject_hash"`
    StartedAt   time.Time    `json:"started_at"`
    CompletedAt *time.Time   `json:"completed_at,omitempty"`
    Steps       []ErasureStep `json:"steps"`
}

type ErasureStep struct {
    System  string `json:"system"`
    Action  string `json:"action"`
    Details string `json:"details"`
}
```

### Anonymization vs Deletion

| Approach | When to Use | Example |
|----------|------------|---------|
| Hard delete | No legal retention requirement, no referential integrity needs | Chat messages, uploaded photos |
| Anonymization | Referential integrity needed or analytics value | User record → `name: "Deleted User"`, `email: hash@erased.local` |
| Pseudonymization | Data needed for ongoing processing but identity not required | Replace `user_id` with random token, keep mapping separate |
| Retained with justification | Legal obligation to keep | Invoices (10 years IT tax), contracts |

## Right to Portability (Art. 20)

Only applies to:
- Data *provided by* the subject (not derived/inferred data)
- Processing based on consent or contract
- Processing carried out by automated means

```go
// GeneratePortableExport creates a machine-readable export.
func GeneratePortableExport(ctx context.Context, subjectID string, format string) ([]byte, string, error) {
    // Collect only data PROVIDED BY the subject
    data := &PortableData{
        Profile:       getProfileData(ctx, subjectID),       // Name, email, etc.
        Content:       getUserContent(ctx, subjectID),        // Posts, uploads
        Transactions:  getTransactionHistory(ctx, subjectID), // Orders, payments
        Preferences:   getUserPreferences(ctx, subjectID),    // Settings
        Communications: getUserCommunications(ctx, subjectID), // Messages sent
    }
    // EXCLUDE: derived analytics, risk scores, internal notes

    switch format {
    case "json":
        b, err := json.MarshalIndent(data, "", "  ")
        return b, "application/json", err
    case "csv":
        b, err := marshalCSV(data)
        return b, "text/csv", err
    default:
        b, err := json.MarshalIndent(data, "", "  ")
        return b, "application/json", err
    }
}
```

## DSR Refusal

You can refuse a request if it is **manifestly unfounded or excessive** (Art. 12(5)). In that case:
- You must inform the subject of the refusal and reasons
- Inform them of their right to complain to the supervisory authority
- Inform them of their right to judicial remedy
- You must respond within the 30-day deadline even to refuse

Document refusal decisions thoroughly — the burden of proof that a request is unfounded/excessive is on you.

## DSR Tracking Dashboard

Track these metrics for compliance evidence:

| Metric | Target | Evidence For |
|--------|--------|-------------|
| Average response time | < 25 days | Art. 12(3) timeliness |
| Requests completed within 30 days | 100% | Art. 12(3) compliance |
| Identity verification rate | 100% | Security of processing |
| Processor notification rate | 100% for erasure | Art. 17(2) obligation |
| Refusal rate with documentation | N/A (track trend) | Art. 12(5) accountability |
