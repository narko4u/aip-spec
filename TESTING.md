# Testing Policy

This document defines **when** tests run and **what is required** of every
change that touches functional code in this repository. It is normative: the
CI pipeline enforces the automated portion, and maintainers enforce the
policy portion during review.

## When tests run

1. **Every pull request** — the `CI` workflow (see
   [`.github/workflows/ci.yml`](.github/workflows/ci.yml)) runs
   `go build ./...`, `go vet ./...` and `go test ./...` on every push to a
   PR branch. A PR that breaks CI cannot be merged.
2. **Every push to `main`** — the same `CI` workflow runs again.
3. **Locally, before opening a PR** — contributors are expected to run the
   checks described in
   [CONTRIBUTING.md](CONTRIBUTING.md#code-contributions-go-reference-implementation)
   before requesting review:

   ```sh
   go build ./...
   go vet ./...
   go test ./...
   ```

4. **Before every release** — the `Release` workflow (see
   [`.github/workflows/release.yml`](.github/workflows/release.yml)) runs
   only on a `v*.*.*` tag; GoReleaser builds the cross-platform binaries,
   generates the SBOM, and signs the artifacts. Tags are created from a
   `main` commit that already passed CI.

## What the test suite covers

Go tests live next to the packages they test (`*_test.go`). The suite
covers the reference implementation under `cmd/` and `internal/`,
including action-schema parsing, contract negotiation flows, and evidence
handling. `go test ./...` runs the full suite; `go vet ./...` runs static
analysis; `go build ./...` verifies every package compiles.

## Policy for major changes

> **Any significant change MUST add or update automated tests in the same
> PR.**

For this project, that means:

- A change to the **Go reference implementation** MUST add or update Go
  tests for the changed behaviour.
- A change to the **specification** MUST update the affected schemas and
  add or update an example that exercises the change (validated in CI).
- A change to the **MCP server** MUST add or update its tests.

Trivial changes (typos, formatting, documentation-only edits) are exempt,
at the maintainer's discretion — but a PR that touches functional code
without updating tests will be blocked in review.

## Enforcement

- CI failing = merge blocked (the `build` check must pass).
- Policy not followed = review comment, PR returned to author.
- Reviewers MUST verify the PR description states which tests were added
  or updated.
