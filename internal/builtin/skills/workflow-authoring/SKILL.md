---
name: "workflow-authoring"
description: "Write workflow harness files that define role, status semantics, feedback loops, and durable delivery rules for disposable ticket workspaces."
---

# Workflow Authoring

## Overview

Use this skill when creating or editing an OpenASE workflow harness file. Its job is to turn a vague workflow idea into a concrete harness that clearly defines what the workflow owns, what counts as completion, how ticket state should move, where feedback comes from, and how work survives beyond a disposable ticket workspace.

This skill is about authoring the workflow file itself, not just the underlying engineering task.

## Runtime Reality You Must Design Around

OpenASE ticket execution is not a persistent personal development environment.

- Each ticket run gets its own workspace.
- That workspace is created when the ticket starts running.
- The local contents of that ticket workspace are disposable and may be cleaned up after the run finishes.
- Any output that must survive the run must be pushed to the repository or written back to durable OpenASE surfaces such as ticket comments or project updates.

A good workflow harness must make that lifecycle explicit. Do not write harnesses that assume a long-lived local scratchpad or future manual cleanup inside the same workspace.

## Use The Shared Platform Contract

Remember that every workflow already receives shared workflow execution rules from the platform. Reuse and specialize those rules instead of fighting them.

In particular, author around these always-on assumptions:

- ticket comments are the durable output channel
- the current workflow pickup status is the active owned lane
- the current workflow finish status must only be used when the workflow deliverable is truly ready
- blocked termination must be reserved for genuinely unprogressable cases
- interactive waiting is not a reliable operating model

Write harness content that sharpens these generic rules for one workflow rather than duplicating all of them verbatim.

## What Every Workflow File Must Define

At minimum, a workflow harness should make these items explicit:

1. The workflow's primary mission.
   - What class of tickets this workflow is for.
   - What it is expected to deliver.
   - What it must not take on.

2. The workflow's capabilities.
   - What role this workflow plays.
   - Which bound skills are expected to be used and for what.
   - Which kinds of repository changes or validations it is competent to perform.

3. Status meaning.
   - What it means for a ticket to be in the pickup status.
   - What the finish status means in this workflow.
   - If a blocked or abnormal finish status exists, what threshold justifies using it.

4. Status order and gates.
   - What must be true before the workflow advances a ticket out of its owned lane.
   - Which validations, deliverables, or feedback must exist before the next state is justified.
   - If the workflow is part of a staged delivery chain, define the gate in terms of proof, not intention.

5. Feedback collection.
   - Where the workflow should look for feedback from humans.
   - Where it should record progress, questions, validations, and risks.
   - How it should absorb ticket comments, project updates, external links, PR review, or other signals.

6. Failure and escalation semantics.
   - What counts as a real blocker.
   - What should become a follow-up ticket instead of inflating the current scope.
   - Which status to use when the workflow cannot responsibly continue.

## Recommended Harness Structure

Use a predictable structure so the workflow remains inspectable and editable:

1. `Role`
2. `Runtime Context`
3. `Mission`
4. `Capabilities`
5. `Status Control`
6. `Feedback Loop`
7. `Execution Rules`
8. `Delivery Standard`
9. `Escalation / Blockers`
10. `Optional Repo or Branch Rules`

The harness does not need these exact headings, but it should cover the same semantics.

## Status Authoring Guidance

When you write workflow state rules, make the semantics operational:

- Define what work the workflow owns while the ticket remains in its pickup status.
- Define what evidence must exist before the workflow can move the ticket to its finish status.
- Define whether the finish status means:
  - ready for the next workflow
  - ready for review
  - ready for deployment
  - globally done
- Never leave the meaning implicit.

Good status rules describe:

- owned lane
- exit gate
- downstream readiness
- blocked threshold

Weak status rules only say "move to Done when finished."

## Feedback Loop Guidance

A workflow should explicitly say where it reads and writes feedback.

When this skill is used for AI suggestions or review, ground the advice in actual project evidence before proposing changes:

- inspect the user's stated goal first
- inspect the current workflow harness and status bindings
- inspect recent tickets that ran through the workflow, especially retries, paused tickets, and repeated failure patterns
- inspect recent activity, ticket comments, and durable ticket descriptions for what humans actually asked for
- prefer explaining which observed pattern caused each suggested harness change

Do not invent a lane split, exit gate, or escalation rule unless the observed workflow history or the user's request supports it.

Common durable sources:

- ticket description
- ticket comments
- the persistent workpad comment
- project updates when high-visibility escalation is needed
- linked PRs, issues, design docs, or other external references

A workflow should also say what belongs in feedback:

- progress updates
- validations run and exact results
- blocker diagnosis
- questions that need human input
- follow-up tasks or residual risks

If the workflow needs human feedback before a state transition, say so clearly and name the source of truth.

## Repository And Branch Rules

These are optional, but when relevant the harness should state:

- which repository or repositories the workflow may change
- whether it should stay within the ticket's repo scope unless explicitly widened
- how branches should be named or updated
- whether commit and push are required before finishing
- whether certain generated files, changelogs, or docs must be kept in sync

Because ticket workspaces are disposable, any workflow that produces repository output should state that unpublished local work is not durable.

## Comment And Reporting Rules

When the workflow needs durable reporting, say what should be written back:

- plan or intent
- implementation progress
- validation commands and outcomes
- review findings
- risk notes
- final readiness statement

Do not rely on terminal output alone. If the result matters after the ticket run ends, require it to be written to a durable platform channel.

## Milestone And Gate Workflows

For gate-style workflows, be even stricter:

- define the phase being proven
- define the integration or connection work expected
- define the exact proof required before the gate can pass
- define what missing evidence keeps the ticket in the active lane

Gate workflows should be written around proof of staged readiness, not vague coordination.

## Common Workflow Patterns

These are common patterns worth documenting directly in workflow harnesses when they apply.

### 1. Coding Workflow

Use this when one workflow owns implementation from ticket pickup through repository delivery.

The harness should state clearly that:

- the workflow accepts tickets from one active implementation lane
- the workflow completes the scoped code change, runs the required validation, and records the results durably
- the workflow commits and pushes all required changes before leaving its owned lane
- the workflow must use a dedicated working branch rather than a protected main branch
- the workflow should link the resulting GitHub PR URL, or the relevant GitHub issue URL when appropriate, back to the ticket
- the workflow must not move the ticket forward while meaningful local workspace changes remain uncommitted or unpublished

Typical gate:

- code changed
- validation passed
- branch pushed
- PR or issue link added to the ticket
- workpad or ticket comments updated with proof

### 2. Split Backend / Frontend Delivery

Use this when backend and frontend work should be handled by different workflows or models.

Typical sequence:

- `Todo`
- `Backend In Progress`
- `Frontend In Progress`
- `In Review`

Author the harnesses so that:

- the backend workflow only owns backend-facing deliverables and does not silently absorb frontend work
- the frontend workflow only starts after the backend gate or contract is truly ready, unless the split is intentionally parallelized around a stable interface
- both workflows document what they must leave behind for the next lane: API shape, branch, ticket comments, workpad notes, screenshots, test output, or follow-up risks
- the review lane expects integrated evidence, not just claims from both sides

This pattern is especially useful when you want to exploit different model strengths or different role-specific skills across the two lanes.

### 3. Deployment Workflow

Use this when deployment is a distinct operational lane rather than an implicit side effect of coding completion.

The harness should specify:

- which deployment skill or operational capability is expected to be used
- which environment may be touched
- which pre-deployment gates must already be satisfied
- which deployment proof is required before the ticket can leave the deployment lane
- where rollback notes, release notes, or environment-specific warnings must be recorded

Typical gate:

- build or artifact prepared
- deployment command executed through the allowed path
- health check or smoke check passes
- deployment result written back to ticket comments or project update

### 4. GitHub Issue Sync Workflow

Use this for operational backlog sync from GitHub into OpenASE.

The workflow should state that it:

- uses `gh` CLI to list or inspect open GitHub issues that are not already closed
- maps those issues into backlog tickets inside OpenASE
- avoids creating duplicates by checking existing external references or linked URLs first
- requires platform permission to create tickets
- writes a durable sync summary showing which issues were imported, skipped, or merged

Typical gate:

- GitHub issues queried successfully
- only non-closed issues considered
- backlog tickets created or linked idempotently
- sync summary recorded with counts and any failures

If this workflow cannot create tickets due to missing scope, the harness should tell the agent to stop in the safest blocked or follow-up status and report that `tickets.create` access is required.

## What Belongs In A Skill Instead

Do not overload the workflow harness with every reusable procedure.

Put cross-workflow, repeatable procedures into skills when they are:

- reusable across multiple workflows
- too detailed for the role-level harness
- operational standards such as review checklists, task breakdown rules, or security checks

The workflow should say which skill matters and why. The skill should carry the reusable procedure body.

## Common Anti-Patterns

Rewrite the harness if any of these are true:

- The workflow mission is broader than one role can safely own.
- The pickup and finish statuses have no operational meaning.
- The harness assumes local workspace state survives after the run.
- The workflow never says where durable progress and validation should be recorded.
- The workflow mixes immediate ticket scope with broad project cleanup.
- The workflow tells the agent to wait for humans by default instead of progressing autonomously.
- The workflow repeats a large reusable checklist that should be a bound skill.

## Suggestion Mode

When the user asks for suggestions instead of only critique, switch from pure review mode into editor mode:

- start with a short diagnosis tied to the user request and the workflow's recent ticket/run history
- then provide either a precise diff plan or a full revised harness draft
- if confidence is high, prefer returning a complete revised harness in a fenced Markdown block so it can be applied directly
- if confidence is lower, separate hard recommendations from open questions and avoid rewriting uncertain sections as if they were settled

Suggestion mode should still preserve existing workflow intent unless the user explicitly wants a role or ownership change.

## Default Deliverable Shape

When authoring or reviewing a workflow file, return these sections:

1. `Workflow Intent`
2. `Status Semantics`
3. `Feedback Contract`
4. `Repo And Branch Rules`
5. `Escalation Rules`
6. `Suggested Harness Outline`
7. `Open Questions`

When the user explicitly asks for an editable suggestion, add one more section:

8. `Suggested Harness Draft`

## Authoring Quality Bar

Before finalizing a workflow harness, check:

- The workflow's main task is explicit.
- The bound skills are named or implied clearly enough for the role.
- The meaning of each relevant status is operational, not ceremonial.
- The order of execution and gates is visible.
- Human feedback sources and durable output channels are explicit.
- Disposable workspace reality is accounted for.
- Optional branch, repo, and comment protocols are stated when they matter.
