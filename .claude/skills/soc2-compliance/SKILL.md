---
name: soc2-compliance
description: SOC2 Type II compliance implementation — Trust Service Criteria, access controls, audit logging, change management, incident response, evidence collection, and audit readiness. Use this skill whenever the user mentions SOC2, SOC 2, trust service criteria, audit readiness, compliance controls, evidence collection for audits, or asks about implementing security/availability/confidentiality/privacy/processing-integrity controls. Also trigger for access review processes, audit logging architecture, change management policies, incident response plans, or preparing for a compliance audit — even if "SOC2" isn't explicitly mentioned.
---

# SOC2 Type II Compliance Skill

This skill helps implement SOC2 Type II controls, prepare for audits, and maintain continuous compliance. It covers all five Trust Service Criteria and provides production-ready patterns.

## How to Use This Skill

1. **Identify the user's phase** — Are they doing a gap assessment, implementing controls, preparing evidence, or responding to audit findings?
2. **Read the relevant reference file** before generating any output:
   - `references/trust-service-criteria.md` — TSC mapping, control objectives, and what auditors look for
   - `references/technical-controls.md` — Implementation patterns (Go, TypeScript), audit logging, RBAC, encryption
   - `references/policies-and-procedures.md` — Policy templates, change management, incident response, vendor management
   - `references/evidence-and-audit.md` — Evidence collection automation, audit prep checklists, common findings, remediation
3. **Always ground advice in the user's actual stack** — don't suggest tools they can't use. Ask about their IAM provider, CI/CD, cloud, monitoring, and logging stack before prescribing solutions.

## Key Principles

- **Controls must be operating, not just designed.** Type II tests operating effectiveness over 6–12 months. A policy nobody follows is a finding.
- **Evidence is everything.** If it's not logged, it didn't happen. Automate evidence collection from day one.
- **Scope matters.** Help the user define the system boundary tightly — fewer systems in scope means fewer controls to maintain.
- **Map controls to TSC criteria explicitly.** Auditors want to see which control satisfies which criterion. Maintain a control-to-TSC matrix.
- **Continuous compliance > audit cramming.** Build controls into CI/CD, IAM, and monitoring so evidence generates itself.

## Quick Reference: TSC Categories

| Category             | Code | Core Question                                      |
| -------------------- | ---- | -------------------------------------------------- |
| Security             | CC   | Are systems protected against unauthorized access? |
| Availability         | A    | Do systems meet SLA commitments?                   |
| Processing Integrity | PI   | Is data processed completely, accurately, timely?  |
| Confidentiality      | C    | Is restricted information properly protected?      |
| Privacy              | P    | Is personal information handled per commitments?   |

## Common User Requests → Action

| Request                           | Reference File               | Key Section                |
| --------------------------------- | ---------------------------- | -------------------------- |
| "Help me implement audit logging" | `technical-controls.md`      | Audit Logging Architecture |
| "Write an incident response plan" | `policies-and-procedures.md` | Incident Response          |
| "What evidence do I need?"        | `evidence-and-audit.md`      | Evidence Matrix            |
| "Do a gap assessment"             | `evidence-and-audit.md`      | Readiness Checklist        |
| "Set up access controls"          | `technical-controls.md`      | Access Control             |
| "Prepare for our SOC2 audit"      | `evidence-and-audit.md`      | Audit Preparation          |
| "Write security policies"         | `policies-and-procedures.md` | Policy Templates           |
| "Change management process"       | `policies-and-procedures.md` | Change Management          |
