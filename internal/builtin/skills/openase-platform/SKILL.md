---
name: "openase-platform"
description: "Platform operations for tickets, projects, and runtime coordination inside OpenASE."
---

# OpenASE Platform Operations

Prefer the wrapper injected into the workspace:

```bash
./.openase/bin/openase ticket list --status-name Todo
```

This wrapper is the `openase` binary with the current workspace's OpenASE platform context already attached. Use it first. Do not hand-roll URLs, guess endpoints, bypass the platform by writing the database directly, or fake platform state.

## What OpenASE Controls

OpenASE is not a small helper that just runs commands. It is an issue-driven automated software engineering platform. Per the PRD, it is responsible for at least:

- Project control: descriptions, statuses, repos, workflows, skills, and scheduled jobs
- Ticket control: lifecycle, status transitions, comment primitives, usage/cost, and external links
- Execution control: agents, providers, machines, runtimes, and orchestration loops
- Controlled autonomy: agents can operate the platform within granted scope and close the loop from claim to execution to platform writeback to follow-up tickets
- Auditability: every platform write goes through the API / ActivityEvent / timeline and stays attributable

For agents, the core purpose of `openase` is not just inspection. It is reading and writing real control-plane state inside platform-enforced boundaries. The tickets you create, project descriptions you update, repos you register, and comments you append all affect later scheduling, UI state, audit trails, and other agents' context.

## Mental Model For Agents

Treat `openase` as the control-plane API for the current engineering project.

- The code repository is only a workspace, not the task system.
- Tickets, projects, workflows, skills, and machines in OpenASE are the real control-plane entities.
- When you need to change platform state, prefer the `openase` CLI. Do not try to express platform state indirectly by editing local files.
- Read before you write: inspect the current state first, then make the smallest necessary change.
- If you exceed scope, the platform returns `403`. That usually means the current harness did not grant the required `platform_access`.

## Execution Model

The runtime injects a capability contract that tells you which principal kind, scopes, and environment variables are actually available in this session. Treat that runtime contract as the source of truth.

Common environment variables include:

- `OPENASE_API_URL`: OpenASE API base URL
- `OPENASE_AGENT_TOKEN`: current agent token
- `OPENASE_PROJECT_ID`: current project UUID
- `OPENASE_TICKET_ID`: current ticket UUID; only present in ticket runtime or ticket-focused Project AI
- `OPENASE_CONVERSATION_ID`: current project conversation UUID; available in Project AI conversations
- `OPENASE_PRINCIPAL_KIND`: current principal kind, such as `ticket_agent` or `project_conversation`
- `OPENASE_AGENT_SCOPES`: current token scopes, comma-separated

Common platform subcommands auto-fill context in this order:

- project scope: `--project-id` -> `OPENASE_PROJECT_ID`
- ticket scope: positional `[ticket-id]` -> `--ticket-id` -> `OPENASE_TICKET_ID`
- API URL: `--api-url` -> `OPENASE_API_URL`
- token: `--token` -> `OPENASE_AGENT_TOKEN`

Important limits:

- Most ID parameters require UUIDs and do not accept human-readable ticket identifiers such as `ASE-42`.
- Output defaults to JSON and can be filtered with `--json`, `--jq`, or `--template`.
- When platform calls fail, the CLI prints the HTTP method, path, status, and API error code directly, so you do not need to guess.
- Tokens are short-lived and scope-bound; not every workflow can modify projects, repos, or scheduled jobs.
- Shared wrapper flags accept both kebab-case and snake_case, such as `--status-name` / `--status_name` and `--body-file` / `--body_file`.

## Top-Level Commands

Below is the current top-level `openase` command surface from source. Not every command is appropriate for agents; the first groups are the ones you will use most often.

### Agent / API Surface

- `api`: raw HTTP passthrough, the fallback entrypoint for any exposed API
- `ticket`: shared platform wrapper for common ticket reads and writes; non-overlapping detail/run/dependency/external-link subcommands still go directly through OpenAPI
- `status`: ticket status board management
- `chat`: ephemeral chat and project conversations
- `project`: shared platform wrapper for update/add-repo; list/get/create/delete still go directly through OpenAPI
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

`serve` / `orchestrate` / `up` / `down` / `restart` / `issue-agent-token` are more about platform operations or control-plane startup and are usually not the first choice for normal ticket execution.

## Safe Default Commands

This is the safest first layer for agents to use. The semantics are stable and suitable for direct workflow / harness calls.

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
- You need to attach follow-up security, testing, or deployment work back to the platform explicitly

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

`openase-platform` only provides the comment primitives here. It does not define workpad semantics directly. When you need persistent workpad maintenance, use the separately bound `ticket-workpad` skill, which builds on top of these `comment list/create/update` primitives.

### 6. Update the project description

```bash
./.openase/bin/openase project update --description "Update the latest project context"
```

Capabilities:

- Calls `PATCH /projects/{projectId}`
- This is the main high-frequency project write operation today

Good fits:

- Product or research roles need to write findings back to the project
- The current ticket uncovers longer-term context that should live in the project description

### 7. Manage the project status board

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
- `status delete` / `status reset` also have typed CLI coverage

Good fits:

- You need to inspect the current status board before moving a ticket
- A project administrator needs to adjust status names, ordering, or the default status

### 8. Register a repo on the project

```bash
./.openase/bin/openase repo create $OPENASE_PROJECT_ID \
  --name "worker-tools" \
  --url "https://github.com/acme/worker-tools.git" \
  --default-branch main \
  --label go \
  --label backend
```

Capabilities:

- Calls `POST /projects/{projectId}/repos`
- `--name` and `--url` are required
- `--default-branch` defaults to `main`
- `--label` is repeatable

Compatibility note:

```bash
./.openase/bin/openase project add-repo \
  --name "worker-tools" \
  --url "https://github.com/acme/worker-tools.git" \
  --default-branch main
```

The older `project add-repo` entrypoint still works, but the newer typed CLI and future docs prefer `repo create` because it expresses more clearly that a repo is an independent control-plane entity under the project.

### 9. Manage project conversations / Project AI

```bash
./.openase/bin/openase chat conversation list --project-id $OPENASE_PROJECT_ID
./.openase/bin/openase chat conversation get $OPENASE_CONVERSATION_ID
./.openase/bin/openase chat conversation entries $OPENASE_CONVERSATION_ID
./.openase/bin/openase chat conversation turn $OPENASE_CONVERSATION_ID --message "Continue with the previous issue"
./.openase/bin/openase chat conversation watch $OPENASE_CONVERSATION_ID
```

Capabilities:

- `chat conversation list` calls `GET /chat/conversations`
- `chat conversation turn` calls `POST /chat/conversations/{conversationId}/turns`
- `chat conversation watch` opens the conversation event stream
- Supports interrupt handling, action proposal execution, and runtime close

Good fits:

- Inspect or resume conversations under the Project AI / project conversation principal
- Read a conversation transcript or workspace diff as control-plane state

### 10. Read the project activity timeline

```bash
./.openase/bin/openase activity list $OPENASE_PROJECT_ID
./.openase/bin/openase activity list $OPENASE_PROJECT_ID --json events
```

Capabilities:

- Calls `GET /projects/{projectId}/activity`
- Reads the project-level business timeline, including ticket changes, workflow changes, runtime lifecycle, and other audit events

Good fits:

- Quickly confirm who recently changed a workflow, status, or project description
- Fill in project context beyond the current ticket
- Confirm whether recent runtime or orchestration business events have been recorded

### 11. Inspect ticket runs / retry / external links

```bash
./.openase/bin/openase ticket run list $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase ticket run get $OPENASE_PROJECT_ID $OPENASE_TICKET_ID $RUN_ID
./.openase/bin/openase ticket retry-resume $OPENASE_TICKET_ID
./.openase/bin/openase ticket external-link add $OPENASE_TICKET_ID \
  --title "PR 482" \
  --url "https://github.com/acme/repo/pull/482"
```

Capabilities:

- `ticket run list` calls `GET /projects/{projectId}/tickets/{ticketId}/runs`
- `ticket run get` calls `GET /projects/{projectId}/tickets/{ticketId}/runs/{runId}`
- `ticket retry-resume` calls `POST /tickets/{ticketId}/retry/resume`
- `ticket external-link add/delete` manage external links on a ticket

Good fits:

- Inspect a ticket's recent execution history, failure reasons, or retry chain
- Explicitly resume execution after a retryable failure
- Attach PRs, issues, docs, incidents, or other external objects back to the ticket

### 12. Inspect workflow harness history / variables / validate

```bash
./.openase/bin/openase workflow harness history $WORKFLOW_ID
./.openase/bin/openase workflow harness variables
./.openase/bin/openase workflow harness validate --input /tmp/harness.json
```

Capabilities:

- `workflow harness history` calls `GET /workflows/{workflowId}/harness/history`
- `workflow harness variables` calls `GET /harness/variables`
- `workflow harness validate` calls `POST /harness/validate`

Good fits:

- Inspect the most recent harness edits before deciding whether to continue changing it
- Check which variables the template may reference
- Run semantic validation before submitting a real harness update

## Full CLI Surface Beyond The Safe Subset

If the high-frequency commands above are not enough, `openase` has a broader typed CLI that follows the OpenAPI contract directly, so you do not need to read source and assemble raw HTTP by hand.

Common namespaces include:

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
./.openase/bin/openase status list $OPENASE_PROJECT_ID
./.openase/bin/openase chat conversation get $OPENASE_CONVERSATION_ID
./.openase/bin/openase repo list $OPENASE_PROJECT_ID
./.openase/bin/openase workflow list $OPENASE_PROJECT_ID
./.openase/bin/openase workflow harness get $WORKFLOW_ID
./.openase/bin/openase workflow harness history $WORKFLOW_ID
./.openase/bin/openase workflow harness variables
./.openase/bin/openase activity list $OPENASE_PROJECT_ID
./.openase/bin/openase ticket run list $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
./.openase/bin/openase ticket retry-resume $OPENASE_TICKET_ID
./.openase/bin/openase scheduled-job list $OPENASE_PROJECT_ID
./.openase/bin/openase machine refresh-health $MACHINE_ID
./.openase/bin/openase machine resources $MACHINE_ID
./.openase/bin/openase provider list $OPENASE_ORG_ID --json providers
./.openase/bin/openase agent output $OPENASE_PROJECT_ID $AGENT_ID
./.openase/bin/openase channel list $OPENASE_ORG_ID
./.openase/bin/openase notification-rule list $OPENASE_PROJECT_ID
./.openase/bin/openase skill list $OPENASE_PROJECT_ID
./.openase/bin/openase watch project $OPENASE_PROJECT_ID
```

These additional typed commands have a few useful properties:

- Parameters and fields come from the OpenAPI contract, not handwritten guesses
- Output defaults to JSON
- You can trim results with `--json`, `--jq`, or `--template`
- They work well for an inspect-first, write-later flow
- A few body fields share names with CLI output flags; for example, the `template` field in `notification-rule create/update` should use `-f template=...` or `--input payload.json` instead

## Raw API Escape Hatch

If a typed command still does not cover what you need, fall back to raw passthrough last:

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

- `api METHOD PATH` is raw HTTP passthrough
- `-f/--field` builds a JSON body from `key=value` pairs
- `--query` appends query-string parameters
- `--input` sends a raw request body; it cannot be mixed with `-f`
- This is the final fallback when no dedicated subcommand exists, not the first choice

## Platform Boundaries And Safety

When using this skill, follow these default boundaries:

- Only operate inside the current project context; do not assume cross-project write access
- Prefer maintaining platform state on the current ticket, such as comments and descriptions, instead of leaving fragmented state everywhere
- Splitting a follow-up ticket is fine, but do not create an endless stream of extra tickets just to look proactive
- Before changing a project, repo, scheduled job, or workflow, confirm that the current role really needs it and that the token scope allows it
- When you hit `403`, check the capability boundary first instead of trying other commands to bypass the platform

## Practical Guidance For Agents

- Read context with `ticket list / get / detail` before deciding whether to write
- When you need execution logs that persist across runtimes, rely on the separate `ticket-workpad` skill; this platform skill only provides the underlying comment primitives
- Prefer `--status-name` when changing ticket status unless you already have the exact status UUID
- When you need the latest machine resources, run `machine refresh-health` first, then inspect `machine resources`
- When you need broader capability, look for a typed command first and fall back to `openase api` only if coverage is missing
- Do not assume the platform accepts a ticket identifier; most commands require UUIDs
