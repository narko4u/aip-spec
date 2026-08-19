# Design: aip-spec

This document describes the design of the Agent Interaction Protocol (AIP)
project: the actors, the actions they perform, and the data flow through the
protocol and its Go reference implementation. It accompanies
[THREAT-ASSESSMENT.md](THREAT-ASSESSMENT.md) (threat model) and
[TESTING.md](TESTING.md) (test policy).

## Purpose

AIP is a protocol for autonomous agent interaction — negotiation, execution,
and settlement of contracts between agents and organizations. It builds on
ACI (discovery) and defines the interaction layer on top.

## Actors

| Actor | Description |
| --- | --- |
| **Agent** | An autonomous software agent (any language) that initiates interaction via the AIP protocol. |
| **Organization / counterparty** | The party on the other side of the contract, reachable through a transport. |
| **Operator** | A user running the `aip` binary (Go reference implementation) or integrating it as a subprocess/HTTP service. |
| **Contract negotiator** | The protocol state machine that moves a contract through negotiation → execution → settlement. |
| **Evidence generator** | Emits Ed25519-signed evidence receipts for executed actions. |

## Actions

| Action | Performed by | Implemented in |
| --- | --- | --- |
| Validate action schema | Agent / Operator | `cmd/aip/`, `pkg/` |
| Negotiate contract (state machine) | Agent / Operator | `pkg/` |
| Dispatch execution via transport | Agent / Operator | `pkg/` |
| Generate Ed25519-signed evidence receipt | Agent / Operator | `pkg/` |
| Record settlement transaction | Agent / Operator | `pkg/` |
| Serve AIP over MCP | Operator | `mcp-server/` |

## Data flow

```
Agent (any language)
  → subprocess/HTTP → aip binary (Go)
    → validates action schema (schemas/)
    → negotiates contract (state machine)
    → dispatches execution via transport
    → generates Ed25519-signed evidence receipt
    → records settlement transaction
```

## Design invariants

1. **Zero external dependencies.** The reference implementation uses only the
   Go standard library (`crypto/ed25519`, `net/http`, `encoding/json`) — a
   single ~8MB static binary with no supply-chain surface.
2. **Cross-platform by construction.** `GOOS`/`GOARCH` cross-compilation
   targets all major platforms; releases ship linux/darwin for amd64/arm64.
3. **Evidence-first.** Every executed action produces an Ed25519-signed
   evidence receipt, so interactions are auditable and verifiable.
4. **Protocol layers stay explicit.** AIP defines negotiation/execution/
   settlement above ACI (discovery); the manifest types remain bound to ACI
   and the layers are documented separately.
