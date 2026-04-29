---
name: "task-breakdown"
description: "Turn a clarified spec into milestone-oriented tickets, dependency edges, and stage-gated delivery lanes."
---

# Task Breakdown

## Overview

Use this skill after a requirement brief or spec already exists. Its job is to convert that clarified scope into a set of OpenASE tickets, parent-child relationships, and dependency edges that maximize safe parallelism without sacrificing phase-level deliverability.

This is not a generic "make a todo list" skill. It is a delivery design skill. It should produce tickets that are independently meaningful, testable, and strict enough that agents cannot declare victory after only shallow local edits.

## When To Use

- A spec from `deep-interview`, a PRD, or another clarified requirement artifact already exists.
- The work is too large for one ticket but still needs strong coordination.
- The project would benefit from explicit blocking edges or parent-child relationships.
- You want to create milestone or integration gates instead of pushing all validation to the very end.

## Do Not Use

- The change is small enough to fit inside one well-scoped ticket.
- The requirements are still ambiguous; clarify them first.
- The plan is already broken down into tickets with clear dependencies and acceptance criteria.

## OpenASE Grounding

Design the breakdown around the platform that already exists:

- Use parent-child ticket relationships for hierarchy.
- Use `blocks` dependencies only for true execution blockers.
- Assume the scheduler may dispatch unrelated tickets in parallel when no blocking edge exists.
- Treat workflow pickup and finish statuses as real stage boundaries; do not advance a ticket unless its deliverable is genuinely ready for the next lane.

The goal is not to create the most tickets. The goal is to create the smallest graph that still preserves correctness, evidence, and throughput.

## Core Principles

1. Break down by deliverable, not by chore.
   - A ticket should represent a meaningful output, not a vague activity.
   - Avoid tickets like "research X", "prepare Y", or "adjust Z" unless the deliverable is explicit and reviewable.

2. Prefer vertical slices over layer buckets.
   - A good slice proves user-visible or system-visible progress across boundaries.
   - Do not split everything into separate frontend, backend, and test piles unless those are true reusable enablers.

3. Introduce milestone gates early.
   - Create explicit gate tickets for integration, connection, and proof of staged delivery.
   - A milestone gate should answer: "What must be true before the next batch of tickets can safely start?"

4. Use dependencies conservatively.
   - Add a `blocks` edge only when one ticket truly cannot start or complete without another.
   - Do not encode mere preference or loose sequencing as hard blockers.

5. Write strict acceptance and proof requirements.
   - Every ticket should define what "done" means.
   - Every important ticket should define how that claim will be proven: commands, screenshots, API responses, logs, migrations, or other artifacts.

6. Prevent lazy completion.
   - If a ticket has no concrete deliverable, no testable acceptance criteria, or no proof path, it is not ready to exist.

## Breakdown Workflow

1. Read the input spec carefully.
   - Capture the objective, non-goals, constraints, acceptance criteria, and major risks.
   - Identify which requirements are mandatory for the first useful delivery slice.

2. Identify milestone states before individual tickets.
   - Ask which phase-level system states matter.
   - Examples: first end-to-end happy path, first integration with auth, first production-safe release candidate.
   - Each milestone should correspond to a state that another engineer can verify, not just a project-management label.

3. Create milestone gate tickets.
   - A gate ticket is a real ticket that proves a stage is integrated and ready.
   - Gate tickets should usually focus on connection work, validation, and missing-gap cleanup across prior tickets.

4. Derive supporting tickets under each milestone.
   - Prefer these ticket kinds:
     - `slice`: vertical feature slice with visible progress
     - `enabler`: shared prerequisite that unlocks multiple later tickets
     - `milestone-gate`: stage proof and integration gate
     - `hardening`: reliability, observability, permission, or regression tightening

5. Build the DAG.
   - Link prerequisite tickets to the tickets they truly unblock.
   - Keep parallel branches open whenever the dependency is soft rather than hard.
   - Collapse unnecessary micro-tickets when they add context switching but no real autonomy.

6. Stress-test the graph.
   - If all integration is deferred to the last ticket, the breakdown is weak.
   - If most tickets are tiny chores, the breakdown is weak.
   - If a milestone gate cannot prove readiness for the next stage, it is not a real gate.

## Ticket Design Rules

Each proposed ticket should define at least:

- `Title`
- `Why this exists`
- `Deliverable`
- `In scope`
- `Out of scope`
- `Acceptance criteria`
- `Proof`
- `Suggested workflow or role`
- `Parent ticket` when applicable
- `Depends on`
- `Blocks`
- `Repository scope` when it can be narrowed safely
- `Kind`: `slice`, `enabler`, `milestone-gate`, or `hardening`

## Anti-Slack Heuristics

Rework the breakdown if any of these are true:

- A ticket can be "completed" without producing an observable deliverable.
- A ticket has acceptance criteria but no proof path.
- More than a minority of tickets are chores rather than slices or gates.
- The first milestone does not produce a vertical, integrated result.
- A late integration gate is carrying all end-to-end validation debt alone.
- Multiple tickets differ only by tiny file edits and should really be one coherent deliverable.

## Milestone Gate Guidance

A strong milestone gate ticket should:

- depend on the slices and enablers required for that phase
- focus on integration, connection, proof, and residual-gap closure
- produce evidence that the phase is genuinely ready
- block later waves until that evidence exists

Typical gate proofs include:

- end-to-end happy path working
- cross-service wiring confirmed
- migration applied and exercised
- auth or permission boundary verified
- regression suite or smoke checks passing for the new slice

## Default Deliverable Shape

Return these sections:

1. `Objective Summary`
2. `Milestones`
3. `Milestone Gates`
4. `Ticket Table`
5. `Dependency Edges`
6. `Parallelism Plan`
7. `Critical Risks`
8. `Creation Order`

Inside `Ticket Table`, include one row or subsection per proposed ticket with:

- temporary key
- title
- kind
- parent
- depends on
- acceptance criteria
- proof
- workflow or role suggestion

## Output Quality Bar

Before finalizing the breakdown, check:

- The graph is as parallel as possible without becoming unsafe.
- Every dependency edge has a clear rationale.
- The first milestone proves real progress, not just preparatory work.
- The final milestone is not the first time the system is ever integrated.
- Each ticket is large enough to matter, but small enough to own and verify.
