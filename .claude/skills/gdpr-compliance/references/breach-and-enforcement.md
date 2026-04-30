# Breach Notification & Enforcement Reference

## The 72-Hour Rule (Art. 33)

The clock starts at **awareness** — the moment you have a reasonable degree of certainty that a breach has occurred. Not when you finish investigating, not when you've determined the impact. Awareness.

### Breach Definition

A "personal data breach" is any breach of security leading to the accidental or unlawful destruction, loss, alteration, unauthorized disclosure of, or access to, personal data. This includes:

- **Confidentiality breach** — unauthorized access or disclosure
- **Integrity breach** — unauthorized alteration
- **Availability breach** — accidental loss or destruction (even if temporary, if it impacts subjects)

### Notification Decision Tree

```
Breach detected
  ↓
Is personal data affected?
  NO → Security incident only (handle internally, no GDPR notification)
  YES ↓

Is there a risk to individuals' rights and freedoms?
  NO → Document internally (Art. 33(5)), no notification required
  YES ↓

Notify the supervisory authority within 72 hours (Art. 33)
  ↓
Is there HIGH risk to individuals?
  NO → Authority notification only
  YES → Also notify affected individuals "without undue delay" (Art. 34)
```

### Risk Assessment for Notification

| Factor | Low Risk | High Risk |
|--------|----------|-----------|
| Data type | Public info, non-sensitive | Financial, health, credentials, codice fiscale |
| Volume | Single record | Hundreds/thousands of records |
| Identifiability | Pseudonymized, encrypted | Plaintext, directly identifying |
| Consequences | Inconvenience | Financial loss, identity theft, discrimination |
| Containment | Immediate, data recovered | Uncontained, data in the wild |
| Recipients | Trusted internal | Unknown external, dark web |

## Breach Response Workflow

### Timeline

```
HOUR 0     Detection / Awareness
           ├── Activate incident response team
           ├── Begin containment
           └── Start breach log (timestamp everything)

HOUR 0-24  Investigation & Assessment
           ├── Determine scope (what data, how many subjects)
           ├── Determine cause (vulnerability, human error, attack)
           ├── Assess risk to individuals
           ├── Begin containment measures
           └── Decide: notify authority? notify subjects?

HOUR 24-48 Preparation
           ├── Draft authority notification
           ├── Draft subject notification (if high risk)
           ├── Legal review
           └── Management approval

HOUR 72    DEADLINE: Authority Notification
           ├── Submit to Garante via telematic system
           │   (https://servizi.gpdp.it/databreach/s/)
           ├── If investigation incomplete: submit what you have,
           │   note "information provided in phases" (Art. 33(4))
           └── You CAN supplement later — but initial notification
               must be within 72 hours

POST-72h   Subject Notification (if high risk)
           ├── Direct communication to affected individuals
           ├── Clear, plain language (Italian for Italian subjects)
           └── Include: what happened, consequences, measures taken,
               contact point, recommendations
```

### Authority Notification Content (Art. 33(3))

The notification to the Garante must include:

1. **Nature of the breach** — categories and approximate number of subjects and records affected
2. **DPO or contact point** — name and contact details
3. **Likely consequences** — description of potential impact
4. **Measures taken** — actions to address the breach and mitigate effects

### Garante Notification (Italy)

Submit via the Garante's online portal: `https://servizi.gpdp.it/databreach/s/`

The form requires:
- Controller identity and contact
- DPO contact
- Breach description (what, when, how)
- Data categories and volume
- Subject categories
- Cross-border impact (if any)
- Measures taken
- Risk assessment
- Whether subjects have been notified

**Keep a copy** of the submission confirmation. This is your evidence of timely notification.

### Subject Notification Content (Art. 34)

Required when there is **high risk** to individuals. Must be in **clear and plain language**:

```markdown
## Data Breach Notification

Dear [Name/Subject],

We are writing to inform you of a personal data breach that may affect
your information.

### What Happened
[Clear description of the breach]

### What Data Was Affected
[Specific categories: name, email, financial data, etc.]

### What We Have Done
[Containment and remediation measures taken]

### What You Can Do
[Specific recommendations: change passwords, monitor accounts, etc.]

### Contact
For questions or concerns, contact our Data Protection Officer:
[DPO name, email, phone]

You also have the right to lodge a complaint with the
Garante per la Protezione dei Dati Personali:
https://www.gpdp.it/
```

**Exception to direct notification (Art. 34(3)):** You don't need to notify subjects individually if:
- You applied encryption/pseudonymization that renders data unintelligible to unauthorized persons
- You took measures that eliminate the high risk
- It would involve disproportionate effort (use public communication instead)

## Breach Register (Art. 33(5))

**Mandatory regardless of whether you notify the authority.** Document ALL breaches, including those assessed as no-risk.

```go
package breach

import "time"

type BreachRecord struct {
    ID                string     `json:"id" bson:"_id"`
    DetectedAt        time.Time  `json:"detected_at" bson:"detected_at"`
    Nature            string     `json:"nature" bson:"nature"`
    DataCategories    []string   `json:"data_categories" bson:"data_categories"`
    SubjectsAffected  int        `json:"subjects_affected" bson:"subjects_affected"`
    RecordsAffected   int        `json:"records_affected" bson:"records_affected"`
    Consequences      string     `json:"consequences" bson:"consequences"`
    MeasuresTaken     []string   `json:"measures_taken" bson:"measures_taken"`
    RiskAssessment    string     `json:"risk_assessment" bson:"risk_assessment"` // "no_risk", "risk", "high_risk"
    AuthorityNotified bool       `json:"authority_notified" bson:"authority_notified"`
    NotifiedAt        *time.Time `json:"notified_at,omitempty" bson:"notified_at,omitempty"`
    SubjectsNotified  bool       `json:"subjects_notified" bson:"subjects_notified"`
    RootCause         string     `json:"root_cause" bson:"root_cause"`
    Remediation       []string   `json:"remediation" bson:"remediation"`
    ClosedAt          *time.Time `json:"closed_at,omitempty" bson:"closed_at,omitempty"`
    PostMortemURL     string     `json:"post_mortem_url,omitempty" bson:"post_mortem_url,omitempty"`
}
```

## Enforcement & Fines

### Fine Tiers (Art. 83)

| Tier | Maximum Fine | Violations |
|------|-------------|-----------|
| Lower | €10M or 2% global turnover (whichever higher) | Controller/processor obligations, certification body obligations, monitoring body obligations |
| Upper | €20M or 4% global turnover (whichever higher) | Processing principles, lawful basis, consent, data subject rights, transfers |

### Factors Affecting Fine Amount (Art. 83(2))

- Nature, gravity, and duration of the infringement
- Intentional or negligent character
- Mitigation measures taken
- Previous infringements
- Categories of personal data affected
- How the authority learned about it (self-reported vs complaint)
- Cooperation with the authority
- Certifications held
- Aggravating/mitigating factors

### Common Garante Enforcement Actions

| Violation | Typical Action | How to Prevent |
|-----------|---------------|---------------|
| No/inadequate cookie consent | Fine + corrective order | Implement Garante-compliant cookie banner |
| Marketing without consent | Fine + processing ban | Consent management system, opt-in only |
| Inadequate security measures | Fine + corrective order | Encryption, access controls, audit logging |
| Failure to respond to DSR | Fine | DSR workflow with deadline tracking |
| Missing or inadequate privacy notice | Corrective order | Complete Art. 13/14 notice |
| No DPA with processors | Fine | Execute DPAs before processing begins |
| Breach notification failure | Fine | Breach response plan with 72-hour timeline |
| Unlawful employee monitoring | Fine + processing ban | DPIA, proportionality assessment, union consultation |
| Video surveillance violations | Fine + corrective order | DPIA, signage, retention limits, proportionality |

### Practical Enforcement Tips

1. **Self-report breaches.** The Garante considers cooperation. Hiding a breach that's later discovered is much worse.
2. **Respond to DSRs on time.** Every DSR refusal or delay is a potential complaint. Even if you need to refuse, respond within 30 days with reasoning.
3. **Cookie compliance is actively enforced in Italy.** The Garante's 2021 cookie guidelines are specific — implement them precisely.
4. **Employee monitoring** is heavily regulated. Before deploying any workplace monitoring (email, location, productivity), consult Art. 4 of the Statuto dei Lavoratori (L. 300/1970) and the Garante's specific provvedimenti. Union agreement or labor inspectorate authorization is typically required.
5. **Video surveillance** requires signage, DPIA, retention limits (typically 24-48 hours per Garante guidelines, max 7 days with justification), and proportionality.
6. **Keep your RoPA current.** The Garante can request it at any time during an inspection.

## Breach Simulation / Tabletop Exercise

Run at least annually. Scenario example:

```markdown
## Tabletop Exercise: Ransomware + Data Exfiltration

### Scenario
At 14:00 on a Wednesday, your monitoring detects unusual outbound traffic
from the production MongoDB server. Investigation reveals:
- An attacker exploited an unpatched dependency to gain shell access
- The user_profiles collection was exported (50,000 records)
- Data includes: name, email, codice_fiscale, phone, address
- The attacker deployed ransomware on the database server
- Backups exist but are 6 hours old

### Discussion Points
1. Who is the Incident Commander?
2. What's the first containment action?
3. When does the 72-hour clock start?
4. What risk level do you assess? (Consider: codice_fiscale = high risk)
5. Do you notify the Garante? (Yes — high risk)
6. Do you notify subjects? (Yes — codice_fiscale exposure = identity theft risk)
7. What do you tell subjects to do?
8. How do you restore service?
9. What's the root cause fix?
10. What systemic changes prevent this class of incident?

### Expected Outputs
- Completed breach register entry
- Draft Garante notification
- Draft subject notification
- Action items for remediation
- Timeline documented
```
