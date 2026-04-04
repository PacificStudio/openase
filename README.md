<h1 align="center">OpenASE<br><sub>Issue-Driven Automated Software Engineering</sub></h1>

<p align="center">
  <strong>OpenASE</strong> is an all-in-one platform that turns tickets into working code — AI agents automatically pick up issues, execute workflows on your machines, and deliver results with full traceability.
</p>

<p align="center">
  <a href="#-from-zero-to-running"><img src="https://img.shields.io/badge/Quick_Start-10_min-blue?style=for-the-badge" alt="Quick Start"></a>
  <a href="docs/guide/index.md"><img src="https://img.shields.io/badge/User_Guide-Docs-ff69b4?style=for-the-badge" alt="User Guide"></a>
  <a href="#-architecture"><img src="https://img.shields.io/badge/Architecture-Monolith-blueviolet?style=for-the-badge" alt="Architecture"></a>
  <a href="#-cli-reference"><img src="https://img.shields.io/badge/CLI-Reference-orange?style=for-the-badge" alt="CLI"></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/SvelteKit-Frontend-FF3E00?logo=svelte&logoColor=white" alt="SvelteKit">
  <img src="https://img.shields.io/badge/Tailwind_CSS-Styling-06B6D4?logo=tailwindcss&logoColor=white" alt="Tailwind">
  <img src="https://img.shields.io/badge/PostgreSQL-Database-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Binary--first-go:embed-2D3748?logo=go&logoColor=white" alt="Embedded">
  <img src="https://img.shields.io/badge/SSE-Real--time-brightgreen" alt="SSE">
</p>

---

## ✨ Key Features

<table align="center" width="100%">
<tr>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>📋 Ticket-Driven Orchestration</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Core-06B6D4?style=for-the-badge&logo=target&logoColor=white" alt="Core" />
</div>

<p align="center"><strong>Kanban Board & List Views</strong></p>
<p align="center"><strong>Parent/Child & Dependency Tracking</strong></p>
<p align="center"><strong>Custom Statuses & Priorities</strong></p>
<p align="center"><strong>Repository Scope Binding</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>🤖 Multi-Agent Support</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Agents-10B981?style=for-the-badge&logo=robot&logoColor=white" alt="Agents" />
</div>

<p align="center"><strong>Claude Code / Codex / Gemini CLI</strong></p>
<p align="center"><strong>Real-time Streaming Output (SSE)</strong></p>
<p align="center"><strong>Agent Lifecycle Management</strong></p>
<p align="center"><strong>Concurrent Execution Control</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>⚡ Workflow Engine</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Automation-8B5CF6?style=for-the-badge&logo=lightning&logoColor=white" alt="Automation" />
</div>

<p align="center"><strong>Markdown Harness Documents</strong></p>
<p align="center"><strong>Skill Binding & Lifecycle Hooks</strong></p>
<p align="center"><strong>Scheduled Cron Jobs</strong></p>
<p align="center"><strong>Built-in Role Templates</strong></p>

</td>
</tr>
<tr>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>🖥️ Machine Management</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Infra-F59E0B?style=for-the-badge&logo=server&logoColor=white" alt="Infra" />
</div>

<p align="center"><strong>SSH / Local / Cloud VMs</strong></p>
<p align="center"><strong>Health Monitoring & Probes</strong></p>
<p align="center"><strong>CPU / Memory / Disk Metrics</strong></p>
<p align="center"><strong>Connectivity Diagnostics</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>🔐 Auth & Security</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Security-EC4899?style=for-the-badge&logo=shield&logoColor=white" alt="Security" />
</div>

<p align="center"><strong>OIDC Browser Login (Auth0, Entra ID)</strong></p>
<p align="center"><strong>Agent Platform Token Auth</strong></p>
<p align="center"><strong>Org & Project RBAC</strong></p>
<p align="center"><strong>GitHub Credential Management</strong></p>

</td>
<td width="33%" align="center" style="vertical-align: top; padding: 15px;">

<h3>📡 Observability</h3>

<div align="center">
  <img src="https://img.shields.io/badge/Realtime-06B6D4?style=for-the-badge&logo=eye&logoColor=white" alt="Observability" />
</div>

<p align="center"><strong>Live Activity Event Stream</strong></p>
<p align="center"><strong>Agent Run Step Tracking</strong></p>
<p align="center"><strong>GitHub Webhook Ingestion</strong></p>
<p align="center"><strong>Project Update Threads</strong></p>

</td>
</tr>
</table>

---

## 🤔 What is OpenASE?

OpenASE is a **single Go binary** that ships an API server, workflow orchestrator, and embedded web UI together. It follows an **issue-driven** model: every piece of work is a ticket, every ticket has a workflow, and AI agents automatically pick up and execute tickets based on status triggers.

```
You create a ticket  →  Orchestrator detects pickup status
    →  Agent claims the ticket  →  Executes workflow on a Machine
    →  Activity stream records every step  →  Ticket completes
```

**No Node.js at runtime** — the SvelteKit frontend is compiled and embedded into the Go binary via `go:embed`.

---

## 🚀 From Zero to Running

This section walks through everything you need on a **fresh machine** — from installing system dependencies to opening the web UI.

### Step 0: System Prerequisites

<details>
<summary><strong>Install Go 1.26+</strong></summary>

```bash
# Download (adjust version and OS/arch as needed)
wget https://go.dev/dl/go1.26.1.linux-amd64.tar.gz

# Extract to /usr/local (requires sudo)
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.26.1.linux-amd64.tar.gz

# Add to PATH — append to ~/.bashrc or ~/.zshrc
export PATH=$PATH:/usr/local/go/bin

# Verify
go version   # go1.26.1 linux/amd64
```

Alternative: if using a project-local toolchain:

```bash
export PATH=$PWD/.tooling/go/bin:$HOME/.local/go1.26.1/bin:$PATH
```

</details>

<details>
<summary><strong>Install Node.js 18+ & pnpm (build-time only)</strong></summary>

Node.js is only needed to build the frontend. It is **not required at runtime**.

```bash
# Option A: via nvm (recommended)
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash
source ~/.bashrc
nvm install 22
nvm use 22

# Option B: via package manager (Ubuntu/Debian)
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt-get install -y nodejs

# Enable corepack for pnpm
corepack enable

# Verify
node --version   # v22.x.x
pnpm --version   # 10.x.x (via corepack)
```

</details>

<details>
<summary><strong>Install PostgreSQL</strong></summary>

You have two choices — let OpenASE setup start a Docker-backed PostgreSQL automatically, or install one yourself.

**Option A: Docker (recommended for local dev)**

```bash
# Install Docker if not present
sudo apt-get update && sudo apt-get install -y docker.io
sudo usermod -aG docker $USER
newgrp docker   # or re-login

# OpenASE setup will create the container for you automatically
```

**Option B: System PostgreSQL**

```bash
# Ubuntu/Debian
sudo apt-get install -y postgresql postgresql-client

# Create database and user
sudo -u postgres psql -c "CREATE USER openase WITH PASSWORD 'openase';"
sudo -u postgres psql -c "CREATE DATABASE openase OWNER openase;"

# Verify
psql postgres://openase:openase@localhost:5432/openase?sslmode=disable -c "SELECT 1;"
```

</details>

<details>
<summary><strong>Install Git & other tools</strong></summary>

```bash
# Ubuntu/Debian
sudo apt-get install -y git make curl wget

# Verify
git --version
make --version
```

</details>

<details>
<summary><strong>(Optional) Install AI Agent CLIs</strong></summary>

OpenASE setup will auto-detect these if present on `PATH`:

| Agent | Install |
|-------|---------|
| **Claude Code** | `npm install -g @anthropic-ai/claude-code` |
| **Codex** | `npm install -g @openai/codex` |
| **Gemini CLI** | `npm install -g @anthropic-ai/gemini-cli` |

These can also be installed later — setup will seed detected providers.

</details>

### Step 1: Clone & Build

```bash
git clone https://github.com/PacificStudio/openase.git
cd openase

# Build frontend + Go binary in one command
make build-web
```

This runs the following under the hood:

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run api:generate
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

Verify the build:

```bash
./bin/openase version
```

### Step 2: Run First-Time Setup

```bash
./bin/openase setup
```

The interactive terminal setup will walk you through:

1. **Database** — start a Docker PostgreSQL automatically, or enter an existing DSN
2. **CLI detection** — checks for `git`, `claude`, `codex`, `gemini` on PATH
3. **Auth mode** — `disabled` (local dev) or `oidc` (browser login)
4. **Service mode** — config-only, or install a `systemd --user` service
5. **Seed data** — creates org, project, ticket statuses, and detected providers

Setup creates the following under `~/.openase/`:

```
~/.openase/
├── config.yaml       # Runtime configuration
├── .env              # Platform auth token
├── logs/             # Service logs
└── workspaces/       # Agent workspaces
```

> **Docker PostgreSQL note:** When choosing Docker, setup uses predictable defaults — container `openase-local-postgres`, port `127.0.0.1:15432`, database `openase`. It generates the password automatically.

### Step 3: Launch

```bash
# All-in-one: API server + orchestrator in a single process
./bin/openase all-in-one --config ~/.openase/config.yaml
```

The control plane is now available at:

```
http://127.0.0.1:19836
```

> **Tip:** Run `./bin/openase doctor --config ~/.openase/config.yaml` to diagnose any issues.

### Step 4: Verify

```bash
# Health checks
curl -fsS http://127.0.0.1:19836/healthz
curl -fsS http://127.0.0.1:19836/api/v1/healthz

# Or use the built-in doctor
./bin/openase doctor --config ~/.openase/config.yaml
```

Open `http://127.0.0.1:19836` in your browser — you should see the OpenASE control plane.

### What's Next?

Now that the platform is running, follow the [User Guide — Quick Start](docs/guide/startup.md) to:

1. Configure ticket statuses and connect a repository
2. Register a machine and an AI agent
3. Create your first workflow and ticket
4. Watch the agent execute automatically

---

## 🔧 Alternative Run Modes

### Managed User Service

Setup can install a `systemd --user` service automatically. You can also manage it manually:

```bash
./bin/openase up      --config ~/.openase/config.yaml   # Install & start
./bin/openase logs    --lines 100                        # Tail logs
./bin/openase restart                                    # Restart
./bin/openase down                                       # Stop & uninstall
```

### Split-Process Mode

Run API server and orchestrator as separate processes:

```bash
./bin/openase serve       --config ~/.openase/config.yaml
./bin/openase orchestrate --config ~/.openase/config.yaml
```

### Environment-Only Mode

If you prefer env vars over config files:

```bash
export OPENASE_DATABASE_DSN=postgres://openase:openase@localhost:5432/openase?sslmode=disable
export OPENASE_SERVER_PORT=19836
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s

./bin/openase all-in-one
```

Or source from `~/.openase/.env`:

```bash
set -a && source ~/.openase/.env && set +a
./bin/openase all-in-one
```

---

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENASE_SERVER_PORT` | `19836` | HTTP server port |
| `OPENASE_DATABASE_DSN` | — | PostgreSQL connection string (**required**) |
| `OPENASE_ORCHESTRATOR_TICK_INTERVAL` | `5s` | Orchestrator polling interval |
| `OPENASE_LOG_FORMAT` | `text` | Log format (`text` or `json`) |
| `OPENASE_LOG_LEVEL` | `info` | Log level |

### Config File Lookup Order

1. `--config <path>` flag
2. `./config.yaml` (or `.yml`, `.json`, `.toml`)
3. `~/.openase/config.yaml`
4. `OPENASE_*` environment variables + built-in defaults

### Authentication

| Mode | Description | Use Case |
|------|-------------|----------|
| `disabled` | No auth required | Local development |
| `oidc` | Browser login via OIDC provider | Production, team use |

OIDC supports standard providers: Auth0, Azure Entra ID, and any OpenID Connect compliant IdP. See [OIDC & RBAC Guide](docs/human-auth-oidc-rbac.md) for setup.

---

## 🏗️ Architecture

### Product Shape

| Principle | Description |
|-----------|-------------|
| **All-Go Monolith** | API server, orchestrator, setup flow, and embedded UI in one binary |
| **Binary-first** | Web UI embedded via `go:embed` — no Node.js at runtime |
| **Issue-driven** | Tickets, workflows, statuses, and activity are the core operating model |
| **Multi-agent** | Adapter-based support for Claude Code, Codex, and Gemini CLI |
| **Git-backed** | Workflow harnesses and skills are repo-aware at runtime |

### Repository Layout

```
openase/
├── cmd/openase/              # CLI entrypoint
├── internal/
│   ├── app/                  # App wiring (serve / orchestrate / all-in-one)
│   ├── httpapi/              # HTTP API, SSE, webhooks, embedded UI
│   ├── orchestrator/         # Scheduling, health checks, retries
│   ├── workflow/             # Workflow service, harness, hooks, skills
│   ├── agentplatform/        # Agent token auth
│   ├── setup/                # First-run setup
│   ├── builtin/              # Built-in role & skill templates
│   └── webui/static/         # Embedded frontend output
├── web/                      # SvelteKit control plane source
├── docs/
│   └── guide/                # User guide (per-module docs)
├── config.example.yaml
├── Makefile
└── go.mod
```

### System Flow

```mermaid
flowchart LR
    U[User / Scheduled Job] --> T[Ticket]
    T --> O[Orchestrator]
    O -->|status trigger| A[Agent]
    A --> W[Workflow + Harness]
    W --> S[Skills]
    A -->|executes on| M[Machine]
    M --> R[Results]
    R --> E[Activity Stream]
    E --> UI[Web Control Plane]
```

---

## 🖥️ Control Plane

The embedded web UI provides a complete project management experience:

| Module | Capabilities |
|--------|-------------|
| **[Tickets](docs/guide/tickets.md)** | Kanban board, list view, filtering, comments, dependencies, repository scoping |
| **[Agents](docs/guide/agents.md)** | Registration, real-time run monitoring, pause/resume/retire lifecycle |
| **[Machines](docs/guide/machines.md)** | SSH/local/cloud registration, health probes, resource metrics |
| **[Workflows](docs/guide/workflows.md)** | Harness editing, status binding, skill binding, version history, impact analysis |
| **[Skills](docs/guide/skills.md)** | Built-in & custom skill management, workflow binding |
| **[Scheduled Jobs](docs/guide/scheduled-jobs.md)** | Cron-based ticket creation, manual trigger, enable/disable |
| **[Activity](docs/guide/activity.md)** | Real-time event stream, type filtering, keyword search |
| **[Updates](docs/guide/updates.md)** | Team progress threads, comments, revision history |
| **[Settings](docs/guide/settings.md)** | Statuses, repositories, notifications, security, archived tickets |

---

## 💻 CLI Reference

OpenASE follows a **GitHub-style dual-layer CLI contract**:

### Resource Commands

```bash
openase ticket list       --status-name Todo --json tickets
openase ticket create     --title "Fix login bug" --description "..."
openase ticket update     --status_name "In Review"
openase ticket comment    create --body "Blocking dependency found"
openase ticket detail     $PROJECT_ID $TICKET_ID

openase workflow create   $PROJECT_ID --name "Codex Worker"
openase scheduled-job trigger $JOB_ID
openase project update    --description "Latest context"
```

### Raw API Escape Hatch

```bash
openase api GET  /api/v1/projects/$PID/tickets --query status_name=Todo
openase api PATCH /api/v1/tickets/$TID --field status_id=$SID
```

### Live Streams

```bash
openase watch tickets $PROJECT_ID
```

### Output Formatting

```bash
--jq '<expr>'              # JQ filter
--json field1,field2       # Select fields
--template '{{...}}'       # Go template
```

Both `--kebab-case` and `--snake_case` flag spellings are accepted.

---

## 🔌 Agent Platform

Agent workers inherit environment variables from the workspace wrapper:

| Variable | Purpose |
|----------|---------|
| `OPENASE_API_URL` | Platform API endpoint |
| `OPENASE_AGENT_TOKEN` | Agent authentication token |
| `OPENASE_PROJECT_ID` | Current project context |
| `OPENASE_TICKET_ID` | Current ticket context |

---

## 🛠️ Development

### Build Commands

```bash
make hooks-install        # Set up git hooks (lefthook)
make check                # Run formatting + backend coverage checks
make build-web            # Build frontend + Go binary
make build                # Build Go binary only (uses existing frontend)
make run                  # Run API server in dev mode
make doctor               # Run local environment diagnostics
```

### Frontend Quality Gates

```bash
make web-format-check     # Prettier formatting
make web-lint             # ESLint checks
make web-check            # Svelte type checking
make web-validate         # All of the above
```

### OpenAPI Contract

```bash
make openapi-generate     # Regenerate api/openapi.json + TS types
make openapi-check        # Verify committed artifacts are up-to-date
```

### Testing

```bash
make test                        # Go test suite
make test-backend-coverage       # Full backend tests + coverage gate
make lint                        # Lint changes since origin/main
make lint-all                    # Full lint suite
```

---

## 📖 Documentation

| Document | Description |
|----------|-------------|
| **[User Guide](docs/guide/index.md)** | Complete guide to every UI module |
| [Getting Started](docs/guide/startup.md) | 5-step quick start for new users |
| [Module Architecture](docs/guide/architecture.md) | How modules work together |
| [FAQ](docs/guide/faq.md) | Common questions & troubleshooting |
| **[Source Build & Run](docs/source-build-and-run.md)** | Detailed build & deployment guide |
| [OIDC & RBAC](docs/human-auth-oidc-rbac.md) | Authentication & authorization setup |
| [Observability](docs/observability-checklist.md) | Monitoring checklist |
| [WebSocket Rollout](docs/remote-websocket-rollout.md) | WebSocket transport guide |
| [Gemini CLI Adaptation](docs/gemini-cli-adaptation-guide.md) | Gemini CLI integration |
| [Stream Protocol](docs/claude-code-stream-protocol.md) | Claude Code streaming protocol |

---

## 📄 License

See [LICENSE](LICENSE).

---

<p align="center">
  <strong>OpenASE</strong>
  <br>
  <em>Create the ticket. The agent does the rest.</em>
</p>
