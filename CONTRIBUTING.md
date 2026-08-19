# Contributing to AIP (Agent Interaction Protocol)

First off, thank you for considering contributing to AIP. We welcome contributions from the community.

## Code of Conduct

This project and everyone participating in it is governed by the [Code of Conduct](CODE_OF_CONDUCT.md). By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Issues

- **Bug reports** - open an issue with steps to reproduce, expected behaviour, and actual behaviour
- **Feature requests** - open an issue describing the use case and proposed behaviour
- **Security issues** - do NOT open a public issue; email **contact@empirelabs.com.au**

### Spec Contributions

The AIP specification is at **draft v0.1**. We're actively iterating:

1. Check the [open questions](README.md#open-questions-to-resolve-before-v02) in the README
2. Open an issue or PR discussing your proposed change before implementing
3. Spec changes should update the README and relevant schemas

### Code Contributions (Go Reference Implementation)

1. Fork the repo
2. Create a feature branch (`git checkout -b feat/your-feature`)
3. Make your changes
4. Run `go build ./...` to verify compilation
5. Add tests for new functionality
6. Run `go vet ./...` and `go test ./...` to verify tests pass
7. Commit with clear messages
8. Open a PR against `main`

> **When tests run.** CI runs `go build`, `go vet` and `go test` on every
> PR and every push to `main`. Major changes MUST add or update automated
> tests in the same PR — see [TESTING.md](TESTING.md) for the full policy.

### Code Style

- Go: standard `go fmt` formatting
- Python: PEP 8 for MCP server
- Spec docs: Markdown with clear section headings
- Schemas: JSON Schema draft-07

## Development Setup

```sh
# Clone the repo
git clone https://github.com/narko4u/aip-spec.git
cd aip-spec

# Build the Go binary
go build -o aip ./cmd/aip/

# Run the demo
./aip demo
```

## Licensing

By contributing, you agree that your contributions will be licensed under the same licenses as this project:
- **Specification (documentation, schemas)**: CC BY 4.0
- **Code (Go, Python)**: MIT
