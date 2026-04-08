<p align="center">
  <img src="docs/imgs/openase.svg" alt="OpenASE" width="80" height="80" />
</p>

<h1 align="center">OpenASE<br><sub>Ticket-Driven Automated Software Engineering</sub></h1>

<p align="center">
  <strong>OpenASE</strong> is an all-in-one platform that turns tickets into working code — AI agents automatically pick up tickets, execute workflows on your machines, and deliver results with full traceability.
</p>

<p align="center">
  <a href="#-from-zero-to-running"><img src="https://img.shields.io/badge/Quick_Start-10_min-blue?style=for-the-badge" alt="Quick Start"></a>
  <a href="docs/guide/en/index.md"><img src="https://img.shields.io/badge/Guide-English-ff69b4?style=for-the-badge" alt="English Guide"></a>
  <a href="docs/guide/zh/index.md"><img src="https://img.shields.io/badge/指南-中文-ff69b4?style=for-the-badge" alt="中文指南"></a>
  <a href="#-architecture"><img src="https://img.shields.io/badge/Architecture-Monolith-blueviolet?style=for-the-badge" alt="Architecture"></a>
  <a href="#-cli-reference"><img src="https://img.shields.io/badge/CLI-Reference-orange?style=for-the-badge" alt="CLI"></a>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Linux-supported-success?logo=linux&logoColor=white" alt="Linux">
  <img src="https://img.shields.io/badge/macOS-supported-success?logo=apple&logoColor=white" alt="macOS">
  <img src="https://img.shields.io/badge/Windows-untested-lightgrey?logo=windows&logoColor=white" alt="Windows">
  <img src="https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/SvelteKit-Frontend-FF3E00?logo=svelte&logoColor=white" alt="SvelteKit">
  <img src="https://img.shields.io/badge/Tailwind_CSS-Styling-06B6D4?logo=tailwindcss&logoColor=white" alt="Tailwind">
  <img src="https://img.shields.io/badge/PostgreSQL-Database-4169E1?logo=postgresql&logoColor=white" alt="PostgreSQL">
  <img src="https://img.shields.io/badge/Binary--first-go:embed-2D3748?logo=go&logoColor=white" alt="Embedded">
  <img src="https://img.shields.io/badge/SSE-Real--time-brightgreen" alt="SSE">
</p>

---

## 🖼️ Product Screenshots

<p align="center">
  The embedded web UI covers ticket orchestration, AI-assisted planning, skill authoring, and live project tracking.
</p>

<p align="center">
  <img src="docs/imgs/active-project-board.png" alt="OpenASE active project board" width="100%" />
</p>
<p align="center">
  <strong>Live execution</strong><br>
  Monitor real project work as tickets move across backlog, todo, in-progress, and review.
</p>

<table align="center" width="100%">
<tr>
<td width="50%" align="center" style="vertical-align: top; padding: 12px;">
  <img src="docs/imgs/kanban-board.png" alt="OpenASE kanban board" width="100%" />
  <p><strong>Ticket board</strong><br>Manage backlog and execution flow with a kanban-style ticket view.</p>
</td>
<td width="50%" align="center" style="vertical-align: top; padding: 12px;">
  <img src="docs/imgs/kanban-board-with-ai-assistant.png" alt="OpenASE project AI assistant beside the ticket board" width="100%" />
  <p><strong>Project AI</strong><br>Break work into tickets and inspect workspace context directly beside the board.</p>
</td>
</tr>
<tr>
<td colspan="2" align="center" style="vertical-align: top; padding: 12px;">
  <img src="docs/imgs/skill-editor-install-codex.png" alt="OpenASE skill editor" width="100%" />
  <p><strong>Skill editor</strong><br>Edit built-in or custom skills that drive repeatable automation workflows.</p>
</td>
</tr>
</table>

---

## 🧭 Why OpenASE Exists

AI coding agents are powerful — but only when humans stay in control. The real question is **how** that control works. OpenASE is built around two complementary modes of human–agent interaction: **synchronous** and **asynchronous**.

### Asynchronous AI — Ticket Agents

When the requirements are clear, the acceptance criteria are well-defined, and a **Harness** (a hard-boundary document that constrains agent behavior) is in place, the **Ticket Agent** executes the entire task autonomously. It follows the Workflow's instructions, updates ticket statuses, and completes the work described in each ticket — humans don't need to babysit.

The Workflow defines the status flow and tells the agent how to transition tickets through each stage. It is a **soft control surface**: you write natural-language instructions in the Workflow to guide how the agent should behave at each status. Here are two common patterns:

- **Fullstack Coder** — A single agent handles the entire lifecycle. Set up a simple status flow like `Todo → In Progress → In Review → Merging → Done`. The agent picks up a ticket, writes the code, opens a PR, and moves it forward. Human review at the `In Review` stage is optional — you can keep it for quality gates or let another agent handle the review.

- **Mixed Relay** — Multiple specialized agents collaborate in a relay. Set up a richer flow like `Design → Backend → Frontend → Testing → In Review → Merging → Done`, and assign different agent roles (product manager, backend engineer, frontend engineer, tester) to each stage. Agents hand off work to the next role as the ticket progresses. Human review checkpoints are optional at any stage — you can insert them where you want oversight, or remove them entirely for full autonomy.

> Both async and sync AI support multiple agent CLIs. **Claude Code** and **Codex** are recommended for production use; **Gemini CLI** is supported but less stable.

### Synchronous AI — Project AI

When your requirements are still vague, the problem space needs exploration, or you're not ready to commit to a formal ticket — start a conversation with **Project AI**.

Project AI is a synchronous, interactive assistant that lives in the sidebar of the control plane. Use it to analyze requirements, explore technical approaches, draft PRDs and documentation, scaffold repositories, and prepare everything before kicking off async execution. Each conversation tab runs in an **independent workspace** — tabs are isolated, can run in parallel, and are easy to manage and review side by side.

**What Project AI can do:**

- Read ticket details, dependencies, activity history, and run status
- Read Workflow/Harness configuration, Skill code, and machine health
- Inspect git workspace diffs (branch, file changes, repo status)
- Create and update tickets, publish project updates
- Modify Workflows/Harnesses and Skills directly
- Trigger agent execution on tickets
- Operate git (commit, branch, push)
- Control agent runtimes (pause, resume, interrupt)

**Multi-agent, isolated workspaces — in one panel:** Each Project AI tab spawns its own isolated workspace. You can run multiple agents side by side in the same panel — each working on a different task, in a different branch, with zero interference. No more juggling terminals or switching between IDEs. One unified view to launch, monitor, and interact with all your agents simultaneously.

**Context-aware focus:** As you navigate the control plane, Project AI automatically switches between 4 focus modes — **Workflow**, **Skill**, **Ticket**, and **Machine** — injecting the relevant context into the conversation so it always understands what you're looking at.

**Best practice:** Start with a fuzzy idea, discuss it with Project AI to converge on the best approach, let it draft documents and initialize the repo. Then set up an automated Fullstack or Mixed Relay Workflow, configure ticket dependencies to control parallelism and blocking, and let the system run. With well-structured Harnesses and dependency graphs, the platform can sustain long periods of high-throughput autonomous work with zero human intervention.

### Skills — Extending What Agents Can Do

Skills are reusable instruction documents that give agents extra capabilities beyond raw coding. Every Workflow automatically binds a **built-in Ticket Skill** that teaches the Ticket Agent how to update ticket statuses on the platform — this is how agents drive the status flow without hardcoded logic.

Beyond the built-in skills, you can:

- **Bind more built-in skills** to any Workflow for common operations (e.g., git conventions, PR creation, code review checklists).
- **Create custom skills** from the Skill Editor to encode project-specific knowledge and procedures.
- **Import skills from your repository** — ask Project AI to pull skill files from your repo and register them in OpenASE.

When a Workflow runs, its bound skills are **injected into the CLI agent's skill directory** at runtime (e.g., `.codex/skills/`, `.claude/skills/`, `.gemini/skills/`), so the agent picks them up natively. Project AI has access to **all skills** regardless of Workflow binding, making it the ideal place to author, edit, and debug skills before deploying them.

**Best practice:** Use Project AI to create, modify, or debug your skills interactively. Once a skill works as expected, save it to OpenASE and bind it to the relevant Workflows — every Ticket Agent that runs under those Workflows will inherit the skill automatically.

### The Interplay

In practice, these two modes are constantly interleaved:

- **Requirements change** → existing Harnesses go stale and need synchronous refinement.
- **Projects iterate** → new Harnesses must be authored to prevent agents from drifting off course.
- **Tech debt accumulates** → scheduled jobs trigger agents to clean it up automatically on a cron.

This interplay of sync and async work deserves a **unified home** — not scattered across terminals, IDEs, and chat windows. OpenASE brings it all together: **multi-agent CLI support** (Claude Code, Codex, Gemini CLI), **multi-machine scheduling**, and a single control plane so you never have to context-switch between tools.

### Organization & Project Management

OpenASE supports **multi-Organization** management out of the box. Each Org contains its own set of Projects, and each Project has independent tickets, workflows, skills, machines, and agent configurations. Cross-Org **team collaboration** (shared projects, role-based permissions across Orgs) is currently **WIP**.

### ⚠️ Security Notice

To maximize unattended Workflow execution efficiency, OpenASE launches CLI agents with **permissive flags by default** (e.g., `--dangerously-skip-permissions` for Claude Code, `--yolo` for Codex). This means agents can read, write, and execute arbitrary commands on the host machine without per-action confirmation. You can change this to standard (interactive) mode per-provider in the **Provider settings** page.

**Be aware of the risks:**
- Only run OpenASE on machines where you trust the agent's scope of access.
- Restrict OS-level permissions (user accounts, filesystem boundaries) as appropriate.
- **This project is NOT designed for public-facing deployment.** It is intended for local development, private networks, and trusted environments.
- By default, OpenASE runs in **Demo Mode** (auth disabled) — no login required, anyone on the network can access the control plane. This is convenient for quick local evaluation, but **for LAN or multi-user deployments, always configure HTTPS + OIDC authentication** to protect your data and agent access. See the [OIDC & RBAC Guide](docs/guide/en/settings.md) for setup instructions.

### The Vision

OpenASE aims to be a **full-lifecycle software engineering platform**: end-to-end iteration from ticket to deployed code, team collaboration across roles, and multi-repository orchestration — all driven by the ticket-first, agent-native model described above.

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

<p align="center"><strong>Local / Direct-Connect / Reverse-Connect Machines</strong></p>
<p align="center"><strong>Websocket Execution + SSH Helper Compatibility</strong></p>
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

OpenASE is a **single Go binary** that ships an API server, workflow orchestrator, and embedded web UI together. It follows a **ticket-driven** model: every piece of work is a ticket, every ticket has a workflow, and AI agents automatically pick up and execute tickets based on status triggers.

```
You create a ticket  →  Orchestrator detects pickup status
    →  Agent claims the ticket  →  Executes workflow on a Machine
    →  Activity stream records every step  →  Ticket completes
```

**No Node.js at runtime** — the SvelteKit frontend is compiled and embedded into the Go binary via `go:embed`.

---

## 📊 Status & Roadmap

### Feature Completion

| Module | Status | Notes |
|--------|--------|-------|
| **Tickets** | ✅ Stable | CRUD, Kanban/list views, comments, dependencies, parent/child, archiving |
| **Agents** | ✅ Stable | Registration, run monitoring, streaming output, lifecycle management |
| **Workflows** | ✅ Stable | Harness editing, status/skill binding, hooks, version history, impact analysis |
| **Skills** | ✅ Stable | Built-in & custom skills, workflow binding, enable/disable |
| **Activity** | ✅ Stable | Real-time SSE event stream, filtering, search |
| **Updates** | ✅ Stable | Threads, comments, revision history |
| **Settings** | ✅ Stable | Statuses, repositories, notifications, security, archived tickets |
| **Scheduled Jobs** | ✅ Stable | Cron-based ticket creation, manual trigger, enable/disable |
| **Machines (Local)** | ✅ Stable | Local machine registration, health probes, resource metrics |
| **CLI** | ✅ Stable | Dual-layer contract, resource commands, raw API, live streams |
| **Setup** | ✅ Stable | Interactive terminal setup, Docker PostgreSQL, managed user service (`systemd --user` on Linux, `launchd` on macOS) |
| **Machines (Remote)** | ✅ Stable | Remote Runtime v1 uses websocket-only execution for direct-connect listeners and reverse-connect daemons; SSH is helper-only for bootstrap and diagnostics |
| **OIDC Auth** | 🚧 WIP | Browser login, session management, RBAC |

### Roadmap

| Priority | Item | Description |
|----------|------|-------------|
| 🟡 Medium | **Remote Runtime Operations** | Expand rollout automation, dashboards, and operator tooling around the websocket-only remote runtime plane |
| 🟡 Medium | **Windows Support** | Native service management and shell-script support outside WSL2 |
| 🟡 Medium | **Notification Channels** | Slack, email, and webhook notification delivery |
| 🟡 Medium | **iOS & Android App** | Mobile control plane for monitoring and managing projects on the go |
| 🟡 Medium | **Desktop All-in-One App** | Standalone desktop application bundling the full OpenASE experience |
| 🟡 Medium | **Kubernetes Runtime** | Run agent workloads on Kubernetes clusters for elastic scaling |
| 🟢 Future | **Multi-org Collaboration** | Cross-organization project sharing and permissions |
| 🟢 Future | **Plugin Ecosystem** | Third-party plugin support for custom tools and integrations |
| 🟢 Future | **Metrics Dashboard** | Agent performance metrics, ticket throughput analytics |

---

## 🚀 From Zero to Running

This section walks through everything you need on a **fresh machine** — from installing system dependencies to opening the web UI.

### Platform Support

| Platform | Status | Notes |
|----------|--------|-------|
| **Linux** (x86_64, arm64) | ✅ Fully supported | Primary development and deployment platform |
| **macOS** (Apple Silicon, Intel) | ✅ Supported | `setup`, `up/down/restart/logs`, and the managed user service use `launchd` with `~/Library/LaunchAgents/com.openase.plist` |
| **Windows** | ⚠️ Untested | Native service management and shell scripts have not been validated. WSL2 is recommended as a workaround |

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
<summary><strong>Install Node.js 22 LTS or 24 LTS & pnpm (build-time only)</strong></summary>

Node.js is only needed to build the frontend. It is **not required at runtime**.

Use an LTS release: **Node 22 LTS** is the default recommendation, and **Node 24 LTS** is also expected to work. Avoid odd-numbered non-LTS releases such as **Node 23**. The current frontend dependency set includes `engines` constraints that can cause `pnpm` to reject versions like `v23.11.1` during install or build.

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

There is no third setup-managed "user-space local database" path. If your user cannot access Docker, you must prepare PostgreSQL yourself before running `openase setup`.

**Option A: Docker (recommended for local dev)**

```bash
# Install Docker if not present
sudo apt-get update && sudo apt-get install -y docker.io
sudo usermod -aG docker $USER
newgrp docker   # or re-login

# OpenASE setup will create the container for you automatically
```

**Option B: Bring Your Own PostgreSQL**

For macOS local development, prefer a native PostgreSQL install managed by Homebrew or Postgres.app if you do not want Docker to own the database lifecycle.

```bash
# macOS via Homebrew
brew install postgresql@16
brew services start postgresql@16

# Verify a DSN that setup can reuse
psql postgres://localhost:5432/postgres?sslmode=disable -c "SELECT 1;"
```

On Linux, a distro package install is the straightforward equivalent:

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
| **Gemini CLI** | `npm install -g @google/gemini-cli` |

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
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

`make build-web` rebuilds the embedded frontend and then compiles the Go binary. It does **not** refresh the committed OpenAPI artifacts. If you changed backend API shapes or want to refresh `api/openapi.json` and `web/src/lib/api/generated/openapi.d.ts`, run `make openapi-generate` separately first.

Verify the build:

```bash
./bin/openase version
```

### Step 2: Run First-Time Setup

```bash
./bin/openase setup
```

The interactive terminal setup will walk you through:

1. **Database** — start a Docker PostgreSQL automatically, or enter an existing PostgreSQL connection (`host`, `port`, `database`, `user`, `password`, `sslmode`)
2. **CLI detection** — checks for `git`, `claude`, `codex`, `gemini` on PATH
3. **Auth mode** — `disabled` (local dev) or `oidc` (browser login)
4. **Service mode** — config-only, or install the managed user service (`systemd --user` on Linux, `launchd` on macOS)
5. **Seed data** — creates org, project, ticket statuses, and detected providers

Setup creates the following under `~/.openase/`:

```
~/.openase/
├── config.yaml       # Runtime configuration
├── .env              # Platform auth token
├── logs/             # Service logs
└── workspaces/       # Agent workspaces
```

> **Docker PostgreSQL note:** When choosing Docker, setup uses predictable defaults — container `openase-local-postgres`, port `127.0.0.1:15432`, database `openase`. It generates the password automatically. If your account does not have Docker access, setup does not offer another local-database fallback; prepare PostgreSQL first and then choose the manual connection path.

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

Now that the platform is running, follow the User Guide — Quick Start ([EN](docs/guide/en/startup.md) | [中文](docs/guide/zh/startup.md)) to:

1. Configure ticket statuses and connect a repository
2. Register a machine and an AI agent
3. Create your first workflow and ticket
4. Watch the agent execute automatically

---

## 🔧 Alternative Run Modes

### Managed User Service

Setup can install the managed user service automatically. You can also manage it manually:

```bash
./bin/openase up      --config ~/.openase/config.yaml   # Install & start
./bin/openase logs    --lines 100                        # Tail logs
./bin/openase restart                                    # Restart
./bin/openase down                                       # Stop
```

The managed service only runs OpenASE itself (`openase all-in-one --config ...`). It does not manage PostgreSQL for you. If you pointed OpenASE at an existing PostgreSQL instance, keep that database running separately. If setup created a Docker PostgreSQL container, that container is still a separate service boundary from OpenASE.

| Platform | Manager | Service definition | Inspect | Restart | Stop | Logs |
|----------|---------|--------------------|---------|---------|------|------|
| Linux | `systemd --user` | `~/.config/systemd/user/openase.service` | `systemctl --user status openase` | `systemctl --user restart openase` | `systemctl --user stop openase` | `journalctl --user -u openase -n 200 -f` |
| macOS | `launchd` | `~/Library/LaunchAgents/com.openase.plist` | `launchctl print gui/$(id -u)/com.openase \|\| launchctl print user/$(id -u)/com.openase` | `launchctl kickstart -k <target>` | `launchctl bootout <target>` | `tail -n 200 -f ~/.openase/logs/openase.stdout.log ~/.openase/logs/openase.stderr.log` |

On macOS, setup prefers the `gui/<uid>/com.openase` target and falls back to `user/<uid>/com.openase` when the login session is attached to that domain instead. Use whichever target `launchctl print` resolves on your machine for `kickstart` and `bootout`.

For long-running Linux server use, `systemd --user` may also need lingering enabled so the user service survives logout and can start as expected after reboot:

```bash
loginctl enable-linger "$USER"
```

Run that once per user account on machines where you expect OpenASE to keep running without an active login session.

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
| `disabled` | No login required; OpenASE uses a stable local admin principal for personal mode | Local development, personal deployments |
| `oidc` | Browser login via OIDC provider | Production, team use |

OIDC supports standard providers: Auth0, Azure Entra ID, and any OpenID
Connect compliant IdP. See the IAM Dual-Mode Contract
([EN](docs/en/iam-dual-mode-contract.md) | [中文](docs/zh/iam-dual-mode-contract.md))
for the product model, and the OIDC & RBAC Guide
([EN](docs/en/human-auth-oidc-rbac.md) | [中文](docs/zh/human-auth-oidc-rbac.md))
for setup.

---

## 🏗️ Architecture

### Product Shape

| Principle | Description |
|-----------|-------------|
| **All-Go Monolith** | API server, orchestrator, setup flow, and embedded UI in one binary |
| **Binary-first** | Web UI embedded via `go:embed` — no Node.js at runtime |
| **Ticket-driven** | Tickets, workflows, statuses, and activity are the core operating model |
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
| **[Tickets](docs/guide/en/tickets.md)** | Kanban board, list view, filtering, comments, dependencies, repository scoping |
| **[Agents](docs/guide/en/agents.md)** | Registration, real-time run monitoring, pause/resume/retire lifecycle |
| **[Machines](docs/guide/en/machines.md)** | SSH/local/cloud registration, health probes, resource metrics |
| **[Workflows](docs/guide/en/workflows.md)** | Harness editing, status binding, skill binding, version history, impact analysis |
| **[Skills](docs/guide/en/skills.md)** | Built-in & custom skill management, workflow binding |
| **[Scheduled Jobs](docs/guide/en/scheduled-jobs.md)** | Cron-based ticket creation, manual trigger, enable/disable |
| **[Activity](docs/guide/en/activity.md)** | Real-time event stream, type filtering, keyword search |
| **[Updates](docs/guide/en/updates.md)** | Team progress threads, comments, revision history |
| **[Settings](docs/guide/en/settings.md)** | Statuses, repositories, notifications, security, archived tickets |

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
make build-web            # Build frontend assets + Go binary (does not refresh OpenAPI artifacts)
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

| Document | EN | 中文 |
|----------|----|----|
| **User Guide** | [English](docs/guide/en/index.md) | [中文](docs/guide/zh/index.md) |
| Getting Started | [English](docs/guide/en/startup.md) | [中文](docs/guide/zh/startup.md) |
| Module Architecture | [English](docs/guide/en/architecture.md) | [中文](docs/guide/zh/architecture.md) |
| FAQ | [English](docs/guide/en/faq.md) | [中文](docs/guide/zh/faq.md) |
| **Source Build & Run** | [English](docs/en/source-build-and-run.md) | [中文](docs/zh/source-build-and-run.md) |
| IAM Dual-Mode Contract | [English](docs/en/iam-dual-mode-contract.md) | [中文](docs/zh/iam-dual-mode-contract.md) |
| WebSocket Runtime Contract | [English](docs/en/websocket-runtime-contract.md) | [中文](docs/zh/websocket-runtime-contract.md) |
| OIDC & RBAC | [English](docs/en/human-auth-oidc-rbac.md) | [中文](docs/zh/human-auth-oidc-rbac.md) |
| Observability | [English](docs/en/observability-checklist.md) | [中文](docs/zh/observability-checklist.md) |
| Remote Runtime v1 Rollout | [English](docs/en/remote-websocket-rollout.md) | [中文](docs/zh/remote-websocket-rollout.md) |
| Gemini CLI Adaptation | [English](docs/en/gemini-cli-adaptation-guide.md) | [中文](docs/zh/gemini-cli-adaptation-guide.md) |
| Claude Code Stream Protocol | [English](docs/en/claude-code-stream-protocol.md) | [中文](docs/zh/claude-code-stream-protocol.md) |

---

## ⭐ Star History

<p align="center">
  <a href="https://star-history.com/#PacificStudio/openase&Date">
    <img src="https://api.star-history.com/svg?repos=PacificStudio/openase&type=Date" alt="Star History Chart for PacificStudio/openase" width="100%" />
  </a>
</p>

---

## 📄 License

See [LICENSE](LICENSE).

## Friendly Link

[linux.do](https://linux.do/latest)

---

<p align="center">
  <strong>OpenASE</strong>
  <br>
  <em>Create the ticket. The agent does the rest.</em>
</p>
