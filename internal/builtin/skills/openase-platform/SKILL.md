---
name: "openase-platform"
description: "Platform operations for tickets, projects, and runtime coordination inside OpenASE."
---

# OpenASE Platform Operations

Prefer the wrapper injected into the workspace:

```bash
./.openase/bin/openase ticket list --status-name Todo
```

This wrapper is the `openase` binary with the current workspace's OpenASE
platform context already attached. Use it first. Do not hand-roll URLs, guess
endpoints, bypass the platform by writing the database directly, or fake
platform state.

## What This Skill Is For

OpenASE is not a small helper that just runs commands. It is an issue-driven
automated software engineering platform. Per the PRD, it is responsible for at
least:

- Project control: descriptions, statuses, repos, workflows, skills, and
  scheduled jobs
- Ticket control: lifecycle, status transitions, comment primitives,
  usage/cost, and external links
- Execution control: agents, providers, machines, runtimes, and orchestration
  loops
- Controlled autonomy: agents can operate the platform within granted scope
  and close the loop from claim to execution to platform writeback to follow-up
  tickets
- Auditability: every platform write goes through the API / ActivityEvent /
  timeline and stays attributable

For agents, the core purpose of `openase` is not just inspection. It is
reading and writing real control-plane state inside platform-enforced
boundaries. The tickets you create, project descriptions you update, repos you
register, and comments you append all affect later scheduling, UI state, audit
trails, and other agents' context.

Treat `openase` as the control-plane API for the current engineering project.

- The code repository is only a workspace, not the task system.
- Tickets, projects, workflows, skills, and machines in OpenASE are the real
  control-plane entities.
- When you need to change platform state, prefer the `openase` CLI. Do not try
  to express platform state indirectly by editing local files.
- Read before you write: inspect the current state first, then make the
  smallest necessary change.
- If you exceed scope, the platform returns `403`. That usually means the
  current harness did not grant the required `platform_access`.

## Runtime Contract First

The runtime injects a capability contract that tells you which principal kind,
scopes, and environment variables are actually available in this session.
Treat that runtime contract as the source of truth.

Common environment variables include:

- `OPENASE_API_URL`: OpenASE API base URL
- `OPENASE_AGENT_TOKEN`: current agent token
- `OPENASE_PROJECT_ID`: current project UUID
- `OPENASE_TICKET_ID`: current ticket UUID; only present in ticket runtime or
  ticket-focused Project AI
- `OPENASE_CONVERSATION_ID`: current project conversation UUID; available in
  Project AI conversations
- `OPENASE_PRINCIPAL_KIND`: current principal kind, such as `ticket_agent` or
  `project_conversation`
- `OPENASE_AGENT_SCOPES`: current token scopes, comma-separated

Common platform subcommands auto-fill context in this order:

- project scope: `--project-id` -> `OPENASE_PROJECT_ID`
- ticket scope: positional `[ticket-id]` -> `--ticket-id` ->
  `OPENASE_TICKET_ID`
- API URL: `--api-url` -> `OPENASE_API_URL`
- token: `--token` -> `OPENASE_AGENT_TOKEN`

Important limits:

- Most ID parameters require UUIDs and do not accept human-readable ticket
  identifiers such as `ASE-42`.
- Output defaults to JSON and can be filtered with `--json`, `--jq`, or
  `--template`.
- When platform calls fail, the CLI prints the HTTP method, path, status, and
  API error code directly, so you do not need to guess.
- Tokens are short-lived and scope-bound; not every workflow can modify
  projects, repos, or scheduled jobs.
- Shared wrapper flags accept both kebab-case and snake_case, such as
  `--status-name` / `--status_name` and `--body-file` / `--body_file`.

### Principal-Specific Constraints

Check `OPENASE_PRINCIPAL_KIND` before assuming a route is available.

When the principal is `ticket_agent`:

- Treat this as the current ticket runtime.
- Current-ticket routes are limited to the ticket identified by
  `OPENASE_TICKET_ID`.
- Project-level writes still depend on the scopes listed above.

When the principal is `project_conversation`:

- Treat this as a project-scoped conversation runtime, not a ticket runtime.
- Use project-scoped ticket mutation routes when `tickets.update` is granted.
- Do not assume current-ticket comment/update/report-usage endpoints are
  available.
- Ticket-runtime-only routes can reject this principal kind even when
  `OPENASE_TICKET_ID` is present.
- `OPENASE_CONVERSATION_ID` is often the stable runtime identity you should
  use when inspecting the current Project AI session.

If you are unsure which write path to use, inspect `OPENASE_PRINCIPAL_KIND`
and `OPENASE_AGENT_SCOPES` first, then pick the smallest typed command that
matches the granted scope.

## Command Selection Rules

Use this order of preference:

1. Prefer a typed `openase` command whose semantics already match the target
   entity.
2. Read current state first, then write the minimum necessary change.
3. Use `openase api` only when there is no suitable typed command.
4. Do not edit local files to "represent" platform state.
5. Do not rely on database access, guessed URLs, or undocumented compatibility
   paths.

This skill is about platform reads and writes. It is not a substitute for the
separate `ticket-workpad` skill, repository code changes, or workflow-specific
execution instructions.

## Top-Level Commands

Below is the current top-level `openase` command surface from source. Not every
command is appropriate for agents; the first groups are the ones you will use
most often.

### Agent / API Surface

- `api`: raw HTTP passthrough, the fallback entrypoint for any exposed API
- `ticket`: shared platform wrapper for common ticket reads and writes;
  non-overlapping detail/run/dependency/external-link subcommands still go
  directly through OpenAPI
- `status`: ticket status board management
- `chat`: ephemeral chat and project conversations
- `project`: shared platform wrapper for update/add-repo; list/get/create/delete
  still go directly through OpenAPI
- `repo`: project repos, GitHub repo discovery, and ticket repo scopes
- `workflow`: workflow and harness reads and writes
- `scheduled-job`: scheduled job management
- `machine`: machine registration, probing, and resource inspection
- `provider`: provider inspection and configuration
- `agent`: agent inspection, pause/resume, output, and step reads
- `activity`: project activity timeline reads
- `channel`: notification channel management and tests
- `notification-rule`: notification rule management
- `skill`: skill inspection, updates, binding, and refresh
- `watch`: SSE watch streams
- `stream`: SSE stream feeds

### Service / Control Plane Operations

- `serve`: start only the HTTP API service
- `orchestrate`: start only the orchestration loop
- `all-in-one`: start the API and orchestrator in one process
- `up`: start the local OpenASE service
- `setup`: initialize the local runtime environment
- `down`: stop the local service
- `restart`: restart the local service
- `logs`: inspect local service logs
- `doctor`: local environment diagnostics

### Admin / Schema / Utility

- `issue-agent-token`: issue an agent token
- `openapi`: export or inspect OpenAPI artifacts
- `version`: inspect the version

In practice, agents most often use these commands inside a workspace:

- `ticket`
- `status`
- `chat`
- `project`
- `repo`
- `workflow`
- `activity`
- `scheduled-job`
- `machine`
- `provider`
- `agent`
- `channel`
- `notification-rule`
- `skill`
- `watch` / `stream`
- `api`

`serve` / `orchestrate` / `up` / `down` / `restart` / `issue-agent-token` are
more about platform operations or control-plane startup and are usually not the
first choice for normal ticket execution.

## Safe Default Commands

This is the safest first layer for agents to use. The semantics are stable and
suitable for direct workflow / harness calls.

### 1. List current project tickets

```bash
./.openase/bin/openase ticket list
./.openase/bin/openase ticket list --status-name Todo --priority high
./.openase/bin/openase ticket list --json tickets
```

Capabilities:

- Calls `GET /projects/{projectId}/tickets`
- Supports multi-value filtering with `--status-name`
- Supports multi-value filtering with `--priority`

Use this before creating follow-up tickets or mutating status. It gives you the
real project board state instead of assuming a ticket name or status lane.

### 2. Create a ticket

```bash
./.openase/bin/openase ticket create \
  --title "Add integration coverage" \
  --description "Split the follow-up work" \
  --priority high \
  --type task \
  --external-ref "PacificStudio/openase#39"
```

Capabilities:

- Calls `POST /projects/{projectId}/tickets`
- `--title` is required
- Optional: `--description`, `--priority`, `--type`, and `--external-ref`

Good fits:

- You discover that a follow-up ticket is needed
- You need to split work that is outside the current scope
- You need to attach follow-up security, testing, or deployment work back to
  the platform explicitly

### 3. Update the current ticket

```bash
./.openase/bin/openase ticket update --description "Record new findings from execution"
./.openase/bin/openase ticket update --status-name Done
./.openase/bin/openase ticket update $OPENASE_TICKET_ID --external-ref "gh-123"
```

Capabilities:

- Calls `PATCH /tickets/{ticketId}`
- Can update `--title`, `--description`, and `--external-ref`
- Can update status via `--status`, `--status-name`, or `--status-id`
- `--status-name` and `--status-id` are mutually exclusive
- At least one update field is required

In `project_conversation` runtimes, do not assume the current-ticket variant is
available. Prefer project-scoped ticket mutation routes or the typed command
shape exposed by the current runtime contract.

### 4. Record usage / cost

```bash
./.openase/bin/openase ticket report-usage \
  --input-tokens 1200 \
  --output-tokens 340 \
  --cost-usd 0.0215
```

Capabilities:

- Calls `POST /tickets/{ticketId}/report-usage`
- Records incremental usage instead of overwriting totals
- Set at least one field: `--input-tokens`, `--output-tokens`, or `--cost-usd`

Use this only when the current runtime exposes a compatible ticket route.
Project-conversation runtimes can lack current-ticket reporting endpoints even
if `OPENASE_TICKET_ID` exists.

### 5. Manage ticket comments

List comments:

```bash
./.openase/bin/openase ticket comment list
```

Create a regular comment:

```bash
./.openase/bin/openase ticket comment create --body "Record the current blocker"
./.openase/bin/openase ticket comment create --body-file /tmp/comment.md
```

Update an existing comment:

```bash
./.openase/bin/openase ticket comment update $OPENASE_TICKET_ID $COMMENT_ID --body-file /tmp/comment.md
```

Capabilities:

- `ticket comment list` calls `GET /tickets/{ticketId}/comments`
- `ticket comment create` calls `POST /tickets/{ticketId}/comments`
- `ticket comment update` calls `PATCH /tickets/{ticketId}/comments/{commentId}`
- Choose exactly one of `--body` or `--body-file`

`openase-platform` only provides the comment primitives here. It does not
define workpad semantics directly. When you need persistent workpad
maintenance, use the separately bound `ticket-workpad` skill, which builds on
top of these `comment list/create/update` primitives.

### 6. Update the project description

```bash
./.openase/bin/openase project update --description "Update the latest project context"
```

Capabilities:

- Calls `PATCH /projects/{projectId}`
- This is the main high-frequency project write operation today

Good fits:

- Product or research roles need to write findings back to the project
- The current ticket uncovers longer-term context that should live in the
  project description

### 7. Register a project repo

Preferred current form:

```bash
./.openase/bin/openase repo create $OPENASE_PROJECT_ID \
  --name "worker-tools" \
  --url "https://github.com/acme/worker-tools.git" \
  --default-branch main \
  --label go \
  --label backend
```

Compatibility form:

```bash
./.openase/bin/openase project add-repo \
  --name "worker-tools" \
  --url "https://github.com/acme/worker-tools.git" \
  --default-branch main
```

Capabilities:

- `repo create` calls `POST /projects/{projectId}/repos`
- `--name` and `--url` are required
- `--default-branch` defaults to `main`
- `--label` can be repeated

Prefer `repo create` when available because it models repos as first-class
project entities. Keep `project add-repo` in examples because older harnesses
and existing skills can still reference it.

### 8. Manage the project status board

```bash
./.openase/bin/openase status list $OPENASE_PROJECT_ID
./.openase/bin/openase status create $OPENASE_PROJECT_ID \
  --name "QA" \
  --stage started \
  --color "#FF00AA"
./.openase/bin/openase status update $STATUS_ID --name "Ready for QA"
```

Capabilities:

- `status list` calls `GET /projects/{projectId}/statuses`
- `status create` calls `POST /projects/{projectId}/statuses`
- `status update` calls `PATCH /statuses/{statusId}`
- `status delete` and `status reset` are also available as typed commands

Use these when the workflow needs status-board visibility rather than only
mutating one ticket.

### 9. Inspect workflows and harnesses

```bash
./.openase/bin/openase workflow list $OPENASE_PROJECT_ID
./.openase/bin/openase workflow harness get $WORKFLOW_ID
./.openase/bin/openase workflow harness history $WORKFLOW_ID
./.openase/bin/openase workflow harness variables
./.openase/bin/openase workflow harness validate --input /tmp/harness.json
```

Capabilities:

- Reads workflow definitions and harness versions
- Exposes the current harness text and version history
- Validates harness payloads before writes

Use this path before editing workflows, binding skills, or assuming a workflow
already grants a specific platform scope.

### 10. Inspect activity, runs, and agent output

```bash
./.openase/bin/openase activity list $OPENASE_PROJECT_ID
./.openase/bin/openase ticket run list $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase ticket run get $OPENASE_PROJECT_ID $OPENASE_TICKET_ID $RUN_ID
./.openase/bin/openase agent output $OPENASE_PROJECT_ID $AGENT_ID
```

Capabilities:

- `activity list` reads project-level business timeline events
- `ticket run list/get` inspects execution history for a ticket
- `agent output` reads streamed or recorded agent output

Use these to understand what already happened before writing new platform
state.

### 11. Inspect machines and providers

```bash
./.openase/bin/openase machine refresh-health $MACHINE_ID
./.openase/bin/openase machine resources $MACHINE_ID
./.openase/bin/openase provider list $OPENASE_ORG_ID --json providers
```

Capabilities:

- Refreshes machine health before reading resources
- Reads current machine resource snapshots
- Lists provider configuration and availability

Run `./.openase/bin/openase machine refresh-health $MACHINE_ID` before making
decisions based on machine capacity.

### 12. Manage project conversations

```bash
./.openase/bin/openase chat conversation list --project-id $OPENASE_PROJECT_ID
./.openase/bin/openase chat conversation get $OPENASE_CONVERSATION_ID
./.openase/bin/openase chat conversation entries $OPENASE_CONVERSATION_ID
./.openase/bin/openase chat conversation turn $OPENASE_CONVERSATION_ID --message "Continue the previous investigation"
./.openase/bin/openase chat conversation watch $OPENASE_CONVERSATION_ID
```

Capabilities:

- Lists project conversations
- Reads a specific conversation and its transcript
- Appends a new turn to a persistent Project AI conversation
- Watches the conversation event stream

This is especially relevant when the principal is `project_conversation`.

### 13. Inspect and refresh skills in the current project

```bash
./.openase/bin/openase skill list $OPENASE_PROJECT_ID --json skills
./.openase/bin/openase skill get $SKILL_ID
./.openase/bin/openase skill refresh $OPENASE_PROJECT_ID \
  -f workspace_root="$PWD" \
  -f adapter_type=codex-app-server
```

Capabilities:

- `skill list` resolves the current project skill catalog and skill IDs
- `skill get` returns the current stored content, bundle files, and history
- `skill refresh` re-projects enabled skills into `.codex/skills`,
  `.claude/skills`, `.gemini/skills`, or `.agent/skills` depending on adapter

This is the preferred path when comparing repo skill bundles with the current
platform copy or after updating a skill and needing the current workspace to
see the new version.

## Relationship To `ticket-workpad`

The `ticket-workpad` skill owns durable execution-log semantics. This skill
only provides the platform primitives and helper script that make that
possible.

- `openase-platform` exposes the underlying ticket comment APIs and ships
  `scripts/upsert_workpad.sh`.
- The `ticket-workpad` skill defines which comment counts as the workpad, how
  sections should be maintained, and why later agents should resume from the
  same persistent comment.
- When you need execution logs that persist across runtimes, rely on the
  separate `ticket-workpad` skill; this platform skill only provides the
  underlying comment primitives.

The helper script is projected into the runtime skill bundle and can be called
directly:

```bash
cat <<'EOF' >/tmp/workpad.md
Plan
- inspect workflow and current ticket

Progress
- reading repository and platform state

Validation
- not run yet

Notes
- none
EOF

./.codex/skills/openase-platform/scripts/upsert_workpad.sh --body-file /tmp/workpad.md
```

Equivalent helper locations can exist under `.claude/skills`,
`.gemini/skills`, or `.agent/skills` depending on the adapter type.

## Maintaining Skills Through The Platform

When you are working on the platform skill library itself, use a different
mental model from normal ticket execution:

- `skill import` is for introducing a new local skill bundle into a project.
- If a skill already exists in the project, especially a built-in skill with
  the same name, prefer `skill get` + `skill update` instead of importing
  again.
- `skill get` is the easiest way to compare repo content with the platform's
  current stored bundle.
- `skill refresh` is what makes the updated bundle appear inside the current
  workspace's projected skill directory.

### Inspect The Existing Skill Record

```bash
./.openase/bin/openase skill list $OPENASE_PROJECT_ID --json skills
./.openase/bin/openase skill get $SKILL_ID --json skill,content,files,history
```

Use `skill list` first to resolve the real UUID for a skill such as
`openase-platform`. Then use `skill get` to inspect:

- the current stored `SKILL.md`
- additional bundle files such as scripts
- version history
- whether the skill is built-in, enabled, and bound to workflows

### Update An Existing Bundle Carefully

For bundle updates, the platform expects a full valid skill bundle. Do not send
only one helper file and assume the server will merge it into the previous
bundle version.

At minimum:

- The update payload must still contain a valid `SKILL.md`.
- If the skill bundle includes helper scripts, keep them in the `files` array
  unless you intentionally want to remove them.
- The `SKILL.md` frontmatter name must still match the existing skill name.

Practical payload generation example:

```bash
python3 - <<'PY' >/tmp/openase-platform-update.json
from __future__ import annotations

import base64
import json
from pathlib import Path

root = Path("internal/builtin/skills/openase-platform")
files = []
for path in sorted(p for p in root.rglob("*") if p.is_file()):
    relative = path.relative_to(root).as_posix()
    files.append(
        {
            "path": relative,
            "content_base64": base64.b64encode(path.read_bytes()).decode(),
            "is_executable": bool(path.stat().st_mode & 0o111),
        }
    )

payload = {
    "description": "OpenASE Platform Operations",
    "files": files,
}

print(json.dumps(payload, ensure_ascii=False))
PY

./.openase/bin/openase skill update $SKILL_ID --input /tmp/openase-platform-update.json
```

After the update succeeds, refresh the projected workspace copy:

```bash
./.openase/bin/openase skill refresh $OPENASE_PROJECT_ID \
  -f workspace_root="$PWD" \
  -f adapter_type=codex-app-server
```

If the target runtime is Claude Code or Gemini instead of Codex, change the
`adapter_type` accordingly.

## Full CLI Surface Beyond The Safe Subset

If the high-frequency commands above are not enough, `openase` has a wider
typed CLI that follows the OpenAPI contract directly. Common namespaces
include:

- `openase ticket ...`
- `openase status ...`
- `openase chat ...`
- `openase project ...`
- `openase repo ...`
- `openase workflow ...`
- `openase scheduled-job ...`
- `openase machine ...`
- `openase provider ...`
- `openase agent ...`
- `openase activity ...`
- `openase channel ...`
- `openase notification-rule ...`
- `openase skill ...`
- `openase watch ...`
- `openase stream ...`

High-value examples:

```bash
./.openase/bin/openase ticket get $OPENASE_TICKET_ID
./.openase/bin/openase ticket detail $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase workflow list $OPENASE_PROJECT_ID
./.openase/bin/openase workflow harness get $WORKFLOW_ID
./.openase/bin/openase workflow harness history $WORKFLOW_ID
./.openase/bin/openase workflow harness variables
./.openase/bin/openase machine refresh-health $MACHINE_ID
./.openase/bin/openase machine resources $MACHINE_ID
./.openase/bin/openase provider list $OPENASE_ORG_ID --json providers
./.openase/bin/openase agent output $OPENASE_PROJECT_ID $AGENT_ID
./.openase/bin/openase skill list $OPENASE_PROJECT_ID
./.openase/bin/openase watch project $OPENASE_PROJECT_ID
```

These typed commands have useful properties:

- Parameters and field names come from the API contract, not hand-written
  guesses.
- Output defaults to JSON.
- `--json`, `--jq`, and `--template` can trim large responses.
- They are a better fit for "inspect first, then decide whether to write."

## Raw API Escape Hatch

If a typed command does not exist yet, use raw passthrough last:

```bash
./.openase/bin/openase api GET /api/v1/tickets/$OPENASE_TICKET_ID

./.openase/bin/openase api GET /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  --query status_name=Todo \
  --query priority=high

./.openase/bin/openase api POST /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  -f title="Follow-up" \
  -f workflow_id="550e8400-e29b-41d4-a716-446655440000"

./.openase/bin/openase api PATCH /api/v1/tickets/$OPENASE_TICKET_ID/comments/$COMMENT_ID \
  --input payload.json
```

Rules:

- `api METHOD PATH` is raw HTTP passthrough.
- `-f/--field` uses `key=value` entries to build a JSON body.
- `--query` appends query-string fields.
- `--input` sends a raw request body and cannot be mixed with `-f`.
- This is the last resort when the typed CLI does not already model the
  operation you need.

## Practical Guidance For Agents

- Start with `ticket list / get / detail`, `activity list`, or `skill get`
  before making assumptions about current state.
- Prefer the smallest write that preserves platform clarity. Update the current
  ticket or project when that is enough; create a follow-up ticket only when it
  is truly separate work.
- When mutating ticket status, prefer `--status-name` unless you already have
  the exact status UUID.
- When inspecting machine capacity, refresh health before reading resources.
- When a `403` happens, inspect capability boundaries and scopes first instead
  of trying alternate endpoints blindly.
- Do not assume a ticket identifier like `ASE-42` will be accepted where a UUID
  is required.
- In `project_conversation` sessions, favor project-scoped routes and do not
  assume current-ticket comment, update, or report-usage endpoints are
  available.
- When comparing repo skill bundles to platform bundles, inspect both sides
  explicitly and preserve non-entrypoint bundle files during updates.
