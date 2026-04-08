# Trust Service Criteria — Detailed Reference

## Control-to-TSC Mapping Matrix

Use this matrix as the backbone of your compliance program. Every control must map to at least one criterion. Auditors will test a sample of controls per criterion.

### Security (Common Criteria — CC)

Security is **mandatory** for every SOC2 engagement. The other four categories are optional.

#### CC1 — Control Environment (COSO-Based)

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC1.1 | Commitment to integrity and ethics | Code of conduct signed by all employees | Signed acknowledgments, HR records |
| CC1.2 | Board/management oversight | Security governance meetings | Meeting minutes, attendee lists |
| CC1.3 | Authority and responsibility | Org chart, RACI for security | Documented roles, job descriptions |
| CC1.4 | Competence commitment | Hiring, training, performance | Background checks, training records |
| CC1.5 | Accountability | Performance reviews include security | Review templates, completed reviews |

#### CC2 — Communication and Information

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC2.1 | Internal communication | Security policies accessible to all | Policy repo access logs, acknowledgment records |
| CC2.2 | External communication | Security commitments to customers | ToS, DPA, security page, customer notifications |
| CC2.3 | Communication with the board | Risk reporting to leadership | Board decks, risk register updates |

#### CC3 — Risk Assessment

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC3.1 | Risk objectives defined | Business objectives linked to risk | Risk assessment document |
| CC3.2 | Risk identification | Threats cataloged systematically | Risk register with likelihood/impact |
| CC3.3 | Fraud risk | Fraud scenarios considered | Fraud risk assessment |
| CC3.4 | Change-driven risk | Significant changes trigger reassessment | Change risk analysis records |

#### CC4 — Monitoring

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC4.1 | Ongoing monitoring | Controls tested regularly | Internal audit reports, control testing results |
| CC4.2 | Deficiency remediation | Findings tracked to resolution | Issue tracker, remediation evidence |

#### CC5 — Control Activities

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC5.1 | Controls selected and developed | Controls map to identified risks | Control matrix |
| CC5.2 | Technology general controls | IT infrastructure controls operating | Config baselines, hardening evidence |
| CC5.3 | Policy deployment | Controls implemented via policy | Policy distribution, system configs |

#### CC6 — Logical and Physical Access

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC6.1 | Logical access controls | RBAC, least privilege enforced | IAM configs, role definitions |
| CC6.2 | Authentication | MFA, password policy, session mgmt | IdP config exports, MFA enrollment rates |
| CC6.3 | Access provisioning/deprovisioning | Joiner/mover/leaver process | Onboarding/offboarding tickets, timing evidence |
| CC6.4 | Access reviews | Periodic entitlement reviews | Review artifacts with approvals |
| CC6.5 | Physical access | Data center/office physical controls | Badge logs, visitor logs, DC audit reports |
| CC6.6 | System boundaries | Firewalls, network segmentation | Network diagrams, firewall rules |
| CC6.7 | Data in transit | TLS/encryption for network comms | TLS config scans, certificate inventory |
| CC6.8 | Threat prevention | Endpoint protection, vulnerability mgmt | AV/EDR configs, vuln scan reports |

#### CC7 — System Operations

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC7.1 | Infrastructure monitoring | Detection of anomalies | Monitoring dashboards, alert configs |
| CC7.2 | Incident detection | Security events identified | SIEM rules, alert history |
| CC7.3 | Incident evaluation | Classification and triage | Incident tickets with severity |
| CC7.4 | Incident response | Containment and resolution | IR runbooks, post-mortems |
| CC7.5 | Recovery | Restoration procedures | DR test results, RTO/RPO evidence |

#### CC8 — Change Management

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC8.1 | Change authorization | Changes approved before deploy | PR reviews, change tickets, approval records |

#### CC9 — Risk Mitigation

| Control ID | Objective | What Auditors Test | Evidence |
|-----------|-----------|-------------------|----------|
| CC9.1 | Risk mitigation | Residual risk accepted or treated | Risk treatment plans |
| CC9.2 | Vendor risk management | Third-party risks assessed | Vendor assessments, SOC2 reports collected |

### Availability (A)

Only in scope if the org makes availability commitments (SLAs).

| Control ID | Objective | Evidence |
|-----------|-----------|----------|
| A1.1 | Capacity planning | Capacity forecasts, scaling configs |
| A1.2 | Environmental protections | DC redundancy, UPS, cooling |
| A1.3 | Recovery testing | DR runbooks, test results with RTO/RPO |

### Processing Integrity (PI)

Only in scope if the org processes transactions or data where accuracy matters.

| Control ID | Objective | Evidence |
|-----------|-----------|----------|
| PI1.1 | Processing completeness | Input/output reconciliation reports |
| PI1.2 | Processing accuracy | Validation checks, error rates |
| PI1.3 | Processing timeliness | SLA adherence metrics |
| PI1.4 | Output delivery | Delivery confirmation logs |
| PI1.5 | Stored data integrity | Checksums, corruption detection |

### Confidentiality (C)

In scope when the org handles confidential information beyond personal data.

| Control ID | Objective | Evidence |
|-----------|-----------|----------|
| C1.1 | Identification of confidential info | Data classification policy, labeled assets |
| C1.2 | Disposal of confidential info | Secure deletion procedures, certificates |

### Privacy (P)

In scope when the org collects/processes personal information. Aligns heavily with GDPR.

| Control ID | Objective | Evidence |
|-----------|-----------|----------|
| P1.1 | Privacy notice | Published privacy policy |
| P2.1 | Consent | Consent collection mechanism |
| P3.1 | Collection limitation | Data minimization practices |
| P4.1 | Use and retention | Retention schedule, deletion automation |
| P5.1 | Access by data subjects | DSR handling process, response times |
| P6.1 | Disclosure to third parties | DPAs, sub-processor list |
| P7.1 | Data quality | Accuracy verification processes |
| P8.1 | Complaint handling | Complaint log, response process |

## Scoping Guidance

**Minimize scope aggressively.** Every system in scope needs controls, evidence, and testing.

Questions to determine scope:
1. What systems store, process, or transmit customer data?
2. What infrastructure supports those systems?
3. What people have access to those systems?
4. What third parties touch that data?

Draw the boundary diagram — systems inside the line are in scope. Everything else is out. Document the justification for exclusions.

## Auditor Interaction Tips

- **Don't volunteer information** beyond what's asked.
- **Have evidence ready before the audit starts** — don't collect during the observation period.
- **Exceptions are OK if documented.** An exception with a compensating control and risk acceptance is better than a gap.
- **Auditors sample, not census.** They'll pick a sample of changes, access reviews, incidents. Make sure every instance is compliant, not just the ones you show them.
- **Respond to requests promptly.** Slow evidence delivery extends the audit timeline and increases cost.
