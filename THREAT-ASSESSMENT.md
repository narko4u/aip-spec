# Security Assessment

Status: assessment performed for the v0.1 release (2026-08-18). This
document records the most likely and impactful potential security problems
for this project and the mitigations in place. It is reviewed before each
release.

## What this project is

The AIP (Agent Interaction Protocol) open specification plus a Go
reference implementation. The protocol defines how agents discover, agree,
and settle actions with autonomous companies; the reference CLI parses
manifests and performs local signing/verification operations.

## Assets

1. **Specification integrity** — the normative text and schemas must not
   silently change (trust anchor for implementations).
2. **Verification correctness** — the reference implementation must not be
   tricked into accepting an invalid manifest or signature.
3. **No foothold from use** — parsing an untrusted manifest must not
   compromise the host.

## Likely and impactful problems

| # | Problem | Likelihood | Impact | Mitigation |
|---|---------|------------|--------|------------|
| 1 | Malicious manifest crafted to exploit parser bugs | Medium | Medium (host compromise) | Stdlib-only parsing with bounded inputs; CLI is local and user-invoked; no network listeners |
| 2 | Signature confusion (wrong key accepted, non-canonical encoding) | Medium | High (breaks verification promise) | Ed25519 verification against declared keys; canonical serialization in hashing; schema validation before verification |
| 3 | Specification ambiguity leading to divergent implementations | Medium | High (ecosystem trust) | Machine-readable schemas; conformance tests in the reference implementation |
| 4 | Typosquatting / impersonation in agent discovery | Medium | Medium | Deployment and registry verification handled by the WitnessOS evidence layer |
| 5 | Dependency supply-chain risk | Low | Low | Zero external runtime dependencies (stdlib only); CI installs from pinned source |

## Threat model scope

- **In scope:** protocol semantics, schema validation, reference CLI input
  handling.
- **Explicitly out of scope:** transport security of agent endpoints
  (implementer's responsibility), identity issuance and delegated
  authorization (AAIF Identity & Trust WG domain).

## Attack surface analysis

- `cmd/`, `internal/`, `pkg/` — manifest parsing, signing, verification.
- `schemas/` — JSON Schema structural validation.
- Protocol normative text — interpretation by implementers.
