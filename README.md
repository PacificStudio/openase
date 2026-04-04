# OpenASE

OpenASE is an issue-driven automated software engineering platform. The current codebase follows the direction in [`OpenASE-PRD.md`](./OpenASE-PRD.md): an all-Go monolith, binary-first deployment, issue/workflow orchestration, and adapter-based multi-agent support.

## Current Slice

The repository has moved beyond the initial scaffold. The current vertical slice includes:

- a single Go binary with `serve`, `orchestrate`, and `all-in-one` modes
- an embedded web control plane served from the Go binary
- first-run setup that seeds `~/.openase/` and scaffolds repository-local `.openase/` assets
- organizations, projects, project repos, agent providers, agents, activity events, and project-level ticket statuses
- ticket CRUD, ticket detail, parent/child relationships, and dependency management
- workflow CRUD plus Git-backed harness documents, validation, hooks, skill binding, and built-in role templates
- per-project SSE streams for tickets, agents, hooks, and activity
- agent platform token auth plus agent-facing `openase ticket ...` and `openase project ...` CLI wrappers
- GitHub webhook ingestion for pull request and review events
- managed user-service lifecycle commands: `up`, `down`, `restart`, and `logs`
- local environment diagnostics via `openase doctor`

## Product Shape

- `All-Go monolith`: API server, orchestrator, setup flow, and embedded UI live in one repository and ship as one binary.
- `Binary-first`: release binaries ship the web UI embedded with `go:embed`, so end users do not need Node.js at runtime.
- `Issue-driven orchestration`: tickets, workflows, statuses, and activity are the core operating model.
- `Multi-agent adapters`: setup currently detects and can seed providers for Claude Code, OpenAI Codex, and Gemini CLI.
- `Git-backed behavior`: workflow harnesses and scaffolded skills live in `.openase/` inside the target repo, not hidden in a database.

## Build

Common local developer entrypoints now live in the root `Makefile`:

```bash
make hooks-install
make check
make build-web
```

For a source build that refreshes the embedded frontend before compiling the Go binary, run:

```bash
make build-web
```

The equivalent explicit commands are:

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

Frontend quality gates can be run from the repo root:

```bash
make web-format-check
make web-lint
make web-check
make web-validate
```

For a complete zero-to-run source deployment guide, see [`docs/source-build-and-run.md`](./docs/source-build-and-run.md).

## Quick Start

### 1. Prepare PostgreSQL

Copy the sample config if you want a file-based setup:

```bash
cp config.example.yaml ./config.yaml
```

The minimum required runtime setting is a PostgreSQL DSN, for example:

```bash
export OPENASE_DATABASE_DSN=postgres://openase:openase@localhost:5432/openase?sslmode=disable
```

### 2. Run setup or start the managed service

First-run setup opens a local wizard that creates the home directory layout, writes config under `~/.openase/`, migrates the database, seeds the initial org/project/provider data, and scaffolds `.openase/` inside the selected project repository.

```bash
./bin/openase setup
```

If you prefer the managed per-user service path, `up` will launch setup on first run and otherwise install/update the service definition that runs `openase all-in-one`. Use the compiled binary here so the managed service points at a stable executable path:

```bash
./bin/openase up --config ~/.openase/config.yaml
```

### 3. Run the platform

```bash
./bin/openase doctor --config ~/.openase/config.yaml
./bin/openase serve --config ~/.openase/config.yaml
./bin/openase orchestrate --config ~/.openase/config.yaml
./bin/openase all-in-one --config ~/.openase/config.yaml --tick-interval 2s
./bin/openase version
```

Managed service helpers:

```bash
./bin/openase up --config ~/.openase/config.yaml
./bin/openase logs --lines 100
./bin/openase restart
./bin/openase down
```

Useful environment overrides:

```bash
export OPENASE_SERVER_PORT=41000
export OPENASE_DATABASE_DSN=postgres://openase:openase@localhost:5432/openase?sslmode=disable
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s
export OPENASE_LOG_FORMAT=json
```

## What Setup Scaffolds

The setup flow seeds both home-directory and repo-local assets.

Under `~/.openase/`:

- `config.yaml` runtime config
- `.env` with the local auth token used by the managed service
- `logs/` and `workspaces/`

Inside the selected project repository:

- `.openase/harnesses/coding.md`
- `.openase/harnesses/roles/*.md` for built-in role templates
- `.openase/skills/*/SKILL.md` for built-in skills
- `.openase/bin/openase` wrapper that forwards to the installed binary while preserving injected agent env vars

## Control Plane and API Surface

The embedded UI is served from `/` and currently exposes onboarding, project selection, Kanban ticket management, workflow/harness editing, skill binding, built-in role creation, agent visibility, and live activity.

Representative HTTP routes:

- `GET /healthz`
- `GET /api/v1/healthz`
- `GET/POST /api/v1/orgs`, `GET/PATCH /api/v1/orgs/:orgId`, and `GET/POST /api/v1/orgs/:orgId/projects`
- `GET/PATCH/DELETE /api/v1/projects/:projectId`, plus repo, agent, activity, and repo-scope endpoints under the project
- `GET/POST /api/v1/projects/:projectId/tickets`, `GET /api/v1/projects/:projectId/tickets/:ticketId/detail`, `GET/PATCH /api/v1/tickets/:ticketId`, and dependency endpoints
- `GET/POST /api/v1/projects/:projectId/workflows`, `GET/PATCH/DELETE /api/v1/workflows/:workflowId`, plus harness read/write and validation
- `GET /api/v1/projects/:projectId/skills`, `POST /api/v1/projects/:projectId/skills/{refresh,harvest}`, and workflow skill bind/unbind endpoints
- `GET /api/v1/roles/builtin`
- `GET /api/v1/projects/:projectId/{tickets,agents,hooks,activity}/stream`
- `POST /api/v1/webhooks/:connector/:provider`
- authenticated agent platform routes under `/api/v1/platform/...`

## CLI Contract

`cmd/openase` now follows a GitHub-style dual-layer contract:

- `openase api METHOD PATH` is the raw escape hatch for any shipped HTTP route.
- shared `openase ticket ...` and `openase project ...` mutations use the platform wrapper semantics for overlapping agent-safe operations.
- the remaining resource commands stay aligned with OpenAPI and group high-frequency operations by resource.
- stream endpoints live under `openase watch ...` instead of being mixed into CRUD trees.

Representative examples:

```bash
openase api GET /api/v1/projects/$OPENASE_PROJECT_ID/tickets --query status_name=Todo
openase api PATCH /api/v1/tickets/$OPENASE_TICKET_ID --field status_id=$OPENASE_STATUS_ID
openase ticket list --status-name Todo --json tickets
openase ticket update --status_name "In Review"
openase ticket comment update $OPENASE_TICKET_ID $OPENASE_COMMENT_ID --body-file /tmp/comment.md
openase ticket detail $OPENASE_PROJECT_ID $OPENASE_TICKET_ID
openase workflow create $OPENASE_PROJECT_ID --name "Codex Worker" --description "Default coding workflow"
openase scheduled-job trigger $OPENASE_JOB_ID
openase watch tickets $OPENASE_PROJECT_ID
```

Raw and typed commands default to JSON output and support:

- `--jq '<expr>'`
- `--json field1,field2`
- `--template '{{...}}'`

## Agent Platform Compatibility

Agent workers still inherit `OPENASE_API_URL`, `OPENASE_AGENT_TOKEN`, `OPENASE_PROJECT_ID`, and `OPENASE_TICKET_ID` from the workspace wrapper. The shared `ticket` / `project` wrapper commands resolve against the agent-platform surface for the overlapping routes and accept both kebab-case and snake_case flag spellings such as `--status-name` / `--status_name` and `--body-file` / `--body_file`.

Examples:

```bash
openase ticket list --status-name Todo
openase ticket create --title "Add integration coverage" --description "Follow-up from coding ticket"
openase ticket update --description "Recorded execution notes"
openase ticket comment create --body "Logged a blocking dependency"
for helper in \
  ./.codex/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.claude/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.gemini/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.agents/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.agent/skills/openase-platform/scripts/upsert_workpad.sh
do
  if [ -x "$helper" ]; then
    "$helper" --body-file /tmp/workpad.md
    break
  fi
done
openase project update --description "Latest project context"
```

## Repository Layout

- `cmd/openase/`: CLI entrypoint
- `internal/app/`: app wiring for serve/orchestrate/all-in-one
- `internal/httpapi/`: HTTP API, SSE, webhook handlers, embedded UI hosting
- `internal/orchestrator/`: scheduling, health checks, retries
- `internal/workflow/`: workflow service, harness registry, hook execution, skill binding, validation
- `internal/agentplatform/`: agent token issuance and authentication
- `internal/setup/`: first-run setup service and wizard
- `internal/builtin/`: built-in role and skill templates
- `internal/webui/static/`: generated frontend output embedded into the binary during source builds
- `web/`: SvelteKit source for the control plane

## Validation

Focused validation commands used frequently during development:

```bash
make check
make web-validate
make web-check
go run ./cmd/openase --help
go run ./cmd/openase project --help
go run ./cmd/openase ticket --help
```

If you change the web app, rebuild `web/` before compiling or running the Go binary so the embedded assets stay in sync.
