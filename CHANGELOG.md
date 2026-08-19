# Changelog

All notable changes to aip-spec are documented here.
Releases are versioned with [Semantic Versioning](https://semver.org/).

## [v0.2.1] - 2026-08-19

### Added
- **OpenSSF Best Practices baseline-3 readiness**: goreleaser CycloneDX SBOMs
  and cosign keyless signing for release assets; testing policy
  ([TESTING.md](TESTING.md)); release verification documentation
  (integrity, authenticity, release-author identity, SBOM) in the README.
- **CI hardening**: syft installed before the goreleaser SBOM step so SBOMs
  are generated on every release.
- **Packaging**: Dockerfile + GHCR publishing workflow; `.dockerignore`.
- **Documentation**: installation section (`go install`), WitnessOS launch
  family cross-links, community feedback window announcement.

### Security
- SAST (CodeQL) and SCA (OSV-Scanner) checks enforced in CI.
- Least-privilege CI permissions (`contents: read` by default).
- SECURITY.md policy sections and MAINTAINERS.md added.

### Fixed
- AIP Dockerfile asset URL version format (no `v` prefix mismatch).

## [v0.2.0] - 2026-08-12

### Added
- **AIP v0.1 reference implementation** (Go): agent interaction protocol with
  action schema validation, contract negotiation state machine, execution
  dispatch, Ed25519-signed evidence receipts, and settlement recording.
- CLI tool (`cmd/aip`) with `demo`, schema validation, and contract
  inspection helpers.
- MCP server (`mcp-server/`) exposing AIP tooling to MCP clients.
- AJSON integration examples.

### Documentation
- README rewritten: core concepts (Action, Contract, Negotiation, Execution,
  Settlement), protocol layers, manifest types bound to ACI, canonical use
  case, design principles.
- Cross-links added to the Empire Stack (ACI → AIP → AJSON).

[v0.2.0]: https://github.com/narko4u/aip-spec/releases/tag/v0.2.0
[v0.2.1]: https://github.com/narko4u/aip-spec/releases/tag/v0.2.1
