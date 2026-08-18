# Security

## Reporting vulnerabilities

**Do not open a public issue.** Email [contact@empirelabs.com.au](mailto:contact@empirelabs.com.au) with:

- A description of the vulnerability
- Steps to reproduce
- Affected versions
- Any proposed fix (optional)

We aim to acknowledge reports within 48 hours and provide a timeline for resolution within 5 business days.

## Scope

Security issues in **this repository only**. For issues in other Empire Labs projects, report them in the affected project's own repository.

## Design principles

Empire Labs projects follow a security-first posture:

1. **Least privilege** - CI and tooling run with minimal permissions
2. **Offline-first where possible** - no unnecessary network calls
3. **Minimal dependencies** - fewer dependencies means a smaller attack surface
4. **Tamper-evident** - where applicable, outputs carry cryptographic evidence

## Supported versions

Security fixes are applied to the latest release. Older releases are patched on a best-effort basis for critical issues only.

## Disclosure policy

We follow coordinated disclosure:

1. Reporter contacts us privately (email above)
2. We acknowledge within 48 hours
3. We work on a fix and timeline
4. We publish a fix, then coordinate public disclosure with the reporter

## Threat model

See the repository documentation for the threat model and trust boundaries applicable to this project.

## End of life

A release is considered end of life once it has been superseded by a newer release. Superseded releases no longer receive security updates except critical-only best-effort patches.

## Secrets and credentials

- Secrets (API keys, tokens, passwords, private keys) must never be committed to this repository. CI and local development use environment-provided credentials only.
- Repository secrets are stored in GitHub encrypted secrets and are scoped to the workflows that need them.
- If a secret is exposed, rotate it immediately, remove it from repository history, and report the exposure to contact@empirelabs.com.au.

## Dependency and static-analysis remediation policy

- Software Composition Analysis (SCA): known-vulnerable dependencies are remediated before any release. Critical and high severity findings are remediated within 30 days; medium within 90 days.
- Static Application Security Testing (SAST): findings are triaged on the same severity thresholds (critical/high within 30 days, medium within 90 days). Findings that cannot be fixed are documented with a justification.
- No release is cut while critical or high severity SCA or SAST findings are unresolved.
