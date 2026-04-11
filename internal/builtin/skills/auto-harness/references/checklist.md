# AutoHarness Checklist

Use this checklist to audit an existing project. Score each item as:
- `0` missing
- `1` partial or manual
- `2` enforced and maintained

## A. Repo Knowledge Layout

- `AGENTS.md` or equivalent entrypoint exists and reflects current practice.
- Top-level docs tell agents where to start and where deeper references live.
- Architecture docs are organized by subsystem or boundary, not one giant file.
- Deep reference material is discoverable through links from entrypoint docs.
- Technical debt or migration notes are stored in stable, searchable locations.

## B. Architecture Boundaries

- Layer boundaries are explicit and easy to name.
- Dependency direction is documented.
- Dependency direction is also enforced by code or checks.
- Ownership of shared contracts is clear.
- Cross-layer adapters are narrow and intentionally placed.

## C. Contract Integrity

- There is a single source of truth for each important API contract.
- Frontend types or clients are generated from the source contract where practical.
- CLI request paths and parameters are covered by tests against the real route shape.
- Permission or capability matrices are centralized and testable.
- Contract freshness is checked in CI.

## D. Test Harness Coverage

- Pure rule logic has unit tests.
- Boundary behavior has integration tests.
- User-visible workflows have e2e coverage where it matters.
- External systems have deterministic fakes, fixtures, or replayable adapters.
- Every historical drift bug has a regression test at the correct boundary.

## E. Tooling and Static Guards

- Type checks are part of the normal developer workflow.
- Lint rules catch the most common architectural footguns.
- Generated artifacts are reproducible from checked-in commands.
- CI checks fail loudly when generated files or schemas are stale.
- Local scripts or make targets make the guardrails easy to run.

## F. Observability and Expensive Integrations

- External integration boundaries emit structured logs with stable keys.
- Logs include correlation identifiers needed for later diagnosis.
- Expensive or rate-limited paths have enough logs to debug without replay.
- Logging shape is consistent enough to assert on in tests or tooling.
- Failures include actionable metadata, not just free-form strings.

## G. Decision Hygiene

- Dependency selection principles are written down.
- Deviations from preferred patterns are tracked with rationale.
- TODOs and debt items are grouped by architectural seam or contract.
- Removal criteria exist for temporary compatibility layers.
- Known weak spots have explicit owners or next actions.

## Audit Output Template

Use this shape in your report:

- `Harness Scorecard` - section totals and weakest categories
- `High-Risk Drift Points` - the semantics most likely to split across layers
- `Missing Hard Guards` - what is still enforced only by convention
- `Recommended First Fix` - the smallest change with durable leverage
- `CI Upgrades` - what to add so the drift cannot silently return
