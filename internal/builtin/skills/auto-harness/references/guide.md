# AutoHarness Guide

## Harness Layer Definition

The harness layer is the project-specific control system around the product code. It is strongly tied to the repository and architecture, but it is not the business feature itself.

Typical harness surfaces:
- `AGENTS.md` and repo operating rules
- repository knowledge layout and documentation index
- architectural boundaries and ownership rules
- lint, format, and type-check constraints
- OpenAPI, JSON schema, protobuf, GraphQL, or generated client contracts
- frontend/backend coupling points
- test scaffolding and fake-system infrastructure
- structured logging and diagnostic conventions
- dependency selection principles
- technical-debt tracking and migration notes

Harness quality determines whether an agent can safely recover context, follow the intended architecture, and avoid repeating the same bug class.

## Core Diagnosis Logic

Use this logic whenever a bug looks like a "small mismatch":

1. Ask whether the same semantic is represented in multiple places.
2. Ask which place is the real source of truth.
3. Ask how the downstream copies are synchronized.
4. Ask what would fail automatically if they drifted.
5. If the answer is "nothing", treat it as a harness gap.

This is the key pattern behind permission drift, route drift, CLI drift, codegen drift, stale docs, stale examples, and missing observability fields.

## Design Principles

### 1. Single Source of Truth

For every drift-sensitive semantic, choose one authoritative definition.

Examples:
- API shape -> OpenAPI spec
- frontend API types -> generated from OpenAPI
- permission capability matrix -> central contract table
- allowed package imports -> explicit boundary rule
- structured log schema -> typed helper or common logger adapter

### 2. Progressive Disclosure

Make the repository a durable memory system, but do not force every task to read every document.

Use layered discovery:
- entrypoint docs: `AGENTS.md`, top-level index, architecture map
- targeted docs: boundary notes, subsystem guides, data contracts
- deep references: migrations, debt logs, incident notes, external protocol details

The agent should be able to find the right reference quickly, not ingest the whole archive.

### 3. Enforce the Architecture in Code

If a boundary matters, encode it.

Examples:
- package import guards
- generated types instead of duplicated request or response structs
- table-driven permission-to-route matrices
- test-only adapters for expensive external systems
- static checks for forbidden dependency direction

### 4. Test the Coupling Points

The highest-value tests usually sit on boundaries:
- backend route <-> auth scope
- OpenAPI <-> generated frontend types
- CLI subcommand <-> final HTTP path
- schema <-> persisted payload
- log emitter <-> downstream diagnostics contract

### 5. Optimize for Maintainability, Not Heroics

Prefer a design where future contributors cannot accidentally do the wrong thing.

Signals of good harness design:
- obvious entrypoints
- generated clients or schemas
- few handwritten duplicate contracts
- narrow architecture seams
- fast local verification
- CI failures that point to the real source of drift

## When to Add Stronger Guardrails

Strengthen the harness immediately when you see any of these:
- the same rule repeated in 3 or more files
- manual sync steps in PR descriptions
- bugs caused by stale generated code or stale docs
- frontend and backend disagreeing on nullability or field shape
- CLI hitting a different route shape than the intended runtime contract
- architecture explained only in prose, not enforced anywhere
- external system failures that are hard to reproduce and poorly logged

## Preferred Guardrail Order

1. Centralize the authority.
2. Generate downstream artifacts.
3. Add focused regression tests around the real boundary.
4. Add CI freshness checks.
5. Improve logs for the cases that still escape tests.

## Anti-Patterns

Avoid these unless there is no better option:
- duplicated handwritten contracts across backend, frontend, and CLI
- architecture rules that exist only in a wiki page
- broad e2e suites used as a substitute for boundary tests
- hidden sync steps such as "remember to update all three places"
- stringly typed log payloads without stable keys
- giant docs that are not indexed and force full-context loading
