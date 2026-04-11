# Guard Patterns

## 1. Source Contract -> Generated Consumer

Use when the same data shape crosses backend and frontend.

Pattern:
- define the contract once
- generate types, clients, or bindings from it
- fail CI when generated outputs are stale

Good fits:
- OpenAPI -> frontend client/types
- JSON schema -> validators or typed models
- protobuf -> backend/frontend RPC bindings

## 2. Central Matrix -> Table-Driven Enforcement

Use when permissions, routes, commands, or states must stay aligned.

Pattern:
- centralize the matrix in one package or data table
- consume it from helpers or adapters
- add table-driven tests that exercise all supported combinations

Good fits:
- scope -> route -> principal-kind mapping
- state transition rules
- command -> route mapping
- feature flag -> dependency wiring

## 3. Architecture Boundary Guard

Use when layering matters.

Pattern:
- encode allowed dependency direction
- fail builds on forbidden imports or forbidden package references
- keep exceptions explicit and rare

Good fits:
- domain cannot import transport layer
- frontend feature modules cannot reach internal infra directly
- generated code cannot be edited manually

## 4. Boundary Regression Test

Use when a bug was caused by two layers disagreeing.

Pattern:
- reproduce the real user path
- assert the exact seam where drift occurred
- keep the test close to the boundary, not just the leaf helper

Good fits:
- CLI subcommand -> request path
- auth scope -> route response
- serializer -> persisted payload
- frontend nullability -> rendered behavior

## 5. Structured Log Contract

Use when full-system replay is expensive.

Pattern:
- define stable log keys
- emit typed or helper-backed log events
- include correlation identifiers and external request identifiers
- test critical emitters where practical

Good fits:
- third-party provider calls
- machine agents and orchestrators
- flaky or rate-limited workflows
- long-running async jobs

## 6. Harness Freshness Check

Use when the repo contains derived artifacts or indexed docs.

Pattern:
- expose one command that regenerates the artifact
- compare working tree after generation in CI
- fail if regeneration changes tracked files

Good fits:
- generated API clients
- docs indices
- schema snapshots
- codegen outputs

## Selection Heuristic

Choose the lightest guard that makes drift impossible, not merely less likely.

Order of preference:
1. generated downstream artifact
2. central matrix plus exhaustive tests
3. static or import-layer check
4. integration or regression test
5. procedural documentation only, as a last resort
