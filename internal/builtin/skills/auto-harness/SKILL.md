---
name: auto-harness
description: "Diagnose and strengthen a repository's harness layer: AGENTS.md rules, knowledge layout, architecture boundaries, lint and type gates, API and generated-client contracts, test scaffolding, structured logging, and technical-debt tracking. Use when Codex needs to audit drift-prone semantics that are repeated across code, docs, schemas, generated artifacts, frontend/backend boundaries, or CI, then design or implement guardrails with progressive disclosure and zero-tolerance contract checks."
---

# Auto Harness

## Overview

Design the project harness so repository rules become executable. Start by mapping where semantics can drift, then convert the risky ones into generated artifacts, contract tests, type or layer guards, and CI checks.

## Workflow

1. Build a harness map before proposing fixes.
   - Read entrypoint materials first: `AGENTS.md`, repo-root docs, `Makefile`, package scripts, CI workflows, API specs, codegen config, lint config, type-check config, and test runners.
   - Avoid bulk-loading every doc. Follow progressive disclosure: read only the references needed for the current risk.
   - Record the harness surfaces that already exist: documentation rules, generated files, test suites, CI gates, architecture constraints, observability standards, and debt tracking.

2. Load the matching reference files.
   - Read `references/checklist.md` to score the current harness and find missing guardrails.
   - Read `references/guide.md` when you need the design principles and recommended repository shapes.
   - Read `references/guard-patterns.md` when you need concrete enforcement strategies.
   - Read `references/fix-plan.md` when you need a staged remediation plan or a deliverable template.
   - Read `references/upstream.md` when you need the builtin bundle provenance or manual sync guidance.

3. Diagnose drift as a systems problem, not a single bug.
   - Search for the same semantic repeated across layers: permissions, routes, CLI commands, OpenAPI, generated clients, frontend assumptions, docs, logging fields, config, and tests.
   - Flag any semantic that is declared in multiple places without a hard synchronization mechanism.
   - Treat "scope list says yes, runtime path says no" and similar mismatches as harness failures, not just implementation bugs.
   - Write the diagnosis in terms of duplicated authority, missing invariants, missing generated artifacts, or missing tests.

4. Prefer hard guardrails over policy prose.
   - Establish a single source of truth for drift-sensitive semantics.
   - Generate downstream artifacts instead of hand-maintaining parallel definitions.
   - Add failing checks in CI for zero-drift contracts: OpenAPI parity, generated client freshness, import-layer guards, JSON schema checks, snapshot parity, or table-driven route or scope coverage.
   - Add type-level or package-boundary checks when architecture depends on layering.
   - Add structured logs when the real system is expensive to exercise and deterministic tests are limited.

5. Use tests to constrain maintainability.
   - Add unit tests for pure rules and table-driven contracts.
   - Add integration tests for boundary behavior, storage wiring, and auth or permission enforcement.
   - Add e2e tests for user-visible workflows and generated frontend/backend integration points.
   - Add regression tests for every bug whose root cause was semantic drift across layers.
   - Favor tests that prove two layers cannot disagree, rather than tests that only validate each layer in isolation.

6. Produce actionable deliverables.
   - Deliver a short harness map.
   - Deliver the top drift risks and why they exist.
   - Deliver a prioritized fix plan with fast wins, foundational changes, and CI gates.
   - If asked to implement, land the smallest durable guardrail first, then expand coverage.

## Operating Rules

- Prefer executable constraints over narrative guidance.
- Prefer generated artifacts over duplicate handwritten definitions.
- Prefer progressive disclosure over dumping every document into context.
- Prefer contract tests that cover real commands, real routes, and real generated clients.
- Treat frontend/backend contract drift as zero-tolerance when codegen or schema generation is feasible.
- Treat missing structured logs as a harness defect when external integrations are costly to replay.
- Keep fixes incremental, but design around the final source-of-truth model.

## Default Deliverable Shape

Return these sections when doing a harness audit:

1. `Harness Map` - current rules, generators, tests, CI gates, and documentation entrypoints.
2. `Drift Risks` - duplicated semantics, weak invariants, or missing sync checks.
3. `Guardrail Plan` - concrete code or test mechanisms that would prevent recurrence.
4. `Fix Order` - what to land first, what to generate later, and what to move into CI.
5. `Residual Risks` - what still depends on convention instead of enforcement.
