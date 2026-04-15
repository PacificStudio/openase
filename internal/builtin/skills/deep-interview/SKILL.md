---
name: "deep-interview"
description: "Clarify ambiguous requests with a focused, Socratic interview before planning or implementation."
---

# Deep Interview

## Overview

Use this skill when the request is broad, ambiguous, or missing concrete acceptance criteria. Its job is to turn a vague idea into an execution-ready requirement brief before deeper planning or implementation begins.

## When To Use

- The user wants to explore a broad idea without making hidden assumptions.
- The request lacks clear scope, non-goals, or success criteria.
- A later planning or execution step would otherwise guess at intent.
- The change touches an existing codebase and the current pattern or boundary is still unclear.

## Do Not Use

- The request already names concrete files, symbols, errors, or acceptance criteria.
- The user explicitly wants to skip clarification and accept the risk.
- A complete requirement brief or implementation plan already exists.

## Workflow

1. Start with context you can gather yourself.
   - Inspect the codebase, docs, or surrounding ticket context before asking the user about project internals.
   - Summarize what is already known and which parts are still assumptions.

2. Ask one question at a time.
   - Do not batch multiple unrelated questions.
   - Ask the highest-leverage unresolved question first.

3. Prioritize requirement clarity in this order.
   - Intent: why the user wants the change.
   - Outcome: what end state should exist when the work is done.
   - Scope: how far the change should go.
   - Non-goals: what should stay out of scope.
   - Decision boundaries: what the agent may decide without asking again.
   - Constraints: technical, business, operational, or timeline limits.
   - Success criteria: how completion will be judged.

4. Pressure-test each important answer.
   - Ask for an example, counterexample, or concrete signal.
   - Surface the hidden assumption behind the answer.
   - Force a boundary or tradeoff when the scope is still fuzzy.
   - If the user is describing symptoms, steer back toward the underlying problem.

5. Keep the interview efficient.
   - Prefer evidence-backed confirmation questions for brownfield work.
   - Do not ask the user for facts you can inspect directly.
   - Stop once the request is clear enough to plan, not after exhausting every possible question.

6. Crystallize the result into a requirement brief.
   - Intent
   - Desired outcome
   - In-scope work
   - Out-of-scope / non-goals
   - Decision boundaries
   - Constraints
   - Testable acceptance criteria
   - Open risks or residual ambiguity

7. Hand off cleanly.
   - If implementation approach still needs review, hand off to a planning step.
   - If the request is now concrete enough to execute, pass the brief forward as the source of truth.

## Operating Rules

- Ask about intent and boundaries before implementation details.
- Do not rotate topics just for coverage when one answer is still vague.
- Revisit at least one earlier answer with a deeper follow-up before declaring the brief complete.
- Do not hand off while non-goals or decision boundaries remain implicit.
- If the user chooses to proceed early, explicitly record the residual risk.

## Default Deliverable Shape

Return these sections when the interview is complete:

1. `Intent`
2. `Desired Outcome`
3. `In Scope`
4. `Out of Scope`
5. `Decision Boundaries`
6. `Constraints`
7. `Acceptance Criteria`
8. `Residual Risks`
