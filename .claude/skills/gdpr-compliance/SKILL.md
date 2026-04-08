---
name: gdpr-compliance
description: GDPR compliance implementation — data subject rights (DSR), lawful basis for processing, DPIA, privacy by design, breach notification (72-hour rule), consent management, cross-border transfers, PII handling, and Records of Processing Activities (RoPA). Use this skill whenever the user mentions GDPR, data protection, privacy compliance, DPA, data subject requests, right to erasure, right of access, data portability, consent management, privacy by design, DPIA, data breach notification, cross-border data transfers, SCCs, PII masking, data retention policies, or any EU/EEA personal data regulation. Also trigger for Italian data protection (D.Lgs. 196/2003 as amended by D.Lgs. 101/2018, Garante per la Protezione dei Dati Personali), NIS2 privacy intersections, or when the user is building systems that process EU personal data — even if "GDPR" isn't explicitly mentioned.
---

# GDPR Compliance Skill

This skill helps implement GDPR-compliant systems, handle data subject requests, prepare for supervisory authority interactions, and maintain ongoing privacy compliance.

## How to Use This Skill

1. **Identify what the user needs** — Are they building a new system (privacy by design), handling a DSR, responding to a breach, or preparing documentation?
2. **Read the relevant reference file** before generating output:
   - `references/data-subject-rights.md` — DSR workflows, identity verification, erasure/access/portability implementation, response deadlines
   - `references/legal-framework.md` — Lawful basis selection, DPIA, RoPA, DPO requirements, Italian specificities (Garante), cross-border transfers, SCCs
   - `references/technical-controls.md` — Privacy by design patterns (Go), PII detection/masking, encryption, consent management, retention automation, pseudonymization
   - `references/breach-and-enforcement.md` — 72-hour breach notification, risk assessment, Garante interaction, fines, common enforcement actions
3. **Always ground advice in the user's stack and jurisdiction.** Ask about their data flows, hosting location, and whether they're a controller or processor before prescribing solutions.

## Key Principles

- **Lawful basis FIRST.** Every processing activity needs a documented lawful basis before it starts. Consent is not always the right choice — contract and legitimate interest are often stronger.
- **Data minimization is a legal obligation, not a best practice.** Collect only what's necessary, retain only as long as needed, and provide access only to those who need it.
- **Privacy by design and by default (Art. 25).** Build privacy into system architecture, don't bolt it on. Default settings must be the most privacy-protective.
- **Accountability (Art. 5(2)).** You must be able to *demonstrate* compliance, not just claim it. Documentation, logs, and records are mandatory.
- **The 72-hour clock is real.** Breach notification to the supervisory authority starts from *awareness*, not from completing the investigation.

## Quick Reference

| Topic | Article(s) | Deadline | Reference File |
|-------|-----------|----------|----------------|
| Data subject requests | Art. 15–22 | 30 days (extendable to 90) | `data-subject-rights.md` |
| Breach notification to authority | Art. 33 | 72 hours from awareness | `breach-and-enforcement.md` |
| Breach notification to subjects | Art. 34 | "Without undue delay" | `breach-and-enforcement.md` |
| DPIA required | Art. 35 | Before processing begins | `legal-framework.md` |
| RoPA maintained | Art. 30 | Continuous | `legal-framework.md` |
| DPO appointment | Art. 37 | Before processing begins | `legal-framework.md` |
| Cross-border transfer safeguards | Art. 44–49 | Before transfer | `legal-framework.md` |
| Consent records | Art. 7 | Continuous | `technical-controls.md` |

## Common User Requests → Action

| Request | Reference File | Key Section |
|---------|---------------|-------------|
| "Implement right to erasure" | `data-subject-rights.md` | Erasure Implementation |
| "Handle a data subject access request" | `data-subject-rights.md` | Access Request |
| "We had a data breach" | `breach-and-enforcement.md` | Breach Response Workflow |
| "Build a consent management system" | `technical-controls.md` | Consent Management |
| "Do we need a DPIA?" | `legal-framework.md` | DPIA |
| "Set up data retention policies" | `technical-controls.md` | Retention Automation |
| "Transfer data to the US" | `legal-framework.md` | Cross-Border Transfers |
| "Mask PII in logs" | `technical-controls.md` | PII Detection & Masking |
| "Write a privacy policy" | `legal-framework.md` | Privacy Notice |
| "GDPR compliance checklist" | `legal-framework.md` | Compliance Checklist |
