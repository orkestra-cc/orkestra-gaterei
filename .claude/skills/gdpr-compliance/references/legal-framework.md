# Legal Framework Reference

## Lawful Basis for Processing (Art. 6)

Choosing the right lawful basis is a **one-time, upfront decision per processing activity**. You cannot retroactively switch basis. Document the choice and reasoning in your RoPA.

### Decision Tree

```
Is processing necessary to perform a contract with the subject?
  YES → Art. 6(1)(b) Contract
  NO ↓

Is processing required by EU or member state law?
  YES → Art. 6(1)(c) Legal obligation
  NO ↓

Is there a legitimate business interest that doesn't override
the subject's rights?
  YES → Art. 6(1)(f) Legitimate interest (do LIA)
  NO ↓

Can you ask for freely given, specific, informed consent?
  YES → Art. 6(1)(a) Consent
  NO ↓

Does it involve vital interests or public interest?
  MAYBE → Art. 6(1)(d) or (e) — rare for private sector
```

### Lawful Basis Comparison

| Basis | Strengths | Weaknesses | Best For |
|-------|-----------|-----------|----------|
| **Consent** (a) | Clear, subject-controlled | Withdrawable anytime, creates ongoing obligation | Marketing, optional analytics, cookies |
| **Contract** (b) | Strong, not withdrawable | Must be genuinely necessary, not just convenient | Service delivery, account management, payment |
| **Legal obligation** (c) | Mandatory, no opt-out | Must identify specific law | Tax records, AML/KYC, employment law |
| **Legitimate interest** (f) | Flexible, doesn't require consent | Requires LIA, subject can object | Security, fraud prevention, B2B marketing, analytics |

### Legitimate Interest Assessment (LIA)

Required whenever you rely on Art. 6(1)(f). Document:

```markdown
## Legitimate Interest Assessment

**Processing activity:** [Description]
**Date:** [YYYY-MM-DD]

### Step 1: Purpose Test
What is the legitimate interest?
- [Describe the business need]
Is it a genuine interest (not just convenient)?
- [Yes/No + reasoning]
Is it lawful?
- [Yes/No]

### Step 2: Necessity Test
Is this processing necessary for the stated purpose?
- [Yes/No + reasoning]
Could the same purpose be achieved with less data or in a less intrusive way?
- [Assessment]

### Step 3: Balancing Test
What is the impact on individuals?
- Nature of data: [ordinary / sensitive]
- Reasonable expectations: [Would subjects expect this?]
- Power imbalance: [Employer/employee, etc.]
- Vulnerability: [Children, patients, etc.]
Does the individual's interest override?
- [Assessment]

### Safeguards
- [Measures to reduce impact: pseudonymization, access controls, transparency]

### Decision
Legitimate interest [is / is not] the appropriate basis.

### Review Date
[Annual review date]
```

### Special Category Data (Art. 9)

Processing biometric data, health data, racial/ethnic origin, political opinions, religious beliefs, trade union membership, genetic data, sex life/orientation requires **both** an Art. 6 basis AND an Art. 9(2) exception:

| Exception | Use Case |
|-----------|----------|
| Explicit consent | Health apps, diversity monitoring |
| Employment/social security law | Italian D.Lgs. 81/2008 workplace safety |
| Vital interests (subject incapacitated) | Emergency medical |
| Substantial public interest | Italian law specific provisions |
| Healthcare | Under medical professional responsibility |

## Records of Processing Activities (RoPA) — Art. 30

**Mandatory** for organizations with 250+ employees OR processing that is not occasional OR involves special categories or criminal data. In practice: maintain a RoPA regardless of size.

### RoPA Template (Controller — Art. 30(1))

| Field | Description | Example |
|-------|-------------|---------|
| Processing activity | Name and description | "Customer account management" |
| Controller identity | Name, contact, DPO | Octolabs S.r.l., info@octo-labs.com |
| Purposes | Why data is processed | Service delivery, invoicing |
| Data subjects | Categories of people | Customers, prospects |
| Data categories | Types of personal data | Name, email, company, VAT number |
| Recipients | Who receives the data | Payment processor, hosting provider |
| Transfers | Non-EEA transfers | US (SCCs), none |
| Retention | How long data is kept | Account lifetime + 10 years (tax) |
| Security measures | Technical/organizational | Encryption, RBAC, audit logging |
| Lawful basis | Art. 6 basis | Contract (b) |

### RoPA Template (Processor — Art. 30(2))

| Field | Description |
|-------|-------------|
| Processor identity | Name, contact |
| Controller(s) | Each controller you process for |
| Processing categories | What you do with the data |
| Transfers | Non-EEA transfers |
| Security measures | Technical/organizational |

**Keep the RoPA as a living document** — update whenever you add a new processing activity, change a processor, or modify retention. Store in version control for audit trail.

## Data Protection Impact Assessment (DPIA) — Art. 35

### When Required

A DPIA is **mandatory** before processing that is "likely to result in a high risk":

1. Systematic and extensive profiling with significant effects
2. Large-scale processing of special category data (Art. 9) or criminal data (Art. 10)
3. Systematic monitoring of publicly accessible areas at large scale
4. Any processing on the **Garante's mandatory DPIA list** (Provvedimento 11 ottobre 2018, n. 467)

**Italian-specific DPIA triggers (Garante list):**
- Evaluation/scoring including profiling
- Automated decisions with legal/significant effects
- Systematic monitoring (video surveillance, location tracking)
- Processing sensitive/judicial data at large scale
- Cross-matching of datasets from different controllers
- Data concerning vulnerable subjects (employees, patients, children, elderly)
- Innovative use of technology (AI, biometrics, IoT)
- Processing that prevents subjects from exercising a right or using a service
- Large-scale data transfers outside the EU

### DPIA Template

```markdown
# Data Protection Impact Assessment

**Project:** [Name]
**Version:** [X.Y]
**Date:** [YYYY-MM-DD]
**Owner:** [Role]
**DPO consulted:** [Yes/No, date]

## 1. Processing Description

### What
- Data categories: [list all personal data]
- Special categories: [if any — Art. 9/10]
- Volume: [approximate number of subjects]
- Storage: [where, how long]

### Why
- Purpose: [specific, explicit, legitimate]
- Lawful basis: [Art. 6 + Art. 9 if applicable]
- Necessity: [why this processing is needed]

### How
- Collection method: [direct from subject, from third party, observed]
- Processing operations: [collection, storage, analysis, sharing, deletion]
- Technology: [systems, algorithms, AI models]
- Access: [who can access, roles]

### Who
- Controller: [identity]
- Processors: [list with DPA status]
- Recipients: [who receives data]
- Transfers: [non-EEA, mechanism]

## 2. Necessity and Proportionality (Art. 35(7)(b))

- [ ] Purpose is specific and legitimate
- [ ] Processing is necessary (no less intrusive alternative)
- [ ] Data minimization applied (only necessary data collected)
- [ ] Storage limitation defined and enforced
- [ ] Accuracy measures in place
- [ ] Data subjects informed (privacy notice)
- [ ] Subject rights can be exercised

## 3. Risk Assessment (Art. 35(7)(c))

### Risk to Individuals

| Risk | Likelihood | Severity | Risk Level | Source |
|------|-----------|----------|-----------|--------|
| Unauthorized access/disclosure | L/M/H | L/M/H | L/M/H | [threat] |
| Unauthorized modification | L/M/H | L/M/H | L/M/H | [threat] |
| Loss of data | L/M/H | L/M/H | L/M/H | [threat] |
| Excessive collection / purpose creep | L/M/H | L/M/H | L/M/H | [threat] |
| Re-identification of pseudonymous data | L/M/H | L/M/H | L/M/H | [threat] |
| Discrimination from automated decisions | L/M/H | L/M/H | L/M/H | [threat] |
| Physical, material, or moral damage | L/M/H | L/M/H | L/M/H | [threat] |

## 4. Mitigation Measures (Art. 35(7)(d))

| Risk | Measure | Residual Risk |
|------|---------|--------------|
| [From above] | [Technical/organizational measure] | [After mitigation] |

### Standard measures checklist:
- [ ] Encryption at rest (AES-256)
- [ ] Encryption in transit (TLS 1.3)
- [ ] Access controls (RBAC, least privilege)
- [ ] Pseudonymization applied where possible
- [ ] Audit logging of all access
- [ ] Data minimization in collection and storage
- [ ] Retention automation with deletion
- [ ] Staff training on data handling
- [ ] Incident response plan covering this processing
- [ ] Regular security testing

## 5. Consultation

- DPO opinion: [assessment and recommendations]
- Data subjects consulted: [if appropriate — Art. 35(9)]
- Supervisory authority prior consultation needed: [Yes/No — Art. 36]

## 6. Decision

- [ ] Processing can proceed with current safeguards
- [ ] Processing can proceed with additional measures: [list]
- [ ] Processing should not proceed — risk too high

**Approved by:** [Name, Role]
**Date:** [YYYY-MM-DD]
**Next review:** [YYYY-MM-DD]
```

## DPO Requirements (Art. 37–39)

### When a DPO is Mandatory

- Public authority or body (except courts)
- Core activities require regular and systematic monitoring of subjects at large scale
- Core activities involve large-scale processing of special categories (Art. 9) or criminal data (Art. 10)

**Italian specificity:** The Garante has provided additional guidance — when in doubt, appoint a DPO. Many Italian businesses in the tech/SaaS space appoint one voluntarily as a trust signal.

### DPO Responsibilities

- Inform and advise controller/processor and employees
- Monitor compliance with GDPR and policies
- Advise on DPIAs
- Cooperate with and be contact point for the Garante
- Must be independent — cannot be instructed on how to perform tasks
- Cannot be dismissed or penalized for performing duties

## Cross-Border Data Transfers (Art. 44–49)

### Transfer Mechanism Selection

```
Is the recipient country covered by an adequacy decision?
  YES → Transfer permitted (Art. 45)
  NO ↓

Can you use Standard Contractual Clauses (SCCs)?
  YES → Execute SCCs + Transfer Impact Assessment (Art. 46(2)(c))
  NO ↓

Are Binding Corporate Rules available (intra-group)?
  YES → Use BCRs (Art. 47)
  NO ↓

Does a derogation apply (Art. 49)?
  - Explicit consent (informed of risks)
  - Necessary for contract with subject
  - Important public interest
  - Legal claims
  YES → Document and proceed
  NO → Transfer not permitted
```

### Current Adequacy Decisions (verify current list)

EU-adequate countries include: UK, Japan, South Korea, Canada (commercial), Israel, Switzerland, New Zealand, Argentina, Uruguay, and others. **The US** has the EU-US Data Privacy Framework (DPF) — only for certified companies. Always verify the recipient is DPF-certified.

### Transfer Impact Assessment (TIA)

Required alongside SCCs since Schrems II. Assess:

1. **Circumstances of the transfer** — data types, frequency, purposes, onward transfers
2. **Laws of the recipient country** — government access powers, surveillance laws, judicial remedies
3. **Supplementary measures** — encryption with EU-retained keys, pseudonymization, contractual restrictions
4. **Overall assessment** — do the supplementary measures effectively supplement the SCCs?

If the laws of the recipient country undermine the SCC protections and supplementary measures cannot compensate → **do not transfer**.

### Key Italian / EU Infrastructure Considerations

- **EU hosting** — hosting in the EU avoids transfer issues entirely. For self-hosted infrastructure (OVHCloud EU, Hetzner EU), document the data residency guarantee.
- **US SaaS tools** — if using US-based SaaS (GitHub, Slack, etc.), verify DPF certification or execute SCCs. Document in the RoPA.
- **Sub-processors** — you're responsible for your processor's transfers too. Review their sub-processor lists.

## Privacy Notice (Art. 13/14)

Must include:
1. Controller identity and contact
2. DPO contact (if appointed)
3. Purposes and lawful basis for each
4. Legitimate interests pursued (if applicable)
5. Recipients or categories of recipients
6. Transfer information (countries, safeguards)
7. Retention periods (or criteria)
8. Subject rights (access, rectification, erasure, etc.)
9. Right to withdraw consent (if consent-based)
10. Right to lodge complaint with Garante
11. Whether data provision is statutory/contractual requirement
12. Automated decision-making info (Art. 22)
13. Source of data (if not from subject — Art. 14)

**Language:** Clear, plain language. Avoid legal jargon. For Italian users, provide in Italian.

## GDPR Compliance Checklist

### Governance
- [ ] Lawful basis documented for each processing activity
- [ ] RoPA maintained and current (Art. 30)
- [ ] DPO appointed (if required) and registered with Garante
- [ ] Privacy notice published and covers all Art. 13/14 requirements
- [ ] DPIA conducted for high-risk processing
- [ ] Data Processing Agreements (DPA) with all processors
- [ ] Sub-processor list maintained and reviewed
- [ ] Employee data protection training completed (annual)
- [ ] Breach response plan documented and tested
- [ ] Internal privacy policies approved and distributed

### Technical
- [ ] Encryption at rest for all personal data
- [ ] Encryption in transit (TLS 1.2+ minimum)
- [ ] Access controls with least privilege
- [ ] Audit logging for all personal data access
- [ ] DSR handling workflow operational
- [ ] Consent management system (if consent-based processing)
- [ ] Automated retention enforcement
- [ ] PII inventory and data flow map
- [ ] Pseudonymization applied where feasible
- [ ] Backup encryption with access controls

### Cross-Border
- [ ] Transfer mechanisms in place for all non-EEA transfers
- [ ] TIA completed for SCC-based transfers
- [ ] DPF certification verified for US recipients
- [ ] Data residency documented for all infrastructure

### Italian Specificities
- [ ] Registered with Garante (if DPO appointed)
- [ ] Cookie banner compliant with Garante guidelines (June 2021)
- [ ] Italian-language privacy notice for Italian users
- [ ] Codice Privacy (D.Lgs. 196/2003, as amended) alignment verified
- [ ] Provvedimenti specifici del Garante reviewed for your sector
