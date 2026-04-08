# Evidence Collection & Audit Readiness

## Evidence Matrix

This is the master list of evidence you need. Automate collection wherever possible — manual evidence gathering is the #1 cause of audit fatigue and missed deadlines.

### Continuous Evidence (Auto-Generated)

| Evidence | Source | Format | TSC Mapping | Automation |
|----------|--------|--------|-------------|------------|
| Audit logs | Application + infra | JSON logs | CC6, CC7 | Ship to centralized log store |
| Change records | Git + CI/CD | PR history, pipeline logs | CC8.1 | GitHub/GitLab audit API |
| Vulnerability scans | CI/CD pipeline | Scan reports (SARIF/JSON) | CC7.1, CC6.8 | Run per-deploy, archive results |
| Uptime metrics | Monitoring | Grafana/Datadog dashboards | A1.1 | Export monthly SLA reports |
| Deployment history | CI/CD | Pipeline execution logs | CC8.1 | Archive pipeline artifacts |
| Secret rotation | Secrets manager | Rotation event logs | CC6.2 | Automated rotation + logging |

### Periodic Evidence (Scheduled)

| Evidence | Frequency | Owner | Format | TSC Mapping |
|----------|-----------|-------|--------|-------------|
| Access reviews | Quarterly | System owners | Signed review spreadsheet/ticket | CC6.4 |
| Backup restore tests | Quarterly | DevOps/SRE | Test execution record with results | A1.3 |
| DR test | Annually | Engineering lead | DR test report (scenario, results, gaps) | A1.3, CC9.1 |
| Penetration test | Annually | External vendor | Pentest report (findings, remediation) | CC7.1 |
| Risk assessment | Annually | Security lead | Risk register (updated) | CC3.1–3.4 |
| Policy review | Annually | Policy owners | Updated policies with revision history | CC1, CC5 |
| Security training | Annually | HR/Security | Completion records, quiz scores | CC1.4, CC2.1 |
| Vendor reviews | Annually | Procurement/Security | Assessment records, SOC2 reports | CC9.2 |
| Business continuity test | Annually | Operations | BCP test report | CC9.1 |

### Event-Driven Evidence

| Evidence | Trigger | Owner | Format |
|----------|---------|-------|--------|
| Incident reports | Per incident | IC | Post-mortem document |
| Exception requests | Per exception | Requester + approver | Approval record with justification |
| Employee onboarding | Per hire | HR + IT | Provisioning ticket, training completion |
| Employee offboarding | Per departure | HR + IT | Deprovisioning ticket with timestamps |
| Vendor onboarding | Per new vendor | Procurement | Assessment, DPA, risk tier assignment |

## Evidence Collection Automation (Go)

```go
package evidence

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "time"
)

type EvidencePackage struct {
    Period      string          `json:"period"`
    GeneratedAt time.Time      `json:"generated_at"`
    Items       []EvidenceItem  `json:"items"`
}

type EvidenceItem struct {
    Category    string    `json:"category"`
    Name        string    `json:"name"`
    TSCMapping  []string  `json:"tsc_mapping"`
    CollectedAt time.Time `json:"collected_at"`
    Source      string    `json:"source"`
    Hash        string    `json:"hash"`       // SHA-256 of the evidence artifact
    FilePath    string    `json:"file_path"`   // Path to stored artifact
    Status      string    `json:"status"`      // "collected", "missing", "overdue"
}

type Collector struct {
    outputDir string
    sources   []EvidenceSource
}

type EvidenceSource interface {
    Name() string
    Collect(ctx context.Context, period string) ([]EvidenceItem, error)
}

// Example sources you'd implement:
// - GitHubAuditLogSource    → fetches PR/merge/review history
// - CICDPipelineSource      → fetches pipeline runs, gate results
// - IAMExportSource         → fetches user list, roles, MFA status
// - MonitoringExportSource  → fetches uptime reports, SLA metrics
// - VulnScanSource          → fetches latest scan results
// - BackupTestSource        → fetches restore test records

func (c *Collector) CollectAll(ctx context.Context, period string) (*EvidencePackage, error) {
    pkg := &EvidencePackage{
        Period:      period,
        GeneratedAt: time.Now().UTC(),
    }

    for _, src := range c.sources {
        items, err := src.Collect(ctx, period)
        if err != nil {
            // Log error but continue — missing evidence is a finding, not a crash
            pkg.Items = append(pkg.Items, EvidenceItem{
                Category: src.Name(),
                Status:   "error",
                Name:     fmt.Sprintf("Collection failed: %v", err),
            })
            continue
        }
        pkg.Items = append(pkg.Items, items...)
    }

    return pkg, c.save(pkg, period)
}

func (c *Collector) save(pkg *EvidencePackage, period string) error {
    path := fmt.Sprintf("%s/evidence-%s.json", c.outputDir, period)
    data, err := json.MarshalIndent(pkg, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}
```

### Cron Schedule

```cron
# Evidence collection automation
# Monthly evidence package
0 2 1 * *   /usr/local/bin/evidence-collector --period=$(date -d 'last month' +%Y-%m) --output=/evidence/

# Weekly access snapshot (for quarterly reviews)
0 3 * * 1   /usr/local/bin/iam-snapshot --output=/evidence/access/

# Daily audit log integrity check
0 4 * * *   /usr/local/bin/audit-chain-verify --alert-on-break
```

## Audit Preparation Checklist

### 8 Weeks Before Audit

- [ ] Confirm audit scope with auditor (systems, TSC categories, observation period dates)
- [ ] Assign internal audit liaison (single point of contact)
- [ ] Verify all policies are current (reviewed within 12 months)
- [ ] Run full evidence collection — identify any gaps
- [ ] Schedule key personnel availability during fieldwork
- [ ] Prepare system description document

### 4 Weeks Before

- [ ] Complete quarterly access review (must fall within observation period)
- [ ] Verify backup restore test completed within observation period
- [ ] Confirm all employees completed security training
- [ ] Review open incidents — ensure all post-mortems completed
- [ ] Prepare evidence binder / shared folder structure
- [ ] Test evidence access — can auditors reach everything they need?

### 2 Weeks Before

- [ ] Pre-populate Information Request List (IRL) if auditor provided one
- [ ] Organize evidence by TSC criterion
- [ ] Brief key personnel on auditor interaction (answer directly, don't volunteer)
- [ ] Review previous audit findings — confirm all remediated

### During Fieldwork

- [ ] Respond to evidence requests within 24 hours
- [ ] Track all requests and delivery in a spreadsheet
- [ ] Escalate blockers to audit liaison immediately
- [ ] Do NOT make changes to address gaps during fieldwork (auditors notice)
- [ ] Document any exceptions or compensating controls

### After Audit

- [ ] Review draft report for factual accuracy
- [ ] Develop remediation plan for any findings
- [ ] Track remediation to completion with deadlines
- [ ] Share final report with stakeholders (customers requesting it)
- [ ] Begin continuous improvement for next period

## System Description Document

The auditor needs a system description. Include:

1. **Company overview** — What you do, key services
2. **System boundaries** — What's in scope (infrastructure, applications, data)
3. **System components** — Infrastructure diagram, technology stack
4. **Data flows** — How data enters, moves through, and exits the system
5. **People** — Roles and responsibilities for in-scope systems
6. **Complementary user entity controls (CUECs)** — What customers must do (e.g., "Customer is responsible for managing their own user accounts")
7. **Subservice organizations** — Cloud providers, SaaS dependencies in scope
8. **Trust service criteria** — Which categories and why
9. **Principal service commitments** — What you promise (SLAs, security commitments)

## Common Findings & Remediation

| Finding | Severity | Root Cause | Fix | Evidence of Fix |
|---------|----------|-----------|-----|----------------|
| Access review not completed | Medium | No process/owner | Implement quarterly review with calendar reminders | Completed review with sign-offs |
| Terminated user access not revoked | High | Manual process, HR/IT gap | Automate deprovisioning via HR→IdP integration | Offboarding tickets with timestamps showing < 24h |
| No MFA on admin accounts | High | Not enforced in IdP | Enable MFA enforcement, no exceptions | IdP config showing MFA required, enrollment report |
| Missing change approval | Medium | PRs merged without review | Branch protection rules, require approval | Git settings screenshot, sample PRs with reviews |
| No encryption at rest | High | Default configs | Enable encryption on DB, storage, backups | Encryption configuration evidence |
| Audit logs deletable | High | App has delete permissions | Separate log store, remove delete permissions | IAM policy for log storage |
| No vulnerability scanning | Medium | Not in pipeline | Add SCA + SAST to CI/CD | Pipeline config, sample scan reports |
| Shared service accounts | Medium | Convenience over security | Individual accounts, deprecate shared | Account inventory showing no shared accounts |
| No incident response plan | High | Never written | Document IR plan, run tabletop | IR plan document, tabletop exercise record |
| Backup not tested | Medium | Never scheduled | Quarterly restore tests | Restore test records with success/fail |
| Missing security training | Low | No LMS/tracking | Implement annual training with tracking | Completion records |
| No vendor risk assessment | Medium | Ad-hoc procurement | Vendor assessment process with risk tiers | Vendor register, assessment records |

## SOC2 Readiness Roadmap

### Phase 1: Gap Assessment (2–4 weeks)

- [ ] Document current state of controls
- [ ] Map existing controls to TSC criteria
- [ ] Identify gaps (missing controls, missing evidence, missing policies)
- [ ] Prioritize remediation by risk (High findings first)
- [ ] Estimate remediation effort and assign owners
- [ ] Choose auditor and TSC scope

### Phase 2: Remediation (2–6 months)

- [ ] Write and approve required policies
- [ ] Implement technical controls (access, logging, encryption, scanning)
- [ ] Set up evidence collection automation
- [ ] Configure monitoring and alerting
- [ ] Complete security awareness training
- [ ] Perform risk assessment
- [ ] Assess critical vendors
- [ ] Test backup restore and DR procedures
- [ ] Conduct tabletop incident response exercise
- [ ] Run internal audit / readiness assessment

### Phase 3: Type I Audit (1–2 months)

- [ ] Point-in-time assessment of control design
- [ ] Auditor evaluates whether controls are suitably designed
- [ ] Address any findings from Type I
- [ ] Begin observation period for Type II

### Phase 4: Type II Observation (6–12 months)

- [ ] Controls operating continuously throughout the period
- [ ] Evidence collected continuously (automated where possible)
- [ ] Quarterly access reviews completed
- [ ] Incidents handled per IR plan, post-mortems completed
- [ ] Changes managed through approved process
- [ ] No gaps in evidence — fill any holes immediately

### Phase 5: Type II Audit (1–2 months)

- [ ] Auditor tests operating effectiveness over the observation period
- [ ] Provide evidence for sampled controls
- [ ] Respond to information requests promptly
- [ ] Review draft report
- [ ] Remediate any findings
- [ ] Receive final report

### Ongoing: Continuous Compliance

- [ ] Evidence collection runs automatically
- [ ] Quarterly reviews on schedule
- [ ] Annual training, risk assessment, DR test, pentest
- [ ] Policy review cycle maintained
- [ ] Findings tracked and remediated
- [ ] Prepare for next observation period
