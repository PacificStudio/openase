# AutoHarness Fix Plan

Use this template when turning a diagnosis into an implementation plan.

## 1. Problem Statement

Capture the bug class, not just the current symptom.

Template:
- `Observed failure:`
- `Underlying drift:`
- `Why the current harness allowed it:`
- `Desired invariant:`

## 2. Authority Model

Name the single source of truth you want after the change.

Template:
- `Authoritative definition:`
- `Derived consumers:`
- `Synchronization mechanism:`
- `What becomes illegal to edit by hand:`

## 3. Staged Plan

### Phase 1 - Stop the bleeding

Land the smallest durable guardrail.

Examples:
- add a regression test on the real boundary
- centralize one duplicated matrix
- add CI freshness check for one generated artifact
- add structured logs around an opaque external integration

### Phase 2 - Remove manual sync

Reduce duplicate authority.

Examples:
- move handwritten client types to generated outputs
- move command routing decisions behind a shared contract helper
- move repeated permission checks into a central matrix or helper

### Phase 3 - Scale the guardrail

Expand from one bug to the whole class.

Examples:
- convert a one-off regression into a table-driven suite
- add import-layer checks across the whole repo
- add generated snapshots or parity checks for every resource group

## 4. Verification Plan

Specify how to prove the harness now holds.

Template:
- `Unit checks:`
- `Integration checks:`
- `E2E checks:`
- `CI freshness checks:`
- `Observability checks:`

## 5. Residual Risks

List what still depends on convention.

Examples:
- remaining handwritten compatibility layers
- boundaries that still lack import guards
- external systems that only have logs, not deterministic tests

## 6. Preferred Output Format

When reporting back, keep the plan in this shape:
- `Root Cause`
- `Single Source of Truth`
- `Guards to Add`
- `Implementation Order`
- `Verification`
- `Follow-up Debt`
