# VEX — Vulnerability Exploitability eXchange

Status: current as of v0.1 (2026-08-18). Reviewed before each release.

This document states the exploitability status of known vulnerabilities in
the software components of this project, per the OSPS VM-04.02 control.
"Not affected" means the vulnerable component is present in the supply chain
but the vulnerable code path cannot be reached or does not affect the
shipped artifact.

## Component inventory

| Component | Type | Version | Runtime? |
|-----------|------|---------|----------|
| AIP specification (protocol, schemas) | Specification | v0.1 | Yes (normative) |
| `cmd/`, `internal/`, `pkg/` (Go reference implementation) | Shipped code | v0.1 | Yes |
| Go standard library | Runtime | 1.22+ | Yes |
| GitHub Actions (go build/vet) | CI-only | pinned by SHA | No |

## Statements

| Component | Vulnerability | Status | Justification |
|-----------|---------------|--------|---------------|
| AIP specification | (any) | Not affected | A text protocol specification plus JSON schemas; it does not execute. Security properties are expressed in the protocol security section |
| Go reference implementation | (any) | Not affected | CLI/schema tooling that performs local parsing and signing checks; it stores no credentials, opens no listeners, and writes nothing to the target |
| Test/build/CI components | (any) | Not affected | Not shipped to end users; only ever run in ephemeral CI on trusted inputs |

## Change policy

- This VEX is updated whenever a new component is added, a vulnerability is
  reported, or a release is prepared.
- New releases must not ship while a High or Medium severity finding in a
  reachable component is unresolved (see `SECURITY.md` remediation
  thresholds).
