# OpenASE — Open Auto Software Engineering

**Ticket-driven fully automated software engineering platform | Product Requirements Document (PRD)**

March 2026 | Open-source license: Apache License 2.0

---
## Chapter 1 Executive Summary

OpenASE (Open Auto Software Engineering) is a fully automated software engineering platform driven by tickets as its core unit. It abstracts work across the software development lifecycle, including coding, testing, documentation, security scanning, and deployment, into standardized tickets, which AI Agents can autonomously claim and execute, enabling end-to-end automation from requirements to code merge.

The platform adopts an all-Go monolithic architecture. The orchestrator and API service share the same codebase, and through an adapter abstraction layer it is compatible with multiple Agent CLIs (OpenAI Codex, Claude Code, Gemini CLI, etc.), with built-in approval governance, lifecycle Hooks, and cost control mechanisms.

**Core value proposition:**

- Enable engineering teams to shift from "supervising AI coding" to "managing the work itself"
- Bring Agent behavior definitions (Workflow / Harness / Skills) under version management in the platform control plane, making them auditable, rollbackable, and stably projectable to any runtime
- Allow teams to define "what counts as done" through the lifecycle Hook system (repository preparation before ticket claim, quality checks during execution, cleanup verification after completion)
- Single-binary deployment (frontend embedded with `go:embed`), ready to use after download, with no Docker or Node.js required

---
## Chapter 2 Background and Problem Statement

### 2.1 Current Industry State

In early 2026, AI coding Agents had already reached the level of being able to independently complete development tasks of moderate complexity. Tools such as OpenAI Codex, Claude Code, and Gemini CLI can read codebases, edit files, run commands, and submit PRs. However, between "an Agent can write code" and "an Agent can reliably deliver software," there is a huge gap.

### 2.2 Core Problems

After researching the source code of two open-source projects, OpenAI Symphony (Elixir/OTP orchestrator) and OpenClaw Mission Control (FastAPI + Next.js operations platform), we identified five core pain points in the current AI Agent coding field:

**Problem 1: Agents require continuous human supervision**

- Current state: After starting a coding Agent, developers need to continuously watch its output and manually judge when to stop it and when to intervene. Compared with traditional development, this does not reduce much human effort; it merely turns "writing code" into "watching an Agent write code."
- Root cause: There is a lack of automated quality validation and closed-loop mechanisms. The Agent does not know when something counts as "done," so humans have to act as quality gatekeepers.

**Problem 2: Parallel handling of multiple tasks is difficult**

- Current state: When a team has dozens of pending items, they can only hand them to an Agent one by one. There is no orchestration mechanism to manage priority, concurrency, or dependencies.
- Root cause: Existing tools are designed for a single-task interactive model and lack a task queue and scheduling system.

**Problem 3: Agent behavior is uncontrollable and not traceable**

- Current state: Agent behavior is determined by the Prompt used each time, with no version control, making it impossible to trace back "why it did this last time." The quality of Prompts provided by different developers varies widely, leading to inconsistent output.
- Root cause: The Agent's working specification (Prompt + configuration) has not been managed as "code."

**Problem 4: Severe coupling to specific Agent CLIs**

- Current state: Symphony supports only Codex, and Mission Control supports only its own protocol. When a team wants to use both Claude Code and Codex for different types of tasks, there is no unified orchestration layer.
- Root cause: Each platform is designed for a specific Agent CLI and does not abstract a general adapter interface.

**Problem 5: A disconnect from coding to delivery**

- Current state: After the Agent finishes writing code, testing, documentation updates, security scanning, and deployment are still fragmented manual processes. No platform manages these as unified software engineering activities.
- Root cause: Existing tools focus on the single step of "coding" rather than the entire software engineering lifecycle.

### 2.3 Our Judgment

The AI Agent coding field is undergoing a paradigm shift from "single-Agent usage" to "Agent fleet management." What the next generation of platforms needs to solve is not "how to make Agents write better code," but "how to enable a team of Agents to reliably deliver software." That is exactly what OpenASE is built to do.

---
## Chapter 3 Product Vision and Goals

### 3.1 Vision

Become the software engineering operating system for the AI era: an automation platform that enables AI Agent teams to work like human engineering teams by taking tasks through tickets, executing according to standards, passing quality gates, and ultimately delivering mergeable code.

### 3.2 Design Principles

**Principle 1: Tickets are everything** — Tickets are the atomic unit of the system and the only collaboration interface between Agents and humans. All work, whether feature development, bug fixes, documentation updates, or security scanning, is expressed as tickets. Creating a ticket means issuing an instruction; closing a ticket means confirming delivery.

**Principle 2: Workflows define how work is done** — Each ticket is attached to a Workflow type (`coding`, `test`, `doc`, `security`, `deploy`, etc.), and the Harness content of the Workflow defines the Agent's handoff process, work standards, and work boundaries. For the same requirement, if you attach a different Workflow, the Agent will handle it differently.

Additional note: A Workflow does not directly "pick a running worker instance"; instead, it binds to an Agent definition. The Agent definition then binds to a Provider. That is:

- Provider = an available Coding Agent CLI adapter configuration on a certain Machine
- Agent = the runtime mode definition of a certain Provider within the project
- Workflow = binds to a certain Agent definition and declares which runtime mode, which machine, and which CLI should drive this type of ticket

**Principle 3: Behavior is a managed asset** — Agent behavior definitions (Workflow Harness, Skills, binding relationships) are stored in the OpenASE control plane rather than in `.openase/` files inside the project repository. Modifying these assets generates in-platform versions and audit records; at runtime, version snapshots are materialized into the workspace at startup to ensure traceability, replayability, and stable recovery.

**Principle 4: Trust, but verify** — The platform does not assume that Agent output is always correct. Quality verification is implemented through lifecycle Hooks: teams define what checks to run at each stage of a ticket (lint, tests, security scanning, etc.), and Hook failures block state transitions. High-risk operations require human approval.

**Principle 5: Progressive automation** — Start with manual Workflow and Agent assignment, then gradually introduce AI-based automatic assignment, automatic approval, and Harness self-optimization. Let teams build trust in the system at their own pace.

### 3.3 Target Users

- **Primary users**: Engineering teams of 3-20 people that are already using AI coding tools (Claude Code, Codex, etc.) but struggle to manage Agent work at scale.
- **Secondary users**: Individual developers who want to use Agents in parallel to handle different tasks across multiple personal projects.
- **Future users**: Enterprise engineering platform teams that need to manage Agent usage, cost, and security consistently at the organizational level.

### 3.4 Success Metrics

| Metric | Definition | Phase 1 Target |
|------|------|-------------|
| Ticket autonomous completion rate | Percentage of tickets completed autonomously by Agents (without human intervention) | ≥ 60% |
| Average ticket cycle time | Average time from `todo` to `done` | < 30 minutes (medium complexity) |
| PR first-pass rate | Percentage of PRs submitted by Agents that are merged without a change request | ≥ 40% |
| Orchestrator availability | Uptime of the orchestration service without failures | ≥ 99.5% |
| Deployment time | Time from `git clone` to service availability | < 10 minutes |

---
## Chapter 4 Core Design Philosophy

### 4.1 The Operating Model of "Tickets Are Everything"

The entire system operates around tickets. Below is the complete lifecycle of a ticket:

1. A human or system creates a ticket (manual creation / scheduled task trigger / external API call)
2. The ticket carries a Workflow type label (a user-defined string label, such as `Product Manager` / `QA` / `Backend Engineer`), and the platform then derives a stable Workflow Family (such as `planning` / `test` / `coding`) based on Role / state semantics / Harness signals for recommendation, analysis, and visual semantics
3. During a scheduling cycle, the orchestrator discovers executable tickets, parses the Agent definition bound to the ticket's Workflow, directly reads the currently published Workflow / Skills versions from the DB and materializes a snapshot for this run; it then prepares the required repo workspace (direct clone/fetch), executes the `on_claim` Hook (deriving a working copy of the ticket, key decryption, dependency preparation, etc.), and then creates an AgentRun
4. The AgentRun performs work according to the Workflow Harness (coding + testing + PR creation)
5. After each Turn, the AgentRun has its ticket state re-read by the orchestrator; as long as the ticket remains in the `pickup/active` state, it continues subsequent Turns in the same session until the state changes, `max_turns` is hit, or it is paused or canceled
6. When the ticket leaves the `pickup/active` state, the orchestrator only stops continuation and releases runtime ownership; business-state progression must come from the Agent's explicit platform actions or from explicit state changes made by humans in the UI/API
7. After a human confirms completion, the ticket is moved to `done`, and the `on_done` Hook is executed (workspace cleanup, notification sending, etc.); the business-level lifecycle enters `ActivityEvent`, fine-grained CLI traces enter `AgentTraceEvent`, and human-readable action stages enter `AgentStepEvent`

### 4.2 Multiple Workflow Labels and Family

`workflow.type` is user-facing and is the raw label in the project's language; inside the platform, a stable `workflow family` is derived for scheduling analysis, HR Advisor, Dashboard aggregation, and visual semantics.

| Workflow Family | User-visible label examples | Responsibilities | Typical outputs |
|--------------|----------------|------|---------|
| planning | Product Manager / PRD / Requirements Analysis | Requirement definition, scope clarification, planning breakdown | Actionable tickets, priority and scope conclusions |
| dispatcher | Dispatcher / Triage / Routing | Ticket dispatching, lane routing, state orchestration | State changes, dispatch results |
| coding | Fullstack Developer / Backend Engineer / Frontend Engineer | Requirement implementation, feature development, bug fixes, refactoring | Git Branch + PR + CI green |
| review | Code Reviewer / Approval | Code review, approval, pre-merge gatekeeping | Review conclusions, blockers |
| test | QA Engineer / Verification | Write tests based on requirements/code to prevent regressions | Test code + coverage report |
| docs | Technical Writer / Documentation | Scan code changes and ensure documentation aligns with the latest implementation | Documentation update PR |
| deploy | Release Captain / DevOps Engineer | Connect to remote machines, update configuration, deploy new versions | Deployment logs + version verification results |
| security | Security Engineer / Audit | Code security analysis, vulnerability detection, PoC writing | Security report + fix PR |
| harness | Harness Optimizer / Prompt Tuning | Automatically optimize Workflow Harness content (meta-workflow) | Updated Harness templates |
| environment | Environment Provisioner / Bootstrap | Environment repair, machine preparation, dependency recovery | Schedulable environment, repair records |
| research | Research Ideation / Experiment Runner | Research, experiment design, hypothesis validation | Research conclusions, experiment results |
| reporting | Report Writer / Writeup | Report organization, conclusion distillation, external writing | Reports, summaries, paper |

### 4.3 Auto Harness Mechanism

The core of each Workflow is a Harness document, persisted and version-managed by the OpenASE control plane. The Harness uses pure Markdown / Gonja body to define the Agent's working specification; control-plane metadata such as Workflow name, type, role, state binding, skill binding, and platform permissions is all stored in the database and is no longer written into Harness frontmatter. At runtime, the current version is materialized into the workspace at startup for the Agent to use.

**Core characteristics of the Harness:**

- **Versioning**: Harness modifications generate a new platform version, and new tickets use the latest published version by default
- **Runtime snapshot**: Each AgentRun records the exact Harness version it used; new runtimes automatically get the latest version, while already running runtimes do not drift implicitly
- **Template variables**: Supports `{{ ticket.identifier }}`, `{{ project.name }}`, `{{ agent.name }}`, `{{ attempt }}`, and so on
- **Version traceability**: Each ticket records the Harness version number used during execution
- **Self-optimization**: The refine-harness meta-workflow analyzes execution history and automatically optimizes Harness content

### 4.4 Phased Strategy

The current phase focuses on the main line of Org → Project → Ticket → Workflow → Agent. Enterprise capabilities such as Team, Role, and Permission will be implemented later:

- **Phase 1**: Get the main pipeline working and prove the automated closed loop is feasible
- **Phase 2**: Git integration + approval governance + multiple Workflows
- **Phase 3**: Scheduled tasks + Auto Harness self-optimization + cost control
- **Phase 4**: Multi-tenancy + pluginization + open API

---
## Chapter 5 Technical Architecture

### 5.1 Architecture Decision Records

**Decision 1: All-Go Monolithic Architecture**

| Option Considered | Advantages | Disadvantages | Conclusion |
|---------|------|------|------|
| Python API + Go orchestrator | Best of both worlds | Two languages: two dependency sets, two CI pipelines, cross-language serialization | Rejected |
| All Python | Unified language, large ecosystem | Fragile subprocess management, GIL limitations, bloated deployment | Rejected |
| Elixir/OTP (Symphony approach) | BEAM is naturally suited for Agent process management | Scarce talent, small ecosystem, steep learning curve | Rejected |
| All-Go (adopted) | Lightweight concurrency with goroutines, single-binary deployment, broad talent pool | Less elegant than BEAM, slightly more CRUD code | Adopted |

**Decision 2: SSE Instead of WebSocket**

| Option Considered | Advantages | Disadvantages | Conclusion |
|---------|------|------|------|
| WebSocket | Bidirectional communication, low latency | Complex connection management, heavy reconnection logic, difficult load balancing | Rejected |
| Pure SSE (adopted) | Native HTTP, automatic reconnection, good browser support | One-way push only | Adopted |

Reason: The Agent monitoring scenario only needs the server to push events to the client; bidirectional communication is not required. The OpenClaw Mission Control source code verified that SSE is fully sufficient in this scenario.

**Decision 3: Workflow / Skill Content Stored in the Platform Control Plane**

| Option Considered | Advantages | Disadvantages | Conclusion |
|---------|------|------|------|
| Git repo `.openase/` as the source of truth | Humans can directly see the files | Strongly depends on an extra local cache layer / branch / working tree state, and couples control-plane assets too deeply to the code repository | Rejected |
| Pure filesystem shared directory | Simple | Poor consistency across multiple machines / instances, weak recovery and auditing | Rejected |
| Database-stored Skill / Harness versioned assets + file manifest + audit tables (adopted) | Clear control-plane source of truth, transactional consistency, can materialize to any runtime by bundle version | Requires self-built versioning, file storage, and export logic | Adopted |

**Decision 4: Adapter Abstraction Layer Instead of a Unified Protocol**

The native protocols of different Agent CLIs differ too much (Codex is JSON-RPC over stdio, Claude Code is an NDJSON stream). Forcing unification would lose each tool's unique capabilities. Therefore, the adapter pattern is used: each Agent CLI has its own native adapter, but exposes a unified Go interface to the orchestrator.

### 5.2 Technology Stack

| Layer | Technology Choice | Reason for Selection |
|------|---------|---------|
| CLI framework | cobra | Subcommand pattern (`serve / orchestrate / all-in-one`) |
| Web framework | Echo v4 | Lightweight and high performance, mature middleware ecosystem, good OpenAPI integration |
| ORM | ent (open sourced by Facebook) | Code generation + type safety + graph traversal queries |
| Database migration | atlas (by Ariga) | Native ent integration, declarative migrations + diff detection |
| Database | PostgreSQL 16 | JSON fields + full-text search + mature reliability; users deploy it themselves, terminal setup guides the connection |
| Inter-process communication | PostgreSQL LISTEN/NOTIFY | Cross-service event notifications (when deployed separately); use Go channel directly when in the same process |
| Scheduled tasks | robfig/cron v3 | The most mature cron library in the Go ecosystem |
| Git operations | go-git v5 | Pure Go implementation, no C dependency |
| File monitoring | fsnotify | workspace file observability / local debugging aid (no longer used to watch Harness as the authoritative source) |
| Process management | os/exec + context | Agent CLI subprocess management + timeout cancellation |
| Logging | slog (standard library) | Built-in structured logging in Go 1.21+ |
| Configuration | viper | Multi-source configuration (file / environment variables / command line) |
| OpenAPI generation | OpenASE built-in exporter + kin-openapi | Export `api/openapi.json` from the Go HTTP contract as the single source of interface truth for frontend-backend handoff |
| Frontend framework | SvelteKit + Tailwind CSS | Compile-time framework with no runtime overhead; adapter-static outputs pure static files, perfectly matching `go:embed`; SSE streaming updates fit naturally with Svelte store |
| Frontend component library | shadcn-svelte (based on bits-ui) | Copy-paste component source with no runtime dependency; Tailwind-native; matches Linear-style aesthetics; Kanban drag-and-drop paired with svelte-dnd-action |
| Frontend icon library | Lucide (lucide-svelte) | Default icon library for shadcn-svelte; each icon is an independent Svelte component, perfect on-demand import tree-shaking; 1400+ icons |
| Frontend API client | openapi-typescript | Generate TypeScript contract types from `api/openapi.json`, paired with a lightweight fetch wrapper + Svelte store |
| Authentication | Dual mode (`disabled` / `oidc`) | Browser login can be disabled by default for local use; standard OIDC is used when human browser login is required |

### 5.3 Service Architecture

| Service | Language | Responsibility |
|------|------|------|
| `openase serve` | Go | ticket CRUD, state machine, Workflow management, SSE push, authentication |
| `openase orchestrate` | Go | ticket polling and scheduling, Agent process management, heartbeat monitoring, retry scheduling, Stall detection, reading published versions and materializing runtime snapshots |
| `openase all-in-one` | Go | Single-process mode, running `serve + orchestrate` concurrently via goroutines |
| web-app | Go (embed) | Static assets built by SvelteKit are compiled into the Go binary and embedded through `go:embed`, with no separate deployment required |
| PostgreSQL 16 | - | The only database. Users deploy it themselves (existing instance or started with a one-line Docker command), and OpenASE is only responsible for connecting |

**Deployment Model: Binary-first**

OpenASE is compiled into a single binary, with frontend static assets embedded through `go:embed`. Users do not need to install Docker, Node.js, or any other runtime. Download the binary → run it → open it in the browser → start using it.

Docker is retained as an optional advanced deployment method (suitable for production environments that need container orchestration), but it is not the default recommendation.

**Inter-process Communication: PostgreSQL Is the Only Shared State**

The main communication medium between the two processes (`serve` and `orchestrate`) is PostgreSQL itself: the orchestrator polls the database on each Tick to get the latest ticket state, and after the API process writes to the database, the orchestrator can naturally read it. Most scenarios do not require additional real-time notifications.

Only a few scenarios require real-time communication (cannot wait for the next Tick):

| Scenario | Direction | Why can't it wait for Tick? | Communication method |
|------|------|-------------------|---------|
| User cancels ticket | serve → orchestrate | A running Agent should stop immediately and must not waste 5 more seconds | PG `NOTIFY cancel_ticket, 'ASE-42'` |
| Agent state change | orchestrate → serve | Frontend SSE needs immediate push so the user can see progress right away | PG `NOTIFY agent_event, '{...}'` |
| Hook execution result | orchestrate → serve | The frontend should immediately show Hook success/failure | PG `NOTIFY hook_result, '{...}'` |
| Ticket state advancement | orchestrate → serve | The frontend board should update immediately | PG `NOTIFY ticket_status, '{...}'` |

**All non-real-time scenarios (90%+) rely entirely on database polling and do not need a notification mechanism:**

| Scenario | Explanation |
|------|------|
| New ticket appears | The orchestrator sees it on the next Tick (default 5s) by polling `SELECT ... WHERE status = 'todo'` |
| Workflow changes | `serve` writes the new Workflow / Skill version to the DB; later, runtime reads the latest published version directly when created and materializes it, with no extra inter-process communication or control-plane sync involved |
| PR link updates in RepoScope | `serve` writes to the database, and the frontend and orchestrator naturally read it; does not depend on external Webhooks |
| Agent registration/configuration | `serve` writes to the database, and the orchestrator naturally reads the Workflow -> Agent -> Provider bindings and concurrency limits on the next dispatch |

**`all-in-one` mode is even simpler:** `serve` and `orchestrate` run in different goroutines within the same process, and all of the real-time communication above uses Go channels instead of PG NOTIFY: zero serialization, zero network, zero latency. This is also why `all-in-one` is the recommended default mode.

**EventProvider abstracts this difference:**

```go
// all-in-one mode -> ChannelBus (Go channel)
// separately deployed mode -> PGNotifyBus (PostgreSQL LISTEN/NOTIFY)
// Business code only calls EventProvider.Publish / Subscribe and does not perceive the underlying implementation
```

### 5.4 `~/.openase/` — User Home Base

OpenASE maintains a `~/.openase/` directory under the user's Home directory as the storage center for global configuration and sensitive information:

```
~/.openase/
├── config.yaml              # Global configuration (database connection, listening port, log level)
├── .env                     # Sensitive environment variables (DB password, API Key, etc.), permission 0600
├── workspace/               # Root directory for ticket workspaces
│   └── {org-slug}/
│       └── {project-slug}/
│           └── {ticket-identifier}/
│               ├── backend/
│               └── frontend/
└── logs/                    # Runtime logs (supplement to journalctl, retain the last 7 days)
```

Service configuration files are automatically installed to standard system locations:
- Linux: `~/.config/systemd/user/openase.service`
- macOS: `~/Library/LaunchAgents/com.openase.plist`

**Sensitive information management**: `~/.openase/.env` is used only to store **sensitive environment variables that need to be read when the local OpenASE service starts**, such as database passwords, local API authentication tokens, OIDC client secrets, third-party Provider API keys, notification webhooks, etc., with file permission `0600` (readable and writable only by the owner). It is **not** the "single on-disk location for all Tokens": Secrets such as outbound GitHub credentials like `GH_TOKEN`, which must be centrally managed by the platform, detected, resolved by scope, and projected into local / remote controlled sessions, must be stored in the platform Secret storage layer rather than written to `~/.openase/.env`. At the same time, Workflow Harness, Skills, and bindings are also no longer treated as authoritative from the `.openase/` directory in the project repository, and are instead managed by the platform control plane.

### 5.5 Backend Layered Architecture (DDD + Provider)

The OpenASE backend uses a four-layer DDD (Domain-Driven Design) architecture, combined with the Provider pattern to handle cross-cutting concerns. The core principle is: **dependency direction always points inward: outer layers depend on inner layers, and inner layers do not know the existence of outer layers.**

```
┌─────────────────────────────────────────────────────────────────┐
│                 Interface / Entry Layer                         │
│  cmd/openase  ·  internal/cli  ·  internal/httpapi             │
│  internal/webui  ·  internal/setup                             │
├─────────────────────────────────────────────────────────────────┤
│              Service / Use-Case Layer                          │
│  internal/service/*  ·  internal/ticket  ·  internal/workflow  │
│  internal/chat  ·  internal/notification  ·  internal/agentplatform │
├────────────────────────────────────────────╥────────────────────┤
│        Domain / Core Types                 ║   Provider         ║
│  internal/domain/*                         ║                    ║
│  internal/types/*                          ║  TraceProvider     ║
│                                            ║  MetricsProvider   ║
│                                            ║  EventProvider     ║
│  The current repo focuses on parse /       ║                    ║
│  value object / pure logic, and does not   ║  ExecutableResolver║
│  require every subpackage to have the      ║  AgentCLIProcessMgr║
│  full entity/repository/service/event set  ║                    ║
│                                            ║  UserServiceMgr    ║
│                                            ║  wired by app/cmd  ║
├────────────────────────────────────────────╨────────────────────┤
│                  Infrastructure Layer                          │
│  internal/repo/     (DB-backed Repository / ent repository adapters) │
│  internal/infra/    (Agent CLI / hook / SSE / workspace implementations) │
│  internal/provider/ (Provider contracts + noop/default pieces) │
└─────────────────────────────────────────────────────────────────┘
```

**Responsibilities of each layer:**

**Domain / Core Types** — in the current repository, this layer mainly carries domain parsing, value objects, pure logic, and a small amount of stable enum mapping.

- `internal/domain/catalog`, `internal/domain/ticketing`, `internal/domain/notification`, etc.: input parsing, value objects, pure business rules, stable data structures
- `internal/types/*`: low-level domain types and database boundary types
- Domain packages expose their own stable types and enums externally, and do not directly leak `ent/*` generated types; database enum and field mappings stay in `internal/repo/*`
- It is no longer assumed that every domain subpackage must strictly correspond to the four-piece set `entity.go / repository.go / service.go / event.go`; actual responsibilities in the current repository take precedence

**Service / Use-Case Layer** — orchestrates use cases and connects repository/provider/domain, and no longer uses the old PRD directory naming of `app/command` and `app/query`.

- Currently mainly corresponds to `internal/service/*`, `internal/ticket`, `internal/workflow`, `internal/chat`, `internal/notification`, `internal/scheduledjob`, and `internal/agentplatform`
- These packages take on the responsibilities of the application layer from the old PRD: orchestrating complete use cases, calling domain parsing results, accessing repositories, and driving providers
- The repository port/interface is owned by the upper-layer package that consumes it; `internal/repo/*` only provides adapter implementations and does not in reverse determine the dependency shape of the service
- Some packages contain both command-style write operations and query-style read operations, but expose them through service objects rather than splitting them by `app/command` and `app/query` directories

**Infrastructure Layer** — implementations of all external dependencies.

- `internal/repo/*`: database-related repository adapters; in the current repository, this part takes on the responsibilities of `infra/persistence/` from the old PRD
- Repository adapters are responsible for mapping ent/client/database details into domain stable types, and do not directly pass persistence types outward
- `internal/infra/adapter/*`: implementations of Agent CLI adapters (Claude Code, Codex, etc.)
- `internal/infra/hook`, `internal/infra/sse`, `internal/infra/workspace`, `internal/infra/event`, etc.: implementations of external system and runtime boundaries
- `internal/provider`: cross-cutting Provider interfaces and default implementations (see 5.6)

**Interface / Entry Layer** — external entry points, kept thin.

- `cmd/openase`: CLI entry point, responsible for startup commands, parameter wiring, and exit code handling
- `internal/httpapi`: Echo HTTP API handlers, route registration, request binding, error mapping, SSE/HTTP entry points; server/runtime wiring remains separated from route/handler registration
- `internal/cli`: CLI subcommands and terminal interaction
- `internal/setup`, `internal/webui`: current terminal setup, legacy web bootstrap, and Web UI entry points

### 5.6 Provider Cross-Cutting Architecture

Provider is the unified pattern OpenASE uses to handle cross-cutting concerns. Each Provider is defined as a Go interface and wired in `cmd/openase` / `internal/app`. Code at any layer can use a Provider, but depends only on the interface, not the implementation.

```go
// All Provider interfaces are defined in internal/provider/
package provider

// TraceProvider — distributed tracing
type TraceProvider interface {
    ExtractHTTPContext(ctx context.Context, header http.Header) context.Context
    InjectHTTPHeaders(ctx context.Context, header http.Header)
    StartSpan(ctx context.Context, name string, opts ...SpanStartOption) (context.Context, Span)
    Shutdown(ctx context.Context) error
}

// MetricsProvider — metrics collection
type MetricsProvider interface {
    Counter(name string, tags Tags) Counter
    Histogram(name string, tags Tags) Histogram
    Gauge(name string, tags Tags) Gauge
}

// EventProvider — inter-process event communication
type EventProvider interface {
    Publish(ctx context.Context, event Event) error
    Subscribe(ctx context.Context, topics ...Topic) (<-chan Event, error)
    Close() error
}

// ExecutableResolver — local executable discovery
type ExecutableResolver interface {
    LookPath(name string) (string, error)
}

// AgentCLIProcessManager — Agent CLI subprocess management
type AgentCLIProcessManager interface {
    Start(ctx context.Context, spec AgentCLIProcessSpec) (AgentCLIProcess, error)
}

// UserServiceManager — platform-related user service management
type UserServiceManager interface {
    Platform() string
    Apply(context.Context, UserServiceInstallSpec) error
    Down(context.Context, ServiceName) error
    Restart(context.Context, ServiceName) error
    Logs(context.Context, ServiceName, UserServiceLogsOptions) error
}
```

In the current repository, the authentication boundary mainly sits at the security setup entry points in `internal/httpapi`; notifications are also no longer exposed through a unified `NotifyProvider`, and are instead managed by the channel adapter / rule engine in `internal/notification`.

**Provider implementation matrix:**

| Provider / Contract | Current default implementation | Optional implementation / extension point | Injection point |
|----------|---------|---------|---------|
| `TraceProvider` | `internal/provider/noop_trace.go` | `internal/infra/otel/trace.go` | interface, service, runtime |
| `MetricsProvider` | `internal/provider/metrics.go` (noop) | `internal/infra/otel/metrics.go` | interface, service, runtime |
| `EventProvider` | `internal/infra/event/channel.go` | `internal/infra/event/pgnotify.go` | service, orchestrator, httpapi |
| `ExecutableResolver` | `internal/infra/executable/path.go` | custom resolver | service |
| `AgentCLIProcessManager` | `internal/infra/agentcli/process.go` | fake / test manager | chat, adapter, orchestrator |
| `UserServiceManager` | `internal/infra/userservice/*.go` | platform-specific implementation | runtime / deploy |

**Dependency injection (wiring):** currently assembled mainly in `internal/app/app.go`, `cmd/openase/main.go`, and CLI subcommands. The concrete implementation is selected according to the configuration in `~/.openase/config.yaml`:

```go
// pseudocode
func buildProviders(cfg Config) Providers {
    var event provider.EventProvider
    if cfg.Event.Driver == "pgnotify" {
        event = pgnotify.New(cfg.Database.DSN)
    } else {
        event = channelbus.New() // all-in-one default channel
    }

    trace := oteltrace.NewOrNoop(cfg.Observability)
    metrics := otelmetrics.NewOrNoop(cfg.Observability)
    resolver := executable.NewPathResolver()

    return Providers{Event: event, Trace: trace, Metrics: metrics, Resolver: resolver}
}
```

### 5.7 Project Directory Structure

```
openase/
├── cmd/openase/
│   └── main.go                  # CLI entry point
│
├── internal/
│   ├── app/                     # startup entry point and runtime wiring
│   ├── domain/                  # domain parsing, pure logic, value objects
│   ├── types/                   # low-level domain types / DB boundary types
│   ├── provider/                # cross-layer provider contracts
│   ├── repo/                    # ent-backed repository adapters
│   ├── service/                 # typical service/use-case packages
│   ├── ticket/                  # ticket service/use-case
│   ├── workflow/                # workflow service/use-case
│   ├── chat/                    # chat service/use-case
│   ├── notification/            # notification service/use-case
│   ├── scheduledjob/            # scheduled job service/use-case
│   ├── agentplatform/           # agent platform service/use-case
│   ├── infra/                   # implementations such as adapter / hook / ssh / workspace / event / otel
│   ├── httpapi/                 # Echo HTTP API, SSE, webhook, OpenAPI handler
│   ├── cli/                     # CLI subcommands
│   ├── orchestrator/            # scheduling and runtime orchestration
│   ├── runtime/                 # runtime support (DB, observability)
│   ├── setup/                   # terminal setup + legacy web bootstrap
│   └── webui/                   # embedded Web UI handler
│
├── web/                         # Svelte frontend (embedded after build)
├── go.mod
└── Dockerfile                   # optional
```

Illustration of ordinary code and script directories in the project's Git repository:

```
your-project/
├── scripts/
│   └── ci/                    # Scripts called by Hooks (part of the repository code)
│       ├── run-tests.sh
│       ├── lint.sh
│       └── cleanup.sh
└── ... (project code)
```

> **Deprecated design note**: In the historical design, `.openase/harnesses/` and `.openase/skills/` under the project repository root were treated as the authoritative source of Workflow / Skill. This design has now been deprecated. The project repository may contain no `.openase/` directory at all, and OpenASE can still run normally; Workflow / Skills are managed by the platform control plane and materialized into the workspace when runtime starts.

---
## Chapter 6 Core Domain Model

### 6.1 Entity Relationship Overview

```
Organization → Project → ProjectRepo (1:N)
                ↓
              Ticket → TicketRepoScope (1:N) → ProjectRepo
                ↓
              Workflow (including Hooks configuration), AgentProvider, Agent,
              ScheduledJob, ActivityEvent
```

A Project is associated with multiple ProjectRepos (multi-repository support). A Ticket declares which Repos it involves through TicketRepoScope, and records the working branch and optional PR link for each Repo. Hook configuration is embedded in Workflow to define automated checks for each lifecycle stage of a ticket.

**JSON Field Usage Principles**: Fields whose contents are known and will be queried or filtered should use structured columns (such as `max_concurrent`, `auto_assign_workflow`) or PostgreSQL native arrays `TEXT[]` (such as `labels`). Only truly dynamic, shape-uncertain, non-queryable data should use JSONB (such as `Ticket.metadata`, `Workflow.hooks`, `AgentProvider.auth_config`, `ScheduledJob.ticket_template`, `ActivityEvent.metadata`, `AgentTraceEvent.payload`).

### 6.2 Organization

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| name | String | Organization name |
| slug | String | URL-friendly identifier |
| status | String | `active` / `archived` |
| default_agent_provider_id | FK (nullable) | Default Agent Provider |

### 6.3 Project

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| organization_id | FK | Owning organization |
| name | String | Project name |
| slug | String | URL-friendly identifier |
| description | Text | Project summary (Markdown) |
| status | String | Project lifecycle string; the database does not use enums. Only the normalized write values `Backlog` / `Planned` / `In Progress` / `Completed` / `Canceled` / `Archived` are allowed |
| default_agent_provider_id | FK | Default Agent Provider |
| max_concurrent_agents | Integer | Project-level maximum concurrent Agent count (default 5) |
| agent_run_summary_prompt | Text (nullable) | Project-level override for the terminal run summary prompt; empty means the platform built-in default prompt is used |

**Project.status Rules:**

- The database layer uses a normal string column (such as `TEXT` / `VARCHAR`), not a database enum.
- The backend is responsible for parsing and normalization at the write boundary; only the following 6 canonical values may be persisted:
  - `Backlog`
  - `Planned`
  - `In Progress`
  - `Completed`
  - `Canceled`
  - `Archived`
- Arbitrary free text must not be written directly into the database; the constraint is enforced by backend code rather than relying on a database enum.
- The API only accepts the 6 exact strings above; any alias, case variation, extra whitespace, or historical value must return `400 Bad Request`.
- The UI must submit only the canonical values above and is not responsible for input correction or alias mapping.

> **v3.2 Change**: `repository_url` and `default_branch` were removed from Project and moved to the new ProjectRepo entity. A Project can be associated with multiple Repos.

> **v3.3 Revision**: Historically, `ProjectRepo.clone_path` mixed two semantics at once: the "repo runtime path" and the "directory name inside the Ticket workspace". This caused conflicts among Workflow, Workspace, and Git sync logic. It is now narrowed into two explicit objects: `ProjectRepo` (remote binding) and `TicketRepoWorkspace` (ticket working copy). A single field must no longer be used to simultaneously express a remote address, a runtime local path, and a Ticket workspace mount path.

### 6.4 ProjectRepo (Project Repository Binding)

A Project can be associated with multiple Git repositories. `ProjectRepo` is the "binding relationship between a project and a remote code repository". It answers "which repos are involved in this project", not "whether this repo is currently cloned on the local machine".

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning project |
| name | String | Repository alias (such as `backend`, `frontend`, `infra`), unique within the project |
| repository_url | String | Remote Git repository URL; always represents remote truth and must never be mixed with a local absolute path |
| default_branch | String | Default branch (such as `main`) |
| workspace_dirname | String | The first-level directory name of this repo inside the Ticket workspace; defaults to `name` and must be a relative path segment |
| labels | TEXT[] | Repository labels (such as `{"go", "backend", "api"}`), PostgreSQL native array with GIN index support |

**Responsibility Boundaries:**

- `ProjectRepo` only stores static binding information and does not store runtime state such as "whether a clone exists on the current machine".
- `repository_url` only represents the remote address; if the platform needs a local working path, it must read `TicketRepoWorkspace.repo_path`.
- `workspace_dirname` only represents the directory name inside the Ticket workspace and must not be treated as a project-level mirror root directory.

**Design Rationale**: In reality, a product is often structured across multiple repositories (frontend, backend, SDK, infrastructure as separate repos). After decoupling Repo from Project, a ticket can declare which Repos it involves, and the orchestrator can create a combined workspace containing all related Repos for the Agent.

**Typical Scenarios**:

- A "user registration feature" ticket needs to modify `backend` (API interface) + `frontend` (registration page) + `sdk` (type definitions)
- A "database migration" ticket only involves the `backend` repository
- A "CI pipeline update" ticket only involves the `infra` repository

### 6.4.1 TicketRepoWorkspace (Ticket Working Copy)

`TicketRepoWorkspace` represents the repo working copy for a Ticket in a particular execution. It is not a configuration entity, but a runtime-derived object. Each working copy is cloned / fetched directly from the remote repository and checked out to the branch and baseline required for this run.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| ticket_id | FK | Owning Ticket |
| agent_run_id | FK | Owning AgentRun |
| repo_id | FK | Associated ProjectRepo |
| workspace_root | String | The workspace root directory for this Ticket run |
| repo_path | String | The absolute path of this repo inside the workspace |
| branch_name | String | Current working branch |
| state | Enum | `planned / materializing / ready / dirty / verifying / completed / failed / cleaning / cleaned` |
| head_commit | String | Current working copy HEAD |
| last_error | Text | Most recent materialize / verify / cleanup error |
| prepared_at | DateTime | Time when the working copy became ready |
| cleaned_at | DateTime | Cleanup completion time |

**State Semantics:**

- `planned`
  - The scheduler has determined that this repo is needed, but preparation of the working copy has not started yet.
- `materializing`
  - The platform is cloning / fetching the repo, checking out the branch, and writing runtime context.
- `ready`
  - The Agent can safely work inside `repo_path`.
- `dirty`
  - The Agent has produced unverified changes.
- `verifying`
  - The platform or Hook is running checks such as tests, lint, or CI aggregation.
- `completed`
  - This execution has completed and is awaiting cleanup.
- `failed`
  - Preparation or verification of this working copy failed.
- `cleaning`
  - The platform is cleaning up this working copy.
- `cleaned`
  - Cleanup is complete; the record is retained for auditing.

**Key Rules:**

- A Ticket Workspace is a short-lived object.
- Multiple retries / continuations of the same Ticket reuse the same `workspace_root` by default, but the working copy state of each repo is still tracked independently.

### 6.4.2 Repo Lifecycle Overview

In OpenASE, a repo has only two required layers of form, and they must be clearly distinguished:

1. **Remote Binding (ProjectRepo)**
   - Source of truth: repo URL, default branch, labels
   - Long lifecycle, basically affected only by user configuration actions
2. **Ticket Working Copy (TicketRepoWorkspace)**
   - Source of truth: directory, branch, verification status for this Ticket / AgentRun
   - Created with execution, cleaned up with execution

**The platform must explicitly manage the following events:**

- `register_repo`
  - Create the ProjectRepo binding; local availability is not guaranteed initially
- `claim_ticket`
  - Prepare the related `TicketRepoWorkspace` directly from the remote repository
- `complete_ticket`
  - Execute verify / cleanup on the working copy
- `delete_repo`
  - First block new runs, then clean related working copy records, and finally delete the binding

**Code Baseline Strategy:**

- The default execution path is: directly fetch / checkout the latest baseline from the remote repository when creating `TicketRepoWorkspace`.
- The platform does not maintain a project-level code cache layer and does not provide capabilities such as cache registration, synchronization, health checks, or path derivation.
- The meaning of "latest code" is determined only by the remote Git repository and the fetch / checkout of the current working copy, without introducing any additional intermediate-layer state machine.

### 6.5 Ticket — Core Entity

A ticket is the absolute core of the entire system. Every unit of work is a Ticket.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning project |
| identifier | String | Human-readable identifier (such as ASE-42), auto-generated |
| title | String | Ticket title |
| description | Text | Ticket description (Markdown) |
| status_id | FK | Associated TicketStatus (user-visible custom status; lifecycle semantics are provided by that status's `stage`) |
| priority | Enum | urgent / high / medium / low |
| type | Enum | feature / bugfix / refactor / chore / epic |
| workflow_id | FK | Assigned Workflow |
| current_run_id | FK | Current active AgentRun; null means not claimed |
| created_by | String | Creator |
| parent_ticket_id | FK | Parent ticket (sub-issue relationship) |
| external_ref | String | External reference (GitHub Issue ID, etc.) |
| attempt_count | Integer | Total attempt count |
| consecutive_errors | Integer | Consecutive failure count (reset to 0 after success or human intervention) |
| next_retry_at | DateTime | Next retry time (calculated with exponential backoff) |
| retry_paused | Boolean | Whether retry is paused (budget exhausted / paused by human) |
| pause_reason | String | Pause reason (`budget_exhausted` / `user_paused`) |
| stall_count | Integer | Stall count |
| retry_token | String | Retry token (prevents stale retries) |
| harness_version | Integer | Harness version used during execution |
| budget_usd | Decimal | Per-ticket budget limit |
| cost_tokens_input | BigInt | Input token count |
| cost_tokens_output | BigInt | Output token count |
| cost_amount | Decimal | Execution cost amount |
| metadata | JSON | Extension fields |
| started_at | DateTime | Execution start time |
| completed_at | DateTime | Completion time |

**Ticket Dependency Relationships:**

| Relationship Type | Description | Behavior |
|---------|------|------|
| blocks | A blocks B: B cannot start before A completes | B does not participate in scheduling until the `stage` corresponding to A's current status enters `completed` or `canceled` |
| sub-issue | A is a child ticket of B | After A is completed, automatically check whether all child tickets of B are completed |

At the backend, only the two structured edges `blocks` / `sub_issue` are persisted; `blocked_by` is not stored separately. `blocked_by` is only the reverse reading semantics of `blocks`, used for UI presentation rather than as an additional data model.

**Ticket External Links (`TicketExternalLink`):**

A ticket can be associated with multiple external Issues (GitHub Issue, GitLab Issue, Jira Ticket, etc.) and multiple PRs. The `external_ref` field is kept as a shortcut for the primary association, while `TicketExternalLink` supports complete 1:N associations.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| ticket_id | FK | Owning ticket |
| link_type | Enum | github_issue / gitlab_issue / jira_ticket / github_pr / gitlab_mr / custom |
| url | String | External link URL (such as `https://github.com/org/repo/issues/42`) |
| external_id | String | Identifier in the external system (such as `org/repo#42`) |
| title | String | Title of the external Issue/PR (optional cache, display only) |
| relation | Enum | resolves / related / caused_by |
| created_at | DateTime | Association creation time |

**Usage Scenarios:**

- A "fix login bug" ticket can be associated with GitHub Issue #42 (bug report) and #45 (related discussion) at the same time, as well as an automatically created PR
- The Agent can see the contents of all associated Issues (description, comments) in the Harness Prompt to help understand context
- OpenASE does not automatically synchronize the status of these external links; they serve only as context references and navigation entry points

**Harness Template Variables**:

```
{{ range .ExternalLinks }}
- [{{ link.type }}] {{ link.title }}: {{ link.url }}
{{ end }}
```

### 6.5.1 TicketComment (Ticket Comment)

`TicketComment` is a human discussion item in the Ticket Detail timeline. It carries handoff, review, decision notes, and supplemental context, and is not mixed together with system Activity.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| ticket_id | FK | Owning ticket |
| author_type | String | `user` / `agent` / `system_proxy` |
| author_id | String | Author identifier (user ID, Agent ID, or external sync identifier) |
| author_name | String | Timeline display name |
| body_markdown | Text | Currently effective Markdown body |
| is_deleted | Boolean | Soft-delete marker |
| deleted_at | DateTime | Deletion time |
| deleted_by | String | Deleting operator |
| created_at | DateTime | Creation time |
| updated_at | DateTime | Most recent update time |
| edited_at | DateTime | Most recent body edit time; null if never edited |
| edit_count | Integer | Number of times the body has been saved, excluding the initial creation |
| last_edited_by | String | Most recent editor |

**Rules:**

- `TicketComment` represents the "current version"; historical versions go into `TicketCommentRevision`.
- When the UI shows `edited`, it must be based on `edited_at != nil` or `edit_count > 0`, and must not rely on string guessing.
- Deletion uses soft delete. By default, the timeline does not show the body and replaces it with the placeholder `Comment deleted`; historical audit can still be retained.
- The current stage does not require reactions, thread replies, or resolved threads, but the data model must not block future extension.

### 6.5.2 TicketCommentRevision (Comment History Version)

Users can view a comment's edit history in Ticket Detail. To guarantee this, every save of a comment body must write a version history entry.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| comment_id | FK | Associated TicketComment |
| revision_number | Integer | Incremented from 1; `1` is the initial published version |
| body_markdown | Text | Snapshot of the body for this version |
| edited_by | String | Submitter of this version |
| edited_at | DateTime | Time this version was created |
| edit_reason | String | Optional edit note; can be empty at the current stage |

**Rules:**

- When creating a comment, the initial version with `revision_number=1` must be written at the same time.
- Every body save must append a new `TicketCommentRevision`; overwriting history in place is forbidden.
- By default, the timeline only shows the current version in `TicketComment.body_markdown`; the "History" panel reads from `TicketCommentRevision`.
- `TicketComment.edit_count = revisions - 1`; this field is a read optimization, not the source of truth.

### 6.5.3 TicketTimelineItem (Detail Page Timeline Projection)

The main Ticket Detail view should not require the frontend to temporarily stitch together `Ticket`, `TicketComment`, and `ActivityEvent` into a timeline by itself. The backend must provide a unified timeline projection object `TicketTimelineItem`.

`TicketTimelineItem` is a **read model / API projection**. It does not require an independently persisted table, but it must have a stable schema.

| Field | Type | Description |
|------|------|------|
| id | String | Stable unique identifier (such as `description:{ticketId}`, `comment:{commentId}`, `activity:{activityId}`) |
| ticket_id | FK | Owning ticket |
| item_type | String | `description` / `comment` / `activity` |
| actor_name | String | Display name of the actor |
| actor_type | String | `user` / `agent` / `system` |
| title | String | Activity title or description title; can be empty for comment |
| body_markdown | Text | Description / comment body; can be empty for activity |
| body_text | Text | Plain-text summary of activity; can be empty for description / comment |
| created_at | DateTime | Item time |
| updated_at | DateTime | Most recent update time for the item |
| edited_at | DateTime | Edit time for comment / description |
| is_collapsible | Boolean | Whether the UI allows collapsing |
| is_deleted | Boolean | Deleted placeholder for comment |
| metadata | JSON | Extra information such as activity icon, status changes, history version count, links |

**Timeline Rules:**

- The Ticket description must appear as the first fixed item in the timeline, semantically equivalent to "author opened this ticket".
- After the description, items are ordered by `created_at` ascending, and new comments / activities are appended at the bottom.
- Comments and activities must share a single timeline, although their visual styles may differ; they must no longer be split into two disconnected main panels.
- Activity items are not editable or deletable; comment items support edit / delete / history / collapse.
- The frontend must not infer the mixed ordering of comments and activities by itself; it must consume the `TicketTimelineItem[]` returned by the backend.

**Ticket Repository Scope (`TicketRepoScope`):**

A ticket can involve one or more Repos under a Project. TicketRepoScope records the repository binding, optional working branch override, and optional PR link for each ticket in each related Repo.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| ticket_id | FK | Owning ticket |
| repo_id | FK | Associated ProjectRepo |
| branch_name | String | Optional working branch override; null means use the system-generated default ticket branch |
| pull_request_url | String | PR URL created in this Repo |

**Relationship Between Ticket Status and RepoScope PR Links:**

- `pull_request_url` is only reference information, not input to the state machine
- Whether a ticket enters states such as `in_review`, `done`, or `canceled` is always determined by an explicit platform action from the Agent or an explicit status change by a human in the UI/API
- OpenASE does not synchronize PR status or CI status; RepoScope does not carry `pr_status` / `ci_status`

**Multi-Repo Workspace Strategy of the Orchestrator:**

When a ticket involves multiple Repos, the orchestrator creates a combined workspace and directly prepares the corresponding `TicketRepoWorkspace` for each repo. The directory structure is as follows:

```
~/.openase/workspace/{org-slug}/{project-slug}/{ticket-identifier}/
├── backend/        # clone of backend repo, checked out to feature branch
├── frontend/       # clone of frontend repo, checked out to feature branch
└── sdk/            # clone of sdk repo, checked out to feature branch
```

All Repo paths and descriptions are injected into the Agent's Harness Prompt:

```
You are working on ticket ASE-42, which involves the following repositories:
- backend (Go API): ~/.openase/workspace/acme/payments/ASE-42/backend/
- frontend (SvelteKit): ~/.openase/workspace/acme/payments/ASE-42/frontend/
- sdk (TypeScript): ~/.openase/workspace/acme/payments/ASE-42/sdk/

Please complete the necessary changes in all related repositories and create a separate PR for each repository.
```

**Branch Naming Convention**: The default working branch for each Repo is uniformly named `agent/{ticket-identifier}`. The branch belongs to the Ticket, not to a specific Agent; the Agent is only the executor and can take over unfinished work from another Agent without changing the branch name. If `TicketRepoScope.branch_name` is explicitly recorded, it represents the working branch override for that Repo; at runtime, the system first tries to reuse a same-named branch that already exists on the remote, and if it does not exist, it creates that working branch from `repo.default_branch`.

**Ticket Workspace Conventions:**

- The workspace path is not a free-form field filled in arbitrarily by the user, but is derived uniformly by the platform
- Default rules:
  - Local machine: `~/.openase/workspace/{org-slug}/{project-slug}/{ticket-identifier}`
  - Remote machine: `{machine.workspace_root}/{org-slug}/{project-slug}/{ticket-identifier}`
- One Ticket corresponds to one independent working directory
- The first-level subdirectories under the working directory are the multiple Repos involved in that Ticket
- Multiple retries / continuations of the same Ticket reuse the same Ticket working directory
- The Ticket working directory is an execution-time copy, and the intermediate layer of project-level code cache no longer exists
- The code baseline of the Ticket Workspace comes directly from the remote Git repository, and `git fetch origin` inside the working copy is exactly how the latest remote code is fetched

### 6.6 Workflow

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning project |
| name | String | Workflow name |
| type | String | User-defined Workflow type label; non-empty, case-preserving, recommended `1..64` |
| role_slug | String | Stable slug of the Workflow role, stored structurally by the control plane |
| role_name | String | Display name of the Workflow role, stored structurally by the control plane |
| role_description | Text | Summary / description of the Workflow role, stored structurally by the control plane |
| agent_id | FK | Bound Agent definition; every execution of this Workflow always uses this Agent as the driver |
| harness_key | String | Stable logical identifier of the Workflow Harness (such as `coding-default`), unique within the project |
| current_version_id | FK | Current published Harness version |
| hooks | JSONB | lifecycle Hook configuration (nested structure: `hook_name → [{cmd, timeout, on_failure}]`, see Chapter 8) |
| pickup_status_ids | NonEmptySet<FK → TicketStatus.id> | Ticket statuses from which this Workflow can claim tickets; at least 1, and all must belong to the same project |
| finish_status_ids | NonEmptySet<FK → TicketStatus.id> | Ticket statuses to which this Workflow may land upon completion; at least 1, and all must belong to the same project |
| platform_access_allowed | String[] | Whitelist of OpenASE Platform API scopes the Agent may request under this Workflow |
| max_concurrent | Integer | Maximum concurrency for this Workflow (default 3) |
| max_retry_attempts | Integer | Maximum retry count (default 3) |
| timeout_minutes | Integer | Timeout minutes per ticket (default 60) |
| stall_timeout_minutes | Integer | Minutes before timeout with no Agent events (default 5) |
| version | Integer | Version number |
| is_active | Boolean | Whether enabled |

Notes:

- Harness body, version history, and audit records are persisted by the platform control plane; files such as `.openase/harnesses/*` in a Git repository are no longer the authority source.
- Workflow metadata and version snapshots are stored structurally in the database; the Harness body no longer serves as a metadata container and no longer depends on YAML frontmatter.
- `workflow.type` is a user-facing raw label and is no longer a hard-coded enum; the platform derives a stable `workflow family` at read time for analysis, recommendation, visualization, and compatibility logic.
- When supporting legacy data, historical values such as `coding` / `test` / `doc` / `security` / `deploy` / `refine-harness` / `custom` may still be read as valid labels and mapped by the classifier to the corresponding family.
- Workflow editing, publishing, and skill binding changes must all be performed through the Platform API; Agents are not allowed to directly modify files in the repo workspace to change platform control plane behavior.
- Any control plane operation that reads / edits Workflow and Skill does not depend on whether the repo workspace already exists. Repository checkout affects only the code execution path, not the viewing, editing, versioning, or binding of Workflow / Skill.
- `pickup_status_ids / finish_status_ids` belong to structured Workflow metadata, are stored in the database, configured by the frontend, and maintained under database reference constraints.
- `pickup_status_ids / finish_status_ids` may reference any TicketStatus within the same project; the platform only validates that the set is non-empty, the references exist, and they belong to the current project, and no longer restricts the bound set by `stage`.
- The orchestrator only reads the status bindings in the database for scheduling; these statuses are no longer redundantly declared inside the Harness as a source of truth.
- When an AgentRun starts, the platform materializes the Harness version pointed to by `current_version_id` and the currently bound Skill versions into that run's workspace. New runtimes use the latest published version by default; runtimes already in progress are not hot-updated automatically unless explicitly refreshed / restarted.

**Deprecated Fields and Compatibility Notes:**

- The historical design of `harness_path` as a Git repository file path is deprecated.
- During the compatibility migration period, the old field may be kept for read-only mapping or export purposes, but it must no longer be used as the location of Workflow content, the source of version truth, or a write target.

### 6.7 AgentProvider (Agent Provider)

AgentProvider represents **a Coding Agent CLI entry point on a specific Machine that can be invoked by OpenASE**. It answers:

- Which external Coding Agent CLI is installed on this machine
- How OpenASE should launch it
- What its login state / environment variables / concurrency limit are on that machine

Therefore, Provider is not as broad as an "abstract tool family", but rather a **machine-bound executable entry point**. Even if two machines both have the Codex CLI installed, they should still be modeled as two separate Providers, because their paths, authentication state, environment, and available concurrency may differ.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| organization_id | FK | Owning organization |
| machine_id | FK | Bound Machine; this Provider can run only on this machine |
| name | String | Name (such as Codex, Claude Code, Gemini CLI) |
| adapter_type | Enum | claude-code-cli / codex-app-server / gemini-cli / custom |
| cli_command | String | Launch command (such as `claude`, `codex`) |
| cli_args | TEXT[] | CLI startup argument array (such as `{"--max-turns", "20", "--verbose"}`) |
| auth_config | JSONB (encrypted) | Encrypted authentication information (structure differs by Provider, truly polymorphic data) |
| model_name | String | Model name (such as `claude-sonnet-4-6`, `gpt-5.3-codex`) |
| model_temperature | Float | Model temperature (default 0.0) |
| model_max_tokens | Integer | Maximum output token count (default 16384) |
| max_parallel_runs | Integer | Provider-level maximum concurrent runs (provider semaphore) |
| pricing_config | JSONB | Provider pricing source of truth, supporting official default prices, user overrides, cached read/write prices, cache storage prices, and tiered / routed pricing metadata |
| cost_per_input_token | Decimal | Input token summary unit price, used for compatibility with old list views and quick display; derived from `pricing_config` and no longer the source of truth for complex billing |
| cost_per_output_token | Decimal | Output token summary unit price, used for compatibility with old list views and quick display; derived from `pricing_config` and no longer the source of truth for complex billing |

**Supplemental Pricing Semantics:**

- When the user selects a built-in model preset, the Provider form should prioritize loading the platform's built-in official default prices rather than requiring the user to manually enter raw token unit prices.
- `pricing_config` must be able to express at least the following pricing dimensions: standard input, output, cache hit read, 5-minute cache write, 1-hour cache write, hourly cache storage billing, and long-context tiered pricing that switches based on prompt token thresholds.
- `source_kind=official` means the Provider is currently using the platform built-in official default price whose source has been verified; `source_kind=custom` means the user has overridden the default.
- For routing presets such as `auto-gemini-*`, fabricating a fixed flat price is not allowed; the UI should clearly show that billing varies according to the concrete model resolved at runtime.
- Ticket cost calculation should prioritize consuming `pricing_config` and structured usage token categories; `cost_per_input_token / cost_per_output_token` are retained only as summary display fields.

**Unified Convention for GitHub Outbound Credentials:**

- All OpenASE **outbound** GitHub capabilities reuse a single platform-managed `GH_TOKEN`, including:
  - GitHub repository `clone / fetch / push` (uniformly using `https://github.com/...git` transport)
  - GitHub Pull Request / Project API calls (if explicitly required by a Workflow or platform action)
- `GH_TOKEN` is an encrypted Secret managed uniformly by the platform and must not be scattered across ProjectRepo, scripts, user shell profiles, or repo files.
- `GH_TOKEN` can come from three sources:
  - The platform initiates GitHub Device Flow authorization
  - Import the current user's existing `gh auth token`
  - The user manually pastes a Personal Access Token for platform management

**Scope of Ownership:**

- The default ownership is **organization-level**: ProjectRepos and GitHub PR automation under the same Organization share one `GH_TOKEN` by default.
- A Project may explicitly configure its own GitHub outbound credential to override the Organization default; if not overridden, it always falls back to the organization-level `GH_TOKEN`.
- A GitHub token as a "machine-level source of truth" is not defined. `gh auth status`, credential helpers, and SSH keys on a Machine are only signals for observability or compatibility, not sources of truth for platform auth configuration.
- At any given time, a Project is allowed to resolve to **exactly one valid GitHub outbound credential**, to avoid different paths like clone / pr hitting different tokens.

**Storage Model:**

- `GH_TOKEN` must be stored in the platform Secret storage layer as **encrypted static configuration**; it must not be stored in plaintext in ProjectRepo, Machine, Workflow, Provider, script repository files, or user shell profiles.
- By default, the UI only displays:
  - Source (`device_flow` / `gh_cli_import` / `manual_paste`)
  - Ownership scope (organization / project)
  - Most recent probe time
  - Permission probe result
  - Most recent error
- The UI does not echo the full token; at most it shows a masked suffix in a fixed format, such as `ghu_xxx...ABCD`.
- Configuration export, diagnostic logs, ActivityEvent, AgentTraceEvent, and Hook output must all apply uniform redaction to `GH_TOKEN`.

**Source Semantics:**

- `device_flow`: the platform completes the GitHub authorization flow itself and ultimately saves a platform-managed token. Before OAuth App / Device Flow wiring is implemented, the UI may explicitly display this as deferred rather than pretending it is already usable.
- `gh_cli_import`: the platform reads the current value of `gh auth token` at import time and copies it for storage; after import completes, that token is **decoupled** from the local `gh` login state.
- `manual_paste`: the user explicitly pastes a token, and the platform stores it.
- Once the token has been saved by the platform, all subsequent scheduling and runtime behavior read from platform Secret storage rather than depending again on the output of the `gh auth token` command on the execution machine.

**Boundary and Observability Rules:**

- `gh auth status` only indicates the GitHub CLI login state on a given machine; it is an observability signal, not the source of truth for platform GitHub outbound authentication.
- Both the local `go-git` path and the remote shell `git` path must explicitly consume the platform-managed `GH_TOKEN`; they must not rely on the implicit assumption that "the machine might already have a credential helper configured".
- At remote runtime, `GH_TOKEN` may only be injected as an environment variable for the current controlled session and must not be written into the user's global shell profile, repository files, or persistent workspace configuration.
- After the platform saves / imports `GH_TOKEN`, it must immediately run one structured permission probe and produce results such as `valid / permissions / repo_access / checked_at`; the UI and scheduler read the probe results, not merely "the string is non-empty".

**Lifecycle State Machine:**

- `missing`
  - No GitHub outbound credential is configured for the current Organization / Project.
- `configured`
  - The token has been saved, but the first probe has not yet completed.
- `probing`
  - The platform is probing permissions / repo access.
- `valid`
  - The token is valid and satisfies the minimum permissions required by currently enabled GitHub capabilities.
- `insufficient_permissions`
  - The token is valid, but lacks the permissions required by currently enabled capabilities.
- `revoked`
  - GitHub returned 401 / the token is invalid / the token has been revoked.
- `error`
  - The platform cannot complete probing (network error, temporary GitHub API failure, inconsistent configuration, etc.).

**State Transition Rules:**

- Create / import token: `missing -> configured -> probing -> valid | insufficient_permissions | revoked | error`
- Manual "retest" or periodic probe: `valid | insufficient_permissions | error -> probing -> ...`
- User deletes token: any state -> `missing`
- User rotates token: the old token exits active configuration immediately, and the new token starts again from `configured -> probing -> ...`

**Runtime Projection Rules:**

- Local workspace management:
  - `go-git` clone / fetch / push must explicitly use the GitHub transport auth resolved by the platform; do not assume the library will automatically read `GH_TOKEN`.
- Remote workspace management:
  - shell `git clone / fetch / push`
  - `gh issue / pr / project`
  The commands above may read `GH_TOKEN` only through **temporary environment variables of the current controlled session**.
- The Agent CLI itself does **not inherit** `GH_TOKEN` by default, unless the current step explicitly needs to call the GitHub API / git transport and the platform explicitly projects it to that subprocess.
- `GH_TOKEN` must not be written into:
  - `.git/config`
  - `.env`
  - workspace files
  - shell profiles
  - Hook script templates

**Capability Matrix:**

- Only using GitHub private repository clone / fetch / push:
  - Requires repo transport to be available
  - Does not require Issue / PR / Project API permission probing to pass
- Using GitHub Issue / PR automation:
  - Requires Issue / PR API permissions to be available
- Using GitHub Project:
  - Additionally requires Project permissions to be available
- When the scheduler determines whether "GitHub capability is available", it must resolve based on the **currently enabled features**, and must not raise all scenarios to the maximum permission requirement

**Minimum Permission Requirements:**

- Classic PAT / `gh` OAuth scope:
  - `repo`
  - `project` (only when OpenASE needs to create / update a GitHub Project)
- Fine-grained PAT:
  - Repository `Contents: write`
  - Repository `Pull requests: write`
  - Repository `Issues: write`
  - Repository `Metadata: read`
  - `Projects: write` (only when OpenASE needs to create / update a GitHub Project)

**Rotation & Revocation:**

- After the user replaces the token:
  - New tasks use only the new token
  - Running sessions are not required to hot-swap, but the old token must not be persisted again or written back to any machine
- If a probe determines that the token has been revoked:
  - The platform marks the state as `revoked`
  - New GitHub clone / pr related tasks are immediately blocked from starting
  - Tasks that have already started may fail and converge, but the error must explain that the GitHub outbound credential has become invalid
- Recommended periodic probe intervals:
  - Success state: between 30 minutes and 6 hours, configured by environment tier
  - Error state: exponential backoff to avoid continuously hitting the GitHub API

**Important: the "availability" of `AgentProvider` is not a static configuration field, but a runtime-derived state.**

- `cli_command`, `cli_args`, `auth_config`, `machine_id`, etc. are **static configuration**
- Whether the Provider can currently be scheduled for execution is a **runtime-derived result**
- The frontend and scheduler **must not** directly interpret "command exists in PATH" or "configuration fields are non-empty" as "Provider is available"

The Provider exposes the following derived fields externally (they can be returned in API responses and do not need to be persisted as configuration columns):

| Field | Type | Description |
|------|------|------|
| availability_state | Enum | `unknown / available / unavailable / stale` |
| available | Boolean | Compatibility boolean field; equivalent to `availability_state == available` |
| availability_checked_at | DateTime (nullable) | Time of the most recent L4 check used to determine Provider availability |
| availability_reason | String (nullable) | Reason for unavailability or staleness (such as `machine_offline`, `cli_missing`, `not_logged_in`, `stale_l4_snapshot`) |

**Rules for Determining Provider Availability:**

- `available`
  - `machine.status == online`
  - The most recent L4 Agent Environment check succeeded and has not expired
  - The corresponding adapter CLI is installed
  - The corresponding CLI authentication state is ready (such as `logged_in` or API key mode)
  - The startup configuration required by the Provider is complete (command, path, remote workspace, etc.)
- `unavailable`
  - The most recent L4 check clearly proves that the Provider cannot run
  - Or the bound Machine is not in `online`
- `unknown`
  - No trusted L4 check has ever completed, and the system does not yet have enough information
- `stale`
  - There was once a successful L4 snapshot, but it has exceeded its validity period and can no longer be used as a scheduling basis

**Default Expiration Window:**

- The default L4 check interval is 30 minutes
- `availability_state` becomes `stale` when `now - availability_checked_at > 2 * L4_interval`
- `stale`, like `unknown`, must not participate in scheduling

### 6.8 Agent (Execution Definition)

The Agent here is not "an occupiable worker in the scheduling pool", but rather "the definition of how a Provider runs in the current project". It answers:

- Which Provider to use
- Which Coding Agent CLI on which machine is bound
- What this role is called
- What static labels / default configuration this role has

A Workflow binds an Agent, meaning "which Agent definition drives execution for this Workflow". The true ephemeral runtime state is not attached to the Agent entity itself, but to an independent runtime record.

**Key Clarification: an Agent is not a single instance that can bind only one Ticket at a time.** The same Agent definition can drive multiple Tickets at different times, and can also produce multiple `AgentRun`s at the same time when concurrency limits allow. What has a one-to-one correspondence with a Ticket is `AgentRun`, not the Agent entity itself.

- A Ticket can have at most one `current_run_id` at any moment
- But the same Agent definition can correspond to multiple `AgentRun`s for multiple Tickets at the same time
- Whether those `AgentRun`s may run in parallel depends on global / provider / stage / workflow concurrency constraints, not on "the Agent can only be single-threaded"

To support quick overview in directory pages and APIs, the Agent definition may additionally expose a read-only aggregated `runtime` summary, such as active run count, summary status, and most recent heartbeat. But this is only a convenience summary, not the source-of-truth runtime view:

- When the same Agent definition has multiple concurrent `AgentRun`s, that summary must not be disguised as a single `current_run_id` / `current_ticket_id`
- Full runtime observability must be based on the `AgentRun` list or detail view
- Any Agent-level `runtime` field must explicitly express `summary / aggregate` semantics

**Agent no longer stores a user-entered `workspace_path`.** Working directories are uniformly derived by the platform at the Ticket level:

- Local Provider: `~/.openase/workspace/{org-slug}/{project-slug}/{ticket-identifier}`
- Remote Provider: `{machine.workspace_root}/{org-slug}/{project-slug}/{ticket-identifier}`

The first-level subdirectories under the directory are the multiple Repos involved in that Ticket.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| provider_id | FK | Owning Provider |
| project_id | FK | Associated Project |
| name | String | Agent definition name (such as `codex-coding`, `claude-reviewer`) |
| is_enabled | Boolean | Whether Workflow is allowed to continue binding and running |
| total_tokens_used | BigInt | Cumulative token consumption attributed to this Agent definition |
| total_tickets_completed | Integer | Cumulative completed ticket count attributed to this Agent definition |

### 6.8.1 AgentRun (Runtime Session)

AgentRun is the real runtime slot. Each Workflow execution creates a new AgentRun, and its lifecycle is managed by the orchestrator. It is the direct target of semaphores and runtime health checks.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| agent_id | FK | Associated Agent definition |
| workflow_id | FK | Workflow used for this run |
| ticket_id | FK | Ticket served by this run |
| provider_id | FK | Redundant record for concurrency statistics and audit |
| status | Enum | launching / ready / executing / completed / errored / terminated |
| session_id | String | Read-only session / thread ID after Runtime is established successfully |
| runtime_started_at | DateTime | Time when Runtime started successfully |
| last_error | String | Most recent Runtime startup / session error (empty when healthy) |
| last_heartbeat_at | DateTime | Last heartbeat time managed by Runtime |
| current_step_status | String (nullable) | Current human-understandable action stage (such as `planning` / `editing` / `running_tests` / `opening_pr`) |
| current_step_summary | String (nullable) | Human-readable summary of the current action stage |
| current_step_changed_at | DateTime (nullable) | Most recent action stage transition time |
| completion_summary_status | String (nullable) | Terminal run summary generation status: `pending` / `completed` / `failed` |
| completion_summary_markdown | Text (nullable) | Human-readable Markdown summary of the terminal run |
| completion_summary_json | JSON | Summary result metadata (provider / model, etc.) |
| completion_summary_input | JSON | Structured summary input frozen at terminal time (steps, commands, approvals, output excerpts, file snapshots, etc.) |
| completion_summary_generated_at | DateTime (nullable) | Time when summary was successfully generated |
| completion_summary_error | Text (nullable) | Reason for summary failure, for troubleshooting |

**Coding Agent Runtime Readiness Contract**

- `launching` only means the scheduler has created this run; it does not mean Codex has started successfully.
- The deterministic conditions for successful startup are: `status == ready || status == executing`, `session_id != ""`, and `last_heartbeat_at` is populated and recent enough.
- Catalog CRUD must not allow users to manually write `session_id`, `runtime_started_at`, `last_error`, or `last_heartbeat_at`; these fields may only be written by the Runtime startup path.
- `current_step_status` is not the same thing as `status`: the former indicates "what the Agent is doing now", while the latter indicates "whether the Runtime has started / is executing / has failed".
- The frontend must use these Runtime fields to show `waiting -> launching -> ready -> failed`, and must not interpret "no activity text" as startup failure.
- When `AgentRun.status` enters `completed` / `errored` / `terminated`, the platform must asynchronously freeze the structured summary input and final file-change snapshot of this run and set `completion_summary_status` to `pending`; this process must not block the original terminal path of the runtime.
- The terminal summary is the platform's own best-effort post-processing and does not change the original `AgentRun.status` / Ticket business status; on failure it only updates `completion_summary_status=failed` and `completion_summary_error`.
- By default, the terminal summary is generated using the same Provider category as the original run; if that provider becomes unavailable after the run ends, the original run remains in its terminal state and the summary enters a failed state.

### 6.9 Manual Review Hold

Tickets that require human confirmation no longer create a separate `ApprovalGate` entity. Entering a manual review state must come from an explicit status update by an Agent or a human; after a normal turn ends, Runtime is only responsible for stopping or continuing to run, and does not automatically move the ticket to `in_review` / `awaiting_review`.

### 6.10 ScheduledJob (Scheduled Task)

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning Project |
| name | String | Task name |
| cron_expression | String | Cron expression |
| workflow_id | FK | Workflow used |
| ticket_template | JSON | Ticket template |
| is_enabled | Boolean | Whether enabled |
| last_run_at | DateTime | Last execution |
| next_run_at | DateTime | Next execution |

### 6.11 AgentTraceEvent (Agent Fine-Grained Runtime Trace)

The large amount of fragmented output generated by the Agent CLI at runtime should not directly pollute the business activity stream. OpenASE uniformly normalizes this kind of fine-grained runtime signal into the `AgentEvent` protocol and persists it as `AgentTraceEvent`.

This layer is the **troubleshooting and real-time observation layer**, not the project activity stream.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning Project |
| ticket_id | FK | Associated ticket |
| agent_id | FK | Associated Agent |
| agent_run_id | FK | Associated AgentRun |
| sequence | BigInt | Strictly monotonic sequence number within the same AgentRun |
| provider | String | Source Provider (`codex` / `claude` / `gemini` / `custom`) |
| kind | String | Normalized event type (such as `assistant_delta`, `assistant_snapshot`, `tool_call_started`, `tool_call_finished`, `command_output_delta`, `runtime_notice`, `error`) |
| stream | String | Logical stream name (`assistant` / `tool` / `command` / `system`) |
| text | Text | Text fragment that can be displayed directly; may be empty |
| payload | JSON | Provider-specific additional information (tool arguments, item id, phase, original stream metadata, etc.) |
| created_at | DateTime | Event time |

**Constraints:**

- `AgentTraceEvent` is only for the Agent console, debugging panels, and replaying runtime details, and does not enter the Dashboard Activity Feed.
- Both `delta` and `snapshot` may be persisted, but `sequence` must guarantee stable ordering within the same `AgentRun`.
- Raw events from any Provider must first be mapped into the unified `AgentEvent`, and then written into `AgentTraceEvent`; the frontend must not directly parse the private protocol of any CLI.

### 6.11.1 AgentStepEvent (Human-Readable Action Stream)

Not every token and not every output delta is appropriate for humans to read. OpenASE needs to extract "action stage transitions" from `AgentTraceEvent` to form a more stable human-readable stream.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning Project |
| ticket_id | FK | Associated ticket |
| agent_id | FK | Associated Agent |
| agent_run_id | FK | Associated AgentRun |
| step_status | String | Action stage (such as `planning` / `editing` / `running_tests` / `opening_pr` / `waiting_review`) |
| summary | Text | Human-readable summary (such as "analyzing repository structure", "running frontend CI", "preparing to create PR") |
| source_trace_event_id | FK (nullable) | Original TraceEvent that triggered this stage change |
| created_at | DateTime | Stage transition time |

**Rules:**

- A new `AgentStepEvent` is appended only when `step_status` changes.
- `AgentRun.current_step_status / current_step_summary / current_step_changed_at` is the current snapshot; `AgentStepEvent` is the historical timeline.
- `AgentStepEvent` is the default main timeline shown on the Agent detail page; it is not the same as business-level Activity.

### 6.11.2 ProjectUpdateThread / ProjectUpdateComment (Project Progress Updates)

The project needs a high-signal-to-noise human progress panel to express "what the current project status is, why that judgment is made, and what to watch next". This line must be an **independent first-class resource**; the body and comments must not be stuffed into `ActivityEvent.metadata`.

**ProjectUpdateThread**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning Project |
| status | Enum | `on_track` / `at_risk` / `off_track` |
| title | String | Update title |
| body_markdown | Text | Currently effective Markdown body |
| created_by | String | Creator identifier |
| created_at | DateTime | Creation time |
| updated_at | DateTime | Most recent update time |
| edited_at | DateTime | Most recent body edit time; null if never edited |
| edit_count | Integer | Number of times the body has been saved, excluding the initial creation |
| last_edited_by | String | Most recent editor |
| is_deleted | Boolean | Soft-delete marker |
| deleted_at | DateTime | Deletion time |
| deleted_by | String | Deleting operator |
| last_activity_at | DateTime | Most recent activity time of this thread; thread edit, status change, and comment append must all advance it |
| comment_count | Integer | Current comment count; read optimization field |

**ProjectUpdateComment**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| thread_id | FK | Owning ProjectUpdateThread |
| body_markdown | Text | Currently effective Markdown body |
| created_by | String | Creator identifier |
| created_at | DateTime | Creation time |
| updated_at | DateTime | Most recent update time |
| edited_at | DateTime | Most recent body edit time; null if never edited |
| edit_count | Integer | Number of times the body has been saved, excluding the initial creation |
| last_edited_by | String | Most recent editor |
| is_deleted | Boolean | Soft-delete marker |
| deleted_at | DateTime | Deletion time |
| deleted_by | String | Deleting operator |

**Version History and Read Rules**

- `ProjectUpdateThreadRevision` stores historical versions of the thread body and status; `revision_number=1` must be written at creation time.
- `ProjectUpdateCommentRevision` stores historical versions of the comment body; `revision_number=1` must be written at creation time.
- Every save of a thread/comment body must append a revision; overwriting history in place is forbidden.
- Deletion uses soft delete; by default the UI preserves the timeline position and shows a deleted placeholder, and must not hard-remove the record directly.
- The project-level `Updates` page must be sorted by `last_activity_at DESC`, so the most recently discussed progress items float to the top.
- `Updates` is the "source of truth for manually curated project progress"; `ActivityEvent` may only append summary-style side-channel events and cannot store thread/comment body content.
- SSE refresh may reuse the project activity stream, but concurrent observers must be able to see new update / comment / delete / status change without requiring manual refresh.

### 6.11.3 ActivityEvent (Business Activity Event)

`ActivityEvent` records only truly important business events at the project, ticket, and orchestration layers. It is the only activity signal source for the Dashboard, project Activity page, and Ticket System Activity.

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning Project |
| ticket_id | FK | Associated ticket (nullable) |
| agent_id | FK | Associated Agent (nullable) |
| event_type | String | Event type |
| message | Text | Event description |
| metadata | JSON | Additional data |
| created_at | DateTime | Event time |

**`event_type` Naming Rules (Strict)**

- Although `event_type` is physically stored as `String`, logically it must be treated as a **controlled enum**; when written to the database, returned through the API, or pushed through SSE, the same canonical catalog must be used
- The naming format is uniformly: `lowercase.dot.separated`
- Mixing synonymous aliases is not allowed; for example, `status_changed` / `ticket.status_changed`, `pr_opened` / `pr.opened`, and `hook_failed` / `hook.failed` must not appear at the same time
- The UI may humanize canonical values into friendlier labels, but it **must not** rewrite the real values at the wire/storage layer
- At repository / service / projector boundaries, values must be parsed into domain enums; writing an unknown type must error, and reading an unknown historical value must explicitly classify it as `unknown` with diagnostic logging, not silently guess

**Canonical ActivityEvent Catalog**

`ActivityEvent` must remain coarse-grained. The only `event_type` values allowed to enter this layer are those in the following table:

| event_type | Meaning | Scope | Required metadata |
|-----------|------|----------|---------------|
| `ticket.created` | Ticket created for the first time | ticket required | Optional: `identifier`, `title`, `created_by` |
| `ticket.updated` | User-visible business fields of the ticket were modified, but no status transition occurred | ticket required | `changed_fields[]`; optional: `updated_by` |
| `ticket.status_changed` | Ticket status transitioned | ticket required | `from_status_id`, `from_status_name`, `to_status_id`, `to_status_name`; optional: `changed_by` |
| `ticket.completed` | Ticket reached a finish status and is considered complete | ticket required | `status_id`, `status_name`; optional: `completed_by` |
| `ticket.cancelled` | Ticket was explicitly cancelled by a human or the platform | ticket required | Optional: `cancelled_by`, `reason` |
| `ticket.retry_scheduled` | The platform has scheduled the next backoff retry for this ticket | ticket required | `attempt_count`, `backoff_seconds`, `next_retry_at`, `reason` |
| `ticket.retry_paused` | Ticket retry has been paused, waiting for human handling or external conditions to recover | ticket required | `pause_reason`; optional: `stall_count`, `consecutive_errors`, `threshold` |
| `ticket.budget_exhausted` | Ticket budget is exhausted and it enters paused-retry state | ticket required | `cost_amount`, `budget_usd` |
| `project_update_thread.created` | Project progress thread created | project required | `thread_id`, `status`; optional: `thread_title`, `created_by` |
| `project_update_thread.edited` | Project progress thread body edited | project required | `thread_id`; optional: `thread_title`, `edited_by` |
| `project_update_thread.deleted` | Project progress thread soft-deleted | project required | `thread_id`; optional: `thread_title`, `deleted_by` |
| `project_update_thread.status_changed` | Risk status of the project progress thread changed | project required | `thread_id`, `from_status`, `to_status`; optional: `thread_title`, `edited_by` |
| `project_update_comment.created` | Project progress comment created | project required | `thread_id`, `comment_id`; optional: `created_by` |
| `project_update_comment.edited` | Project progress comment edited | project required | `thread_id`, `comment_id`; optional: `edited_by` |
| `project_update_comment.deleted` | Project progress comment soft-deleted | project required | `thread_id`, `comment_id`; optional: `deleted_by` |
| `agent.claimed` | An Agent successfully claimed the ticket | ticket + agent required | Optional: `run_id`, `agent_name` |
| `agent.launching` | The runtime/session for this ticket is starting | ticket + agent required | Optional: `run_id`, `session_id` |
| `agent.ready` | The runtime/session is ready and can continue executing turns | ticket + agent required | Optional: `run_id`, `session_id` |
| `agent.paused` | The runtime/session was paused by a human or the system | ticket + agent required | Optional: `run_id`, `reason` |
| `agent.failed` | The runtime/session failed to start or execute and needs human attention or a retry path | ticket + agent required | Optional: `run_id`, `error` |
| `agent.completed` | The Agent successfully completed the execution goal for this ticket | ticket + agent required | Optional: `run_id`, `result` |
| `agent.terminated` | The runtime/session has ended and ownership has been released | agent required, ticket should usually be provided | Optional: `run_id`, `reason` |
| `hook.started` | A Hook started executing | project required; ticket/workflow optional depending on scope | `hook_name`, `hook_scope` (`ticket` / `workflow`) |
| `hook.passed` | A Hook executed successfully | project required; ticket/workflow optional depending on scope | `hook_name`, `hook_scope`; optional: `duration_ms` |
| `hook.failed` | A Hook execution failed | project required; ticket/workflow optional depending on scope | `hook_name`, `hook_scope`, `error`; optional: `duration_ms`, `blocking` |
| `pr.linked` | A RepoScope associated with the ticket recorded a PR link | ticket required | `repo_id` or `repo_name`; optional: `pull_request_url`, `pull_request_number` |

**Additional Constraints:**

- `message` is a human-readable summary; it is not a machine contract, and the frontend must not infer the type from `message` text
- Structured semantics must go into `metadata`; for example, a status change must include `from/to`, and cannot be only a sentence like "moved to done" in `message`
- A single business fact may have only one canonical type; for example, "claim ticket" must be uniformly recorded as `agent.claimed`, and an equivalent `ticket.assigned` must not be emitted elsewhere
- `ActivityEvent` is an append-only audit stream; once written, an event must not be "rewritten into another type"

**The following content must not be written directly into ActivityEvent:**

- Token-level or text-delta-level CLI output
- Continuous scrolling `stdout/stderr` text
- Raw Provider reasoning / internal chain-of-thought text
- High-frequency heartbeat mirror text, unless promoted into an explicit business alert

### 6.11.4 Three-Layer Event Model and Page Consumption Rules

OpenASE uses a strict three-layer event model:

1. `AgentTraceEvent`
   For "machines and troubleshooting", recording fine-grained CLI runtime traces.
2. `AgentStepEvent`
   For "operator understanding", recording human-readable action stage transitions.
3. `ActivityEvent`
   For "product and business", recording truly important system activities.

**Page Consumption Rules:**

- Output/Trace panel in the Agent console: reads `AgentTraceEvent`
- Action Timeline in the Agent console: reads `AgentStepEvent` + `AgentRun.current_step_*`
- Dashboard Activity Feed: reads only `ActivityEvent`
- Project-level Activity page: reads only `ActivityEvent`
- Main timeline of the Ticket detail page: reads the `TicketTimelineItem` projection; that projection is composed from `Ticket.description`, `TicketComment`, and `ActivityEvent` filtered by `ticket_id`
- Comment history panel of the Ticket detail page: reads `TicketCommentRevision`
- Hook History in the Ticket detail page: preferentially shows Hook execution records; when necessary it can be projected from `ActivityEvent(event_type like 'hook.%')`

**This means:**

- No matter how many Agents there are, the Dashboard will not be flooded by token-level output
- Ticket Activity only carries "what happened", not "what scrolled in the terminal"
- The primary Ticket Detail experience is a GitHub Issue-style timeline, not a two-panel composition of "comments in one block, system events in another"
- If you want details, go to the Agent console and look at Trace; if you want process, look at Step; if you want business results, look at Activity

### 6.11.4 Runtime Event Projection Pipeline

To ensure consistent behavior across multiple CLI Providers (Codex / Claude / Gemini / Custom), runtime events must flow in the following order:

1. The adapter receives raw provider events from the CLI runtime stream.
2. The Event Normalizer maps provider-private events into the unified `AgentEvent` protocol.
3. The Trace Persister appends each `AgentEvent` into `AgentTraceEvent(sequence++)`.
4. The Step Projector derives `current_step_status` / `current_step_summary` from the latest `AgentEvent`; only when the stage actually changes does it update `AgentRun.current_step_*` and append `AgentStepEvent`.
5. The Lifecycle / Business Projector writes into `ActivityEvent` only when a coarse-grained business event occurs, such as `agent.claimed`, `ticket.status_changed`, `hook.failed`, `pr.opened`.
6. The EventProvider / SSE layer dispatches separately by event layer:
   - Trace stream -> `AgentTraceEvent`
   - Step stream -> `AgentStepEvent`
   - Activity stream -> `ActivityEvent`

**Forbidden Behavior:**

- The frontend directly consuming raw provider events and guessing state by itself
- The UI side aggregating multiple traces and then inferring business Activity in reverse
- Using `ActivityEvent` to backfill or replace `AgentRun.current_step_*`
- Incorrectly promoting token / delta output into business Activity

---
## Chapter 7 The Relationship Between the Orchestrator and Tickets

### 7.1 Core Principle: The Orchestrator Does Not Handle Status Transitions

**OpenASE is not a project management tool; it is an orchestrator.** It does not care how tickets move across the Kanban board—the user can define any number of `Custom Status` values and freely drag tickets between columns. The orchestrator only cares about one question: **Should this ticket be taken for execution?**

The answer is determined by the workflow configuration, not by a hard-coded state machine.

### 7.2 Workflow Defines Pickup / Finish Rules

Each Workflow configures two required status sets in the database:

```json
{
  "pickup_status_ids": ["Todo", "Ready for AI"],
  "finish_status_ids": ["Done", "Needs Review"]
}
```

The status names above are only for semantic illustration. Real configuration is stored as a non-empty set of `TicketStatus.id` in the database and is maintained via UI selection.

**The orchestrator's scheduling logic is simplified to:**

```go
func (s *Scheduler) runTick(ctx context.Context) {
    // Get all active workflows
    workflows, _ := s.workflowRepo.ListActive(ctx)

    for _, wf := range workflows {
        // Each workflow has its own pickup status set
        pickupStatusIDs := wf.PickupStatusIDs

        // Scan tickets in any pickup status
        candidates, _ := s.ticketRepo.ListByStatusIDs(ctx, wf.ProjectID, pickupStatusIDs)

        for _, t := range candidates {
            // Check dependencies, concurrency, Agent definitions bound to the workflow, and provider semaphore
            // as well as whether the currently matched pickup status itself still has capacity
            if s.canDispatch(ctx, t, wf) {
                s.dispatch(ctx, t, wf)
            }
        }
    }
}
```

**That simple.** There is no hard-coded state transition diagram. The orchestrator only does three things:

1. **Scan**: Find tickets with `status ∈ workflow.pickup_status_ids`
2. **Run**: Create AgentRun according to Agent definitions bound to the workflow, run Hook
3. **Finish**:
   - If `finish_status_ids` has only 1 status, the orchestrator automatically moves the ticket to that status
   - If `finish_status_ids` contains multiple statuses, the Agent must explicitly choose one as target status through Platform API
   - If AgentRun errors, it is retried automatically with exponential backoff (there is no `failed` terminal state)

**There is no `"failed"` state.** A ticket can only finish in two ways: reaching `finish` (success), or being `cancel`ed by a human (voluntary abandonment). Agent execution failure only triggers retries—exponential backoff, budget deduction, error-rate alerts—but the ticket stays in pickup status and waits for the next attempt. Only a human can decide to give up on a ticket.

What status a ticket is in between, how users drag it, and how many columns the board has—the orchestrator does not care about specific names. The scheduling entry is explicitly defined by the workflow's `pickup_status_ids`, and `stage` is used only for terminal-state checks and whether dependencies have been cleared.

### 7.3 Typical Configuration Examples

**Standard software development flow:**

```yaml
pickup_statuses: ["Todo"]   # User drags ticket to Todo column → orchestrator takes over
finish_statuses: ["Done"]   # Agent completes → ticket jumps to Done column
# No fail status—failures auto-retry, only human cancel terminates
```

**Research flow (experiment verifier):**

```yaml
status:
  pickup: "Pending Experiment"    # After user finishes reviewing the idea, move to "Pending Experiment" → Agent takes over
  finish: "Experiment Completed"  # Agent finishes experiment → ticket jumps to "Experiment Completed"
  # Experiment failure is non-terminal—Agent sets status via Platform API (e.g. "Fail Pending Analysis")
```

**Ops flow (DevOps):**

```yaml
status:
  pickup: "Ready for Deploy"
  finish: "Deployed"
  # No fail—deploy failures auto-retry, alert notifications are sent to humans
```

**Code review flow:**

```yaml
status:
  pickup: "Ready for Review"      # After PR submission, user drags to "Ready for Review" → review Agent takes over
  finish: "Review Complete"
  # No fail—errors auto-retry
```

**Key insight: Different workflows in the same project can have different pickup/finish statuses.** This means different columns on the board are entry points for different roles: the "In Development" column is the pickup for the coding workflow, the "In Testing" column is the pickup for the test workflow, and the "Ready for Deployment" column is the pickup for the deploy workflow. Ticket movement on the board naturally forms a pipeline:

```
Backlog → In Development → [coding Agent runs] → In Testing → [test Agent runs] → Ready for Deployment → [deploy Agent runs] → Deployed
                ↑ pickup                         ↑ pickup                             ↑ pickup
                coding Workflow                  test Workflow                        deploy Workflow
```

### 7.4 Status During Execution

The orchestrator needs to know whether a ticket is currently being processed (to avoid duplicate dispatching). This is not implemented through Custom Status, but through the ticket’s `current_run_id` field:

- `current_run_id == null`: not claimed, can be dispatched
- `current_run_id != null`: already claimed, skipped

The semantics here is **“each Ticket allows only one active current run”**, not **“each Agent is allowed to process only one Ticket at a time”**. `current_run_id` is a ticket-level occupancy marker to prevent the same ticket from being claimed repeatedly; it does not impose Agent-level concurrency limits.

```go
func (s *Scheduler) canDispatch(ctx context.Context, t *ticket.Ticket, wf *workflow.Workflow) bool {
    // Already claimed → skip
    if t.CurrentRunID != "" {
        return false
    }
    // Retry paused (budget exhausted / human paused) → skip
    if t.RetryPaused {
        return false
    }
    // In backoff (retry time has not reached yet) → skip
    if t.NextRetryAt != nil && time.Now().Before(*t.NextRetryAt) {
        return false
    }
    // Blocked by dependency → skip
    if s.ticketSvc.IsBlocked(ctx, t.ID) {
        return false
    }
    // Active run limit reached for current matched pickup status → skip
    if matchedPickupStatus.MaxActiveRuns > 0 &&
       s.pool.ActiveCountForStatus(matchedPickupStatus.ID) >= matchedPickupStatus.MaxActiveRuns {
        return false
    }
    // Concurrency limit reached → skip
    if s.pool.ActiveCountForWorkflow(wf.ID) >= wf.MaxConcurrent {
        return false
    }
    return true
}
```

After Agent completes:

```go
// Agent completed successfully
t.CurrentRunID = ""                        // Release the current AgentRun slot
t.AttemptCount = 0                         // Reset count
t.ConsecutiveErrors = 0                    // Reset consecutive error count
if len(wf.FinishStatusIDs) == 1 {
    t.StatusID = wf.FinishStatusIDs[0]     // Single finish: auto-move to the only target status
}
// Multiple finishes: Agent must select one target status from workflow.finish_status_ids via Platform API
s.ticketRepo.Save(ctx, t)
```

After Agent error—**there is no `failed` terminal state, only retries**:

```go
// Agent error (hook failure, CLI crash, stall timeout, etc.)
t.CurrentRunID = ""                        // Release the current AgentRun slot
t.AttemptCount++
t.ConsecutiveErrors++

// Compute backoff: 10s × 2^(attempt-1), max 30 minutes
backoff := min(10 * time.Second * (1 << (t.AttemptCount - 1)), 30 * time.Minute)
t.NextRetryAt = time.Now().Add(backoff)

// Ticket stays in pickup status, does not move away. After backoff reaches, next tick reclaims
// No max_attempts limit—Agent keeps retrying until:
//   1. success (move to finish)
//   2. budget exhausted (pause retrying and notify human)
//   3. human manually cancels

// Budget check
if t.CostAmount >= t.BudgetUSD && t.BudgetUSD > 0 {
    t.RetryPaused = true   // Pause retries, not failed
    t.PauseReason = "budget_exhausted"
    // Notify human: "ASE-42 budget exhausted ($5.00/$5.00), add more budget?"
    s.notifyBudgetExhausted(ctx, t)
}

// High error rate alert (sliding window)
if t.ConsecutiveErrors >= 3 {
    // 3 consecutive failures → notify human, but do not stop retries
    s.notifyHighErrorRate(ctx, t)
}

s.ticketRepo.Save(ctx, t)
```

**Backoff strategy:**

| Attempt | Backoff | Cumulative wait |
|---------|---------|-----------------|
| 1 | 10s | 10s |
| 2 | 20s | 30s |
| 3 | 40s | 1m10s |
| 4 | 80s | 2m30s |
| 5 | 160s | 5m10s |
| 6 | 320s | 10m30s |
| 7+ | 30m (cap) | retry every 30 minutes |

**Three non-terminal pause mechanisms:**

| Pause reason | Trigger | Behavior | How to resume |
|--------------|---------|----------|---------------|
| Budget exhausted | `cost_amount >= budget_usd` | Stop retries and notify human | Human adds budget: `openase ticket set-budget ASE-42 10.00` |
| Human pause | User clicks “pause” in UI | Stop retries | User clicks “resume” in UI |
| Dependency blocked | A blocked ticket has not completed | Not participating in scheduling | Auto-resumes after dependency ticket is completed |

Note: **all three are not `failed`—the ticket remains on the board and humans can resume it anytime.**

**Critical rule: any `status_id` change clears `current_run_id`.** Whether the orchestrator auto-changes status, Agent changes status through Platform API, or a human manually drags on the board—whenever `status_id` changes, `current_run_id` is cleared. This ensures that the ticket can be re-claimed whenever it returns to any pickup column.

```go
// internal/httpapi/ticket_api.go — when a human drags a ticket in the UI
func (h *TicketHandler) UpdateStatus(c echo.Context) error {
    ...
    if newStatusID != t.StatusID {
        t.StatusID = newStatusID
        t.CurrentRunID = ""       // Clear current run occupancy when status changes
        t.ConsecutiveErrors = 0   // Reset error count after human intervention
        t.RetryPaused = false     // Human drag = manual resume
    }
    ...
}
```

### 7.5 Approval

Approval is no longer modeled as a separate entity. The `on_complete` lifecycle Hook is only a quality gate before explicit status advancement; the ticket must move to a non-`pickup` state (usually `in_review`) through an explicit status update from an Agent or a human.

```go
// Explicit status-advance requests are allowed into manual review state only after
// passing on_complete Hook validation
t.StatusID = requestedStatusID
s.ticketRepo.Save(ctx, t)
s.notifier.Send(ctx, "ticket.in_review", ...)
```


## Chapter 8 Hook System: Workflow Hook and Ticket Hook

OpenASE has two completely different types of Hooks, and they must be strictly distinguished:

- **Workflow Hook**: lifecycle events of the Workflow/Harness itself (load, activate, deactivate)
- **Ticket Hook**: events during the execution process of an individual ticket (claim, start, complete, error, cancel)

Both types of Hooks are defined in the Harness YAML Frontmatter, and the script files follow the project repository.

### 8.1 Workflow Hook (Harness lifecycle)

Workflow Hook is triggered when the **Harness itself is loaded/unloaded**, independent of a specific ticket.

| Hook | Trigger time | Typical use |
|------|---------|---------|
| `workflow.on_activate` | When a Workflow is first published or re-enabled | Validate dependency environment (whether Agent CLI is available, repository reachability); warm up cache |
| `workflow.on_deactivate` | When a Workflow is disabled or a Harness is deleted | Clean up global resources; notify the team that this role has been decommissioned |
| `workflow.on_reload` | When a new version of Workflow is released | Validate new configuration legality; notify the team that the Harness has been updated |

```yaml
---
workflow_hooks:
  on_activate:
    - cmd: "claude --version"    # Verify Claude Code is available
      on_failure: block          # If unavailable, the Workflow does not activate
    - cmd: "git ls-remote origin"
      on_failure: warn
  on_reload:
    - cmd: "echo 'Harness v{{ workflow.version }} loaded'"
      on_failure: ignore
---
```

**The execution context of a Workflow Hook is not the ticket workspace**—it runs in the service-managed project-level runtime context. If a Hook needs access to a code repository, the platform can explicitly check out the repo required by the Hook declaration into a lightweight project workspace; however, Workflow/Skill control plane editing itself does not rely on whether any repo workspace pre-exists.

**Prerequisites:**

- Workflow Hook triggers come from workflow lifecycle events in the control plane, not from repository `.openase/harnesses` file watching
- Opening, editing, and publishing Workflows and Skills does not require any repo workspace to pre-exist
- If a Workflow Hook command itself needs repository access, the platform should separately resolve the required code context before executing that Hook; this is a Hook execution dependency and is not a precondition of Workflow control plane storage
- `workflow.on_activate` can validate whether git transport, CLI, and dependency caches are available, but does not decide whether the Workflow can be persisted

### 8.2 Ticket Hook (ticket execution lifecycle)

Ticket Hook is triggered during **each execution of each ticket**, and each ticket run is an independent execution context.

| Hook | Trigger time | Blocking behavior | Typical use |
|------|---------|---------|---------|
| `ticket.on_claim` | After the orchestrator takes ownership of a ticket and before Agent startup | On failure, release the ticket and retry with backoff | Prepare working copy, install dependencies, supplement runtime context, decrypt secrets |
| `ticket.on_start` | Before Agent CLI subprocess starts | On failure, trigger retry | Fetch latest code, check branch conflicts, validate Agent availability |
| `ticket.on_complete` | When Agent declares task completion | On failure, block progress, Agent receives feedback and continues | Run tests, lint, type checks, security scan |
| `ticket.on_done` | After ticket successfully reaches finish state | Does not block | Workspace cleanup, notification |
| `ticket.on_error` | After a single Agent execution error (before each retry) | Does not block | Error logging, alert notification, partial cleanup |
| `ticket.on_cancel` | After user manually cancels a ticket | Does not block | Stop Agent, clean workspace, close unmerged PR |

```yaml
---
ticket_hooks:
  on_claim:
{% for repo in repos %}
    - cmd: "git fetch origin && if git rev-parse --verify origin/{{ repo.branch }} >/dev/null 2>&1; then git checkout -B {{ repo.branch }} origin/{{ repo.branch }}; else git checkout -B {{ repo.branch }} origin/{{ repo.default_branch }}; fi"
      workdir: "{{ repo.name }}"
      timeout: 60
{% endfor %}
    - cmd: "pnpm install --frozen-lockfile"
      workdir: "web"
      timeout: 300
  on_complete:
    - cmd: "bash scripts/ci/run-tests.sh"
      timeout: 600
      on_failure: block
    - cmd: "bash scripts/ci/lint.sh"
      timeout: 120
      on_failure: block
  on_done:
    - cmd: "bash scripts/ci/cleanup.sh"
      on_failure: ignore
  on_error:
    - cmd: "echo 'Error on attempt $OPENASE_ATTEMPT, retry after backoff' >> /tmp/errors.log"
      on_failure: ignore
---
```

In multi-repo tickets, `on_claim` should enter each repo in the `repos` list one by one and prepare branches in the corresponding working directory; it should no longer assume that only one default repo exists. In single-repo projects, this loop naturally expands to one command.

**Ticket Hook executes under the ticket workspace directory** (that is, the `~/.openase/workspace/{org}/{project}/{ticket}/` directory prepared by `on_claim`).

**Responsibility boundaries:**

- `ticket.on_claim` can execute preparatory actions such as `git fetch`, `checkout`, and dependency installation in an existing working copy
- `ticket.on_claim` is not responsible for creating any project-level cache layer; the scheduler prepares working copies directly from the remote repository
- In `ticket.on_start`, “fetch latest code” refers to `fetch` to the remote repository performed by the current working copy

### 8.2.1 Which operations require the latest code baseline

The platform uniformly fetches the latest code baseline directly from the remote repository.

- **Default path: direct fetch/checkout**
  - Create or restore Ticket Workspace
  - Execute Workflow/Ticket Hook that requires the latest baseline in a repository context
  - Any path that requires Agent to create a working branch based on the latest default branch
The core purpose of this definition is: by default, use the simplest and correct remote repository semantics, and stop maintaining additional intermediate-state state machines.

### 8.3 Key differences between the two Hook types

| Dimension | Workflow Hook | Ticket Hook |
|------|-------------|---------|
| Trigger granularity | Each Workflow lifecycle | Each ticket each execution |
| Execution context | Project-level runtime context (explicit repo checkout if necessary) | Ticket workspace |
| Trigger frequency | Very low (when Harness changes) | High (triggered by every ticket) |
| YAML key name | `workflow_hooks:` | `ticket_hooks:` |
| Available environment variables | `OPENASE_PROJECT_ID`, `OPENASE_WORKFLOW_NAME` | Full ticket context (`ticket_id`, `repos`, `agent`, etc.) |

### 8.4 Scripts follow repository

**All scripts called by Hooks are in the project repository**, and the platform directly checks out the corresponding repo when preparing the ticket workspace, so the working copy naturally has all Hook capabilities:

```
your-project/
├── scripts/
│   └── ci/
│       ├── run-tests.sh        # Ticket Hook call
│       ├── lint.sh
│       ├── typecheck.sh
│       └── cleanup.sh
├── .openase/
│   └── ...                     # This directory is no longer the authoritative source for Workflow / Skill
└── src/                        # Project source code
```

### 8.5 Ticket Hook execution environment

Each Ticket Hook command injects the following environment variables:

| Environment variable | Description | Example value |
|---------|------|--------|
| `OPENASE_TICKET_ID` | ticket UUID | `550e8400-...` |
| `OPENASE_TICKET_IDENTIFIER` | Human-readable ticket identifier | `ASE-42` |
| `OPENASE_WORKSPACE` | Ticket workspace root directory | `/home/openase/.openase/workspace/acme/payments/ASE-42` |
| `OPENASE_REPOS` | Repository list JSON | `[{"name":"backend","path":"/home/openase/.openase/workspace/acme/payments/ASE-42/backend"}]` |
| `OPENASE_AGENT_NAME` | Agent name | `claude-01` |
| `OPENASE_WORKFLOW_TYPE` | Workflow type | `coding` |
| `OPENASE_ATTEMPT` | Current attempt count | `1` |
| `OPENASE_HOOK_NAME` | Current Hook name | `on_complete` |

The `workdir` field specifies the subdirectory to execute (multi-repo scenario):

```yaml
ticket_hooks:
  on_claim:
    - cmd: "pnpm install --frozen-lockfile"
      workdir: "frontend"    # /home/openase/.openase/workspace/acme/payments/ASE-42/frontend/
    - cmd: "go mod download"
      workdir: "backend"     # /home/openase/.openase/workspace/acme/payments/ASE-42/backend/
```

Exit code: `0` = pass, `non-0` = failure. on_failure strategies: `block` (default) / `warn` / `ignore`.

### 8.6 Each ticket = independent Agent session (aligned with Symphony)

**Each ticket execution starts a brand-new, independent Agent CLI subprocess.** This aligns with Symphony design:

```
Ticket ASE-42 is claimed
  │
  ├── ticket.on_claim Hook (derive workspace copy, checkout branch, install dependencies)
  │
  ├── Start Agent CLI subprocess (bash -lc "claude -p ... --output-format stream-json")
  │   │
  │   ├── Turn 1: Send rendered Harness Prompt
  │   │   Agent reads requirements → writes code → submits
  │   │
  │   ├── Turn complete → Check whether ticket is still in pickup state
  │   │   Yes → Turn 2: send continuation guidance (do not resend the original Prompt, reuse thread context)
  │   │   No → end
  │   │
  │   ├── Turn 2: Agent continues working (same thread, keeps context)
  │   │   ...up to max_turns times
  │   │
  │   └── Agent CLI subprocess exits
  │
  ├── ticket.on_complete Hook (make test, lint, ...)
  │   pass → move to finish state
  │   fail → back off, re-claim after 1 second, Agent receives "previous on_complete failure reason" as continuation context
  │
  └── ticket.on_done Hook (cleanup, notification)
```

**Key details (see Symphony Section 10.2-10.3):**

- **Turn 1 sends full Prompt**: Harness content rendered by Jinja2, including ticket description, repository path, and work boundaries
- **Turn 2+ sends continuation guidance**: does not resend the original Prompt (already in thread history), only sends “previous on_complete failed due to ... please fix and retry” or “continue with remaining work”
- **Same `thread_id` reuse**: Agent CLI subprocess stays alive across multiple Turns and is not restarted
- **session_id = `<thread_id>-<turn_id>`**: each Turn has an independent ID for tracing and logging
- **Stall detection**: 5 minutes with no Agent events → kill subprocess → after backoff restart a new subprocess

**Additional Symphony-level runtime semantics to add:**

- **Start Session only once**
  - `initialize -> initialized -> thread/start`
  - Create only one thread within the same worker lifecycle
- **Run Turn multiple times**
  - Repeatedly call `turn/start` on the same thread
  - After each completed turn, re-read the latest status of that ticket from the tracker
- **Normal exit does not equal ticket completion**
  - After a turn completes normally, if the ticket is still in active state, the scheduler should schedule a continuation retry after 1 second
  - Only when the ticket leaves active state, enters a terminal state, or reaches `max_turns` does the upper layer take over, and only then is this worker lifecycle truly ended
- **max_turns is the upper bound for a single worker lifecycle**
  - Reaching the limit does not directly finish the ticket
  - Instead, control is handed back to the orchestrator, which decides whether to continue the next agent run
- **Turn-to-turn context has exactly two sources**
  - Codex thread history
  - Current issue workspace / workpad / tracker latest state
  - The full original Prompt should not be re-concatenated in continuation turns

### 8.7 Relationship with Agent CLI internal Hooks

| Layer | Hook source | Control scope |
|------|---------|---------|
| **OpenASE Workflow Hook** | Harness `workflow_hooks:` | Workflow lifecycle (load/unload) |
| **OpenASE Ticket Hook** | Harness `ticket_hooks:` | Ticket execution lifecycle (claim → done) |
| **Agent CLI Hook** | `.claude/settings.json` PreToolUse/PostToolUse | Agent internal tool calls (before/after each Edit/Bash) |

The three layers are complementary: Workflow Hook ensures environment availability, Ticket Hook ensures ticket delivery quality, and Agent CLI Hook ensures real-time execution compliance in Agent behavior.

---
## Chapter 9 Observability Design

OpenASE manages the execution of real engineering tasks by AI Agents—cost, quality, and efficiency all need to be quantified. Observability is not monitoring added afterward; it is part of the product’s core functionality.

### 9.1 Three-Pillar Architecture

OpenASE adopts OpenTelemetry as a unified observability framework, supporting the three pillars of Traces, Metrics, and Logs. Data is exported to user-selected backends (Jaeger, Prometheus, Grafana, Datadog, etc.), and OpenASE does not include built-in storage itself—like the database, the observability backend is infrastructure provided by the user.

For individual users who do not need complete observability, the default implementation is NoopTracer + in-memory Metrics (shown only in the Web UI dashboard), with zero external dependencies.

### 9.2 Core Metric System

**Ticket Metrics——Measure Output**

| Metric Name | Type | Labels | Description |
|--------|------|------|------|
| `openase.ticket.created_total` | Counter | project, workflow_type, source | Total number of tickets created (by source: manual / scheduled / github_issue / api) |
| `openase.ticket.completed_total` | Counter | project, workflow_type, outcome | Total number of completed tickets (outcome: done / cancelled) |
| `openase.ticket.cycle_time_seconds` | Histogram | project, workflow_type | Ticket cycle time (total duration from todo to done) |
| `openase.ticket.agent_time_seconds` | Histogram | project, workflow_type | Actual agent execution time (cumulative in_progress duration, excluding wait-for-approval time) |
| `openase.ticket.attempts` | Histogram | project, workflow_type | Distribution of attempt count per ticket |
| `openase.ticket.queue_depth` | Gauge | project, workflow_type, status | Number of tickets in each status (real-time snapshot) |
| `openase.ticket.stall_total` | Counter | project, workflow_type | Number of stalls |

**Agent Metrics——Measure Resources**

| Metric Name | Type | Labels | Description |
|--------|------|------|------|
| `openase.agent.active` | Gauge | project, provider, adapter_type | Current active AgentRun count |
| `openase.agent.utilization_ratio` | Gauge | project, provider | Provider concurrency utilization (active_runs / max_parallel_runs) |
| `openase.agent.session_duration_seconds` | Histogram | provider, adapter_type | Duration of a single Agent session |
| `openase.agent.tokens_used_total` | Counter | provider, model, direction | Token consumption (direction: input / output) |
| `openase.agent.cost_usd_total` | Counter | provider, model, project | Cumulative API cost (USD) |
| `openase.agent.cost_usd_per_ticket` | Histogram | provider, workflow_type | Per-ticket cost distribution |
| `openase.agent.heartbeat_age_seconds` | Gauge | agent_id | Seconds since last heartbeat (>300 triggers stall) |

**Hook Metrics——Measure Quality Gates**

| Metric Name | Type | Labels | Description |
|--------|------|------|------|
| `openase.hook.execution_total` | Counter | hook_name, outcome | Total number of hook executions (outcome: pass / error / timeout) |
| `openase.hook.duration_seconds` | Histogram | hook_name | Distribution of hook execution durations |
| `openase.hook.block_total` | Counter | hook_name | Number of times state progression was blocked by a Hook (on_complete failure preventing ticket from entering review) |

**Orchestrator Metrics——Measure System Health**

| Metric Name | Type | Labels | Description |
|--------|------|------|------|
| `openase.orchestrator.tick_duration_seconds` | Histogram | — | Duration of each scheduling Tick |
| `openase.orchestrator.tickets_dispatched_total` | Counter | workflow_type | Number of tickets dispatched in each Tick |
| `openase.orchestrator.tickets_skipped_total` | Counter | reason | Number of skipped tickets (reason: blocked / no_agent / max_concurrency) |
| `openase.orchestrator.workers_active` | Gauge | — | Current active Worker count |
| `openase.orchestrator.retry_total` | Counter | strategy | Total retries (strategy: quick / exponential / stall_recovery) |
| `openase.orchestrator.harness_publish_total` | Counter | — | Times a newly published Harness version was recognized by the scheduler |

**PR Metrics——Measure Delivery Quality**

| Metric Name | Type | Labels | Description |
|--------|------|------|------|
| `openase.pr.opened_total` | Counter | project, repo | Total PRs created by the Agent |
| `openase.pr.merged_total` | Counter | project, repo | Total PRs successfully merged |
| `openase.pr.first_pass_rate` | Gauge | project | PR first-pass rate (merged without changes_requested / total merged) |
| `openase.pr.time_to_merge_seconds` | Histogram | project, repo | Time from open to merge for PRs |
| `openase.pr.review_rounds` | Histogram | project | Number of review rounds a PR goes through |

**System Metrics——Measure Runtime**

| Metric Name | Type | Labels | Description |
|--------|------|------|------|
| `openase.system.goroutines` | Gauge | — | Current number of goroutines |
| `openase.system.db_connections_active` | Gauge | — | Number of active database connections |
| `openase.system.db_query_duration_seconds` | Histogram | operation | Database query duration |
| `openase.system.sse_connections_active` | Gauge | — | Current number of SSE connections |
| `openase.system.uptime_seconds` | Gauge | — | Service uptime |

### 9.3 Distributed Tracing

The full lifecycle of each ticket forms a Trace, and each key operation inside it is a Span:

```
Trace: ticket/ASE-42
├── Span: orchestrator.dispatch          (dispatch)
├── Span: hook.on_claim                  (pre-claim Hook)
│   ├── Span: hook.exec clone-repos.sh
│   └── Span: hook.exec decrypt-secrets.sh
├── Span: agent.session                  (agent execution)
│   ├── Span: adapter.claudecode.start
│   ├── Span: adapter.claudecode.turn.1
│   ├── Span: adapter.claudecode.turn.2
│   └── Span: adapter.claudecode.stop
├── Span: hook.on_complete               (post-completion Hook)
│   ├── Span: hook.exec run-tests.sh
│   └── Span: hook.exec lint-check.sh
└── Span: hook.on_done                   (wrap-up Hook)
    └── Span: hook.exec cleanup.sh
```

Each Span carries standard attributes: `ticket.id`, `ticket.identifier`, `workflow.type`, `agent.name`, `agent.provider`. This allows users to filter and analyze by ticket, by Agent, and by Workflow type in Jaeger/Grafana.

### 9.4 Structured Logs

All logs are output as structured JSON via slog, with Trace ID and Span ID automatically injected to correlate logs with tracing:

```json
{
  "time": "2026-03-18T10:30:00Z",
  "level": "INFO",
  "msg": "ticket dispatched to workflow-bound agent definition",
  "ticket_id": "ASE-42",
  "agent_name": "claude-01",
  "workflow_type": "coding",
  "trace_id": "abc123...",
  "span_id": "def456..."
}
```

### 9.5 Built-in Web UI Dashboard

Even without configuring external Prometheus/Grafana, OpenASE’s Web UI includes a lightweight built-in dashboard (based on in-memory Metrics) that displays core indicators:

| Dashboard Panel | Content |
|-----------|---------|
| Ticket Throughput | Trends over 24h / 7d for ticket creation count, completion count, and success rate |
| Average Cycle Time | Ticket cycle time trend by Workflow type |
| Agent Utilization | Concurrent occupancy and current tasks for each Provider / AgentRun |
| Cost Tracking | Token consumption and cost by project / Agent / model |
| Hook Health | Pass rate, average duration, and recent failures for each Hook |
| PR Quality | First-pass rate, average review rounds, merge duration |

---
## Chapter 10 Orchestrator

### 10.1 Scheduling Loop

The core of the orchestrator is a periodic scheduling loop (refer to Symphony's Tick pattern). On each Tick:

1. **Reconciliation**: Check all tickets in running state — is the corresponding AgentRun still running? Is it stalled? Is the ticket canceled?
2. **Reconcile Workflow/Skill versions**: Check whether new published Workflow / Skill versions are available for use by subsequent new runtimes; already running runtimes do not drift implicitly.
3. **Collect candidate tickets**: Query all tickets in todo state, excluding those that are blocked.
4. **Priority sorting**: Sort by priority → created_at.
5. **Concurrency checks**: Check global run semaphore, Provider-level semaphore, and per-Workflow concurrency bit.
6. **Dispatch execution**: Read the Agent definition bound to the Workflow, create an AgentRun, create/reuse workspace, inject the Harness Prompt, and start execution via the corresponding Provider adapter.

#### 10.1.1 In-flight Ticket Reconciliation Rules in Symphony Style

OpenASE's reconciliation should not be just black-box heartbeat checks; it should maintain a single authoritative in-memory runtime state like Symphony and perform three kinds of reconciliation against this state on each Tick:

1. **Stall Reconciliation**
   - For each running AgentRun, record:
     - `started_at`
     - `last_codex_timestamp`
     - `last_codex_event`
     - `session_id`
     - `turn_count`
   - `last_codex_timestamp` should prefer the timestamp of the most recent Codex event; if no event exists, fall back to `started_at`.
   - If `now - last_codex_timestamp > codex.stall_timeout_ms`
     - Immediately terminate worker / Codex subprocess
     - Keep workspace
     - Enter retry backoff as an abnormal exit

2. **Tracker State Reconciliation**
   - Batch-fetch the latest status of all running issues: `fetch_issue_states_by_ids(running_ids)`
   - For each issue:
     - If it has entered terminal state: stop the corresponding AgentRun, release semaphore, clean up the workspace for that issue
     - If it is no longer routed to the current Workflow: stop the corresponding AgentRun, release semaphore, but do not force workspace cleanup
     - If still in active state: only refresh the in-memory issue snapshot for next round continuation / retry use
     - If it has left active state but not entered terminal state: stop the corresponding AgentRun, release semaphore, and do not trigger completion flow

3. **Runtime Fact Reconciliation**
   - AgentRun `:DOWN` / subprocess exit does not equal ticket completion.
   - On normal exit:
     - First record completion totals for this session (especially runtime seconds)
     - Then re-check whether issue is still in active state
     - If issue is still active, perform a 1-second continuation retry instead of finishing directly
   - On abnormal exit:
     - Enter exponential backoff retry
     - Record latest error reason in runtime state / activity / SSE

#### 10.1.2 Revalidate Before Dispatch

A key lesson from Symphony is: **the candidate ticket list is not the final truth; a second confirmation is required before dispatch**.

Before OpenASE truly claims + launches, it should execute:

1. Select an issue/ticket from the candidate set.
2. Sort by `priority -> created_at -> identifier`.
3. Read the corresponding Ticket Workflow, and that Workflow's bound Agent definition and Provider.
4. Validate dispatch preconditions:
   - Not claimed
   - Not currently in running map
   - Bound Agent definition exists and is enabled
   - Global run semaphore is not full
   - Provider-level `max_parallel_runs` semaphore is not full
   - If current hit pickup status has configured `max_active_runs`, that status semaphore is not full
   - Per-workflow concurrency is not full
   - If `A` blocks `B` and blocker `A` has not entered terminal stage, `B` cannot be dispatched
5. Before dispatch, fetch the latest status of the ticket again through repository/API
   - If status has changed, dependencies have changed, or ticket is no longer visible, abandon this dispatch directly

The purpose of this step is to prevent stale dispatch where the ticket is executable when scanned, but already expired when actually claimed.

### 10.2 Retry Strategy

| Scenario | Backoff Strategy | Description |
|------|---------|------|
| AgentRun normal exit | 1-second fast retry | The turn may have completed but the ticket is not finished |
| AgentRun abnormal exit | 10s × 2^(attempt-1), up to 5 minutes | Exponential backoff |
| Stall timeout | Kill Worker + handle as abnormal exit | 5 minutes with no event trigger |
| 3 consecutive Stalls | Pause retry (`retry_paused=true`), notify human | Prevents endless resource consumption |
| Ticket canceled | Stop immediately, no retry | User-initiated cancel |

**Retry Token mechanism**: A Ticket maintains the current `Retry Token`. All delayed retry intents must carry this Token; the scheduler or retrier must compare it again before actual dispatch, and silently discard if it does not match.

- The following transitions must rotate the `Retry Token`:
  - Any Ticket `status` change
  - Orchestrator creates a new delayed retry intent: abnormal-exit retry, stall-recovery retry, turn-limit continuation retry
  - Healthy progress clears the current retry baseline: normal completion, manual state advancement, cancel the current retry cycle
- Status changes or health progress must, while rotating the Token, also clear the old `next_retry_at / retry_paused / pause_reason / consecutive_errors` baseline to prevent stale retry intents from being revived by the new state.

### 10.3 Internal Architecture

| Component | Implementation | Responsibility |
|------|------|------|
| Scheduler | Single goroutine + ticker | Serialize all scheduling decisions |
| RuntimeRegistry | Orchestrator in-memory map | Manage active runtime / session lifecycle |
| RuntimeRunner | One goroutine per AgentRun | Manage individual Agent CLI subprocess and event pump |
| Harness Loader | DB + filesystem reads | Parse Harness content and provide input for runtimes |
| EventBus | Go channel (in-process) / PG LISTEN/NOTIFY (separate deployment) | Event communication with API service |
| HealthChecker / RetryService | Periodic goroutine | Periodically check runtime heartbeat, clean abnormal states, drive retries |

#### 10.3.1 Concurrency Model

OpenASE does not maintain an “agent instance pool” that can be scheduled. The schedulable object is Ticket; Agent is merely the execution definition bound to Workflow. The actual resource pool is expressed by semaphore:

**Explicit requirement: The system must support parallelism.** Here, parallelism includes both “multiple different Agent definitions processing multiple Tickets in parallel” and “the same Agent definition producing multiple `AgentRun`s simultaneously to process multiple Tickets when Provider / Workflow / Status / Global concurrency limits permit.” It is forbidden to implement Agent as a single worker slot that can occupy only one Ticket at a time.

1. **Global Run Semaphore**
   - Limit the total number of concurrent runs in the system
   - Prevent uncontrolled launching of external CLI subprocesses on one machine / one project
2. **Provider Semaphore**
   - Each AgentProvider maintains its own `max_parallel_runs`
   - For example, Codex allows up to 8 concurrent runs, Claude Code allows up to 4
3. **Status Semaphore**
   - Each TicketStatus can optionally configure `max_active_runs`
   - Represents the maximum number of Tickets currently in that status and already claimed by the orchestrator that can run at the same time
   - If multiple Workflows' pickup statuses hit the same status, they share this one status semaphore
   - Typical example: setting `Todo` to 1 means only 1 AgentRun can be working in that entry queue at any moment
4. **Workflow Max Concurrent**
   - Business-level throttling, not resource-pool definition
   - Indicates the maximum number of Tickets that this Workflow can drive at the same time

These three layers apply simultaneously:

- Whether it can run depends on global semaphore + provider semaphore + status semaphore
- Whether this many should run depends on workflow.max_concurrent
- The Agent itself is only the execution definition; what is actually claimed, run, completed, and reclaimed is `AgentRun`
- No implementation may incorrectly narrow “Ticket has only one `current_run_id`” into “Agent can handle only one Ticket at the same time”

---
## Chapter 11 Agent Adapter System

### 11.1 Unified Interface

All adapters implement the same Go interface:

```go
type AgentAdapter interface {
    Start(ctx context.Context, cfg AgentConfig) (Session, error)
    SendPrompt(ctx context.Context, s Session, prompt string) error
    StreamEvents(ctx context.Context, s Session) (<-chan AgentEvent, error)
    Stop(ctx context.Context, s Session) error
    Resume(ctx context.Context, sessionID string) (Session, error)
}
```

### 11.2 Claude Code Adapter (Three-Stage Evolution)

**Phase 1: CLI subprocess**

Start a Claude Code subprocess via `os/exec`:

```bash
claude -p "{{harness_prompt}}" \
  --verbose \
  --output-format stream-json \
  --allowedTools "Bash,Read,Edit,Write,Glob,Grep" \
  --max-turns 20 \
  --max-budget-usd 5.00 \
  --append-system-prompt "{{workflow_constraints_prompt}}"
```

- Parse NDJSON event streams and map them to `AgentEvent`
- Use `--resume session_id` to continue across multiple Turns
- For newer versions of Claude Code, `-p/--print + --output-format stream-json` must be used together with `--verbose`

**Phase 2: Agent SDK integration**

- Use Go to call the Claude Agent SDK to obtain Hooks + Subagents + MCP capabilities
- PreToolUse Hook intercepts dangerous operations
- PostToolUse Hook injects quality feedback
- Stop Hook prevents the Agent from stopping too early (checks whether a PR has been submitted)
- `--json-schema` enforces structured output

**Phase 3: Deep integration with Claude Code Hooks**

Configure native Claude Code Hooks through `.claude/settings.json` to form a two-layer quality safety net with OpenASE’s orchestrator-layer Hook:

- **PreToolUse**: Check whether it exceeds workspace boundaries (modifying unrelated files is not allowed)
- **PostToolUse**: Automatically run lint + type-check after Edit/Write; feed failures back to the Agent
- **Stop**: Check whether all PRs related to the repo have been completed and submitted
- **TaskCompleted**: Run full test suite; block task completion if it fails

### 11.3 Codex Adapter

OpenASE’s Codex adapter should directly adopt the already validated Symphony **stdio request/response + notification** model, rather than abstracting to a black box of just “SendPrompt and wait for result.”

#### 11.3.1 Session / Thread / Turn Three-Layer Model

- **Session**
  - Corresponds to a long-lived Codex app-server subprocess
  - The process is reused within one worker lifecycle of one ticket
- **Thread**
  - Created via `thread/start`
  - Multiple turns reuse the same `thread_id` during one worker run of one ticket
- **Turn**
  - Each `turn/start` generates a new `turn_id`
  - `session_id = <thread_id>-<turn_id>`
  - `turn_count` increments within the same worker lifecycle

#### 11.3.2 Startup Handshake (reference: Symphony `Codex.AppServer`)

1. `initialize`
   - `clientInfo.name/version/title`
   - `capabilities.experimentalApi = true`
2. Wait for `initialize` response (match by request id)
3. Send `initialized`
4. `thread/start`
   - `cwd`
   - `approvalPolicy`
   - `sandbox`
   - `dynamicTools`
5. Wait for `thread/start` response and extract `thread.id`

**Engineering requirements:**

- While waiting for response, non-related notifications/log lines must be allowed and ignored; do not error just because other messages arrive first
- Non-JSON lines should only be logged and must not interrupt the session
- Response recognition must be based on request id, not guessed from method name

#### 11.3.3 Standard Way to Send a Prompt Per Turn

Each turn is sent through `turn/start` with:

- `threadId`
- `input: [{type: "text", text: prompt}]`
- `cwd`
- `title`
- `approvalPolicy`
- `sandboxPolicy`

Specifically:

- **Turn 1**
  - Send the full Harness Prompt
  - Include ticket description, repository information, boundaries, acceptance criteria, and current attempt
- **Turn 2+**
  - Do not resend the original full prompt
  - Send only continuation guidance
  - Clearly tell the Agent: the current thread has preserved context and it should continue from the current workspace/workpad

This is a core Symphonic insight: **multi-turn continuation relies on thread reuse, rather than repeating the same task prompt every time.**

#### 11.3.4 Codex notifications/requests that OpenASE must support

| Method | Direction | Description |
|------|------|------|
| `initialize` | OpenASE → Codex | Initialize connection |
| `initialized` | OpenASE → Codex | Handshake-complete notification |
| `thread/start` | OpenASE → Codex | Create thread |
| `turn/start` | OpenASE → Codex | Initiate turn |
| `item/tool/call` | Codex → OpenASE | Dynamic tool-call request |
| `item/commandExecution/requestApproval` | Codex → OpenASE | Command execution approval |
| `item/fileChange/requestApproval` | Codex → OpenASE | File change approval |
| `item/tool/requestUserInput` | Codex → OpenASE | Request interactive user input |
| `item/agentMessage/delta` | Codex → OpenASE | Agent-visible text output delta |
| `item/commandExecution/outputDelta` | Codex → OpenASE | Command execution output delta |
| `item/completed` | Codex → OpenASE | Item completion snapshot (fallback when there is no delta) |
| `thread/tokenUsage/updated` | Codex → OpenASE | Thread-level token usage stream |
| `turn/completed` | Codex → OpenASE | Turn completed normally |
| `turn/failed` | Codex → OpenASE | Turn failed |
| `turn/cancelled` | Codex → OpenASE | Turn canceled |

#### 11.3.5 Approval, user input, and interaction modes

The Codex adapter must distinguish two runtime modes:

- **Unattended orchestration mode**
  - For ticket worker / orchestrator
  - `approval_policy == "never"`
  - Auto-respond to approval requests with `acceptForSession` / `approved_for_session`
  - Return a standardized `operator unavailable / default answer` for `item/tool/requestUserInput`
- **Project Conversation interactive mode**
  - For `project_sidebar` in Chapter 31
  - Allow `approval_policy != "never"`
  - Must expose approval requests / requestUserInput as resumable interrupts, instead of auto-approving

Project Conversation adapter requirements:

- `item/commandExecution/requestApproval`
  - Persist as `pending_interrupt(kind=command_execution_approval)`
  - Wait for user decision
- `item/fileChange/requestApproval`
  - Persist as `pending_interrupt(kind=file_change_approval)`
  - Wait for user decision
- `item/tool/requestUserInput`
  - Persist as `pending_interrupt(kind=user_input)`
  - Wait for user answer

Engineering constraints:

- Interrupt resume must route precisely by request id, not heuristic logic like “usually only one pending request per turn”
- Provider-native decision options must be preserved; OpenASE should only normalize the envelope and must not flatten Codex-native decision space into an abstract “allow/deny”
- If the current session is clearly non-interactive and no human input can be obtained
  - Return a standardized unavailable / unsupported response, rather than waiting indefinitely

#### 11.3.6 Token reconciliation rules (must be followed)

OpenASE should adopt the token accounting rules already documented by Symphony:

- **Prefer absolute cumulative values**
  - `thread/tokenUsage/updated.params.tokenUsage.total`
  - or `total_token_usage` from Codex core events
- **Do not treat `turn/completed.usage` as a blindly additive total**
  - It is event-specific usage and not necessarily equal to thread cumulative total
- **Maintain high-water mark per thread**
  - `delta = next_total - prev_reported_total`
  - Only accumulate when the high-water mark advances
- **At session completion**
  - Only record `seconds_running`
  - Do not add tokens again to avoid double count

#### 11.3.7 Minimal practical plan to fill current implementation gaps

Based on the current OpenASE codebase, the real gap is not “whether a Codex session can be started,” but that the following three chains are still not closed:

1. `scheduler -> runtimeLauncher`
   - Currently only claims ticket, starts app-server, and obtains `thread_id`
2. `runtime ready -> start turn`
   - There is still no real ticket runner calling `SendPrompt` / `turn/start`
3. `turn finished -> reconcile -> continuation / finish / retry`
   - There is no convergence of turn results back into ticket / agent lifecycle

Therefore, it is recommended to close gaps with the following **minimal implementable architecture**:

- **Add an `AgentRunner` component**
  - Place it in the orchestrator layer
  - Input: agent with `running + runtime_phase=ready + current_ticket_id != nil`
  - Responsibilities:
    - Find the already-started session for this agent
    - Generate the prompt for this round
    - Call `session.SendPrompt(...)`
    - Consume `session.Events()` until `turn/completed` / `turn_failed`
    - After turn ends, reload latest ticket status and decide next steps

- **Change orchestrator main loop into five stages**
  - `healthChecker`
  - `machineMonitor`
  - `scheduler`
  - `runtimeLauncher`
  - `agentRunner`

- **Separate first-turn prompt and continuation prompt**
  - Turn 1:
    - Reuse `workflow.BuildHarnessTemplateData + RenderHarnessBody`
    - This is OpenASE’s existing full prompt source of truth
  - Turn 2+:
    - Use a fixed continuation builder
    - Input should include at least:
      - `turn_number`
      - `max_turns`
      - `last_error`
      - `ticket.attempt_count`
      - `ticket.status`
    - Only add “what to continue / why the previous round did not finish,” do not resend the full harness

- **Minimal runtime phase expansion**
  - Currently only `none / launching / ready / failed`
  - At minimum, expand to:
    - `none`
    - `launching`
    - `ready`
    - `executing`
    - `failed`
  - `executing` means “a turn is currently running,” preventing concurrent `turn/start` on the same session

- **Event-driven heartbeat instead of periodic fake heartbeat**
  - Current `refreshHeartbeats()` writes heartbeat periodically for `ready` agents, which only proves the orchestrator process is alive, not that the Codex turn is still progressing
  - Fix:
    - Update `last_heartbeat_at` whenever any Codex notification / tool call / token update / turn event arrives
    - Use the latest Codex event time as the primary signal for stall detection
    - Periodic heartbeat can only be a fallback and cannot be the primary source of stall truth

- **Post-turn state convergence rules**
  - `turn/completed`
    - Reload ticket + workflow
    - If ticket has left pickup/active state:
      - Stop continuation
      - Release or terminate session
    - If ticket is still active and `turn_count < max_turns`:
      - Start the next continuation turn directly on the same session
    - If ticket is still active but has hit `max_turns`:
      - Stop current worker lifecycle
      - Keep ticket as active
      - Transfer control back to orchestrator through retry/continuation mechanism
  - `turn_failed`
    - Record `last_error`
    - Stop session
    - Proceed with retry backoff

- **Minimal approval / tool / token support**
  - Adapter cannot only recognize `item/tool/call`
  - It must also support at least:
    - `item/commandExecution/requestApproval`
    - `item/fileChange/requestApproval`
    - `item/tool/requestUserInput`
    - `thread/tokenUsage/updated`
  - Default to unattended mode in MVP:
    - `approval_policy = never`
    - Auto-approve approval requests
    - Return standardized unavailable answer for request user input

- **Minimal token reconciliation implementation**
  - Does not need full cost center integration in the first version, but at least:
    - Keep `prev_total` in an in-memory map keyed by `thread_id`
    - Calculate delta when `thread/tokenUsage/updated.total` is received
    - Record delta via `ticket.RecordUsage(...)`
    - Do not add tokens again when turn completes

- **Minimal runtime observability**
  - `item/agentMessage/delta` and `item/commandExecution/outputDelta` must be immediately normalized into fine-grained `AgentEvent` and persisted as `AgentTraceEvent`
  - `item/completed` should only be a fallback snapshot “when there is no delta,” avoiding duplicate persistence of the same item text
  - `/api/v1/projects/{projectId}/agents/{agentId}/output` and `/output/stream` can keep their names during compatibility period, but their single source of truth must be `AgentTraceEvent`, not mixed reads from `ActivityEvent`
  - Elevate to `AgentStepEvent` additionally on step state changes
  - Lifecycle/token usage events must not masquerade as output; output must also not directly masquerade as business Activity

OpenASE should not attempt to solve complex interactive approvals, Hook Gate, or complicated pause/resume all at once in phase 1. First make four things solid: “real turns run, can continue, can retry on failure, and do not double count usage.” Then the chain will move from `runtime ready` to truly executable.

#### 11.3.8 Organization Token Analytics Snapshot rules

The token trend chart and calendar heatmap on the Organization Dashboard must be based on **run terminal-state daily snapshots**, not inferred from ticket creation date or current cumulative values.

- Attribute token usage by the **UTC day** of `AgentRun.terminal_at`
- Only runs that reach terminal state can be included in org-level daily snapshots
- Daily snapshots should retain at least:
  - `input_tokens`
  - `output_tokens`
  - `cached_input_tokens`
  - `reasoning_tokens`
  - `total_tokens`
  - `finalized_run_count`
- When a new run reaches terminal state, it should be incrementally materialized into the corresponding org/day snapshot
- For the first historical-window query, allow on-demand lazy backfill of missing dates instead of performing a one-time full historical migration
- Organization Overview should default to the latest 30 days of token analytics and offer quick windows of 7d / 30d / 90d / 365d

### 11.4 Gemini CLI Adapter

Like Claude Code, integrate via CLI subprocess + stdio stream.

- The orchestration runtime provides a Gemini adapter through `internal/orchestrator/agent_adapter_gemini.go`, supporting provider selection, session start, continuous turn execution, failure propagation, and unified `agentEvent` normalization.
- The current implementation adopts a minimum deliverable semantics of “starting one Gemini CLI process per turn + maintaining conversation history in OpenASE memory,” so it can meet orchestrator continuous execution requirements.
- `internal/chat/runtime_gemini.go` continues to serve instant conversation semantics; both paths share the same Gemini CLI non-interactive JSON output mode.
- Gemini session resume across OpenASE orchestration processes is not supported yet; if crash-recovery-level recovery is needed in the future, transcript persistence or Gemini-native resume capability will be needed.
## Chapter 12 Git Repository and PR Link Integration

### 12.1 Current scope

- **RepoScope Binding**: A ticket can explicitly bind one or more Repos; each RepoScope records the repository, working branch, and optional PR link.
- **PR Link Recording**: An Agent or a human can write back a repo’s PR URL to the corresponding TicketRepoScope so it can be viewed and jumped to on the Ticket Detail page.
- **Multi-Repo Display**: When a ticket involves multiple Repos, the frontend ticket detail page shows the list of branches and PR links for all Repos.
- **Out of scope**: The current version does not receive GitHub/GitLab webhooks, does not sync GitHub Issues, does not sync PR status, and does not sync CI status.

### 12.2 Relationship between ticket and PR link

| Ticket status | Repo/PR info | Trigger condition |
|---------|------------|---------|
| todo | none | Ticket created, waiting to be claimed |
| in_progress | Branches for all involved Repos have been created | Agent is claimed and creates branches in each Repo |
| in_review | One or more RepoScope entries have PR links recorded (optional) | Human or Agent explicitly moves the ticket to review |
| done | Whether a PR exists or is merged is not an automatic judgement condition | Human or Agent explicitly completes the ticket |

**Explicit constraints:**

- `pull_request_url` is used only for reference and navigation, and does not drive any automatic state advancement
- PR merged / closed / changes requested do not automatically change Ticket status
- CI results are not written back to TicketRepoScope, and are not used as Hook or state machine input
## Chapter 13 UI Design

OpenASE’s frontend is not a “collection of back-office admin forms,” but an engineering control plane where users continuously monitor project running status, schedule work quickly, and intervene in incidents immediately. The overall experience is inspired by Linear: **clean, restrained, high density, fast interactions, keyboard-first by default**, but with information density skewed more toward “engineering operations” than just task management.

Core design goals:

- **Dashboard-first**: The first screen after entering the product is not a blank list but the Dashboard, showing first whether the project is healthy, where it is blocked, and whether Agents are working.
- **Project-centric**: Organization is the management boundary, and Project is the work context. In the UI, users should always be clear about “which organization and which project I am currently viewing.”
- **Board-native**: The board is the default work interface, not a secondary view. State transitions, drag-and-drop, real-time push, and exception alerts all revolve around the board.
- **Real-time but calm**: SSE real-time updates should be visible but must not create visual noise.
- **Deep work friendly**: Reduce layered modals and page jumps; use sidebars, drawers, inline editing, and command panels more.

### 13.1 Visual Style and Brand Tone

The overall visual language follows Linear’s “low-noise professional feel,” while being enhanced for the engineering console scenario:

| Design Dimension | Design Requirements |
|---------|---------|
| Visual tone | Calm, professional, restrained; avoid marketing-style large color blocks |
| Layout density | High information density, with controlled hierarchy, whitespace, and borders to reduce reading pressure |
| Color strategy | Neutral colors as the base, status colors only to express status, not for large decorative areas |
| Motion strategy | Short, light, minimal animations, only for state transitions, drag feedback, and real-time update cues |
| Icon strategy | Use Lucide consistently, linear thin strokes, avoid excessive decoration |
| Component style | Low radius, small shadows, thin borders; cards and panels should feel like tools, not marketing pages |

**Recommended visual tokens:**

| Token | Suggested Value | Use |
|------|-------|------|
| Background color | `#0F1115` or light variant `#F7F8FA` | App outer background |
| Panel background | `#151922` / `#FFFFFF` | Cards, sidebars, tables, drawers |
| Divider | `rgba(255,255,255,0.08)` / `#E6E8EC` | Panel borders, list dividers |
| Primary text | `#F3F4F6` / `#111827` | Primary information |
| Secondary text | `#9CA3AF` / `#6B7280` | Descriptions, auxiliary info |
| Accent | `#5E6AD2` | Current selection, focus, primary CTA |
| Success | `#22C55E` | done, hook passed |
| Warning | `#F59E0B` | retry, stalled, budget near limit |
| Error | `#EF4444` | hook failed, agent error |
| Info | `#38BDF8` | SSE real-time updates, highlight prompts |

**Typography recommendations:**

- For Chinese UI, prefer `Inter + Noto Sans SC` or an equivalent stack, keeping numbers and English compact with stable Chinese readability.
- Page title 20–24px, section title 14–16px, body 13–14px, secondary info 12px.
- Slightly compact line heights: headings 1.2–1.3, body 1.5, table/list 1.4.
- Use 8pt spacing system globally, compact lists allow 4pt micro spacing.

### 13.2 Information Architecture

The frontend adopts two context layers:

1. **Organization Context**: determines which projects, members, channels, budgets, and governance rules are visible
2. **Project Context**: determines which project board, Workflow, Agent, notifications, and configuration are visible

Primary user path:

```text
Sign in / First Entry
  → Dashboard (default entry)
  → Select Organization
  → Select Project
  → Enter Board / Updates / Workflows / Agents / Settings via sidebar within the project
  → View and drag tickets on the Board
  → Drill into a single ticket in the right drawer / detail page
```

**Navigation hierarchy:**

| Level | Primary entity | Typical pages |
|------|---------|---------|
| Global level | Dashboard, org switch, global search, personal settings | Dashboard, notification center, command panel |
| Project level | Board, progress updates, Workflow, Agent, approval, activity, cost, settings | Project Dashboard, Board, Updates, Agents, Settings |
| Entity level | Ticket, Harness, Approval, Channel | Ticket detail, Harness editor, approval detail |

### 13.3 Global App Shell

The web UI uses a fixed App Shell to avoid context loss from frequent page jumping:

```text
┌──────────────────────────────────────────────────────────────────────────────┐
│ Top Bar: Organization Switcher | Project Switcher | Search/Cmd+K | New | SSE Status | User Menu │
├───────────────┬──────────────────────────────────────────────┬───────────────┤
│ Left Sidebar  │ Main Content                                 │ Right Panel   │
│ Global/Project Navigation │ Dashboard / Board / Settings / Editor │ Detail Drawer/AI      │
│ Recent Projects/Pinned   │                                          │ Context Panel       │
└───────────────┴──────────────────────────────────────────────┴───────────────┘
```

**Design principles:**

- Top Bar handles “context switching + global actions”
- Left Sidebar handles “navigation under current context”
- Main Content handles “main workspace”
- Right Panel handles “non-blocking deep dives,” such as Ticket details, AI assistant, filters, advanced info

### 13.4 Top Bar Design

The top bar is the context-switching and high-frequency action layer. It should remain single-line, compact, and stable without large jumps across different pages.

**Proposed layout from left to right:**

1. OpenASE Logo / Home
2. Organization Switcher
3. Project Switcher
4. Current page title / breadcrumb
5. Global search entry + `Cmd+K`
6. Primary action button: `+ New Ticket`
7. Real-time status area: SSE online status, number of running Agents, pending approvals
8. User menu: personal settings, theme, logout

**Top bar interaction details:**

| Element | Interaction requirements |
|------|---------|
| Organization switcher | Supports search, recently visited, pinned organizations; refreshes project list and org-level dashboard after switching |
| Project switcher | Shows only projects under current organization; supports recent projects, pinned projects, sorting by health state |
| Cmd+K | Search pages, tickets, projects, Workflows, command actions |
| SSE status | Subtle hint when normal; thin top warning bar with “Reconnecting...” when disconnected |
| New button | Supports dropdown: New Ticket / New Project / New Workflow |

**Recommended fields for project switcher display:**

- Project name
- Health status (Healthy / Warning / Blocked)
- Number of running tickets
- Last activity time

### 13.5 Left Sidebar Design

The sidebar is the product’s skeleton. It should be as stable, restrained, and collapsible as Linear, with one extra layer of “engineering console” characteristics.

**Global sidebar structure:**

```text
[OpenASE]

Dashboard
My Pending
Approvals Center

Projects
  Alpha Platform
  Mobile App
  Infra

Pinned
  High-priority projects this week
  Cost-exception projects

Settings
```

**Project sidebar structure after entering a project:**

```text
Project: Alpha Platform

Overview
Board
Tickets
Workflows
Agents
Approvals
Activity
Insights
Settings
```

**Project navigation description:**

| Entry | Description |
|------|------|
| Overview | Project dashboard, default project entry |
| Board | Multi-column board, default work view |
| Tickets | List view, suitable for batch filtering and search |
| Workflows | Workflow list, Harness editor, version history |
| Agents | Agent definitions, Provider configuration, running AgentRun |
| Activity | Audit logs, system events, Hook execution records |
| Insights | Cost, throughput, success rate, retry trends |
| Settings | Project configuration, Repo, status columns, notifications, machines |

**Sidebar design details:**

- Keep current project name fixed at top, with project health and last update time below
- Use a thin left accent bar plus background fill for current page highlight, avoid exaggerated color blocks
- Show pending approvals, running Agent count, and failure alert count as small badges on navigation items
- Sidebar can collapse to narrow mode, keeping icons and tooltips
- Persistent on wide screens; on narrow screens, it opens via a drawer

### 13.6 Dashboard as the default entry

**Dashboard is the default entry of the system.** It is split into Organization-level Dashboard and Project-level Dashboard.

#### 13.6.1 Organization-level Dashboard

After entering the system, if no specific project is focused, users default to the organization-level Dashboard to evaluate “is the whole team healthy now.”

Suggested panels:

| Panel | Display information |
|------|---------|
| Project Health | Health list of all projects: Healthy / Warning / Blocked |
| Running Now | Current number of AgentRun running, active tickets, pending approvals |
| Delivery Funnel | Tickets created this week, completed tickets, average cycle time, PR merge rate |
| Exceptions | Hook failures, retry pauses, machine offline, budget exceeded |
| Recent Activity | 20 most recent cross-project activity items |
| Cost Snapshot | Today / This week cost, top projects, top Agents |

**Organization-level Dashboard wireframe:**

```text
┌─────────────────────────────────────────────────────────────────────┐
│ Dashboard                                            Today / Week switch │
├───────────────────────┬───────────────────────┬─────────────────────┤
│ Projects Health       │ Running Now           │ Cost Snapshot       │
│ Alpha  Healthy        │ 12 Agents Running     │ Today: $24.3        │
│ Infra  Warning        │ 37 Active Tickets     │ Week:  $183.5       │
│ Mobile Blocked        │ 4 Approvals Pending   │ Top: Alpha          │
├───────────────────────┴───────────────────────┬─────────────────────┤
│ Delivery Funnel                                │ Exceptions         │
│ Created / Done / Merge Rate / Avg Cycle Time   │ Hook Failed: 3     │
│                                                │ Budget Alert: 2    │
├────────────────────────────────────────────────┴─────────────────────┤
│ Recent Activity                                                        │
└─────────────────────────────────────────────────────────────────────┘
```

#### 13.6.2 Project-level Dashboard (Project Overview)

After selecting a specific project, users default to the project-level Dashboard rather than going directly into a configuration page.

Suggested panels:

| Panel | Description |
|------|------|
| Project status summary | Project health, number of bound Repos, active state columns |
| Board Snapshot | Number of tickets per column + risk hints |
| Active Agents | Running Agents, current processing ticket, runtime |
| PR / Hook status | Multi-repo PR overview, recent Hook failures |
| Approval Queue | Pending tickets, waiting duration |
| Cost & Throughput | Cost in last 24h, completion count, average attempt |
| Activity Feed | Recent ticket creation, status changes, Agent completion, webhook writeback |

**Value of project Dashboard:**

- First answer “Is the project operating normally now?”
- Then guide the user into the Board for concrete scheduling
- Expose exceptions first, so users do not have to click many layers to find problems

### 13.7 Board Design

The Board is the most important page within a project and has higher priority than table views. It must satisfy both “intuitive drag-and-drop” and “high information density.”

#### 13.7.1 Page structure

```text
┌────────────────────────────────────────────────────────────────────────────┐
│ Board                                  Filters  Grouping  View Switch  + New Ticket   │
├────────────────────────────────────────────────────────────────────────────┤
│ Workflow: All  Agent: All  Priority: All  Show exceptions only [toggle]     │
├────────────┬────────────┬────────────┬────────────┬────────────┬───────────┤
│ Backlog(8) │ Todo(12)   │ In Progress(4) │ Review(3)  │ Done(27)   │ Cancelled │
│ status tip │ WIP 12      │ 2 Agents    │ 1 blocked  │ today +5   │           │
├────────────┼────────────┼────────────┼────────────┼────────────┼───────────┤
│ ASE-42     │ ASE-51      │ ASE-38      │ ASE-33     │ ASE-11     │           │
│ Fix login  │ Add audit   │ Reconnect   │ Review PR  │ Export CSV │           │
│ high · bug │ coding      │ agent: c-1  │ 2 PRs      │ merged     │           │
│ hook fail  │ backend     │ 3m ago      │ waiting    │            │           │
└────────────┴────────────┴────────────┴────────────┴────────────┴───────────┘
```

#### 13.7.2 Column design

Global Search is not exposed in the shell top bar in the current stage. This entry should only be restored after a unified backend search contract and usable result stream are implemented; until then, keep local filtering within the page.

Each column should show at the top:

- Status name + icon + color
- Ticket count
- Optional WIP info
- Current column risk summary, such as “2 retrying” or “1 pending approval”

Column header actions:

- Rename column
- Modify color / icon
- Set as default status
- Add columns before/after
- Delete column
- View filtered view for this column

#### 13.7.3 Ticket Card Design

A single ticket card must allow users to determine within 1–2 seconds: “what this is, who is handling it, and whether it is risky.”

**Suggested card fields:**

| Information | Default display | Description |
|------|-------------|------|
| `ASE-42` + title | Required | Primary identification info |
| Priority | Required | Highlight `urgent` / `high` with color points |
| Workflow / role | Required | coding / testing / security |
| Current Agent | Show when available | Agent definition currently driving this ticket |
| Repo/PR links | Show when available | `2 repos`, `1 PR linked` |
| Exception status | Highlight when present | `retrying`, `Hook failed`, `pending approval`, `budget exhausted` |
| Updated time | Suggested | `2m ago` |

**Card interactions:**

- Single click: open right-side Ticket detail drawer without leaving the board
- Double click or `Enter`: open full Ticket detail page
- Drag: change status column
- Hover: show more metadata, such as creator, dependencies, and recent activity summary

#### 13.7.4 Drag-and-drop interactions

Drag experience must be clean and explicit, with no “I don’t know whether it worked” ambiguity.

| Scenario | Interaction feedback |
|------|---------|
| Start drag | Card lifts slightly, shadow deepens, placeholder outline appears at original position |
| Drag to target column | Target column header highlights, insertion placeholder appears inside column |
| Drag submit success | Card settles, column count updates immediately, short toast appears at top |
| SSE sync updates | Changes by other users should flash once with soft highlight, no heavy animation |
| Drag failure | Card bounces back, error reason shown, such as “no permission” or “status does not exist” |

#### 13.7.5 Board supporting capabilities

- Quick filters at top: Workflow, Agent, priority, tags, exception status
- View switching: Board / List / My Tickets
- Save filtered views, such as “Hook failed only” and “Pending approval only”
- Support horizontal scrolling + sticky column headers
- Support empty column placeholder copy, such as “No tickets yet, drag one here to start”

### 13.8 Ticket Detail Experience

The board is not the endpoint. Users need to drill into details without losing context, so Ticket detail adopts a “drawer-first, page-second” model.

**Opening methods:**

- Click a board card once to enter the right-side detail drawer
- Click from search results, activity feed, or notifications to go directly to full detail page

**Suggested structure of the right-side drawer:**

| Module | Content |
|------|------|
| Header | Ticket ID, title, status, priority, primary action button |
| Summary | Description, Workflow, current Agent definition, involved Repos |
| Execution | Agent real-time output, runtime, attempt_count, cost |
| Hooks | `on_claim` / `on_complete` / `on_done` history |
| PRs | Multi-repo PR list, status, links |
| Dependencies | Parent-child tickets, blocking relationships |
| Activity | Timeline, comments, system events |

#### 13.8.1 Ticket Timeline (GitHub Issue style)

The information organization of Ticket Detail should be close to GitHub Issue, not a loose stack of “description block, then comments block, then activity block.” The main view must be **a unified timeline**:

1. Show ticket description as the first root entry at the top.
2. After the root entry, display comments and system activity in chronological order (oldest to newest).
3. Provide a new comment input area at the bottom.

This timeline is the primary reading path for Ticket Detail.

**Layout contract:**

- Header: Ticket title, status, priority, primary action button
- Root Description Entry: author, creation time, Markdown body, edit state
- Timeline: comments and activity mixed in chronological order
- Composer: located at timeline bottom for submitting new comments
- Auxiliary sections may include Modules such as Dependencies / Repositories & PRs / Hook History, but cannot replace the main timeline

**Description entry requirements:**

- Always display first, semantically equivalent to “author opened this ticket”
- The card must show:
  - Author
  - Creation time
  - Markdown body
  - `edited` if it has been edited
- Description can enter an edit flow separately, but it is not a normal comment and cannot be deleted

**Comment entry requirements:**

- A comment, like a GitHub issue comment, is a top-level entry in the timeline
- Each comment must show:
  - Author
  - Publish time
  - Markdown body
  - `edited` marker
  - Last edited time
  - Action buttons: `Edit` / `Delete` / `History` / `Collapse`
- `History` should show that comment’s historical versions and at least include:
  - revision number
  - edited by
  - edited at
  - body snapshot
- `Collapse` only affects the reading state of the current detail surface and does not alter data

**System Activity entry requirements:**

- System activity is also a top-level timeline entry, but visually lighter than comments
- System Activity must present the canonical business facts corresponding to `ActivityEvent.event_type` directly, and must not invent aliases in the UI
- Typical events include:
  - `ticket.created` / `ticket.status_changed` / `ticket.completed`
  - `agent.claimed` / `agent.launching` / `agent.ready` / `agent.failed` / `agent.completed`
  - `hook.started` / `hook.passed` / `hook.failed`
  - `pr.opened` / `pr.merged` / `pr.closed`
- At minimum, an activity entry must show:
  - Icon or type marker
  - Summary text
  - Time
  - Metadata when needed (such as status `from/to`, agent name, PR link)
- Activity entries are not editable or deletable by default, but long metadata can be collapsed

**Edit and history requirements:**

- Comment edits must preserve historical versions and must not overwrite old body content
- Timeline main view should show only the latest version by default; historical versions are viewed via `History`
- Comment deletion must have a clear destructive affordance; after deletion, the timeline should keep a placeholder and not disturb order
- If a comment or description is edited, the UI must show `edited` with the corresponding time, not only the latest `updated_at`

**Markdown constraints:**

- Support common Markdown: paragraph, heading, list, link, blockquote, inline code, code block
- Rendered output must pass frontend security sanitization; scripts and dangerous attributes must be blocked
- At this stage, no requirement for @mention, reaction, attachments, or resolved threads

**Real-time refresh requirements:**

- After new comment, comment edit, comment delete, and activity append, an open Ticket Detail must refresh in place
- Refresh semantics should be at the timeline-entry level, not full page flicker reload
- After comment history updates, the current comment’s `edited` marker, edit time, and history count must be synchronized

**Acceptance criteria:**

- Opening any Ticket Detail from Board or Tickets shows the description root entry first
- Below the description, comments and system activity appear in chronological order
- After adding a comment, the entry is appended to the bottom timeline immediately
- Existing comments can be edit / delete / history / collapse
- After a comment is edited, timeline shows `edited` and edit time, and history versions are viewable
- system activity and comments coexist on one timeline, but with clearly different visual hierarchy
- The overall reading order is close to GitHub Issue, rather than split into multiple disjointed panels

### 13.9 Workflows and Harness Editor Design

The Workflow management page is the project’s “control-rule center,” and should feel efficient like an IDE rather than like a standard CMS text field.

**Page layout:**

- Left: Workflow list
- Middle: Harness edit area
- Right: Variable dictionary / AI assistant / Preview panel tab

**Suggested Workflow list fields:**

- Workflow name
- Role name
- Bound Agent definition
- Bound Provider
- pickup / finish status
- Last modified time
- Recent run result (success rate in the past 24h)

**Workflow base configuration must support:**

- Selecting bound Agent definition
- Read-only display of Provider / Machine / Model after selecting Agent
- Simultaneously display this Agent’s Ticket workspace convention (derived by platform, not manually fillable)
- Modify pickup / finish status
- Modify concurrency and timeout limits

The frontend should not provide an interaction to “temporarily pick an idle Agent instance for Workflow.” A Workflow binds to a static Agent definition; actual runtime occupancy is created as AgentRun by the orchestrator during execution.

**Harness editor must support:**

- YAML Frontmatter and Markdown dual-region syntax highlighting
- Auto-completion for variables, filters, snippets
- Preview real render result
- Diff compare historical versions
- Undefined variable warnings
- AI-assisted patch application

### 13.10 Agents Page Design

The Agent page is split into “definition perspective” and “execution perspective” layers, to avoid mixing static Agent configuration and transient runtime status.

#### 13.10.1 Agent execution page

Shows “which AgentRun processes are running now and how they are doing”:

| Field | Description |
|------|------|
| Agent name | such as `codex-coding` |
| Provider / Model | Claude Code / Codex / Gemini |
| Current state | launching / ready / executing / errored / stalled |
| Current ticket | if running, show ticket ID |
| Recent heartbeat | used to judge health |
| Related Workflow | indicates which Workflow triggered this run |
| Today’s cost | operational metric |

Supported actions:

- View live output
- Terminate current run
- View recent failed runs

#### 13.10.2 Agent definition page

This is the page corresponding to the “agent configuration and sidebar entry” you mentioned. It is a first-level entry under project Settings and can also be entered directly from the project sidebar.

Suggested grouping:

| Group | Contents |
|------|------|
| Providers | Integration configuration for Claude Code / Codex / Gemini CLI |
| Defaults | default Provider, default model, concurrency cap |
| Agent Definitions | Agent name, bound Provider, corresponding Machine, which Workflows use it |
| Execution Defaults | default Provider, concurrency constraints, and entry points for future extensible execution strategies |
| Budget | Single Agent / per-project budget cap |
| Safety | approval policy, allowed platform operation scope |

### 13.11 Project Settings Information Architecture

Project settings should not be one huge long form. It should be a two-level navigation structure:

```text
Settings
  General
  Repositories
  Status Columns
  Workflows
  Agents
  Notifications
  Machines
  Security
```

**Common design principles for each settings page:**

- Small navigation on the left, form or list on the right
- Separate dangerous operations from normal configuration
- All “test connection” and “validate configuration” actions complete in place without page jumps
- Immediate feedback after save, and show warnings such as “service will restart” or “new config only affects new tickets” when necessary

In this vertical slice, `Security` is initially delivered as a project-level security posture page: it shows Agent-scoped token, webhook signature verification, and implemented secret redaction boundaries; human login/OIDC, RBAC, and a more complete governance panel are deferred to a dedicated control plane implementation.

### 13.12 Real-time Status and Feedback Design

OpenASE is a real-time system, and it must clearly communicate “what the system is doing,” while avoiding constant page flicker.

**Real-time feedback rules:**

| Scenario | UI behavior |
|------|---------|
| Agent starts execution | Card shows running status dot; corresponding row in Agent page becomes active |
| New Agent log output | Detail drawer output stream appends automatically, default follows to bottom; auto-scroll can be paused |
| Hook failure | Card badge turns red; Hook tab in detail page automatically displays failure prompt |
| Approval generated | Top bar pending approval count +1; Approvals navigation item shows badge |
| SSE disconnected | Top bar shows reconnect status without blocking current browsing |
| Data delay | Top right shows subtle hint “last synced 12s ago” |

**Prohibited approaches:**

- Re-rendering and flickering entire columns on every SSE update
- Replacing in-page state feedback with excessive toasts
- Writing errors only in console without marking them in UI

### 13.13 Keyboard-first and efficiency design

Following Linear, OpenASE should natively support high-frequency keyboard operations:

| Shortcut | Action |
|-------|------|
| `Cmd+K` / `Ctrl+K` | Open command panel |
| `C` | New ticket |
| `G` `B` | Jump to Board |
| `G` `W` | Jump to Workflows |
| `G` `A` | Jump to Agents |
| `[` / `]` | Move between previous/next ticket |
| `Esc` | Close right drawer / cancel operation |
| `.` | Open ticket action menu |

### 13.14 Responsive and mobile strategy

OpenASE is primarily desktop-oriented, but must allow key status visibility on mobile; it does not require completing heavy operations fully on a phone.

| Device | Strategy |
|------|------|
| Desktop (>=1280px) | Full three-column layout: left navigation + main workspace + right drawer |
| Laptop (1024-1279px) | Default two-column: left navigation collapsible, right drawer expands as overlay |
| Tablet (768-1023px) | Sidebar becomes drawer, board supports horizontal swiping |
| Mobile (<768px) | Keep only Dashboard, Board browsing, Ticket detail, approval handling; direct users to desktop for complex configuration |

### 13.15 Accessibility and readability requirements

- Keyboard accessibility: all key buttons, cards, drawers, toggles should be keyboard reachable
- Color should not be the sole signal: error and success states must also use icons/text
- Provide non-drag fallback for dragging: status can be modified via a dropdown in the detail drawer
- SSE updates should be screen-reader friendly; important status changes should be announced with low-frequency aria-live updates

### 13.16 Core page list (Revised)

| Page | Positioning | Key content |
|------|------|---------|
| Dashboard | Default entry | Project health, running Agents, exceptions, cost, activity |
| Project Overview | Project entry | Project status summary, Board Snapshot, PR/Hook/approval overview |
| Board | Core workspace | Multi-column states, drag-and-drop, filtering, real-time updates |
| Ticket Detail | Deep work area | Description, execution, Hooks, PR, dependencies, activity |
| Workflows | Rule center | Workflow list, Harness editor, history, preview |
| Agents | Execution console | Agent definitions, running AgentRun, heartbeat, output, Provider configuration |
| Approvals | Risk governance | Approval queue, context, approve/reject |
| Activity | Audit and replay | Cross-entity event timeline |
| Insights | Operational analytics | Cost, throughput, completion rate, retries, SLA |
| Settings | Configuration center | Project, Repo, status columns, notifications, machines, security |
## Chapter 14 New User Onboarding Design

Even if a tool is very powerful, it will be abandoned if a new user cannot experience value in the first hour. OpenASE needs to separate **local runtime bootstrap** from broader product onboarding: what is currently delivered is terminal-first local setup; empty-project onboarding, team collaboration onboarding, and enterprise governance onboarding are still part of later product experience design and should not be written together with the current `openase setup`.

### 14.1 Current Default Path: Terminal-first Local Setup

The current local startup default entry is to explicitly run `openase setup`; `openase up` is only a compatibility entrypoint and delegates to the same terminal setup flow on first run when no configuration is found.

```bash
./openase setup

# Or
./openase up
```

The default setup flow stays entirely in the terminal, does not auto-open a browser, and does not require users to enter the Web UI first. The currently implemented steps are:

1. Choose database source: automatically launch a local Docker PostgreSQL, or manually provide an existing PostgreSQL connection.
2. Validate database connectivity and perform schema initialization in setup.
3. Check whether key local CLIs (such as `git`, `codex`, `claude`) are available.
4. Select browser authentication mode: `disabled` or `oidc`; if `oidc` is selected, setup directly collects fields such as issuer, client ID/secret, redirect URL, and scopes.
5. Choose the runtime mode after setup ends: `config-only`, or install the current-user managed service for the local platform (`systemd --user` on Linux, `launchd` on macOS).
6. Write `~/.openase/config.yaml`, `~/.openase/.env`, and create `~/.openase/logs/`, `~/.openase/workspaces/`.
7. Initialize default local Organization / Project, ticket statuses, and detected provider seed data.

Current setup **no longer** requires or defaults to the following actions:

- Does not treat Web Setup Wizard as the main path.
- Does not auto-open a browser or auto-redirect to `/setup` / Dashboard after completion.
- Does not collect repo URL, default branch, collaboration mode, or “individual / team / enterprise” mode cards during setup.
- Does not generate a repo-local `.openase/` scaffold in the repository root.

### 14.2 Compatibility Path: Legacy Web Setup and `openase up`

The code still keeps a legacy browser troubleshooting path:

```bash
openase setup --web --host 127.0.0.1 --port 19836
```

This entrypoint starts a lightweight HTTP server, exposes `/setup`, and attempts to open a browser; but it is a **hidden and deprecated compatibility path**, used only for troubleshooting or transition, and should no longer be described in PRD as the current default onboarding.

Similarly, `openase up` also needs to be described with two-phase semantics:

1. If `~/.openase/config.yaml` does not exist, it enters terminal setup.
2. If configuration already exists, it is only responsible for installing or refreshing the current-user service definition and is not equivalent to re-entering a multi-step Setup Wizard.

Therefore, `/setup`, automatic browser opening, and automatic post-completion redirection are all compatibility/transitional surface behaviors, not part of the core contract for current local setup.

### 14.3 Current Service Management Contract: Config-only + Current-user Managed Service

Current setup promises only two runtime options:

- `config-only`: setup only writes `~/.openase/*`, and the user then manually runs `openase all-in-one --config ~/.openase/config.yaml`.
- `current-user managed service`: setup installs and validates the managed service after writing configuration when the local platform supports the shipped user-service backend (`systemd --user` on Linux, `launchd` on macOS).

The current contract for related commands is as follows:

```bash
openase setup
openase up --config ~/.openase/config.yaml
openase all-in-one --config ~/.openase/config.yaml
openase logs --lines 200
openase restart
openase down
```

Three boundaries must be clear here:

- `openase up/down/restart/logs` are current-user service management commands; they are not browser onboarding flows.
- When setup inline-installs a service, the current main path is platform-specific: Linux uses `systemd --user`, macOS uses `launchd`, and unsupported environments explicitly fall back to `config-only`.
- The implementation layer retains a shared user-service abstraction so the command surface stays consistent across platforms, but macOS `launchd` is part of the current shipped contract rather than a compatibility-only or future-only path.

### 14.4 Current Scope, Compatibility Scope, and Non-goals

To prevent setup/service onboarding from drifting again, this chapter defines boundaries as follows:

- Already implemented:
  - terminal-first local setup
  - `disabled` / `oidc` authentication modes
  - `config-only` / current-user managed service runtime options (`systemd --user` on Linux, `launchd` on macOS)
  - initialization of config, logs, workspace, and default control plane data under `~/.openase/`
- Compatibility paths:
  - `openase setup --web` legacy browser flow
  - `openase up` dual-mode behavior of entering setup when no config exists / installing service when config exists
  - cross-platform fallback from managed service installation to `config-only` when the local login session cannot support the expected service manager
- Non-goals / not-yet-delivered commitments:
  - Using Web Setup Wizard as the current primary path
  - automatic browser redirect after setup completion
  - repo binding, project template selection, or persona routing during setup
  - mixing broader team/enterprise onboarding into local runtime bootstrap

Sections 14.5+ describe OpenASE product-level onboarding planned after startup and are meant to define later UI/empty-state experiences; they are not equivalent to behavior already provided by the current `openase setup`.

### 14.5 First-Time Tour Experience for an Empty Project (Product-level Subsequent Onboarding, Not Equivalent to Current local setup)

Empty project onboarding is the **second-stage product onboarding** after OpenASE is running. Current `openase setup` is responsible for bringing the platform up; empty-project onboarding is responsible for moving a newly created project into a **state where an Agent can start working**.

#### 14.5.1 Trigger Conditions and Entry

When the following conditions are met, the frontend must enter `empty_project_onboarding` mode, rather than dropping the user into an ordinary but empty project Dashboard:

- The current user has just created a new project, or is entering a project for the first time that has not completed initialization
- Current project `tickets.count == 0`
- Current project has not completed the first bootstrap round (see checklist below)

The entry behavior must be fixed:

- After completing terminal setup and starting OpenASE, the user enters the project-level Dashboard for the first time
- The left sidebar automatically focuses the newly created project
- The top displays a welcome strip: `Project created. Next, we'll help you configure it into a runnable state`
- The main area renders an **unmissable Onboarding panel** that presents, in order:
  1. Configure GitHub Token
  2. Create or link a Repo
  3. Configure at least one Provider
  4. Automatically create the first Agent and Workflow
  5. Create the first Ticket
  6. Observe Agent updates in the Ticket
  7. Experience Project AI and Harness AI

This panel is not a “suggested list,” but the **main workspace for the empty-state project**. Users may leave the page, but returning to that project Dashboard must resume from the first unfinished step.

#### 14.5.2 Step A — GitHub Token Must be Completed First

The first hard gate of empty-project onboarding is GitHub outbound credentials. Without a valid `GH_TOKEN`, continuation to the Repo create/link step is not allowed.

Completion criteria:

- The platform-hosted `GH_TOKEN` exists in the current project scope (project-level scope preferred, otherwise organization default)
- `GH_TOKEN` probe status reaches `valid`
- User completes one GitHub identity confirmation

Page requirements:

- The first card title in the Onboarding panel is fixed as: `Connect GitHub`
- Provide two primary paths:
  - `Import automatically from local gh`
  - `Paste Token manually`
- The manual paste path must clearly explain how to obtain a token:
  - If not logged into GitHub CLI yet: run `gh auth login` first
  - Get current token: run `gh auth token`
- If the machine has `gh` detected and can successfully return a token, the UI provides one-click import; after clicking, the platform saves the token with `gh_cli_import` semantics.
- If the user takes the manual paste path, the platform saves the token with `manual_paste` semantics.

Required forced actions after saving:

1. The platform immediately performs a GitHub probe.
2. UI shows `Verifying GitHub identity and permissions...`
3. After successful probe, UI must display the detected GitHub username/login, for example `GitHub account octocat will be used`
4. User must click confirmation once: `This is the GitHub identity I want to use for this project`

Failure and exception rules:

- If the probe has not reached `valid`, the step remains incomplete and downstream Repo steps are disabled.
- If the username does not match user expectation, users can re-import/replace the token.
- The UI must not mark this step complete just because the token string is non-empty; completion must be based on probe results.

#### 14.5.3 Step B — Guide the User to Create or Link a Repo

After GitHub Token completion, onboarding automatically advances to the Repo step. The goal here is not to make users “know that multiple repos are supported,” but to ensure the project has at least one real accessible code repository.

Completion criteria:

- The current project has at least one `ProjectRepo`
- Every newly created or linked repo has passed GitHub permission and accessibility checks

Page requirements:

- Card title fixed as: `Create or Link Code Repository`
- Must provide both paths:
  - `Create New Repository`
  - `Link Existing Repository`
- `Create New Repository` must require at least:
  - Repo name
  - Visibility (`private` / `public`)
  - Default branch (default `main`)
- `Link Existing Repository` must require at least:
  - Repo name
  - Git URL

Interaction rules:

- If the user chooses to create a new repo, the platform creates the repo through GitHub API and immediately registers it to the project.
- If the user chooses to link an existing repo, the platform must first validate that the current `GH_TOKEN` has at least minimum required access to that repo, then persist it as `ProjectRepo`.
- Single-repo projects allow completion in one step; multi-repo projects allow adding multiple repos consecutively.
- If the user’s project has a typical frontend/backed structure, UI may recommend `backend` / `frontend` templates, but recommendation does not mean automatic creation.

After this step is complete, Dashboard should update:

- The top welcome strip shows repo connection status.
- Board Snapshot continues to show empty columns, but instead of prompting “connect repo first,” it moves on to Provider step.

#### 14.5.4 Step C — Configure Claude Code / Codex / Gemini CLI

After Repo completion, the user must be guided to configure at least one runnable Provider. Provider onboarding is the third hard gate of empty-project onboarding.

Completion criteria:

- Current project has at least one Provider with `availability_state = available`
- User explicitly selects one as the default execution Provider for the current project

Page requirements:

- Card title fixed as: `Select and Configure AI Provider`
- By default, display three Provider cards:
  - `Claude Code`
  - `OpenAI Codex`
  - `Gemini CLI`
- Each card must display:
  - Whether CLI is detected
  - Current availability status (`available` / `unavailable` / `stale` / `unknown`)
  - Recommended model
  - Whether logged in
  - `Set as Default` or `Continue Setup`

Interaction rules:

- On page entry, automatically perform local PATH detection and provider availability checks.
- If a CLI is available, the corresponding card shows the primary button `Use This Provider`.
- If a CLI is not fully configured, clicking its card opens a provider-specific tutorial drawer, rather than only showing a single “unavailable” message.

provider-specific tutorial must at least cover:

- Claude Code:
  - Install Claude Code CLI
  - Login / authentication
  - How to verify `claude --version` and login status
- OpenAI Codex:
  - Install Codex CLI
  - Login / API authentication
  - How to verify `codex --version` and availability
- Gemini CLI:
  - Install Gemini CLI
  - Login / API authentication
  - How to verify `gemini --version` and availability

Selection rules:

- If multiple CLIs are available, users can directly click any available card.
- If no CLI is available, progression to Agent/Workflow step is not allowed.
- After selection, the default Provider for the current project must be explicitly set and cannot be left empty for system guesswork.

#### 14.5.5 Step D — Automatically Create the First Agent and Workflow Based on Project Status

Once at least one Provider is available, onboarding enters the bootstrap step. This does not require users to understand abstractions like Harness, Workflow, or Agent first; instead, it requires the system to automatically generate a set of **immediately runnable built-in presets** based on `project.status`.

Completion criteria:

- Platform has created 1 Agent
- Platform has created and published 1 Workflow
- Newly created Workflow is bound to the Agent above, and Agent is bound to the user-selected Provider

Branch rules must be hardcoded as follows:

- When `project.status = Planned`
  - Recommended role: `Product Manager`
  - Built-in `role_name = product-manager`
  - Workflow type: `custom`
  - Default pickup status: `Backlog`
  - Default finish status: `Done`
  - Suggested default Agent name: `product-manager-01`
- When `project.status = In Progress`
  - Recommended role: `Coder`
  - Built-in `role_name = fullstack-developer`
  - Workflow type: `coding`
  - Default pickup status: `Todo`
  - Default finish status: `Done`
  - Suggested default Agent name: `coder-01`

Rules for other project statuses:

- `Backlog`: process as `Planned` by default, recommend `Product Manager`
- `Completed` / `Canceled` / `Archived`: do not automatically create execution roles; show explanation and require user to change project status first

UI interaction requirements:

- UI must clearly display a summary of objects to be created:
  - Provider: which one
  - Agent: name, concurrency, enabled status
  - Workflow: role, pickup, finish, built-in Harness used
- After user confirmation, the platform completes in one go:
  1. Create Agent
  2. Create Workflow
  3. Bind Agent ↔ Provider
  4. Bind Workflow ↔ Agent
  5. Immediately publish the current Workflow version

This step must be “auto-create preset,” not dropping the user into a Workflow editor to build from scratch.

#### 14.5.6 Step E — Guide the User to Create the First Ticket, and Immediately Jump to Ticket Detail to Observe Execution

After Agent and Workflow creation, the next step in empty-project onboarding is the first Ticket.

Completion criteria:

- Current project has created at least 1 Ticket
- User has seen at least one in-progress event from the Agent in Ticket detail page

Ticket creation form requirements:

- Title input automatically focused
- Default recommendation is the newly created bootstrap Workflow
- Single-repo projects auto-populate the unique repo; multi-repo projects require explicit repo scope selection
- Form footer must explain:
  - Which pickup status the ticket will enter
  - When orchestrator will claim it
  - Where Agent updates will appear

Default examples by project status:

- `Planned` / `Backlog`
  - Default sample Ticket: `Review project requirements and produce the first PRD draft`
  - Default entry into `Product Manager` Workflow pickup status
- `In Progress`
  - Default sample Ticket: `Implement the project's first core feature`
  - Default entry into `Coder` Workflow pickup status

Post-create page flow must be fixed:

1. After successful creation, automatically navigate to Ticket detail page instead of staying on Dashboard.
2. On first entry to Ticket detail, a guide strip appears at top: `Once the Agent starts working, updates will appear here in real time`
3. Use highlight prompts to direct users to three areas:
  - Ticket status and current Workflow
  - Activity / Step timeline
  - Agent live output area
4. This step is marked complete only after the system receives the first `AgentStepEvent` or equivalent runtime output.

CLI path is still kept, but not the primary path:

```bash
openase ticket create --title "Add input validation to login form" --workflow coding
openase watch ASE-1
```

#### 14.5.7 Step F — Let Users Immediately Experience Project AI and Harness AI

After the first Ticket has started executing, onboarding should not end immediately; it should continue to lead users to two high-value AI entry points in OpenASE:

- `Project AI`
- `Harness AI`

The goal here is not to explain concepts again, but to let users complete the two most important “next actions” in the same project.

Project AI onboarding requirements:

- After the first Agent update appears, Dashboard or project sidebar shows CTA: `Let Project AI continue breaking down the requirements`
- Clicking opens `project_sidebar`
- First prefilled question suggestions:
  - `Based on the current project and existing Tickets, help me break down 3 more follow-up tickets`
  - `What should I do next?`
- After user confirmation, Project AI can help create follow-up tickets via `platform_command_proposal`; compatibility with legacy `action_proposal` is still maintained during migration.

Harness AI onboarding requirements:

- When a newly created bootstrap Workflow exists, show CTA: `Use Harness AI to adjust how this role works`
- Clicking opens the newly created Workflow editor and Harness AI sidebar directly
- First prefilled question suggestions:
  - `Help me optimize this Workflow so it fits the current project better`
  - `Help me add clearer acceptance criteria for this role`

Completion rules:

- Empty project onboarding can only be marked fully complete when the user completes at least one Project AI interaction and has opened Harness AI at least once.
- After completion, Dashboard switches from “strong guidance mode” to regular project Dashboard.

#### 14.5.8 Fixed Layout on First Dashboard Entry

In `empty_project_onboarding` mode, project-level Dashboard is not a normal overview page, but an operation panel with a clear sequence. The layout must be fixed as:

| Area | Content |
|------|---------|
| Top welcome strip | Current project name, project status, GitHub connection status, default Provider status |
| Onboarding Checklist | GitHub Token → Repo → Provider → Agent/Workflow → First Ticket → Observe Ticket → Project AI / Harness AI |
| Board Snapshot | Initial columns are empty, but each column’s purpose is described |
| Help entry | GitHub token help, CLI installation docs, sample Harness, CLI examples |

Checklist rules:

- Incomplete steps show primary CTA
- Completed steps show summary and “Reconfigure”
- Later steps may be visible but must be disabled until preceding steps are complete
- After page refresh or project switch, Checklist must restore to current true completion progress, not lose state

### 14.6 Progressive Unlock

The initial experience is intentionally simplified; complex features are progressively unlocked in Web UI via “milestone prompts”:

| Milestone | Trigger condition | Prompt message |
|-----------|-------------------|----------------|
| First ticket completed | First ticket is done | "Your first ticket is complete! Try creating multiple tickets so Agents can work in parallel." |
| 5 tickets completed | Cumulative 5 done | "Harness has already run 5 times. Want to optimize the working spec on the Workflow management page?" |
| First lifecycle Hook failure | Any lifecycle Hook returns non-zero | "The on_complete Hook failed. Check the Hook log in Ticket detail." |
| 10 tickets completed | Cumulative 10 done | "Want to check the cost dashboard to understand Agent performance and token usage?" |
| First multi-repo ticket | Involves 2+ repos | "A multi-repo ticket has been created! The Agent will modify all repositories together in the combined workspace." |
| Cost reaches $10 | Cumulative cost > 10 | "You have already consumed $10 in API cost. Configure a budget cap in Settings?" |

### 14.7 `openase doctor` — Environment Diagnostics

When users encounter issues, both CLI and Web UI provide diagnostic entry points:

```
$ openase doctor

🔍 OpenASE Environment Diagnostics

  ✅ Git 2.44.0
  ✅ Claude Code installed and usable
  ⚠️  Codex not installed (optional)
  ✅ PostgreSQL 16 connected (localhost:5432)
  ✅ ~/.openase/ config directory complete
  ✅ 2 Workflow Harness versions initialized
  ✅ 3 Hook scripts configured
  ⚠️  on_complete Hook "run-tests.sh" missing executable permission
     → Fix: chmod +x scripts/ci/run-tests.sh

Summary: 1 warning, 0 errors
```
## Chapter 15 Engineering Standards and Pre-commit

Coding standards are not documentation—they are automatically enforced. OpenASE uses **lefthook** (a Go-native Git hooks manager, faster than Python pre-commit, zero external dependencies, aligned with the Go toolchain) to enforce all standards.

### 15.1 Go Code Standards

**Formatting: no room for debate**

| Tool | Purpose | Rules |
|------|---------|-------|
| `gofmt` | Basic formatting | Go standard, not configurable |
| `goimports` | import sorting + automatic import completion | Grouped as stdlib → external → internal, separated by blank lines |

```go
import (
    "context"          // stdlib
    "fmt"

    "github.com/labstack/echo/v4"    // external dependency
    "go.opentelemetry.io/otel"

    "github.com/BetterAndBetterII/openase/internal/domain/ticketing" // internal package
    "github.com/BetterAndBetterII/openase/internal/httpapi"
)
```

**Lint: golangci-lint (strict configuration)**

```yaml
# .golangci.yml
run:
  timeout: 5m

linters:
  enable:
    # Required
    - errcheck         # unchecked error return values
    - govet            # built-in go vet checks
    - staticcheck      # most comprehensive static analysis
    - unused           # unused code
    - gosimple         # simplifiable code
    - ineffassign      # ineffective assignments
    - typecheck        # type checking

    # Code quality
    - gocritic         # code style and performance suggestions
    - revive           # configurable golint replacement
    - misspell         # misspellings in comments and strings
    - prealloc         # preallocatable slices
    - unconvert        # unnecessary type conversions
    - unparam          # unused function parameters

    # Security
    - gosec            # security issue detection
    - bodyclose        # HTTP response body not closed

    # DDD architecture guard
    - depguard         # dependency direction constraints (critical!)

linters-settings:
  depguard:
    rules:
      domain-no-infra:
        files:
          - "domain/**"
        deny:
          - pkg: "github.com/openase/openase/infra"
            desc: "domain layer must not depend on infra layer"
          - pkg: "github.com/openase/openase/api"
            desc: "domain layer must not depend on interface layer"
          - pkg: "github.com/openase/openase/app"
            desc: "domain layer must not depend on application layer"
      app-no-infra-direct:
        files:
          - "app/**"
        deny:
          - pkg: "github.com/openase/openase/infra"
            desc: "application layer must not directly depend on infra layer (inject through interface)"
          - pkg: "github.com/openase/openase/api"
            desc: "application layer must not depend on interface layer"

  revive:
    rules:
      - name: exported
        arguments: [checkPrivateReceivers]
      - name: unexported-return
      - name: blank-imports
      - name: context-as-argument
      - name: error-return
      - name: error-naming
      - name: increment-decrement
```

**depguard / architecture guard is the key architecture guard**—it enforces the repository’s dependency direction during linting: `internal/domain` / `internal/types` cannot depend upward on `repo/service/httpapi/app wiring`, `internal/service` cannot import `internal/httpapi` / `internal/setup` / `cmd/openase`, and new `ent/*` entries are forbidden from entering the otherwise clean domain/service boundary. Violations fail the check and the PR cannot be merged.

**Naming standards**

| Scenario | Rule | Example |
|----------|------|---------|
| Package name | Lowercase words, no underscores/camelCase | `ticket`、`claudecode`、`gitops` |
| Interface | Verb/noun, no `I` prefix | `Repository`、`Adapter`、`Executor` |
| Interface implementation | Noun phrase | `EntTicketRepo`、`ClaudeCodeAdapter` |
| Error variable | `Err` prefix | `ErrTicketNotFound`、`ErrHookFailed` |
| Context | First parameter | `func (s *Service) Claim(ctx context.Context, ...) error` |
| Constructor | `New` + type name | `NewScheduler(cfg Config) *Scheduler` |
| DTO | `XxxRequest` / `XxxResponse` | `CreateTicketRequest`、`TicketDetailResponse` |
| Use-case Input | `XxxInput` / `CreateXxx` / `UpdateXxx` | `PullRequestStatusInput`、`CreateInput`、`UpdateAgentProvider` |
| Domain Event | Past tense | `TicketClaimed`、`HookFailed`、`AgentStalled` |

**Error handling**

```go
// ✅ Always wrap errors and add context
return fmt.Errorf("claim ticket %s: %w", ticketID, err)

// ✅ Use custom types for domain errors
var ErrTicketAlreadyClaimed = errors.New("ticket already claimed")

// ❌ Do not ignore error
_ = db.Close()  // golangci-lint errcheck will block this

// ❌ Do not panic (except truly unrecoverable cases in init phase)
```

**Testing standards**

| Type | Location | Naming | Tool |
|------|----------|--------|------|
| Unit tests | Same directory as tested code | `*_test.go` | `testing` + `testify/assert` |
| Integration tests | `tests/integration/` | `*_integration_test.go` | `testcontainers-go` (PostgreSQL) |
| Architecture tests | With golangci-lint | depguard rules | `golangci-lint` |

```go
// Unit test naming: Test_<MethodName>_<Scenario>_<Expected>
func Test_StateMachine_ClaimTicket_BlockedByDependency(t *testing.T) { ... }
func Test_StateMachine_CompleteTicket_HookFailure_StaysInProgress(t *testing.T) { ... }
```

### 15.2 SvelteKit Frontend Code Standards

**Formatting: Prettier**

```json
// .prettierrc
{
  "semi": false,
  "singleQuote": true,
  "trailingComma": "all",
  "printWidth": 100,
  "tabWidth": 2,
  "useTabs": false,
  "plugins": ["prettier-plugin-svelte", "prettier-plugin-tailwindcss"],
  "overrides": [
    { "files": "*.svelte", "options": { "parser": "svelte" } }
  ]
}
```

`prettier-plugin-tailwindcss` automatically sorts Tailwind classes, so the team does not need to discuss class order.

**Lint: ESLint (flat config)**

```js
// eslint.config.js
import svelte from 'eslint-plugin-svelte'
import ts from '@typescript-eslint/eslint-plugin'

export default [
  ...svelte.configs['flat/recommended'],
  {
    rules: {
      // Strict TypeScript mode
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/strict-boolean-expressions': 'warn',

      // Svelte specific
      'svelte/no-at-html-tags': 'error',       // Prevent XSS
      'svelte/require-each-key': 'error',       // {#each} must include key
      'svelte/no-unused-svelte-ignore': 'error',

      // General
      'no-console': ['warn', { allow: ['warn', 'error'] }],
      'prefer-const': 'error',
    }
  }
]
```

**TypeScript: strict mode**

```json
// key tsconfig.json settings
{
  "compilerOptions": {
    "strict": true,
    "noUncheckedIndexedAccess": true,
    "noImplicitOverride": true,
    "exactOptionalPropertyTypes": true
  }
}
```

**Svelte file structure convention**

```svelte
<!-- 1. script (logic on top) -->
<script lang="ts">
  import { Badge } from '$lib/components/ui/badge'
  import type { Ticket } from '$lib/api/types'

  // props → state → derived → lifecycle → functions
  let { ticket }: { ticket: Ticket } = $props()
  let loading = $state(false)
  let statusLabel = $derived(ticket.status.toUpperCase())

  function handleClaim() { ... }
</script>

<!-- 2. markup (structure in middle) -->
<div class="flex items-center gap-2">
  <Badge variant="outline">{statusLabel}</Badge>
  <button onclick={handleClaim}>Claim</button>
</div>

<!-- 3. style Prefer Tailwind as much as possible, avoid <style> -->
```

**Frontend directory structure convention**

```
web/src/
├── routes/                    # SvelteKit page routes
│   ├── (app)/                 # Auth-required layout group
│   │   ├── tickets/
│   │   ├── workflows/
│   │   ├── agents/
│   │   └── settings/
│   └── (setup)/               # Setup Wizard layout group
├── lib/
│   ├── components/
│   │   ├── ui/                # shadcn-svelte primitive components
│   │   ├── ticket/            # ticket-related business components
│   │   ├── agent/             # Agent-related business components
│   │   └── layout/            # Shell, Sidebar, Header
│   ├── api/
│   │   ├── types.ts           # openapi-typescript generated
│   │   ├── client.ts          # fetch wrapper
│   │   └── sse.ts             # low-level SSE connector (reconnect / parse), no project-level state aggregation
│   ├── features/project-events/ # project-level passive event bus runtime / store
│   ├── stores/                # global Svelte stores
│   └── utils/
└── app.css                    # Tailwind entry
```

#### 15.2.1 Component layering and reuse standards

The frontend must follow component layering from **general to specific** to avoid page-level growth into one giant template tree.

| Layer | Suggested directory | Responsibility | Forbidden |
|-------|--------------------|----------------|-----------|
| UI Primitive layer | `lib/components/ui/` | Basic components like Button, Card, Dialog, Tabs, Badge | Must not include business semantics or API calls |
| App Shell layer | `lib/components/layout/` | Sidebar, TopBar, PageHeader, RightDrawer, EmptyState | Must not directly know ticket/workflow business details |
| Feature component layer | `lib/features/<feature>/components/` | BoardColumn, TicketCard, AgentList, ProjectHealthPanel | Must not arbitrarily depend on each other’s internal implementation across features |
| Feature state layer | `lib/features/<feature>/stores.ts` | Feature-local state, filters, view modes, derived selectors | Must not perform DOM rendering |
| Runtime boundary layer | `lib/features/project-events/` | project passive event bus, project-level in-memory state and subscription boundary | Must not directly render pages |
| Feature data layer | `lib/features/<feature>/api.ts` | API calls, response parsing, feature query encapsulation | Must not do page layout |
| Route assembly layer | `routes/**/+page.svelte` | Assemble page sections, bind route params, organize features | Must not carry extensive business implementation details |

**Hard rules:**

- `routes/**/+page.svelte` should only assemble; it cannot simultaneously carry large data model definitions, SSE protocol parsing, business rule branching, and complex rendering details
- Place business components primarily under `lib/features/`, not dump everything indiscriminately into `lib/components/`
- `ui/` components must not depend on any `feature/` or `routes/`
- `feature/` components may not import private files from other features, only public components/types they expose

#### 15.2.2 Feature-first directory design

As page complexity grows, organizing strictly by “page files” quickly becomes unmanageable. OpenASE frontend should be organized feature-first:

```
web/src/lib/features/
├── board/
│   ├── api.ts
│   ├── stores.ts
│   ├── types.ts
│   ├── mappers.ts
│   └── components/
│       ├── board-view.svelte
│       ├── board-toolbar.svelte
│       ├── board-column.svelte
│       ├── ticket-card.svelte
│       └── board-filters.svelte
├── ticket-detail/
│   ├── api.ts
│   ├── stores.ts
│   ├── types.ts
│   └── components/
├── dashboard/
├── agents/
└── workflows/
```

**Why this design:**

- Board, Ticket Detail, Dashboard, Agents are all evolving large features
- Keeping types, API, store, and components together improves context concentration during changes
- Reduces the tendency for page files to “know everything”

#### 15.2.3 Route files are assembly only, not the brain

Each page follows this responsibility split:

| File | Responsibility |
|------|----------------|
| `+page.ts` / `+layout.ts` | Read route params, perform first-screen load, handle URL search params |
| `+page.svelte` | Assemble page sections, connect feature stores, pass minimal props |
| `lib/features/*/api.ts` | Request and parse remote data |
| `lib/features/*/stores.ts` | Local state, filters, derived views |
| `lib/features/project-events/*` | project passive bus transport, event parsing, shared in-memory subscriptions |
| `lib/features/*/components/*.svelte` | Single visual or interactive blocks |

**Route files must avoid these smells:**

- Defining dozens of domain types directly in `+page.svelte`
- Writing multiple API fetch wrappers directly in `+page.svelte`
- Handling protocol parsing for multiple SSE topics directly in `+page.svelte`
- Maintaining 20+ `$state` variables in `+page.svelte` simultaneously
- One page rendering all UI while also containing all mappers, formatters, and error handling

#### 15.2.4 Recommended page decomposition pattern

Using the project Dashboard as an example, `+page.svelte` should not contain all card implementations; it should be assembled like this:

```svelte
<script lang="ts">
  import ProjectOverviewHeader from '$lib/features/dashboard/components/project-overview-header.svelte'
  import ProjectHealthPanel from '$lib/features/dashboard/components/project-health-panel.svelte'
  import RunningAgentsPanel from '$lib/features/dashboard/components/running-agents-panel.svelte'
  import ExceptionPanel from '$lib/features/dashboard/components/exception-panel.svelte'
  import ActivityFeedPanel from '$lib/features/dashboard/components/activity-feed-panel.svelte'
  import { createProjectDashboardStore } from '$lib/features/dashboard/stores'

  let { data } = $props()
  const dashboard = createProjectDashboardStore(data.projectId)
</script>

<ProjectOverviewHeader project={dashboard.project} />
<div class="grid gap-4 xl:grid-cols-[2fr,1fr]">
  <ProjectHealthPanel summary={dashboard.summary} />
  <RunningAgentsPanel agents={dashboard.agents} />
  <ExceptionPanel items={dashboard.exceptions} />
  <ActivityFeedPanel items={dashboard.activity} />
</div>
```

**Goal:** a page file should read like a page map, not like a giant program novel.

#### 15.2.5 File Budget

To prevent single-page bloat, the frontend introduces an explicit file-size budget:

| File type | Soft cap | Hard cap | Handling requirement |
|-----------|----------|----------|---------------------|
| `routes/**/+page.svelte` | 150 lines | 250 lines | Must split into sections above soft cap; above hard cap is not mergeable |
| `routes/**/+layout.svelte` | 180 lines | 300 lines | Complex layout logic should be moved into layout components |
| `lib/features/**/components/*.svelte` | 200 lines | 300 lines | Split into smaller panel / row / item components when exceeded |
| `lib/components/ui/*.svelte` | 150 lines | 250 lines | Primitive components must stay minimal |
| `*.ts` store / api files | 200 lines | 300 lines | Excess length indicates the feature split is unsuitable |
| Single function | 40 lines | 60 lines | Extract helper or smaller functions when exceeded |

**Exception rules:**

- Very few compound editor components may exceed hard caps, but reasons must be explained in the PR
- Over-limit files need explicit justification in code review why they cannot be split

#### 15.2.6 Frontend state management standards

Preventing page bloat is not only splitting UI, but also splitting state and data responsibilities.

**Rules:**

- Only cross-page shared state should go into `lib/stores/`
- Single-page but complex state goes into feature stores; do not pile it into route files
- Project-level passive SSE transport only enters project event runtime; feature/page layers subscribe to its in-memory state or derived events only
- Parse API responses into domain-friendly frontend types at the boundary, then pass to components
- Presentation components should be as “pure renderers”: receive props and output UI, no direct fetches

**Recommended practice:**

- `retainProjectEventBus(projectId)` maintains one passive connection in the project shell
- `createBoardStore(projectId)` centrally manages columns, cards, filters, drag updates, and consumes project event bus derived state
- `TicketCard.svelte` focuses only on how to render a single card, not whether data came from HTTP or SSE

#### 15.2.7 Frontend dependency direction

The frontend must follow the same dependency discipline as the backend:

```text
ui primitives
  → layout
  → features
  → routes
```

and:

```text
types / mappers
  → api / stores
  → components
  → routes
```

**Forbidden:**

- route being reverse-dependent on feature
- feature A directly imports route files of feature B
- UI primitive importing feature store
- Similar logic being copy-pasted among page components without abstracting a shared section

#### 15.2.8 What lint/tools can prevent runaway pages?

**Yes, and they should be layered:**

1. `ESLint` for static rules
2. `dependency-cruiser` or `eslint-plugin-boundaries` for dependency boundary checks
3. Custom `check-file-budgets` script for line-limit enforcement
4. CI blocking on budget violations before merge

**Recommended ESLint rules:**

```js
// eslint.config.js additional rules
import sonarjs from 'eslint-plugin-sonarjs'
import importPlugin from 'eslint-plugin-import'

export default [
  ...svelte.configs['flat/recommended'],
  sonarjs.configs.recommended,
  importPlugin.flatConfigs.recommended,
  {
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: true, skipComments: true }],
      'max-lines-per-function': ['warn', { max: 60, skipBlankLines: true, skipComments: true }],
      'complexity': ['warn', 10],
      'max-depth': ['warn', 4],
      'import/no-cycle': 'error',
      'sonarjs/cognitive-complexity': ['warn', 15],
    }
  },
  {
    files: ['src/routes/**/*.svelte'],
    rules: {
      'max-lines': ['error', { max: 250, skipBlankLines: true, skipComments: true }]
    }
  }
]
```

**Notes:**

- `max-lines` can detect “overly long files”, and works for `.svelte`, but it only flags outcomes and does not tell you how to split
- `complexity` and `sonarjs/cognitive-complexity` catch overweight script logic
- `import/no-cycle` prevents circular dependencies created after splitting pages

#### 15.2.9 ESLint is not enough, file budget scripts are needed

ESLint checks generic rules, but for team constraints like “different budgets per directory,” a custom script is a better layer, such as:

```bash
node scripts/check-file-budgets.mjs
```

Script responsibilities:

- Check whether `src/routes/**/*.svelte` exceeds 250 lines
- Check whether `src/lib/features/**/*.svelte` exceeds 300 lines
- Check whether `src/lib/components/ui/**/*.svelte` exceeds 250 lines
- Output files over the budget and return non-zero in CI

These scripts are more direct than lint and are suitable for team governance guardrails.

#### 15.2.10 Frontend structure checklist for Code Review

In addition to functional correctness, every frontend PR must pass structural review:

- Whether the page route file is only assembly, instead of packing all logic in one place
- Whether reusable components were added instead of duplicating similar UI
- Whether API, SSE, formatter, and UI state are mixed in the same file
- Whether route files are 250+ lines or feature components are 300+ lines
- Whether cross-layer dependencies, circular dependencies, or private feature-to-feature imports were introduced
- Whether there is a “too-lazy” case where a set of panels is written into one page component

**Current repository risk signals:**

- `web/src/routes/+page.svelte` has reached thousands of lines; this file size must be treated as an architectural warning, not “we will split it later”
- If Dashboard, Board, Settings, and Onboarding are all implemented without these constraints, pages will continuously evolve into unmaintainable giant files

#### 15.2.11 Recommended new scripts and commands

```json
// Suggested additions to package.json scripts
{
  "scripts": {
    "lint": "eslint .",
    "lint:structure": "node scripts/check-file-budgets.mjs",
    "lint:deps": "depcruise src --config .dependency-cruiser.cjs",
    "check": "svelte-kit sync && svelte-check --tsconfig ./tsconfig.json",
    "ci": "pnpm run lint && pnpm run lint:structure && pnpm run lint:deps && pnpm run check && pnpm run build"
  }
}
```

This is how “component reuse standards” and “non-inflating pages” become automated checks instead of verbal requirements.

### 15.3 Pre-commit configuration (lefthook)

```yaml
# lefthook.yml

pre-commit:
  parallel: true
  commands:
    # ── Go ──
    go-fmt:
      glob: "*.go"
      run: goimports -w {staged_files} && git add {staged_files}
      stage_fixed: true

    go-lint:
      glob: "*.go"
      run: golangci-lint run --new-from-rev=HEAD~1 ./...

    go-test:
      glob: "*.go"
      run: go test ./internal/domain/... ./internal/service/... ./internal/ticket ./internal/workflow ./internal/httpapi -short -count=1

    go-vet:
      glob: "*.go"
      run: go vet ./...

    go-mod-tidy:
      glob: "go.{mod,sum}"
      run: go mod tidy && git diff --exit-code go.mod go.sum

    # ── Frontend ──
    fe-format:
      root: "web/"
      glob: "*.{ts,svelte,css,json}"
      run: pnpm exec prettier --write {staged_files} && git add {staged_files}
      stage_fixed: true

    fe-lint:
      root: "web/"
      glob: "*.{ts,svelte}"
      run: pnpm exec eslint {staged_files}

    fe-typecheck:
      root: "web/"
      glob: "*.{ts,svelte}"
      run: pnpm exec svelte-check --tsconfig ./tsconfig.json

    # ── General ──
    no-secrets:
      run: |
        if git diff --cached --diff-filter=ACM | grep -iE '(password|secret|api_key|token)\s*[:=]' | grep -v 'test\|mock\|example'; then
          echo "Possible hard-coded secret detected, please check"; exit 1
        fi

    no-large-files:
      run: |
        for f in $(git diff --cached --name-only --diff-filter=ACM); do
          size=$(wc -c < "$f")
          if [ "$size" -gt 1048576 ]; then
            echo "$f exceeds 1MB"; exit 1
          fi
        done

commit-msg:
  commands:
    conventional-commit:
      run: |
        MSG=$(cat {1})
        if ! echo "$MSG" | grep -qE '^(feat|fix|refactor|docs|test|chore|ci|perf)(\(.+\))?: .{1,72}$'; then
          echo "Commit message does not follow the convention"
          echo "Format: type(scope): description"
          echo "Example: feat(ticket): add multi-repo support"
          exit 1
        fi
```

**Execution flow (parallel, total time < 10s):**

```
git commit
  ├── [parallel] go-fmt + fe-format      → auto-fix formatting, re-stage
  ├── [parallel] no-secrets              → scan for hard-coded secrets
  ├── [parallel] no-large-files          → reject files over 1MB
  ├── [parallel] go-lint (includes depguard)   → architecture guard + code quality
  ├── [parallel] go-vet + go-test        → static checks + unit tests
  ├── [parallel] fe-lint + fe-typecheck  → ESLint + type checking
  ├── [parallel] go-mod-tidy             → go.mod consistency
  └── [commit-msg] conventional-commit   → commit message format
```

### 15.4 Conventional Commits

| Type | Meaning | Scope (aligned with DDD domain model) |
|------|---------|--------------------------------------|
| `feat` | New feature | `ticket`、`workflow`、`agent`、`project`、`approval`、`hook` |
| `fix` | Bug fix | `orchestrator`、`adapter`、`api`、`web`、`setup` |
| `refactor` | Refactor | same as above |
| `docs` | Documentation | `harness`、`readme` |
| `test` | Testing | same as feat |
| `chore` | Chores | `deps`、`ci`、`config` |
| `perf` | Performance | same as feat |

### 15.5 Pre-commit vs CI responsibility split

| Check item | Pre-commit (local, fast) | CI (remote, comprehensive) |
|------------|---------------------------|----------------------------|
| gofmt + goimports | Auto-fix | Check only (no auto-fix) |
| golangci-lint | Incremental (changes in this commit) | Full |
| go test (unit) | `internal/domain` + service/use-case + `internal/httpapi`, `-short` | All packages, including coverage |
| go test (integration) | Skipped | testcontainers-go |
| Prettier | Auto-fix | Check |
| ESLint + svelte-check | Incremental | Full |
| Secret scan | simple grep | full gitleaks |
| depguard architecture guard | with golangci-lint | with golangci-lint |
| SvelteKit build | Skipped | `pnpm run build` |
| OpenAPI contract regenerate + diff | run `make openapi-generate` as needed | enforce `make openapi-check`, requiring `api/openapi.json` and `web/src/lib/api/generated/openapi.d.ts` to be committed |
| Coverage report | Skipped | `go test -coverprofile` |

**Backend-frontend handoff rules**:

1. Once backend HTTP handlers, request structures, or response structures change, regenerate `api/openapi.json` first.
2. The frontend should only consume type contracts generated from `api/openapi.json` and should no longer maintain a separate independently handwritten interface definition set.
3. Before PR merge, CI must pass `make openapi-check`, using generated diff to guarantee backend and frontend interfaces are synchronized.

---
## Chapter 16 Core Data Flow (Pseudocode)

This chapter uses pseudocode to show the full data flow, cross-layer call relationships, and notification methods of the six most critical paths in OpenASE. The pseudocode follows a DDD layered structure: `handler → command/query → domain service → repository / provider`.

### 16.1 Path 1: Create ticket (User → System)

This is the most basic write path, showing the standard call chain of Interface → Application → Domain → Infrastructure.

```
┌──────────┐   POST /api/v1/projects/:id/tickets   ┌──────────────┐
│  Web UI  │ ────────────────────────────────────→ │ internal/httpapi │
│ (Svelte) │    createTicketRequest                │  (Interface)     │
└──────────┘                          └──────┬───────┘
                                             │
                  ┌──────────────────────────┘
                  ▼
         ┌────────────────────────┐
         │ ticket/service.go      │
         │ or service/* package   │  (Service / Use-Case Layer)
         └──────────┬─────────────┘
                  │
                  ▼
```

```go
// internal/httpapi/ticket_api.go — Interface / Entry layer
func (h *TicketHandler) Create(c echo.Context) error {
    ctx := c.Request().Context()
    ctx, span := h.tracer.StartSpan(ctx, "handler.ticket.create")  // Provider: Trace
    defer span.End()

    var req createTicketRequest
    if err := c.Bind(&req); err != nil {
        return echo.NewHTTPError(400, err.Error())
    }

    // Call the Service / Use-Case layer
    result, err := h.ticketService.Create(ctx, req.toCreateInput())
    if err != nil {
        return mapDomainError(err)  // domain error → HTTP status
    }

    return c.JSON(201, result)
}
```

```go
// internal/ticket/service.go — Service / Use-Case Layer
func (s *Service) Create(ctx context.Context, input CreateInput) (Ticket, error) {
    ctx, span := s.trace.StartSpan(ctx, "ticket.create")
    defer span.End()

    params, err := parseCreateInput(input)
    if err != nil {
        return Ticket{}, err
    }

    created, err := s.repo.Create(ctx, params)
    if err != nil {
        return Ticket{}, fmt.Errorf("create ticket: %w", err)
    }

    return created, nil
}
```

```go
// internal/domain/ticketing/retry.go — Domain / Core Types layer (pure business rules)
func NextRetryAt(attempt int, baseDelay time.Duration, now time.Time) time.Time {
    if attempt < 1 {
        attempt = 1
    }
    delay := baseDelay * time.Duration(1<<(attempt-1))
    return now.Add(delay)
}
```

**Notification method:** `EventProvider.Publish("ticket.events", ...)` → orchestrator receives via `EventProvider.Subscribe("ticket.events")` → next tick discovers new ticket. In-process uses Go channel (zero latency), while separately deployed uses PG LISTEN/NOTIFY.

---

### 16.2 Path 2: Orchestrator schedules Tick (system internal core loop)

This is the scheduler loop the orchestrator runs every N seconds, showing how the orchestrator interacts with service/use-case, domain rules, and provider boundaries.

```go
// internal/orchestrator/scheduler.go
func (s *Scheduler) runTick(ctx context.Context) {
    ctx, span := s.tracer.StartSpan(ctx, "orchestrator.tick")
    defer span.End()
    tickStart := time.Now()

    // ── Step 1: Reconciliation ──
    // Check if all running tickets' Agents are still alive
    for ticketID, worker := range s.pool.ActiveWorkers() {
        if worker.IsStalled(s.stallTimeout) {
            s.logger.Warn("agent stalled", "ticket", ticketID, "last_event", worker.LastEventAt)
            worker.Kill()
            s.handleRetry(ctx, ticketID, RetryReasonStall)
        }
    }

    // ── Step 2: Sync cache for latest published Workflow / Skill versions ──
    published := s.workflowCatalog.SyncPublishedVersions(ctx)
    s.metrics.Counter("openase.orchestrator.harness_publish_total").Add(len(published.Harnesses))

    // ── Step 3: Fetch candidate tickets ──
    candidates, err := s.ticketRepo.ListByStatus(ctx, ticket.StatusTodo)
    if err != nil {
        s.logger.Error("list candidates", "err", err)
        return
    }

    // ── Step 4: Filter + sort ──
    var eligible []*ticket.Ticket
    for _, t := range candidates {
        // Check dependency: skip if blocked
        if blocked, _ := s.ticketSvc.IsBlocked(ctx, t.ID); blocked {
            s.metrics.Counter("openase.orchestrator.tickets_skipped_total",
                provider.Tags{"reason": "blocked"}).Inc()
            continue
        }
        eligible = append(eligible, t)
    }
    // Sort by priority + creation time
    ticket.SortByPriorityAndAge(eligible)

    // ── Step 5: Concurrent check + dispatch ──
    for _, t := range eligible {
        if s.pool.ActiveCount() >= s.maxConcurrent {
            s.metrics.Counter("openase.orchestrator.tickets_skipped_total",
                provider.Tags{"reason": "max_concurrency"}).Inc()
            break
        }

        // Dispatch!
        if err := s.dispatch(ctx, t); err != nil {
            s.logger.Error("dispatch failed", "ticket", t.Identifier, "err", err)
        } else {
            s.metrics.Counter("openase.orchestrator.tickets_dispatched_total",
                provider.Tags{"workflow_type": t.WorkflowType()}).Inc()
        }
    }

    // Record tick duration
    s.metrics.Histogram("openase.orchestrator.tick_duration_seconds").
        Observe(time.Since(tickStart).Seconds())
}
```

**Notification method:** Scheduler is a pure internal loop and does not notify external systems directly. Status changes are sent via database updates + EventProvider broadcast, and the SSE endpoint listens to EventProvider and pushes to the frontend.

---

### 16.3 Path 3: Ticket dispatch → Hook → Agent execution (core mainline)

This is OpenASE’s core path: the full flow from dispatching a ticket to the Agent actually starting work.

```
Scheduler.dispatch()
    │
    ▼
┌────────────────────┐  on_claim Hook   ┌──────────────┐   Agent CLI   ┌───────────┐
│ ticket.Service     │ ───────────────→ │ HookExecutor │ ────────────→ │  Adapter  │
│ Claim / Assign     │ (workspace prep, etc.)     │  (Shell)     │   (Start)      │(ClaudeCode)│
└────────────────────┘                  └──────────────┘              └───────────┘
```

```go
// internal/orchestrator/scheduler.go
func (s *Scheduler) dispatch(ctx context.Context, t *ticket.Ticket) error {
    ctx, span := s.tracer.StartSpan(ctx, "orchestrator.dispatch",
        trace.WithAttributes("ticket.id", t.Identifier))
    defer span.End()

    // 1. Select agent (capability matching + load balancing)
    agent, err := s.agentSvc.SelectForTicket(ctx, t)
    if err != nil {
        return fmt.Errorf("select agent: %w", err)
    }

    // 2. Execute Claim through Service / Use-Case layer (includes on_claim Hook)
    err = s.ticketService.Claim(ctx, t.ID, agent.ID)
    if err != nil {
        return fmt.Errorf("claim ticket: %w", err)
    }

    // 3. Start worker goroutine
    s.pool.Start(t.ID, func(workerCtx context.Context) {
        s.runWorker(workerCtx, t, agent)
    })

    return nil
}
```

```go
// internal/ticket/service.go — Claim / Status transition includes Hook execution
func (s *Service) Claim(ctx context.Context, ticketID uuid.UUID, run *runtime.AgentRun, ag *agent.Agent) error {
    ctx, span := s.trace.StartSpan(ctx, "ticket.claim")
    defer span.End()

    t, err := s.repo.Get(ctx, ticketID)
    if err != nil {
        return err
    }

    // 1. Domain layer status transition (pure rule validation)
    if err := t.TransitionTo(ticket.StatusInProgress); err != nil {
        return err  // e.g., ErrAlreadyClaimed, ErrBlockedByDependency
    }
    t.CurrentRunID = run.ID

    // 2. Prepare hook environment variables
    repos, _ := s.repoScopeRepo.ListByTicket(ctx, t.ID)
    hookEnv := hook.Env{
        TicketID:         t.ID,
        TicketIdentifier: t.Identifier,
        WorkflowType:     t.WorkflowType(),
        AgentName:        ag.Name,
        Repos:            repos,
    }

    // 3. Execute on_claim Hook (blocking: fails means not claimed)
    workflow, _ := s.workflowRepo.Get(ctx, t.WorkflowID)
    results, err := s.hookExec.RunAll(ctx, workflow.Hooks.OnClaim, hookEnv)
    if err != nil {
        // Hook failed → log it → ticket remains unclaimed in todo
        s.eventBus.Publish(ctx, "ticket.events", ticket.HookFailedEvent{
            TicketID: t.ID,
            Hook:     "on_claim",
            Error:    err.Error(),
            Results:  results,
        })
        s.metrics.Counter("openase.hook.block_total",
            provider.Tags{"hook_name": "on_claim"}).Inc()
        return fmt.Errorf("on_claim hook failed: %w", err)
    }

    // 4. Persist status change
    if err := s.repo.Save(ctx, t); err != nil {
        return err
    }

    // 5. Broadcast event
    s.eventBus.Publish(ctx, "ticket.events", ticket.ClaimedEvent{
        TicketID: t.ID,
        AgentID:  agentID,
    })

    return nil
}
```

```go
// internal/orchestrator/runtime_runner.go — Agent execution by Worker (conceptual sketch)
func (s *Scheduler) runWorker(ctx context.Context, t *ticket.Ticket, agent *agent.Agent) {
    ctx, span := s.tracer.StartSpan(ctx, "worker.run",
        trace.WithAttributes("ticket.id", t.Identifier, "agent.name", agent.Name))
    defer span.End()

    // 1. Set up compound workspace (multi-repo clone + checkout feature branch)
    workspace, err := s.workspaceMgr.Setup(ctx, t)
    if err != nil {
        s.handleFailure(ctx, t, err)
        return
    }
    defer s.workspaceMgr.Cleanup(ctx, workspace)

    // 2. Render Harness Prompt
    harness, _ := s.harnessLoader.Load(ctx, t.WorkflowID)
    prompt, _ := harness.Render(harness.TemplateData{
        Ticket:  t,
        Project: workspace.Project,
        Agent:   agent,
        Repos:   workspace.Repos,
        Attempt: t.AttemptCount,
    })

    // 3. Start agent through adapter
    adapter := s.adapterRegistry.Get(agent.Provider.AdapterType)
    session, err := adapter.Start(ctx, agent.Config())
    if err != nil {
        s.handleFailure(ctx, t, err)
        return
    }

    // 4. Send prompt and stream events
    adapter.SendPrompt(ctx, session, prompt)
    events, _ := adapter.StreamEvents(ctx, session)

    for event := range events {
        // Update heartbeat
        agent.LastHeartbeatAt = time.Now()
        s.agentRepo.Save(ctx, agent)

        // Record token usage
        s.metrics.Counter("openase.agent.tokens_used_total", provider.Tags{
            "provider": agent.Provider.Name,
            "direction": "output",
        }).Add(event.TokensOut)

        // Broadcast event to frontend SSE
        s.eventBus.Publish(ctx, "agent.events", AgentProgressEvent{
            TicketID:  t.ID,
            AgentName: agent.Name,
            Event:     event,
        })

        // Check whether it has finished (result-type event)
        if event.Type == "result" {
            break
        }
    }

    // 5. After normal turn completion, runtime rereads ticket status
    //    If ticket is still active: continue next continuation turn
    //    If ticket has left active: stop current worker, but do not auto-change business status
}
```

**Revision:** OpenASE should be implemented using Symphony’s AgentRunner pattern, instead of remaining this simplified “single Turn worker” version above.

A pseudocode version closer to the target implementation should be:

```go
// runtime/agent_runner.go
func (r *AgentRunner) Run(ctx context.Context, ticket *ticket.Ticket, agent *agent.Agent) error {
    workspace, err := r.workspaceMgr.Setup(ctx, ticket)
    if err != nil {
        return err
    }
    defer r.workspaceMgr.AfterRun(ctx, workspace, ticket)

    session, err := r.adapter.Start(ctx, agent.Config())
    if err != nil {
        return err
    }
    defer r.adapter.Stop(ctx, session)

    for turnNumber := 1; turnNumber <= cfg.Agent.MaxTurns; turnNumber++ {
        prompt := r.buildTurnPrompt(ticket, turnNumber)

        turn, err := r.adapter.StartTurn(ctx, session, StartTurnInput{
            Prompt:          prompt,
            Workspace:       workspace.Path,
            ApprovalPolicy:  cfg.Codex.ApprovalPolicy,
            SandboxPolicy:   cfg.Codex.TurnSandboxPolicy,
            DisplayTitle:    fmt.Sprintf("%s: %s", ticket.Identifier, ticket.Title),
        })
        if err != nil {
            return err
        }

        for event := range turn.Events {
            r.runtimeRepo.UpdateHeartbeat(ctx, agent.ID, event.Timestamp)
            r.runtimeRepo.IntegrateCodexEvent(ctx, agent.ID, session.ThreadID, turn.ID, event)
            r.eventBus.Publish(ctx, "agent.events", event)
        }

        refreshed, err := r.ticketRepo.RefreshState(ctx, ticket.ID)
        if err != nil {
            return err
        }
        ticket = refreshed

        if !ticket.IsActiveState() {
            return nil
        }
    }

    // max_turns reached: hand control back to the orchestrator; continuation retry will later decide whether to continue
    return nil
}
```

This revised version has four behaviors that must be preserved:

1. `Start()` and `thread/start` are done only once, and the same thread is reused across the whole worker lifecycle.
2. `StartTurn()` is called multiple times on the same thread, and the latest ticket status is reloaded after each turn.
3. Continuation turns only send continuation guidance and do not re-send the full original prompt.
4. After `max_turns` is reached, do not automatically mark the ticket as complete; hand control back to the orchestrator.

**Notification method:**
- `on_claim` hook failure → `EventProvider.Publish("ticket.events", HookFailedEvent)` → SSE → frontend shows “claim failed”.
- Agent running → `EventProvider.Publish(agent progress event)` → SSE → frontend live stream.
- Agent or human explicitly advances status → state transition path in `internal/ticket` executes `on_complete` Hook as needed → on success, `EventProvider.Publish(ticket status changed event)` → SSE → frontend status update.

---

### 16.4 Path 4: on_complete Hook execution + explicit status progression

This path shows how Hook acts as a quality gate to block or allow explicit status progression. A normal turn ending does not directly enter this path.

```go
// internal/ticket/service.go — simplified pseudocode for explicit status progression path
func (s *Service) AdvanceAfterExplicitAction(ctx context.Context, ticketID uuid.UUID, requestedStatusID uuid.UUID) error {
    ctx, span := s.trace.StartSpan(ctx, "ticket.advance_after_explicit_action")
    defer span.End()

    t, _ := s.loadTicket(ctx, ticketID)
    workflow, _ := s.loadWorkflow(ctx, t.WorkflowID)

    // 1. Execute on_complete Hook (blocking)
    hookEnv := hook.Env{
        TicketID:         t.ID,
        TicketIdentifier: t.Identifier,
        Workspace:        t.WorkspacePath(),
        Repos:            t.RepoScopes(),
    }
    results, err := s.runOnCompleteHooks(ctx, workflow, hookEnv)

    // Record each Hook result
    for _, r := range results {
        s.metrics.Histogram("openase.hook.duration_seconds",
            provider.Tags{"hook_name": r.Name}).Observe(r.Duration.Seconds())
        s.metrics.Counter("openase.hook.execution_total",
            provider.Tags{"hook_name": r.Name, "outcome": r.Outcome}).Inc()
    }

    if err != nil {
        // ── Hook failed: ticket stays in in_progress ──
        //    Return failure info as "feedback" to Agent so it can fix and retry
        s.publishHookFailed(ctx, t.ID, "on_complete", err, results)
        return fmt.Errorf("on_complete hook failed: %w", err)
    }

    // ── All hooks passed ──

    // 2. Check whether PRs for all repos have been submitted
    scopes, _ := s.listRepoScopes(ctx, t.ID)
    allPRsOpen := true
    for _, scope := range scopes {
        if scope.PRStatus == "none" {
            allPRsOpen = false
            break
        }
    }
    if !allPRsOpen {
        return ErrPRNotSubmitted  // Some repos have not submitted PRs yet
    }

    // 3. Explicit status transition (for example, to in_review)
    if err := t.TransitionTo(requestedStatusID); err != nil {
        return err
    }
    s.saveTicket(ctx, t)

    // 4. Notify reviewer/frontend that ticket has explicitly entered review state
    s.notificationEngine.NotifyTicketReadyForReview(ctx, t, len(results))

    // 5. Broadcast
    s.publishMovedToReview(ctx, t.ID, results)

    s.metrics.Histogram("openase.ticket.agent_time_seconds",
        provider.Tags{"workflow_type": t.WorkflowType()}).
        Observe(time.Since(t.StartedAt).Seconds())

    return nil
}
```

**Notification method:**
- Hook fails → `EventProvider` → SSE → frontend marks in red + Agent receives feedback and retries.
- Hook passes → `internal/notification` / `NotificationEngine` → Slack/Email/Webhook → reviewer receives notification.
- Human moves ticket to finish → triggers `on_done` → ticket is completed.

---

### 16.5 Path 5: PR link binding (internal explicit write)

This path shows how PR links in RepoScope are recorded. It is a normal internal write path and does not depend on any GitHub/GitLab webhook.

```
Agent / Human
    │
    ▼
POST /api/v1/projects/:projectId/tickets/:ticketId/repo-scopes/:scopeId
    │
    ▼
httpapi / catalog service
    │
    ▼
Update TicketRepoScope.pull_request_url
    │
    ▼
Write ActivityEvent(pr.linked)
    │
    ▼
SSE push to frontend
```

```go
// internal/httpapi/catalog.go — Interface / Entry layer (conceptual sketch)
func (h *RepoScopeHandler) UpdateRepoScope(c echo.Context) error {
    input := parseRepoScopePatch(c)
    scope, err := h.catalog.UpdateTicketRepoScope(c.Request().Context(), input)
    if err != nil {
        return writeError(c, err)
    }
    return c.JSON(200, scope)
}
```

```go
// internal/catalog/service.go — Service / Use-Case Layer（conceptual sketch）
func (s *Service) UpdateTicketRepoScope(ctx context.Context, input UpdateRepoScopeInput) error {
    scope := s.repoScopeRepo.Get(ctx, input.ScopeID)
    scope.PullRequestURL = input.PullRequestURL
    s.repoScopeRepo.Save(ctx, scope)

    s.activityRepo.Append(ctx, ActivityEvent{
        Type: "pr.linked",
        TicketID: scope.TicketID,
        Metadata: map[string]any{
            "repo_id": scope.RepoID,
            "pull_request_url": scope.PullRequestURL,
        },
    })
    return nil
}
```

**Clear constraints:**
- Binding a PR link does not automatically advance ticket status.
- PR merged / closed / review / CI result are not automatically written back to OpenASE.
- If ticket completion is needed, ticket status still must be explicitly changed by a human or Agent.

---

### 16.6 Path 6: SSE real-time push (system → frontend)

This path shows how events flow from backend to frontend and supports multiple browsers being online simultaneously.

For passive state synchronization at project scope, OpenASE must enforce a single semantic model:

- Each mounted project shell may have at most one passive SSE connection.
- The connection must be `GET /api/v1/projects/:projectId/events/stream`.
- page / drawer / sidebar cannot each open a project-scoped stream; they can only subscribe to the project event bus in-memory state.
- project-scoped filtering must be done on the server side and converged by `project_id` before the event reaches the browser.
- Request-owned interactive streams such as `GET /api/v1/chat/conversations/:conversationId/stream` are not part of the project passive bus and cannot carry shared project state.

Backend shutdown semantics must also be fixed clearly: when `openase serve` / `openase all-in-one` enters shutdown, OpenASE must prioritize **bounded shutdown**, actively terminating project passive SSE, request-owned interactive streams, and reverse websocket machine channels first; preserving the continuity of existing connections is lower priority than exiting the process within the configured timeout, and clients should automatically reconnect after the service recovers.

```
Domain Event  ──→  EventProvider  ──→  SSE Hub (fan-out)  ──→  Browser A
                   (Go channel)       (one channel per connection)   ──→  Browser B
                                                          ──→  Browser C
```

**Key: fan-out broadcast.** A single event is published once and all SSE connections receive it. The SSE Hub manages connection registration/unregistration and event broadcast:

```go
// infra/sse/hub.go — SSE connection manager
type Hub struct {
    mu          sync.RWMutex
    connections map[string]map[chan Event]struct{}  // projectID → set of channels
}

func (h *Hub) Register(projectID string) chan Event {
    ch := make(chan Event, 64)  // buffered to avoid slow consumers blocking broadcast
    h.mu.Lock()
    if h.connections[projectID] == nil {
        h.connections[projectID] = make(map[chan Event]struct{})
    }
    h.connections[projectID][ch] = struct{}{}
    h.mu.Unlock()
    return ch
}

func (h *Hub) Unregister(projectID string, ch chan Event) {
    h.mu.Lock()
    delete(h.connections[projectID], ch)
    close(ch)
    h.mu.Unlock()
}

// Broadcast: all connections subscribed to the project receive it
func (h *Hub) Broadcast(event Event) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    for ch := range h.connections[event.ProjectID] {
        select {
        case ch <- event:
        default:
            // Buffer full → skip the slow consumer (do not block other connections)
        }
    }
}
```

**SSE Hub listens to EventProvider and broadcasts to project passive bus connections:**

```go
// At startup: Hub subscribes to EventProvider and fan-outs to all SSE connections
func (h *Hub) Run(ctx context.Context, eventBus provider.EventProvider) {
    events, _ := eventBus.Subscribe(ctx, "ticket.events", "agent.events",
        "hook.events", "activity.events", "agent.trace.events", "agent.step.events")
    for event := range events {
        h.Broadcast(event)
    }
}
```

**SSE endpoint: register the project bus endpoint once, filter by `project_id` on server side, then write to browser:**

```go
// internal/httpapi/sse.go
func (h *SSEHandler) ProjectEventStream(c echo.Context) error {
    projectID := c.Param("projectId")
    ctx := c.Request().Context()

    c.Response().Header().Set("Content-Type", "text/event-stream")
    c.Response().Header().Set("Cache-Control", "no-cache")
    c.Response().Header().Set("Connection", "keep-alive")

    // Subscribe to multiple topics, then converge by project_id into one canonical project bus
    ch := h.hub.Register(projectID)
    defer h.hub.Unregister(projectID, ch)

    pingTicker := time.NewTicker(15 * time.Second)
    defer pingTicker.Stop()

    for {
        select {
        case <-ctx.Done():
            return nil

        case event := <-ch:
            if event.ProjectID != projectID {
                continue
            }
            data, _ := json.Marshal(event)
            fmt.Fprintf(c.Response(), "event: %s\ndata: %s\n\n", event.Type, data)
            c.Response().Flush()

        case <-pingTicker.C:
            fmt.Fprintf(c.Response(), ": ping\n\n")
            c.Response().Flush()
        }
    }
}
```

**Frontend project event runtime:**

```typescript
// web/src/lib/features/project-events/project-event-bus.ts
export function retainProjectEventBus(projectId: string) {
  return retainBus(projectId, () =>
    connectEventStream(`/api/v1/projects/${projectId}/events/stream`, {
      onMessage(frame) {
        const event = parseProjectEventFrame(frame)
        projectEventStore.publish(projectId, event)
      },
    }),
  )
}
```

The Project bus must cover passive updates for `ticket`, `agent`, `hook`, `activity`, and ticket run lifecycle / trace / step; pages and drawers should only consume shared in-memory state, not recreate transport.

```svelte
<!-- web/src/routes/(app)/tickets/+page.svelte -->
<script lang="ts">
  import { createTicketStream } from '$lib/api/sse'
  import TicketCard from '$lib/components/ticket/TicketCard.svelte'

  const { tickets } = createTicketStream(projectId)
</script>

<!-- Ticket board: UI updates automatically when tickets store changes -->
{#each [...$tickets.values()] as ticket (ticket.id)}
  <TicketCard {ticket} />
{/each}
```

**Notification summary table:**

| Event source | Propagation path | Final recipient |
|--------|---------|----------|
| User creates ticket | Handler → Cmd → EventProvider → SSE Hub → all browsers | Frontend board adds card in real time |
| User drags ticket | Handler → Cmd → EventProvider → SSE Hub → all browsers | Other users’ boards update in real time |
| Orchestrator dispatches | Scheduler → ClaimCmd → EventProvider → SSE Hub | Frontend sees Agent claimed |
| Agent execution progress | Worker → EventProvider → SSE Hub | Frontend real-time output stream |
| Hook fails | HookExec → Cmd → EventProvider → SSE Hub | Frontend highlights red + Agent receives feedback |
| Hook passes + requires approval | Cmd → EventProvider → SSE Hub + NotificationEngine → Telegram/Enterprise WeChat | Frontend + external notification |
| GitHub PR merged | Webhook → Cmd → EventProvider → SSE Hub | Frontend status updates in real time |
| Agent error retry | Worker → EventProvider → SSE Hub + NotificationEngine | Frontend shows retry status + alert |
| Configuration change | terminal setup / manual config edit → restart hosted service | Service restarts, frontend SSE reconnects automatically |


## Chapter 17 System Boundaries and Information Consistency

This chapter answers a fundamental question: **what OpenASE is responsible for, what it is not responsible for, where information is stored, where inconsistencies may occur, and how to fix them when they occur.**

### 17.1 System Boundary Overview

```
                        OpenASE Responsibility Boundary
┌──────────────────────────────────────────────────────────┐
│                                                          │
│   Ticket lifecycle management    Workflow / Harness / Hook Orchestration
│   Agent scheduling and monitoring    Approval governance    Cost tracking    Activity audit
│                                                          │
└────────┬──────────────┬──────────────┬───────────────────┘
         │              │              │
    ┌────▼────┐   ┌─────▼─────┐  ┌────▼────┐
    │ Git platform │   │ Agent CLI │  │  OIDC   │
    │ GitHub   │   │ Claude    │  │ Provider│
    │ GitLab   │   │ Codex     │  │         │
    │          │   │ Gemini    │  │         │
    └─────────┘   └───────────┘  └─────────┘
     External system ①        External system ②      External system ③

    ┌─────────┐   ┌───────────┐  ┌─────────┐
    │PostgreSQL│   │ Notification channels │  │  OTel   │
    │ (self-managed) │   │ Slack     │  │ Collector│
    │          │   │ Email     │  │ (optional) │
    └─────────┘   └───────────┘  └─────────┘
     Infrastructure ①       External system ④      Infrastructure ②
```

**OpenASE responsibilities (inside system):**

- Ticket full lifecycle (create → assign → execute → approve → complete)
- Definition and execution of Workflow, Harness, and Hook
- Agent scheduling, process management, heartbeat monitoring
- Ticket and Git branch/PR association
- Cost tracking and budget control
- Activity audit logs
- Frontend UI and SSE push

**OpenASE does not handle (external system management):**

- Git repositories themselves (code contents, branch permissions, CI/CD workflows)
- Installation, upgrade, and API Key management for Agent CLI (user-managed, OpenASE only invokes)
- PostgreSQL operations (backup, upgrade, high availability)
- Authentication identity source (management of OIDC Provider)
- Notification channel configuration (Slack workspace, Email server)
- Observability backend (operations of Jaeger, Prometheus)

### 17.2 External System Integration List

| External system | Integration mode | OpenASE-side interface | Data direction | Integration phase |
|---------|---------|---------------|---------|---------|
| **GitHub / GitLab** | Git + optional REST API | ProjectRepo / runtime git&gh integration | Bidirectional (code and PR links), no state synchronization | Phase 2 |
| **Claude Code** | CLI subprocess (NDJSON stream) | `internal/infra/adapter/claudecode/` | Bidirectional | Phase 1 |
| **OpenAI Codex** | JSON-RPC over stdio | `internal/infra/adapter/codex/` | Bidirectional | Phase 1 |
| **Gemini CLI** | CLI subprocess (stdio stream) | `internal/orchestrator/agent_adapter_gemini.go` + `internal/chat/runtime_gemini.go` | Bidirectional | Phase 1 |
| **PostgreSQL** | SQL (ent ORM) + LISTEN/NOTIFY | `internal/repo/` + `internal/infra/event/pgnotify.go` | Bidirectional | Phase 1 |
| **OIDC Provider** | OIDC Discovery + JWT verification | Unified OIDC adapter has not yet been implemented in this repository; see `internal/httpapi/security_settings_api.go` for security boundary notes | Read | Phase 4 |
| **Slack / Telegram / Webhook / WeCom** | Webhook / Bot API | `internal/notification/` | Write | Phase 2 |
| **OTel Collector** | OTLP gRPC/HTTP | `internal/infra/otel/` | Write | Phase 2 |

### 17.3 Information Ownership and Single Source of Truth

This is the most important section. Each critical piece of information has one authoritative source, while other locations are caches or projections. In case of inconsistency, the source of truth prevails.

| Information | Single source of truth | Storage location | Who writes | Who reads |
|------|---------|---------|------|------|
| **Ticket status** | OpenASE DB | PostgreSQL `tickets.status` | serve (API) / orchestrate (state machine) | serve (API/SSE), orchestrate (scheduling) |
| **Ticket description/metadata** | OpenASE DB | PostgreSQL `tickets.*` | serve (API) | all |
| **Workflow definition** | OpenASE DB | PostgreSQL `workflows.*` | serve (API) | orchestrate (scheduling) |
| **Harness content** | OpenASE DB | PostgreSQL `workflow_versions.*` (or equivalent version table) | serve (API) / refine-harness Agent | orchestrate (runtime materializer; reads published version directly at runtime creation) |
| **Skill bundle content** | OpenASE DB | PostgreSQL `skills.*` + `skill_versions.*` + `skill_version_files.*` (or equivalent bundle version table) | serve (API) / Agent Platform API | orchestrate (runtime materializer; reads published version directly at runtime creation) |
| **Hook scripts** | Git repository | Project repo `scripts/ci/*` (referenced in Harness) | Humans (git push) | orchestrate (HookExecutor) |
| **Agent registration info** | OpenASE DB | PostgreSQL `agents.*` + `agent_providers.*` | serve (API) | orchestrate (scheduling) |
| **Agent runtime lifecycle** | OpenASE DB | PostgreSQL `agent_runs.*` + `tickets.current_run_id` | orchestrate (runtime runner / scheduler) | serve (API/SSE), orchestrate |
| **Agent current action phase** | OpenASE DB | PostgreSQL `agent_runs.current_step_*` | orchestrate (step projector) | serve (Agent console / API) |
| **Agent CLI fine-grained output** | OpenASE DB | PostgreSQL `agent_trace_events.*` | orchestrate (adapter event normalizer) | serve (Agent output API / trace stream) |
| **Project/ticket business activity stream** | OpenASE DB | PostgreSQL `activity_events.*` | serve / orchestrate / hook executor | serve (dashboard / ticket activity / project activity) |
| **Agent process handle** | Orchestrator memory | runtime registry (active session / worker map) | orchestrate (runtime runner) | orchestrate (health checker / shutdown) |
| **Agent session ID** | Agent CLI | Agent CLI internal management | Agent CLI | orchestrate (Adapter read) |
| **Git branch existence** | Git platform | GitHub / GitLab repository | Agent (git push) | orchestrate / humans |
| **PR link** | OpenASE DB | PostgreSQL `ticket_repo_scopes.pull_request_url` | serve (API) / orchestrate (platform operation) | serve (API/SSE), humans |
| **Token consumption** | Agent CLI return value | Agent event stream `cost_usd` | Agent CLI → orchestrate | orchestrate → DB |
| **Accumulated cost** | OpenASE DB | PostgreSQL `tickets.cost_amount` | orchestrate (calculation) | serve (dashboard) |
| **Hook execution result** | Orchestrator runtime | script exit code + stderr | HookExecutor | orchestrate → DB (`ActivityEvent` / Hook record) |
| **Human review result** | OpenASE DB | PostgreSQL `tickets.status_id` | serve (Reviewer action) | serve (API/SSE), orchestrate |
| **User identity** | OIDC Provider | External IdP | OIDC Provider | serve (JWT verification) |
| **User identity cache** | OpenASE DB | PostgreSQL `users.*` | serve (first-login sync) | serve (API) |
| **Global config** | filesystem | `~/.openase/config.yaml` | `openase setup` / manual edit | serve + orchestrate (read on startup) |
| **Sensitive config** | filesystem | `~/.openase/.env` | `openase setup` / manual edit | serve + orchestrate (read on startup) |

### 17.4 Consistency Analysis: Where Inconsistencies Occur and How to Fix Them

**Strong consistency areas (single-writer, no inconsistency):**

| Information | Why it does not become inconsistent |
|------|----------------|
| Ticket status | All state changes go through the persistence path and rules of `internal/ticket` / `internal/ticketstatus`, written in the same PostgreSQL transaction |
| Workflow definition | Only serve process API can write |
| Cumulative cost | Only orchestrate process worker accumulates in Agent events |

**Eventually consistent areas (delayed but self-converging):**

| Information | Inconsistency window | Auto-repair mechanism | Worst case |
|------|-----------|------------|---------|
| Published Harness / Skill version | Between new version publication and next runtime creation (usually < 5s) | Read current published version directly from DB and materialize at next dispatch/runtime creation | Single read failure: retry on next tick, no extra sync needed |
| Agent heartbeat | During heartbeat interval (configurable, default 30s) | Worker reports continuously | Agent crashes without reporting: HealthChecker has 5-minute Stall timeout fallback |
| User identity cache | Between rename/disable changes on OIDC side and synchronization to OpenASE | Refresh user info on every JWT verification | Active old JWT for disabled user until expiry: token expiration as fallback (recommended ≤ 1h) |
| Frontend UI state | Between backend state change and SSE push (< 1s) | SSE event stream continuous push | SSE disconnect/reconnect: fetch full latest state after reconnect |

**Potential inconsistency risks (requiring proactive defense):**

**Risk 1: Git branch exists but DB does not know**

- Scenario: Agent creates a Git branch, but OpenASE crashes before writing TicketRepoScope
- Source-of-truth conflict: Git platform says branch exists, DB says branch_name is empty
- Defense: On worker startup, first check whether remote branch already exists (`git ls-remote`); if it exists, recover the TicketRepoScope record
- Compensation: `openase reconcile` CLI command, scanning all running tickets and comparing Git remote with DB state

**Risk 2: Agent completed but OpenASE did not receive completion event**

- Scenario: Agent CLI subprocess is OS-killed (OOM, etc.), and the final result event is lost
- Source-of-truth conflict: Agent actually completed work (PR has been submitted), but OpenASE still thinks the ticket is in in_progress
- Defense: Stall detection (5 minutes of no events → kill + retry). On retry, Agent will detect PR already exists and report completion directly
- Compensation: In on_claim Hook, check whether PR already exists; if so, skip coding and enter review flow directly

**Risk 3: Missing or stale PR link**

- Scenario: Agent created PR, but did not write the link back to RepoScope; or PR link was manually changed later
- Design decision: `pull_request_url` is reference information only and is not a state machine input, so missing or stale values do not block ticket progression
- Compensation: Humans can manually complete or correct PR link in Ticket Detail; if automation is needed later, it must be established as a separate capability

**Risk 4: Double-write contention—serve and orchestrate modify the same ticket simultaneously**

- Scenario: User cancels ticket in UI while orchestrator is marking ticket as complete
- Source-of-truth conflict: both processes write different values to the same `ticket.status`
- Defense: PostgreSQL optimistic locking. Add `version` field to ticket table, use `WHERE version = ?` on every update, retry on conflict
- Implementation: `field.Int("version").Default(0)` in ent schema + update with `UpdateOne(t).Where(ticket.Version(currentVersion))`

**Risk 5: Harness version mismatch**

- Scenario: Ticket claimed with Harness v3; while it is executing, Harness is updated to v4
- Design decision: **In-flight tickets are not affected**—Worker caches a Harness snapshot when claiming, and ticket records the `harness_version` field. New tickets use v4, old tickets continue using v3
- No repair needed: this is expected behavior, not an inconsistency

### 17.5 Reconciliation Strategy

For the inconsistency risks above, OpenASE adopts a two-layer repair mechanism:

**Automatic repair (background Reconciler):**

```go
// internal/orchestrator/health_checker.go / retry_service.go — periodic checks (conceptual diagram)
func (r *Reconciler) Run(ctx context.Context) {
    // 1. Orphan runtime cleanup
    //    Check whether tasks in active runtime registry still exist (not deleted/cancelled)
    r.cleanOrphanWorkers(ctx)

    // 2. Stuck ticket detection
    //    in_progress longer than 1 hour without heartbeat → pause retry (retry_paused=true), notify humans
    r.detectStuckTickets(ctx)

    // 3. Git branch reconciliation
    //    TicketRepoScope of running tickets → check whether remote branch exists
    r.reconcileGitBranches(ctx)
}
```

**Manual repair (CLI command):**

```bash
openase reconcile              # reconcile everything
openase reconcile --branches   # reconcile only Git branches
openase reconcile --stuck      # handle only stuck tickets
openase reconcile --dry-run    # report inconsistencies only, without repair
```

### 17.6 Information Flow Overview Diagram

```
                 Source of truth: OpenASE DB + Git repositories
                   ┌─────────────┐
                   │ Harness / Skill versions │──publish───→ Next runtime reads and materializes directly
                   │ Skill versions   │
                   └─────────────┘
                           │
                           │
                    ┌─────────────┐
                    │ Hook scripts   │
                    │ Code branch/PR  │──Webhook───→ serve → DB (cache)
                    └─────────────┘               ↑
                                          Reconciler compensating polling

                    Source of truth: Agent CLI
                   ┌─────────────┐
                   │ Session state  │
                   │ Token consumption   │──NDJSON/RPC──→ orchestrate Worker
                   │ Execution result │                    │
                   └─────────────┘                    Write to DB
                                                       │
                    Source of truth: OpenASE DB                     ▼
                   ┌─────────────┐              ┌──────────────┐
                   │ Ticket status   │◄─────────────│ PostgreSQL    │
                   │ Approval decisions │              │  (single durable source)  │
                   │ Cost data      │─────────────→│              │
                   │ Activity logs  │              └──────┬───────┘
                   └─────────────┘                      │
                                                   LISTEN/NOTIFY
                    Source of truth: OIDC Provider           │
                   ┌─────────────┐                      ▼
                   │ User identity│──JWT verification──→ serve ──SSE──→ Frontend (projection)
                   └─────────────┘

Legend:
  ──→  Data direction
  Source-of-truth  The authoritative source for this information; take precedence when inconsistent
  (cache)  Non-authoritative internal copy in OpenASE, rebuildable from the source
  (projection)  Read-only view derived from the source
```

### 17.7 Boundary Design Principles

1. **Single source of truth**: Each information item has exactly one writer. Ticket status is written only by the state machine, PR links in RepoScope are written only via explicit platform write paths, and Harness is stored only in Git repositories
2. **References do not drive state machine**: External links and PR links provide context only and do not participate in automatic ticket state advancement
3. **Optimistic locking prevents double writes**: If two processes may write the same ticket at the same time, use a `version` field for optimistic concurrency control
4. **Reconciler fallback**: Background Reconciler is only responsible for local runtime and Git branch consistency, without relying on GitHub/GitLab Webhooks
5. **Idempotent operations**: All state transition operations are designed to be idempotent—repeating the same state transition does not cause side effects
6. **Crash recovery**: After restart, orchestrator scans all in_progress tickets, restores Workers or marks them for retry
## Chapter 18 REST API Design

### 18.1 API Overview

All APIs are prefixed with `/api/v1`, return JSON, and use standard HTTP status codes. Authentication is via the `Authorization: Bearer <token>` header. Pagination uses cursor-based pagination (`?cursor=xxx&limit=20`).

### 18.2 Resource Endpoints

**Organization**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/orgs` | List all organizations |
| POST | `/api/v1/orgs` | Create organization |
| GET | `/api/v1/orgs/:orgId` | Get organization details |
| PATCH | `/api/v1/orgs/:orgId` | Update organization |
| DELETE | `/api/v1/orgs/:orgId` | Archive organization and automatically archive all underlying projects (soft delete) |

**Project**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/projects` | List projects |
| POST | `/api/v1/orgs/:orgId/projects` | Create project |
| GET | `/api/v1/projects/:projectId` | Get project details |
| PATCH | `/api/v1/projects/:projectId` | Update project |
| DELETE | `/api/v1/projects/:projectId` | Archive project (soft delete) |

**Project Updates**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/projects/:projectId/updates` | List project update threads, sorted by `last_activity_at DESC` |
| POST | `/api/v1/projects/:projectId/updates` | Create a project update thread |
| PATCH | `/api/v1/projects/:projectId/updates/:threadId` | Edit a project update thread (including status changes) |
| DELETE | `/api/v1/projects/:projectId/updates/:threadId` | Soft-delete a project update thread |
| GET | `/api/v1/projects/:projectId/updates/:threadId/revisions` | Read thread revision history |
| POST | `/api/v1/projects/:projectId/updates/:threadId/comments` | Create a project update comment |
| PATCH | `/api/v1/projects/:projectId/updates/:threadId/comments/:commentId` | Edit a project update comment |
| DELETE | `/api/v1/projects/:projectId/updates/:threadId/comments/:commentId` | Soft-delete a project update comment |
| GET | `/api/v1/projects/:projectId/updates/:threadId/comments/:commentId/revisions` | Read comment revision history |

**ProjectRepo**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/projects/:projectId/repos` | List project repositories |
| POST | `/api/v1/projects/:projectId/repos` | Add repository |
| PATCH | `/api/v1/projects/:projectId/repos/:repoId` | Update repository configuration |
| DELETE | `/api/v1/projects/:projectId/repos/:repoId` | Remove repository |

**Ticket (core)**

| Method | Path | Description | Parameters |
|------|------|------|------|
| GET | `/api/v1/projects/:projectId/tickets` | List tickets | `?status_name=Todo,In+Progress&priority=high&workflow_type=coding&cursor=xxx&limit=20` (`status_name` uses custom status names and is not hardcoded enum) |
| POST | `/api/v1/projects/:projectId/tickets` | Create ticket | body: `{title, description, priority, type, workflow_id?, repo_scopes?}` |
| GET | `/api/v1/tickets/:ticketId` | Ticket detail (basic detail) | |
| GET | `/api/v1/projects/:projectId/tickets/:ticketId/detail` | Ticket detail aggregated view (including description entry, timeline, RepoScopes, dependencies, pickup_diagnosis) | |
| GET | `/api/v1/projects/:projectId/tickets/:ticketId/runs` | List run sessions for the ticket (latest first, with attempt, current step summary, and completion summary status) | |
| GET | `/api/v1/projects/:projectId/tickets/:ticketId/runs/:runId` | Get transcript data for one ticket run (`AgentRun + AgentTraceEvent + AgentStepEvent + completion summary`) | |
| PATCH | `/api/v1/tickets/:ticketId` | Update ticket (title, description, priority) | |
| POST | `/api/v1/tickets/:ticketId/transition` | State transition | body: `{to_status_id: "uuid" or to_status_name: "To Test", comment?}` (pass custom status name or ID) |
| POST | `/api/v1/tickets/:ticketId/cancel` | Cancel ticket | body: `{reason?}` |
| POST | `/api/v1/tickets/:ticketId/retry` | Trigger retry manually | |
| GET | `/api/v1/tickets/:ticketId/activity` | Ticket activity logs | `?cursor=xxx&limit=50` |

**Ticket Comments**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/tickets/:ticketId/comments` | List current ticket comments |
| POST | `/api/v1/tickets/:ticketId/comments` | Create comment |
| PATCH | `/api/v1/tickets/:ticketId/comments/:commentId` | Edit comment |
| DELETE | `/api/v1/tickets/:ticketId/comments/:commentId` | Delete comment |
| GET | `/api/v1/tickets/:ticketId/comments/:commentId/revisions` | Read comment revision history |

**Ticket Dependencies**

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/tickets/:ticketId/dependencies` | Add dependency `{target_ticket_id, type: "blocks"\|"sub_issue"}` |
| DELETE | `/api/v1/tickets/:ticketId/dependencies/:depId` | Remove dependency |

**Ticket External Links**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/tickets/:ticketId/links` | List external links (Issues + PRs) |
| POST | `/api/v1/tickets/:ticketId/links` | Add external link `{link_type, url, relation}` |
| DELETE | `/api/v1/tickets/:ticketId/links/:linkId` | Remove external link |

**Workflow**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/projects/:projectId/workflows` | List workflows |
| POST | `/api/v1/projects/:projectId/workflows` | Create workflow |
| GET | `/api/v1/workflows/:workflowId` | Workflow details (including Harness content, Hook configuration) |
| PATCH | `/api/v1/workflows/:workflowId` | Update workflow |
| GET | `/api/v1/workflows/:workflowId/harness` | Get raw Harness content currently in release (Markdown) |
| PUT | `/api/v1/workflows/:workflowId/harness` | Update Harness and generate a new version |
| GET | `/api/v1/workflows/:workflowId/harness/history` | Harness version history (platform release records) |

**AgentProvider & Agent**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/providers` | List Agent Providers (including `availability_state / available / availability_checked_at / availability_reason`) |
| POST | `/api/v1/orgs/:orgId/providers` | Register provider |
| PATCH | `/api/v1/providers/:providerId` | Update provider |
| GET | `/api/v1/providers/:providerId` | Provider details (including runtime availability derived fields) |
| GET | `/api/v1/projects/:projectId/agents` | List agents |
| POST | `/api/v1/projects/:projectId/agents` | Register agent |
| GET | `/api/v1/agents/:agentId` | Agent details (state, current ticket, heartbeat, token consumption) |
| GET | `/api/v1/projects/:projectId/agents/:agentId/output` | Fine-grained agent runtime output; compatible naming, reads underlying `AgentTraceEvent` |
| GET | `/api/v1/projects/:projectId/agents/:agentId/steps` | Human-readable agent action stream; reads `AgentStepEvent` |
| DELETE | `/api/v1/agents/:agentId` | Deregister agent |

**ScheduledJob**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/projects/:projectId/scheduled-jobs` | List scheduled jobs |
| POST | `/api/v1/projects/:projectId/scheduled-jobs` | Create scheduled job |
| PATCH | `/api/v1/scheduled-jobs/:jobId` | Update scheduled job |
| DELETE | `/api/v1/scheduled-jobs/:jobId` | Delete scheduled job |
| POST | `/api/v1/scheduled-jobs/:jobId/trigger` | Trigger one run manually |

**SSE Endpoints**

| Method | Path | Event Types |
|------|------|---------|
| GET | `/api/v1/projects/:projectId/events/stream` | Canonical project passive bus; aggregates `ticket.*`, `agent.*`, `hook.*`, `activity.*`, `ticket.run.lifecycle`, `ticket.run.trace`, `ticket.run.step` |
| GET | `/api/v1/projects/:projectId/agents/:agentId/output/stream` | Fine-grained agent output stream; pushes only `AgentTraceEvent` |
| GET | `/api/v1/projects/:projectId/agents/:agentId/steps/stream` | Agent action phase stream; pushes only `AgentStepEvent` |

**Coding Agent Launch Verification**

- `claimed + runtime_phase=none`: displays “Claimed, waiting for launcher”
- `claimed/running + runtime_phase=launching`: displays “Launching Codex session”
- `running + runtime_phase=ready + session_id`: displays “Ready”
- `failed` or `runtime_phase=failed`: displays failed state and `last_error`
- When the activity panel is empty, display “No business activity yet”; do not treat zero output rows as not started or failed
- When the Agent output panel is empty, display “No trace events yet,” indicating there is currently no fine-grained runtime output and not that runtime failed to start
- The Ticket detail Runs panel defaults to showing the latest run transcript; discussion/comments and execution transcript are rendered separately, with the former based on comment/activity and the latter on ticket-native run data path
- The Ticket detail Runs panel additionally shows a terminal completion summary card above the transcript; must support `pending` / `completed` / `failed`, and the summary is only a high signal-to-noise abstraction, not a replacement for raw transcript evidence
- The runtime card in the Ticket detail sidebar must use the backend-provided `pickup_diagnosis` as the source of truth and explicitly state whether the ticket is `runnable / waiting / blocked / running / completed / unavailable`, instead of letting the frontend reconstruct scheduler judgment on its own
- `pickup_diagnosis` must include stable reason code, primary description, next action hint, and structured workflow / agent / provider / capacity / blocked_by / retry information for UI display; when `next_retry_at` is in the future, the frontend must show a real-time countdown and a deterministic UTC absolute time
- Black-box acceptance must cover: create an idle agent + pickup ticket, observe `claimed`, then observe `running + ready + session_id + heartbeat`, and receive `agent.ready`; then see `AgentTraceEvent` and `AgentStepEvent` separately in the output/step streams

**Webhook Inbound**

The current version does not define GitHub / GitLab inbound Webhook APIs. OpenASE does not synchronize Issue, PR status, or CI status via webhook.

**System Management**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/system/health` | Health check (200 = healthy, 503 = unhealthy) |
| GET | `/api/v1/system/status` | System status (orchestrator status, Worker count, queue depth) |
| POST | `/api/v1/system/reconcile` | Trigger manual reconciliation (equivalent to `openase reconcile`) |
| GET | `/api/v1/system/metrics` | Prometheus-format metrics export |

### 18.3 Unified Response Format

```json
// Success (single resource)
{
  "data": { ... },
  "meta": { "request_id": "req_xxx" }
}

// Success (list, cursor-based pagination)
{
  "data": [ ... ],
  "meta": {
    "request_id": "req_xxx",
    "cursor": "eyJpZCI6...",
    "has_more": true,
    "total": 142
  }
}

// Error
{
  "error": {
    "code": "TICKET_BLOCKED",
    "message": "Ticket ASE-42 is blocked by ASE-30",
    "details": { "blockers": ["ASE-30"] }
  },
  "meta": { "request_id": "req_xxx" }
}
```

### 18.4 Error Code System

| HTTP Status | Error Code | Description |
|-------------|-----------|------|
| 400 | `VALIDATION_ERROR` | Request parameter validation failed |
| 400 | `INVALID_TRANSITION` | Invalid state transition (such as backlog → done) |
| 401 | `UNAUTHORIZED` | Missing or invalid token |
| 403 | `FORBIDDEN` | Insufficient permissions |
| 404 | `NOT_FOUND` | Resource not found |
| 409 | `CONFLICT` | Optimistic lock conflict (version mismatch) |
| 409 | `TICKET_BLOCKED` | Ticket blocked by dependency |
| 409 | `AGENT_BUSY` | Agent is executing another ticket |
| 409 | `APPROVAL_PENDING` | Pending approval exists |
| 422 | `HOOK_FAILED` | Hook execution failed and blocked the operation |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL_ERROR` | Internal server error |
| 503 | `SERVICE_UNAVAILABLE` | Service temporarily unavailable (starting up / database disconnected) |

---
## Chapter 19 Error Handling and Fault Tolerance Strategies

### 19.1 Layered Error Handling

```
Domain / Core Types Layer → Return domain errors (ErrTicketBlocked, ErrInvalidTransition)
                            Pure domain semantics, no HTTP concerns
          ↓
Service / Use-Case Layer → Wrap context (fmt.Errorf("claim ticket %s: %w", id, err))
                            Do not swallow errors, do not convert them
          ↓
Interface / Entry Layer → Map to HTTP status codes + unified error response
                            domain error → 4xx, infra error → 5xx
```

```go
// internal/httpapi/server.go / ticket_api.go
func ErrorHandler(err error, c echo.Context) {
    var domainErr *domain.Error
    if errors.As(err, &domainErr) {
        // Domain error → 4xx
        c.JSON(domainErr.HTTPStatus(), ErrorResponse{
            Code:    domainErr.Code(),
            Message: domainErr.Message(),
        })
        return
    }
    // Unknown error → 500 (log full stack, response returns only request_id)
    logger.Error("unhandled error", "err", err, "request_id", c.Get("request_id"))
    c.JSON(500, ErrorResponse{Code: "INTERNAL_ERROR", Message: "internal error"})
}
```

### 19.2 Database Fault Tolerance

| Scenario | Behavior |
|------|------|
| Connection loss | ent connection pool retries automatically; API request returns 503; orchestrator Tick skipped, retry on next Tick |
| Connection pool exhaustion | New requests wait in queue (default timeout 5s); timeout returns 503; alerting metric `db_connections_active` triggers an alarm |
| Migration failure | Startup aborts; logs output the specific migration error, and does not enter serving state |
| Slow query | `db_query_duration_seconds` Histogram exceeds threshold triggers alert; if orchestrator Tick times out, that Tick is skipped |
| Deadlock | ent transaction default sets `lock_timeout = 5s`; rollback and retry on timeout (up to 3 attempts) |

### 19.3 Agent Process Fault Tolerance

| Scenario | Behavior |
|------|------|
| Agent CLI process OOM kill | Worker goroutine receives exit signal via `cmd.Wait()` → marks abnormal exit → exponential backoff retry |
| Agent CLI zombie process | Worker waits 10 seconds after Kill; if process still exists, sends SIGKILL; records ActivityEvent |
| Agent CLI startup failure (command not found) | Worker returns error → triggers on_error Hook → exponential backoff retry → ActivityEvent records error |
| stdout stream interruption (broken pipe) | Worker detects EOF → treated as abnormal exit → retry |
| NDJSON parse error | Skip that line, log warning, do not interrupt session |
| File descriptor leak | Worker closes stdin/stdout/stderr pipes in defer; each Tick checks `/proc/self/fd` count as alerting metric |

### 19.4 Disk Space Fault Tolerance

| Scenario | Behavior |
|------|------|
| Workspace too large | on_done / on_fail / on_cancel Hook responsible for cleaning workspace; Reconciler periodically scans and cleans orphan workspaces |
| Insufficient disk space | On orchestrator Tick, check free space on partition containing `~/.openase/workspace/`; when below threshold (default 1GB), pause dispatching new tickets |
| Log file bloat | `~/.openase/logs/` retains last 7 days, via logrotate or built-in cleanup |

### 19.5 Graceful Shutdown

Shutdown sequence after receiving `SIGTERM` or `SIGINT`:

```
1. Stop accepting new HTTP requests (Echo graceful shutdown, wait for in-flight requests to complete, timeout 30s)
2. Stop orchestrator Ticker (no longer dispatch new tickets)
3. Send cancellation signal (context.Cancel) to all active Worker
4. Wait for all Workers to finish current operations or timeout (default 60s)
   - Worker receives cancel → sends SIGTERM to Agent CLI → waits for exit → updates ticket status to "interrupted"
   - Worker fails to exit within timeout → sends SIGKILL to Agent CLI → forcefully sets ticket status
5. Close SSE connections
6. Close database connection pool
7. Write final logs, exit
```

Tickets remain in their original state after graceful shutdown (`current_run_id` is cleared). On next startup, the orchestrator will scan these tickets through the crash-recovery flow and redispatch them.

### 19.6 Crash Recovery

Recovery check executed on process startup:

```go
func (s *Scheduler) recoverOnStartup(ctx context.Context) {
    // 1. Find all in_progress tickets without active Workers
    orphans, _ := s.ticketRepo.ListByStatus(ctx, ticket.StatusInProgress)

    for _, t := range orphans {
        // 2. Check Runtime readiness / heartbeat, instead of host PID
        if !s.isRuntimeReady(t.AgentRuntimePhase, t.LastHeartbeatAt) {
            // 3. Reset to todo, redispatch on next Tick
            t.TransitionTo(ticket.StatusTodo)
            t.AttemptCount++  // Count as one failed attempt
            s.ticketRepo.Save(ctx, t)
            s.logger.Warn("recovered orphan ticket", "ticket", t.Identifier)
        }
    }
}
```

---
## Chapter 20 Database Indexes and Performance

### 20.1 Core indexes

```sql
-- Core query polled every 5 seconds by the orchestrator
-- SELECT * FROM tickets WHERE project_id = ? AND status = 'todo' ORDER BY priority, created_at
CREATE INDEX idx_tickets_dispatch ON tickets (project_id, status, priority, created_at);

-- Ticket board page: grouped by project and status
CREATE INDEX idx_tickets_board ON tickets (project_id, status);

-- Ticket dependency check
CREATE INDEX idx_ticket_deps_blocker ON ticket_dependencies (target_ticket_id, type);
CREATE INDEX idx_ticket_deps_source ON ticket_dependencies (source_ticket_id);

-- TicketRepoScope: Webhook matching (find ticket by branch name)
CREATE INDEX idx_repo_scopes_branch ON ticket_repo_scopes (repo_id, branch_name);
CREATE INDEX idx_repo_scopes_ticket ON ticket_repo_scopes (ticket_id);

-- Agent heartbeat query
CREATE INDEX idx_agents_heartbeat ON agents (project_id, status, last_heartbeat_at);

-- Activity log: query by ticket and time
CREATE INDEX idx_activity_ticket ON activity_events (ticket_id, created_at DESC);
CREATE INDEX idx_activity_project ON activity_events (project_id, created_at DESC);

-- Scheduled job: next execution time
CREATE INDEX idx_scheduled_next ON scheduled_jobs (next_run_at) WHERE is_enabled = true;
```

### 20.2 JSON field indexes

```sql
-- If you need to query by fields in metadata (such as GitHub Issue ID from external_ref)
CREATE INDEX idx_tickets_external_ref ON tickets ((metadata->>'external_ref'));

-- ProjectRepo label query (native TEXT[] array)
CREATE INDEX idx_repos_labels ON project_repos USING GIN (labels);
```

### 20.3 ActivityEvent archiving strategy

ActivityEvent is an append-only table and will continue to grow. Use monthly partitioning + automatic archiving:

```sql
-- Partition by month
CREATE TABLE activity_events (
    id UUID,
    project_id UUID,
    ticket_id UUID,
    event_type TEXT,
    message TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
) PARTITION BY RANGE (created_at);

-- Automatically create monthly partitions (via pg_partman or app-layer cron)
CREATE TABLE activity_events_2026_03 PARTITION OF activity_events
    FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
```

Archiving strategy: keep 90 days of online data, detach older partitions with `DETACH` and archive to cold storage or directly `DROP` them (depending on compliance requirements).

### 20.4 Pagination approach

All list interfaces use cursor-based pagination (not offset-based, to avoid deep pagination performance issues):

```go
// cursor is a base64-encoded (created_at, id) tuple
// SELECT * FROM tickets
//   WHERE project_id = ? AND status = ? AND (created_at, id) < (cursor_time, cursor_id)
//   ORDER BY created_at DESC, id DESC
//   LIMIT 21  -- fetch one extra row to determine has_more
```

---
## Chapter 21 Complete Harness Template Example

### 21.1 Coding Harness (Refer to Symphony WORKFLOW.md)

The following is a production-level complete Coding Harness example. This content is persisted by the platform control plane; at runtime, the corresponding version will be materialized into temporary files in the workspace at startup for the Agent to use:

````markdown
---
# ═══ Agent Configuration ═══
agent:
  max_turns: 20                    # Maximum turn count
  timeout_minutes: 60              # Single-ticket timeout
  max_budget_usd: 5.00             # Per-ticket budget cap

# ═══ Branch Standards ═══
git:
  branch_pattern: "agent/{{ ticket.identifier }}"
  commit_convention: conventional   # conventional / freeform
  auto_push: true

# ═══ Hook Configuration ═══
hooks:
  on_claim:
{% for repo in repos %}
    - cmd: "git fetch origin && if git rev-parse --verify origin/{{ repo.branch }} >/dev/null 2>&1; then git checkout -B {{ repo.branch }} origin/{{ repo.branch }}; else git checkout -B {{ repo.branch }} origin/{{ repo.default_branch }}; fi"
      workdir: "{{ repo.name }}"
      timeout: 60
{% endfor %}
    - cmd: "pnpm install --frozen-lockfile"
      workdir: "frontend"
      timeout: 300
      on_failure: warn
  on_complete:
    - cmd: "make lint"
      timeout: 120
      on_failure: block
    - cmd: "make test"
      timeout: 600
      on_failure: block
    - cmd: "make typecheck"
      timeout: 120
      on_failure: block
  on_done:
    - cmd: "bash scripts/ci/cleanup.sh"
    - cmd: "bash scripts/ci/notify-slack.sh"

---

# Coding Workflow

You are a professional software engineer Agent, handling ticket `{{ ticket.identifier }}`.

## Ticket Context

- Identifier: {{ ticket.identifier }}
- Title: {{ ticket.title }}
- Priority: {{ ticket.priority }}
- Type: {{ ticket.type }}

{{ if .Ticket.Description }}
### Ticket Description

{{ ticket.description }}
{{ else }}
(No description, infer requirements from the title)
{{ end }}

{{ if gt .Attempt 1 }}
## Handoff Context

This is attempt {{ attempt }}. There may already be partial results from the previous attempt in the workspace.
Continue from the current workspace state; do not start from scratch. Check the reason for the previous failure and fix it.
{{ end }}

## Involved Repositories

{{ range .Repos }}
- **{{ repo.name }}** ({{ repo.labels | join(", ") }}): `{{ repo.path }}`
{{ end }}

## Workflow

1. **Analyze requirements**: Read the ticket description carefully and understand what needs to be done
2. **Assess impact scope**: Determine which files need to be changed and which modules are involved
3. **Implement changes**:
   - Follow the project's existing coding style and architectural patterns
   - Keep changes minimal and only modify what is necessary to complete the requirements
   - Commit meaningful changes in a timely manner (conventional commit format)
4. **Write/update tests**:
   - Write tests for new functionality
   - Ensure existing tests still pass
   - Goal: key paths of new code have test coverage
5. **Create PR**:
   - PR title format: `{{ ticket.identifier }}: <concise description>`
   - PR description includes: changes made, test methods, related ticket
   - Create a separate PR for each involved repository
6. **Self-check**:
   - Run lint, tests, and typecheck
   - Check for missing files and uncleaned debug code
   - Confirm all PRs have been submitted and linked to the ticket

## Scope

- Do not modify files unrelated to the ticket
- Do not delete existing tests (unless the ticket explicitly requests it)
- Do not change project architecture (unless the ticket explicitly requests it)
- Do not introduce new third-party dependencies (unless the ticket explicitly requests it)
- Do not modify CI/CD configuration files
- In case of ambiguity, choose the most conservative implementation
````

### 21.2 Security Harness Example

````markdown
---
agent:
  max_turns: 15
  timeout_minutes: 45
hooks:
  on_complete:
    - cmd: "make security-report"
      timeout: 300
      on_failure: warn
  on_done:
    - cmd: "bash scripts/ci/notify-security-team.sh"
---

# Security Scan Workflow

You are a security engineer Agent, responsible for conducting security audits on code related to ticket `{{ ticket.identifier }}`.

## Scan Scope

{{ range .Repos }}
- **{{ repo.name }}**: `{{ repo.path }}`
{{ end }}

## Workflow

1. **Static analysis**: scan the code for common security issues (injection, XSS, hardcoded keys, insecure dependencies)
2. **Dependency audit**: check dependencies for known vulnerabilities (CVE)
3. **PoC writing**: write proof-of-concept exploit code for high-risk issues found
4. **Generate report**: output a structured security report (findings, severity, remediation recommendations)
5. **Create fix tickets**: for every issue that needs fixing, create a sub-ticket

## Output Format

Append a security report to the ticket description in this format:

```
## Security Report
- [CRITICAL] SQL Injection in auth/login.go:42 — Remediation: Use parameterized queries
- [HIGH] Hardcoded API key in config/secrets.go:15 — Remediation: Move to environment variable
- [MEDIUM] Outdated dependency lodash@4.17.20 (CVE-2021-xxxx) — Remediation: Upgrade to 4.17.21
```
````
## Chapter 22 Configuration Schema

### 22.1 Full schema for `~/.openase/config.yaml`

```yaml
# ═══ Service ═══
server:
  host: "0.0.0.0"               # Listening address (default 0.0.0.0)
  port: 19836                    # Listening port (default 19836)
  mode: "all-in-one"             # all-in-one (default) | serve | orchestrate
  # mode determines which components are started:
  #   all-in-one = API + orchestrator + frontend (recommended, most common scenarios)
  #   serve      = API + frontend only (requires starting orchestrate separately)
  #   orchestrate = orchestrator only (requires starting serve separately)

# ═══ Database ═══
database:
  host: "localhost"              # required
  port: 5432                     # default 5432
  name: "openase"                # default openase
  user: "openase"                # required
  password: "${DB_PASSWORD}"     # reference .env variable
  ssl_mode: "disable"            # disable | require | verify-full
  max_connections: 20            # max connection pool size (default 20)
  lock_timeout: "5s"             # deadlock timeout

# ═══ Authentication ═══
auth:
  mode: "local"                  # local | oidc
  local:
    token: "${OPENASE_AUTH_TOKEN}"  # at least 50 characters
  oidc:                          # required when mode=oidc
    issuer_url: ""
    client_id: ""
    client_secret: "${OIDC_CLIENT_SECRET}"

# ═══ Orchestrator ═══
orchestrator:
  tick_interval: "5s"            # scheduling interval (default 5s)
  max_concurrent_agents: 5       # global max concurrency (default 5)
  stall_timeout: "5m"            # agent no-event timeout (default 5m)
  max_retry_backoff: "30m"       # max exponential backoff (default 30 minutes)
  error_alert_threshold: 3       # notify a human after N consecutive errors (default 3, do not stop retries)
  workspace_root: "~/.openase/workspace"  # Base root directory for ticket workspace
  min_disk_free_gb: 1            # disk free threshold to pause dispatching (default 1GB)

# ═══ Event transport ═══
event:
  driver: "auto"                 # auto (default) | channel | pgnotify
  # auto = selected automatically by server.mode:
  #   all-in-one → channel (Go channel, zero overhead)
  #   serve / orchestrate → pgnotify (PostgreSQL LISTEN/NOTIFY)
  # Can also override manually: force pgnotify (for example, all-in-one but want to test distributed transport)

# ═══ Notifications ═══
# Notification channels and subscription rules are configured through Web UI / API (Chapter 33), not in config.yaml
# Only global default channels are configured here (defaults can be written by setup)
notify:
  default_channel: "log"         # fallback when no NotificationRule is configured: log | slack
  slack:
    webhook_url: "${SLACK_WEBHOOK_URL}"

# ═══ Observability ═══
observability:
  tracing:
    enabled: false               # disabled by default
    endpoint: ""                 # OTel Collector gRPC address (e.g., localhost:4317)
  metrics:
    enabled: true                # enabled by default (in-memory metrics, used by Web UI dashboard)
    export:
      prometheus: false          # whether to expose /api/v1/system/metrics
      otlp_endpoint: ""          # OTel Collector address

# ═══ Logging ═══
log:
  level: "info"                  # debug | info | warn | error
  format: "json"                 # json | text (text is suitable for local development)
  output: "stdout"               # stdout | file
  file_path: "~/.openase/logs/openase.log"
  max_age_days: 7                # log retention days

# ═══ Git ═══
git:
  author_name: "OpenASE"
  author_email: "openase@localhost"
```

### 22.2 Full schema for `~/.openase/.env`

```bash
# Permissions 0600, stores only sensitive information; config.yaml references variables with ${VAR}

# Database
DB_PASSWORD=your_db_password

# Authentication
OPENASE_AUTH_TOKEN=your_local_auth_token_at_least_50_characters_long_xxxxx
OIDC_CLIENT_SECRET=                    # required when mode=oidc

# Agent CLI API Keys
ANTHROPIC_API_KEY=sk-ant-xxx           # used by Claude Code
OPENAI_API_KEY=sk-xxx                  # used by Codex
GOOGLE_API_KEY=xxx                     # used by Gemini

# Notifications
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/xxx

# Note:
# - GitHub outbound credential GH_TOKEN is not persisted in .env.
# - GH_TOKEN must be stored in the platform secret store and resolved by org/project scope,
#   then temporarily projected for local go-git or remote controlled sessions to use.

# Observability (optional)
OTEL_EXPORTER_OTLP_ENDPOINT=
```

### 22.3 `config.yaml` validation rules

```go
// Validate at startup; fail fast and output a clear error message on failure
func validateConfig(cfg Config) error {
    // required fields
    if cfg.Database.Host == "" { return fmt.Errorf("database.host is required") }
    if cfg.Database.User == "" { return fmt.Errorf("database.user is required") }

    // valid auth.mode values
    if cfg.Auth.Mode != "disabled" && cfg.Auth.Mode != "oidc" {
        return fmt.Errorf("auth.mode must be one of: disabled, oidc")
    }

    // OIDC completeness
    if cfg.Auth.Mode == "oidc" && cfg.Auth.OIDC.IssuerURL == "" {
        return fmt.Errorf("auth.oidc.issuer_url is required when auth.mode=oidc")
    }

    // port range
    if cfg.Server.Port < 1024 || cfg.Server.Port > 65535 {
        return fmt.Errorf("server.port must be between 1024 and 65535")
    }

    // valid server.mode values
    validModes := map[string]bool{"all-in-one": true, "serve": true, "orchestrate": true}
    if !validModes[cfg.Server.Mode] {
        return fmt.Errorf("server.mode must be one of: all-in-one, serve, orchestrate")
    }

    // valid event.driver values + cross-validation
    validDrivers := map[string]bool{"auto": true, "channel": true, "pgnotify": true}
    if !validDrivers[cfg.Event.Driver] {
        return fmt.Errorf("event.driver must be one of: auto, channel, pgnotify")
    }
    // channel is only valid in all-in-one mode (Go channel can only be shared within one process)
    if cfg.Event.Driver == "channel" && cfg.Server.Mode != "all-in-one" {
        return fmt.Errorf("event.driver=channel requires server.mode=all-in-one (Go channel cannot cross process boundary)")
    }

    return nil
}
```

---
## Chapter 23 Upgrade and Migration Strategy

### 23.1 Version Numbering Specification

OpenASE follows Semantic Versioning (SemVer): `MAJOR.MINOR.PATCH`

| Type | Meaning | Database Migration | Configuration Changes |
|------|---------|-------------------|----------------------|
| PATCH (0.1.1 → 0.1.2) | Bug fix | None | None |
| MINOR (0.1.x → 0.2.0) | New features | Possible (automatically backward compatible) | Possible new config items (with defaults) |
| MAJOR (0.x → 1.0) | Breaking change | Possible (review required) | Possible incompatible changes |

### 23.2 Upgrade Process

```bash
# 1. Stop the service
openase down

# 2. Back up the database (strongly recommended)
pg_dump -U openase openase > backup_$(date +%Y%m%d).sql

# 3. Replace binary
# Download new version or go install
mv openase-new /usr/local/bin/openase

# 4. Start service (automatically detect and run migrations)
openase up
```

### 23.3 Automatic Migration Mechanism

OpenASE automatically detects database version and executes migrations when it starts:

```go
func (s *Server) startup(ctx context.Context) error {
    // 1. Connect to the database
    client, err := ent.Open("postgres", cfg.Database.DSN())

    // 2. Compare current schema version vs target version
    // Atlas versioned migration mode
    dir, _ := iofs.New(migrations, "migrations")
    err = client.Schema.Create(ctx,
        schema.WithDir(dir),
        schema.WithDropColumn(false),  // Never drop columns automatically
        schema.WithDropIndex(false),   // Never drop indexes automatically
    )
    if err != nil {
        return fmt.Errorf("migration failed: %w\n\nRun 'openase migrate status' for details", err)
    }

    // 3. Record migration event
    s.logger.Info("database migration complete", "version", currentVersion)

    return nil
}
```

Key principles:

- **Add only, never delete**: Automatic migration only performs `ADD COLUMN`, `CREATE INDEX`, `CREATE TABLE`. It never automatically runs `DROP COLUMN` or `DROP TABLE`.
- **Backward compatibility**: The new version schema must be safely readable by the old binary (new columns have default values, old code ignores unknown columns).
- **Migration failure blocks startup**: The service will not run when the schema is in an inconsistent state.

### 23.4 Rollback Strategy

If the new version has issues and needs rollback:

```bash
# 1. Stop the new version
openase down

# 2. Restore old binary
mv openase-old /usr/local/bin/openase

# 3. Roll back schema if needed (may only be required for MAJOR upgrades)
psql -U openase openase < backup_20260318.sql

# 4. Start old version
openase up
```

Because of the add-only migration strategy, MINOR version rollbacks usually do not require restoring the database—the old code will ignore new columns it does not recognize. Only MAJOR upgrades (which may rename columns or change data formats) require database rollback.

### 23.5 Migration CLI Commands

```bash
openase migrate status    # View current schema version and pending migrations
openase migrate up        # Manually run all pending migrations (same as those run automatically at startup)
openase migrate history   # View migration execution history
```

### 23.6 Configuration Compatibility

- New configuration items added in new versions must have reasonable defaults, and users should not need to manually edit `config.yaml`.
- Deprecated configuration items should remain supported for 3 MINOR versions (with warning logs), then be removed.
- If interactive config completion is needed after upgrade, users may rerun `openase setup --force`; however, new configuration items should be safe defaults first, rather than relying on Web Setup Wizard prompts.

---
## Chapter 24 Testing Strategy

### 24.1 Testing Layer Overview

A core value of DDD layering is testability—each layer has clear input/output boundaries, dependencies are injected through interfaces, and external systems can all be mocked.

```
┌────────────────────────────────────────────────────────────────┐
│ E2E Tests (few, covering only critical paths)                   │
│ Real HTTP → Real DB → Real Agent CLI (mock server)               │
├────────────────────────────────────────────────────────────────┤
│ Integration Tests (moderate, verifying inter-component cooperation)│
│ Real DB (testcontainers) + mock Agent CLI + mock Git               │
├────────────────────────────────────────────────────────────────┤
│ Unit Tests (many, covering all business logic)                    │
│ Pure in-memory mocks, zero external dependencies, millisecond execution│
└────────────────────────────────────────────────────────────────┘
```

### 24.2 Layer-by-Layer Testing Strategy

**Domain / Core Types Layer — Target 100% Coverage, All Pure Unit Tests**

This layer is pure Go code with zero external dependencies and zero interface calls. Tests instantiate entities directly, invoke methods, and assert results. Nothing needs mocking because the domain layer does not depend on any interfaces.

| Test target | Test coverage | Mock requirement | Coverage target |
|-------------|---------------|------------------|-----------------|
| `internal/domain/ticketing/retry.go` | Exponential backoff, budget pause checks | None | 100% |
| `internal/domain/ticketing/cost.go` | token/cost parsing, amount rounding | None | 100% |
| `internal/domain/catalog/*.go` | Input parsing, UUID/limit/enum parsing, machine/provider pure rules | None | 100% |
| `internal/domain/notification/channel.go` | Notification channel types, config normalization, message structure | None | 100% |
| `internal/domain/notification/rule.go` | Subscription rule parsing, matching logic | None | 100% |
| `internal/domain/issueconnector/connector.go` | Historical external sync parsing logic (out of current scope) | None | 100% |
| `internal/types/pgarray/string_array.go` | PostgreSQL array edge cases | None | 100% |

```go
// internal/domain/ticketing/retry_test.go / internal/ticket/*_test.go — Example
func Test_Transition_TodoToInProgress_Success(t *testing.T) {
    ticket := NewTicket("ASE-1", "Fix bug")
    ticket.Status = StatusTodo

    err := ticket.TransitionTo(StatusInProgress)

    assert.NoError(t, err)
    assert.Equal(t, StatusInProgress, ticket.Status)
}

func Test_Transition_BacklogToDone_Rejected(t *testing.T) {
    ticket := NewTicket("ASE-1", "Fix bug")
    ticket.Status = StatusBacklog

    err := ticket.TransitionTo(StatusDone)

    assert.ErrorIs(t, err, ErrInvalidTransition)
}

func Test_Transition_InProgressToInReview_BlockedByDependency(t *testing.T) {
    parent := NewTicket("ASE-1", "Parent")
    parent.Status = StatusInProgress // not completed

    child := NewTicket("ASE-2", "Child")
    child.Status = StatusInProgress
    child.AddDependency(parent.ID, DependencyBlocks)

    err := child.CanTransitionTo(StatusInReview)

    assert.ErrorIs(t, err, ErrBlockedByDependency)
}
```

**Service / Use-Case Layer — Target 95%+ Coverage, Mock Repository + Provider**

This layer orchestrates use cases: call domain service → call repository → call provider. All dependencies are interfaces, all mocked.

| Test target | mocked interfaces | Key validation points |
|-------------|-------------------|-----------------------|
| `internal/service/catalog/*.go` | `internal/repo/catalog.Repository`, `provider.ExecutableResolver`, `MachineTester` | Orchestrating catalog use cases, resource probing, default and interdependent updates |
| `internal/ticket/*.go` | Ent client / repository boundaries, event bus, status template dependencies | ticket creation/status transitions/dependency relationships/budget and external-link logic |
| `internal/workflow/*.go` | repo / filesystem / provider boundaries | Harness validation, template rendering, skill installation, workflow orchestration |
| `internal/chat/*.go`, `internal/notification/*.go` | adapter / provider / service mock | conversation orchestration, notification sending, side-effect propagation |

```go
// internal/service/catalog/agent_catalog_test.go — Example
func TestCreateAgentProviderRejectsMissingExecutable(t *testing.T) {
    // Arrange
    svc := New(&stubRepository{}, stubExecutableResolver{}, nil)

    // Act
    _, err := svc.CreateAgentProvider(context.Background(), domain.CreateAgentProvider{
        OrganizationID: uuid.New(),
        Name:           "Gemini",
        AdapterType:    entagentprovider.AdapterTypeGeminiCli,
        ModelName:      "gemini-2.5-pro",
        AuthConfig:     map[string]any{},
    })

    // Assert
    assert.ErrorIs(t, err, ErrInvalidInput)
}
```

**Infrastructure Layer — Per-Component Testing Strategy**

This layer involves real external systems, so testing strategy varies significantly by component:

| Component | Testing method | mock / real | Coverage target |
|----------|----------------|-------------|-----------------|
| `internal/repo/` (ent-backed repository adapters) | Integration tests | **Real PostgreSQL** (testcontainers-go) | 90% |
| `internal/infra/adapter/claudecode/` | Unit tests | **Mock CLI subprocess** (fake NDJSON stream) | 85% |
| `internal/infra/adapter/codex/` | Unit tests | **Mock JSON-RPC server** (stdin/stdout pipe) | 85% |
| `internal/infra/hook/shell_executor.go` | Unit + integration | **Real shell** (executes test scripts) | 90% |
| `internal/infra/workspace/` | Integration tests | **Real filesystem** (temp dir) | 80% |
| `internal/infra/sse/hub.go` | Unit tests | **Mock HTTP ResponseWriter / fake subscribers** | 90% |
| `internal/infra/otel/*.go` | Unit tests | fake exporter / noop provider | 80-90% |
| `internal/infra/event/channel.go` | Unit tests | no external dependencies (pure Go channel) | 100% |
| `internal/infra/event/pgnotify.go` | Integration tests | **Real PostgreSQL** | 85% |
| `internal/notification/*` | Unit tests | **Mock HTTP** (httptest) | 90% |

```go
// internal/infra/adapter/claudecode/adapter_test.go — mock CLI subprocess
func Test_ClaudeCodeAdapter_StreamEvents_ParsesNDJSON(t *testing.T) {
    // Construct a fake claude process that emits predefined NDJSON
    fakeOutput := strings.Join([]string{
        `{"type":"system","subtype":"init","session_id":"sess-123"}`,
        `{"type":"assistant","content":"I'll fix the bug..."}`,
        `{"type":"tool_use","tool":"Edit","input":{"file":"auth.go"}}`,
        `{"type":"result","result":"Bug fixed","cost_usd":0.03}`,
    }, "\n")

    adapter := claudecode.NewAdapter(claudecode.Config{
        // Use echo to simulate the Claude CLI
        Command: "echo",
        Args:    []string{fakeOutput},
    })

    session, _ := adapter.Start(ctx, agentConfig)
    events, _ := adapter.StreamEvents(ctx, session)

    var collected []agent.AgentEvent
    for e := range events {
        collected = append(collected, e)
    }

    assert.Len(t, collected, 4)
    assert.Equal(t, "system", collected[0].Type)
    assert.Equal(t, "result", collected[3].Type)
    assert.InDelta(t, 0.03, collected[3].CostUSD, 0.001)
}
```

```go
// internal/repo/catalog/repo_test.go — Real PostgreSQL
func Test_TicketRepo_ListByStatus_WithPagination(t *testing.T) {
    if testing.Short() { t.Skip("requires PostgreSQL") }

    // testcontainers starts a temporary PostgreSQL
    pg := testutils.StartPostgres(t)
    client := ent.NewClient(ent.Driver(pg.Driver()))
    client.Schema.Create(ctx)

    repo := persistence.NewTicketRepo(client)

    // Insert test data
    for i := 0; i < 25; i++ {
        repo.Save(ctx, testutils.NewTicket(fmt.Sprintf("ASE-%d", i), ticket.StatusTodo))
    }

    // Act: paginated query
    page1, cursor, _ := repo.ListByStatus(ctx, ticket.StatusTodo, persistence.Page{Limit: 10})
    page2, _, _ := repo.ListByStatus(ctx, ticket.StatusTodo, persistence.Page{Limit: 10, Cursor: cursor})

    // Assert
    assert.Len(t, page1, 10)
    assert.Len(t, page2, 10)
    assert.NotEqual(t, page1[0].ID, page2[0].ID) // no overlap
}
```

**Orchestrator — Mixed Unit + Integration Testing**

| Component | Testing method | mocked interfaces | Key validation points |
|-----------|----------------|-------------------|-----------------------|
| `internal/orchestrator/scheduler.go` | Unit tests + integration tests | event provider, ent fixture, service boundary | Tick scheduling logic, blocking checks, concurrency limits, machine/agent selection |
| `internal/orchestrator/runtime_launcher.go` / `runtime_runner.go` | Unit tests | `AgentCLIProcessManager`, `TraceProvider`, filesystem boundary | runtime startup, event pumping, session lifecycle |
| `internal/orchestrator/health_checker.go` | Unit tests | ent fixture / fake clock | stall detection threshold, zombie runtime cleanup |
| `internal/orchestrator/machine_monitor.go` | Unit tests + integration tests | SSH/process boundary | remote machine availability, auth state, monitoring events |
| `internal/orchestrator/retry_service.go` | Unit tests | ticket/retry data fixture | backoff, recovery, pause conditions |
| `internal/orchestrator/connector_syncer.go` | Integration tests | real DB + connector fake | historical external synchronization flow (out of current scope) |
| Full scheduling loop | Integration tests | real DB + mock Adapter | ticket end-to-end flow from todo → claimed → running → review |

```go
// internal/orchestrator/scheduler_test.go — Simplified illustration
func TestSchedulerRunTickSkipsBlockedTickets(t *testing.T) {
    fixture := newSchedulerFixture(t)
    fixture.createBlockedCandidate("ASE-2")
    fixture.createRunnableCandidate("ASE-3")

    report, err := fixture.scheduler.RunTick(context.Background())

    require.NoError(t, err)
    assert.Equal(t, 1, report.TicketsSkipped["blocked"])
    assert.Equal(t, 1, report.TicketsDispatched)
}
```

**Interface / Entry Layer — Thin Layer, Testing HTTP Contracts**

| Component | Testing method | mock | Key validation points |
|-----------|----------------|------|-----------------------|
| `internal/httpapi/*.go` | Unit tests + integration tests | service/use-case boundary, provider | HTTP status codes, request parameter binding, response format, error mapping |
| `internal/httpapi/tracing.go` | Unit tests | `TraceProvider` | span creation, request_id injection |
| `internal/httpapi/sse.go` | Integration tests | `EventProvider` (ChannelBus) | SSE event format, ping keepalive, filtering logic |
| `cmd/openase/main.go`, `internal/cli/*.go` | Unit tests | command/service mock | entrypoint parameters, exit codes, error passthrough |

```go
// internal/httpapi/ticket_api_test.go — httptest (simplified illustration)
func Test_CreateTicket_Returns201_WithIdentifier(t *testing.T) {
    server := newHTTPServerFixture(t)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-1/tickets",
        strings.NewReader(`{"title":"Fix bug","priority":"high"}`))
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    server.echo.ServeHTTP(rec, req)

    assert.Equal(t, 201, rec.Code)
    assert.Contains(t, rec.Body.String(), "\"identifier\":\"ASE-1\"")
}

func Test_CreateTicket_Returns400_WhenTitleMissing(t *testing.T) {
    server := newHTTPServerFixture(t)

    req := httptest.NewRequest(http.MethodPost, "/api/v1/projects/proj-1/tickets",
        strings.NewReader(`{"priority":"high"}`)) // Missing title
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()
    server.echo.ServeHTTP(rec, req)

    assert.Equal(t, 400, rec.Code)
}
```

### 24.3 Which Interfaces Can Be Mocked and Which Cannot

**Can and should be mocked (in unit tests):**

| Interface | Definition location | Mock generation approach |
|-----------|--------------------|--------------------------|
| `catalog.Repository` | `internal/repo/catalog/repo.go` | `mockery` or hand-written stub |
| `MachineTester` | `internal/service/catalog/service.go` | hand-written stub / mock |
| `provider.TraceProvider` | `internal/provider/trace.go` | `mockery` or `NoopTracer` |
| `provider.MetricsProvider` | `internal/provider/metrics.go` | `mockery` or `NoopMetrics` |
| `provider.EventProvider` | `internal/provider/event.go` | `mockery` or `ChannelBus` (real but lightweight) |
| `AgentCLIProcessManager` | `internal/provider/agentcli.go` | `mockery` or fake manager |
| `UserServiceManager` | `internal/provider/service.go` | `mockery` |

A principle: **all interfaces exposed on current service / provider boundaries should have corresponding mocks or stubs.** Prefer generating mocks on stable provider / repository boundaries; for local service dependencies, hand-written stubs are often easier to control.

**Should not be mocked (must use real implementation in integration tests):**

| Component | Why it cannot be mocked |
|-----------|-------------------------|
| PostgreSQL | ent query behavior, transaction isolation, LISTEN/NOTIFY cannot be faithfully simulated in mocks. Use `testcontainers-go` to start temporary PostgreSQL |
| Filesystem (workspace, harness) | File permissions, symlinks, concurrent writes and boundary behavior must be tested on a real filesystem |
| Shell Hook execution | Subprocess management, exit codes, stdin/stdout/stderr pipes behave differently in mocks |
| go-git operations | Git branch/merge/conflict resolution logic must be tested on a real repository (can be initialized in a temp dir) |

### 24.4 Required Integration Test Paths That Must Pass

The following 6 paths are the core backbone of the system and must have end-to-end integration test coverage; they cannot rely on mocks alone:

**Path 1: Ticket mainline (most critical)**

```
Create ticket → orchestrator Tick discovery → on_claim lifecycle Hook → Agent Adapter starts
→ multi-turn Agent execution / continuation → Agent explicitly requests state progression → on_complete lifecycle Hook → state advances to in_review
```

Test environment: real PostgreSQL + mock Agent Adapter (returns predefined NDJSON stream) + real lifecycle Hook (simple `exit 0` script)

**Path 2: Hook failure blocking**

```
ticket in_progress → Agent completes → on_complete lifecycle Hook fails (exit 1)
→ ticket remains in in_progress → Agent receives feedback → retries
```

Test environment: real PostgreSQL + mock Adapter + real Hook (`exit 1` script)

**Path 3: Multi-Repo PR aggregation**

```
ticket involves 2 repos → Agent submits 2 PRs → Webhook callback PR merged
→ first merged (ticket still in in_review) → second merged (ticket moves to done)
```

Test environment: real PostgreSQL + mock platform write path (HTTP/API calls)

**Path 4: Stall detection and recovery**

```
Agent starts → no events for 5 minutes → HealthChecker marks Stall → Kill Worker
→ retry (attempt_count + 1) → continuous Stall → notify human, continue backoff retry
```

Test environment: real PostgreSQL + mock Adapter (no events emitted after start) + reduced `stall_timeout` to 1 second

**Path 5: Crash recovery**

```
ticket in_progress → process is killed → restart
→ recoverOnStartup scans orphan tickets → reset to todo → redistributed on next Tick
```

Test environment: real PostgreSQL + manually set ticket state to in_progress (simulate crash)

**Path 6: SSE realtime push**

```
Establish SSE connection → create ticket → SSE receives ticket.created event
→ ticket status change → SSE receives ticket.status_changed event
```

Test environment: real HTTP service (`httptest.Server`) + ChannelBus + real PostgreSQL

### 24.5 Frontend Testing Strategy

| Test type | Tooling | Coverage scope | Coverage target |
|-----------|---------|----------------|-----------------|
| Component unit tests | vitest + @testing-library/svelte | UI component rendering, props behavior, event triggering | 80% |
| Store logic tests | vitest | State management and SSE event handling in Svelte stores | 90% |
| API client tests | vitest + msw (Mock Service Worker) | Request construction, error handling, retry in fetch wrapper | 90% |
| UX smoke / perf regression | Playwright | Lightweight regressions for 6 critical interaction paths; fixed fixtures + mock data; record key step latency and assert budget | Critical path coverage; required for PRs |

#### 24.5.1 Frontend Interaction Smoothness Regression (Non-functional Requirement)

OpenASE frontend must treat “interaction responsiveness” as an explicit non-functional requirement, not as a subjective post-release judgment. To avoid making the Playwright suite heavy and flaky as a full end-to-end suite, PR stages keep only one **lightweight UX smoke / perf regression** suite with the following goals:

- Run on every frontend-related PR
- Short runtime, stable outcomes, and clear failure signals
- Cover only fixed high-frequency critical paths; no dependency on real external services
- Use mock / fixture / stable test data; no need for a complete backend orchestration chain
- Verify functional usability and confirm critical interactions do not become noticeably sluggish

This Playwright regression currently covers only the following 6 paths:

1. Main workspace navigation switching: after switching among `Board / Machines / Agents / Settings / Scheduled Jobs / Workflows` in the project page, the main content should become interactive quickly
2. Machines list view and edit: open Machines page, click strip card, open drawer, edit, and save
3. Machines quick actions: trigger `Test connection`, verify immediate feedback, completion feedback, and stable resource info
4. Repositories list view and edit: open repository list, open drawer, edit, and save
5. Core Agents page interactions: open key sheets/drawers such as provider/registration, verify primary action flows
6. Scheduled Jobs / Workflows operations: open Scheduled Jobs or Workflow management page, enter create/edit flow, and complete submission

In OpenASE, Playwright serves as an “interaction regression budget guard,” not a full-scale realistic load test. It must record and assert these metrics:

| Metric | Definition | Purpose |
|--------|------------|---------|
| `route_to_interactive_ms` | Time from navigation click or `goto` to main interactive elements of new page becoming visible and operable | Measure whether page transitions feel slow |
| `action_feedback_ms` | Time from click to first clear user feedback (drawer opens, loading, toast, status change) | Measure whether there is an immediate response |
| `action_complete_ms` | Time from action start to actual completion and UI stabilizing | Measure end-to-end action path latency |
| `stability_assertions` | Whether key UI remains consistent before and after action, with no disappearing elements, state mismatch, or meaningless jitter | Measure interaction stability |
| `continuation_ready` | Whether the user can immediately proceed to the next action after current action completes | Measure flow continuity |

Initial default budgets are as follows as PR regression thresholds:

| Scenario | Budget |
|----------|--------|
| Main navigation to interactive main page content | `p75 < 800ms` |
| Card click to drawer visible | `p75 < 150ms` |
| Drawer open to first editable input | `p75 < 250ms` |
| Save click to loading / disabled / first feedback appears | `p75 < 100ms` |
| Save click to successful toast or success state appears | `p75 < 1500ms` in local/CI fixed fixture environment |
| Local filtering or lightweight list update | `p75 < 150ms` |
| First feedback for test connection | `p75 < 150ms` |

To keep results stable, the Playwright suite in PR stage must satisfy these constraints:

- Do not depend on real GitHub, real SSH, real Agent CLI, or real external network
- Do not depend on large preloaded databases or long startup chains
- Prefer reading frontend instrumentation `performance.mark/measure` instead of fragile sleeps or eyeballed timing approximations
- Default to smoke + regression only, not full end-to-end real-environment business acceptance
- Failure reports must indicate whether navigation switching, drawer opening, save feedback, or list updates exceeded budget; avoid “only says it failed” without clear slowdown location

Recommended frontend instrumentation naming convention follows and should be shared by Playwright and production RUM:

- `nav:start` / `nav:interactive`
- `drawer:start` / `drawer:visible` / `drawer:ready`
- `save:start` / `save:feedback` / `save:success` / `save:error`
- `filter:start` / `filter:applied`
- `test_connection:start` / `test_connection:feedback` / `test_connection:done`

This requirement is non-functional: **if a change makes critical interaction paths “functionally still correct” but clearly slower, more sluggish, or less predictable, the change should not be considered complete.**

```typescript
// web/src/lib/api/sse.test.ts — SSE store test
import { describe, it, expect, vi } from 'vitest'
import { createTicketStream } from './sse'

describe('createTicketStream', () => {
  it('updates store on ticket.created event', () => {
    // mock EventSource
    const mockES = { addEventListener: vi.fn(), onerror: null, onopen: null, close: vi.fn() }
    vi.stubGlobal('EventSource', vi.fn(() => mockES))

    const { tickets } = createTicketStream('project-1')

    // Simulate SSE event
    const handler = mockES.addEventListener.mock.calls.find(c => c[0] === 'ticket.created')[1]
    handler({ data: JSON.stringify({ ticketId: 'ASE-1', ticket: { id: 'ASE-1', status: 'backlog' } }) })

    let value
    tickets.subscribe(v => value = v)
    expect(value.get('ASE-1')).toEqual({ id: 'ASE-1', status: 'backlog' })
  })
})
```

### 24.6 Coverage Targets and Reality

**Is 100% coverage achievable?**

Answered by layer:

| Layer | Can 100% be done? | Realistic target | Notes |
|-------|--------------------|------------------|-------|
| Domain / Core Types | **Yes, required** | 100% | Mainly corresponds to pure logic and parsing code in `internal/domain/*` and `internal/types/*` |
| Service / Use-Case | Nearly possible | 95%+ | In current repository, mainly `internal/service/*`, `internal/ticket`, `internal/workflow`, `internal/chat`, and similar service packages; most dependencies can be mocked |
| Infrastructure | Not realistic | 80-90% | Full coverage of edge cases in external system interaction is difficult (network timeouts, concurrency races, etc.) |
| Repository / Persistence | Near feasible | 90% | In current repository, mainly `internal/repo/*`, well-suited for integration tests with real PostgreSQL |
| Interface / Entry | Near feasible | 90%+ | In current repository, mainly `internal/httpapi`, `internal/cli`, `cmd/openase`; keep HTTP handlers thin and wire-level entrypoints accounted separately |
| Orchestrator | Not realistic | 85% | Involves goroutine concurrency, timers, and subprocess management; some races are hard to trigger deterministically |
| Frontend | Not realistic | 80% | UI interaction edge cases (browser compatibility, animation timing) are hard to fully cover |

**Overall coverage target: 75%+, with domain layer at 100%.**

Do not pursue 100% overall coverage—it leads to meaningless tests written only to chase the score (such as testing getter methods). Use `go test -coverprofile` in CI, set a 75% threshold, and require domain layer 100% as a hard constraint.

By default, the repository runs backend tests and coverage gating via `make check`; both CI and local push gates call `scripts/ci/backend_coverage.sh` consistently. The default requirement is “full backend tests pass + domain/core coverage threshold passes”; to run an additional full backend scope total coverage check, you must explicitly set `OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true`.

### 24.7 Mock Generation and Testing Toolchain

```yaml
# Defined in Makefile
test-unit:             ## Run unit tests (domain/core + service/use-case + httpapi)
	go test ./internal/domain/... ./internal/service/... ./internal/ticket ./internal/workflow ./internal/chat ./internal/notification ./internal/httpapi -short -count=1 -coverprofile=coverage-unit.out

test-integration:      ## Run integration tests (repository + infra + orchestrator, needs PostgreSQL)
	go test ./internal/repo/... ./internal/infra/... ./internal/orchestrator ./internal/runtime/... -count=1 -coverprofile=coverage-integration.out

test-all:              ## Run all tests
	go test ./... -count=1 -coverprofile=coverage-all.out

test-backend-coverage: ## Run full backend tests + domain/types 100% coverage gate (when OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true, also requires overall 75%+)
	./scripts/ci/backend_coverage.sh

test-coverage:         ## Coverage report
	go tool cover -func=coverage-all.out | tail -1
	@echo "Domain/Core coverage:"
	go test ./internal/domain/... ./internal/types/... -coverprofile=coverage-domain.out
	go tool cover -func=coverage-domain.out | tail -1

mock-generate:         ## Generate mocks (mockery)
	mockery --all --dir=./internal --output=./mocks --outpkg=mocks

test-frontend:         ## Frontend tests
	cd web && pnpm run test

test-e2e:              ## Playwright UX smoke / perf regression (required for PR, lightweight fixed fixtures)
	cd web && pnpm exec playwright test
```

### 24.8 Test Directory Structure

```
openase/
├── internal/
│   ├── domain/
│   │   ├── ticketing/
│   │   │   ├── retry.go
│   │   │   └── retry_test.go             # Pure logic unit tests
│   │   ├── notification/
│   │   └── ...
│   ├── service/
│   │   ├── catalog/
│   │   │   ├── service.go
│   │   │   └── agent_catalog_test.go     # Service layer unit tests
│   │   └── ...
│   ├── repo/
│   │   ├── catalog/
│   │   │   ├── repo.go
│   │   │   └── repo_test.go              # Repository integration tests (testcontainers / Postgres)
│   │   └── ...
│   ├── infra/
│   │   ├── adapter/
│   │   │   ├── claudecode/
│   │   │   │   ├── adapter.go
│   │   │   │   └── adapter_test.go       # Unit tests (fake NDJSON)
│   │   │   └── ...
│   │   └── ...
│   ├── orchestrator/
│   │   ├── scheduler.go
│   │   ├── scheduler_test.go             # Unit/integration mixed tests
│   │   └── ...
│   ├── httpapi/
│   │   ├── ticket_api.go
│   │   └── ticket_api_test.go            # HTTP contract tests
│   └── ...
├── mocks/                                # mockery-generated files
│   ├── catalog_repository.go
│   ├── agent_adapter.go
│   ├── event_provider.go
│   └── ...
├── tests/
│   ├── integration/
│   │   ├── ticket_lifecycle_test.go      # Path 1: ticket mainline
│   │   ├── hook_blocking_test.go         # Path 2: Hook blocking
│   │   ├── multi_repo_pr_test.go         # Path 3: Multi-repo PR
│   │   ├── stall_recovery_test.go        # Path 4: Stall detection
│   │   ├── crash_recovery_test.go        # Path 5: crash recovery
│   │   └── sse_realtime_test.go          # Path 6: SSE push
│   ├── testutils/
│   │   ├── postgres.go                   # testcontainers wrapper
│   │   ├── fixtures.go                   # test data factory
│   │   └── fake_adapter.go               # fake Agent Adapter
│   └── testdata/
│       ├── harnesses/                    # test harness files
│       └── hooks/                        # test hook scripts
└── web/
    ├── src/lib/
    │   ├── api/sse.test.ts               # SSE store tests
    │   ├── api/client.test.ts            # API client tests
    │   └── components/ticket/
    │       └── TicketCard.test.ts        # Component tests
    └── tests/
        └── e2e/                          # Playwright UX smoke / perf regression
            ├── navigation.spec.ts
            ├── machines.spec.ts
            ├── repositories.spec.ts
            ├── agents.spec.ts
            ├── workflows.spec.ts
            └── perf.ts
```

---
## Chapter 25 Multi-Machine Support (SSH Control Plane)

### 25.1 Scenarios and Motivation

Research and engineering teams usually have multiple heterogeneous Linux machines—GPU training servers, high-memory data-processing nodes, general development boxes, and local laptops. Different tickets have different compute requirements: model training needs GPUs, data cleaning needs large memory, and code refactoring can run on almost any machine.

OpenASE needs to support:

- Binding a ticket to a specific machine for execution (or letting the orchestrator automatically select based on resource needs)
- Starting an Agent CLI child process on remote machines and streaming events back in real time
- Allowing the executing Agent to access other machines via SSH during execution (such as copying training results from a GPU machine to a storage machine)
- Monitoring resource status across machines in real time (CPU, memory, GPU, disk)

### 25.2 Architecture: Single Control Plane + SSH Execution Plane

```
┌──────────────────────────────────────────────────────────┐
│ Control Plane (openase all-in-one)                        │
│ Runs on a single machine only and manages all state           │
│                                                          │
│ ┌────────────┐  ┌──────────────┐  ┌───────────────────┐  │
│ │ API Server │  │ Orchestrator │  │ Machine Monitor   │  │
│ │ + Web UI   │  │ (Scheduler)  │  │ (SSH health check)│  │
│ └────────────┘  └──────┬───────┘  └────────┬──────────┘  │
│                        │                   │             │
└────────────────────────┼───────────────────┼─────────────┘
                         │ SSH               │ SSH
              ┌──────────┼───────────────────┼──────────┐
              ▼          ▼                   ▼          ▼
         ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
         │ local   │ │ gpu-01  │ │ gpu-02  │ │ storage │
         │ (self)  │ │ A100×4  │ │ H100×8  │ │ 64GB    │
         │         │ │ 256GB   │ │ 512GB   │ │ 16TB    │
         └─────────┘ └─────────┘ └─────────┘ └─────────┘
           Worker      Worker      Worker      (No Agent,
         on local     started via  started via   access-only)
         executes     SSH         SSH
          via         Agent CLI   Agent CLI

```

Key design decision: **the control plane is not distributed.** The OpenASE main process runs on only one machine, and PostgreSQL also runs on that machine (or a machine reachable by it). Remote machines do not run any OpenASE components—they only need SSH reachability and the Agent CLI installed. In this way, remote machines are stateless execution nodes, and failures on them do not affect the system.

### 25.3 Machine Entity

Add `Machine` as a first-class entity:

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| organization_id | FK | Owning organization |
| name | String | Machine alias (such as `gpu-01`, `storage`, `local`), unique within the organization |
| host | String | SSH endpoint address (such as `10.0.1.10` or `gpu-01.lab.internal`). `local` means the control-plane host itself |
| port | Integer | SSH port (default 22) |
| ssh_user | String | SSH username |
| ssh_key_path | String | SSH private key path (relative to `~/.openase/`, such as `keys/gpu-01.pem`) |
| description | Text | Machine description (Markdown), for reference during Agent execution (for example, `"A100 GPU × 4 for training"`) |
| labels | TEXT[] | Resource labels (such as `{"gpu", "a100", "cuda-12"}` or `{"high-memory", "data-processing"}`) |
| status | Enum | online / offline / degraded / maintenance |
| workspace_root | String | Remote ticket workspace root directory (such as `/home/openase/.openase/workspace`) |
| env_vars | TEXT[] | Environment variables injected during remote execution (such as `{"CUDA_VISIBLE_DEVICES=0,1", "HF_HOME=/data/huggingface"}`) |
| last_heartbeat_at | DateTime | Last health check time |
| resources | JSONB | Latest collected resource snapshot (dynamic data, suitable for JSONB) |

**Machine `status` semantic contract:**

- `online`
  - Indicates only **machine-level execution surface health**
  - The machine's most recent L1 reachability check succeeded
  - and no infrastructure-level fault that blocks scheduling has been triggered
  - `online` **does not mean** any specific adapter is installed, logged in, or guaranteed to start successfully
- `degraded`
  - Machine is reachable, but there are issues requiring operations attention
  - For example: low disk space, L2/L4/L5 check failures, partial environment anomalies
  - Not considered for automatic scheduling by default, unless a future product-level strategy flag explicitly enables “degraded is schedulable”
- `offline`
  - Machine is not reachable at the infrastructure level and cannot be used as an execution node
- `maintenance`
  - Manual maintenance status; no scheduling is allowed regardless of monitoring results

**Key constraints:**

- Machine `status` answers only “is this machine healthy as an execution node”
- Provider `availability_state` answers only “whether a specific Agent CLI entry on this machine can execute”
- The scheduler must check both `machine.status` and `provider.availability_state` and must not rely on only one
- `workspace_root` controls only the ticket workspace root; it must not be reused as another repository-path semantic

**`resources` JSONB structure** (collected periodically by Machine Monitor):

```json
{
  "cpu_cores": 32,
  "cpu_usage_percent": 45.2,
  "memory_total_gb": 256,
  "memory_used_gb": 120,
  "memory_available_gb": 136,
  "disk_total_gb": 2000,
  "disk_available_gb": 1200,
  "gpu": [
    {"index": 0, "name": "A100-80G", "memory_total_gb": 80, "memory_used_gb": 12, "utilization_percent": 0},
    {"index": 1, "name": "A100-80G", "memory_total_gb": 80, "memory_used_gb": 0, "utilization_percent": 0}
  ],
  "agent_cli": {
    "claude_code": {"installed": true, "version": "1.2.3", "path": "/usr/local/bin/claude"},
    "codex": {"installed": false},
    "gemini": {"installed": false}
  },
  "collected_at": "2026-03-19T10:30:00Z"
}
```

### 25.4 `local` Machine — Default Zero Configuration

The system automatically creates a `Machine` record with `name=local` and `host=local` during initialization. **When the Machine table contains only the `local` record, all behavior is exactly the same as before**—the Agent runs as a local subprocess with no SSH involvement. Users only add remote machines when multi-machine support is needed.

This ensures single-node users do not need to be aware of multi-machine features at all.

### 25.5 Binding Relationship: Workflow × Agent × Provider × Machine

Multi-machine semantics change to **explicit binding** rather than runtime “automatically guessing which machine to use”:

- Workflow binds to Agent definition
- Agent definition binds to Provider
- Provider binds to Machine
- Therefore each Workflow run is naturally pinned to a fixed Machine

In other words, when users configure an Agent, they are really selecting “which Coding Agent CLI entry on which machine.” The scheduler at runtime only performs availability checks and semaphore control; it no longer performs automatic matching across multiple machines based on `required_machine_labels`.

**Scheduling admission rules:**

- `machine.status == online`
- `provider.availability_state == available`
- Provider semaphore not full
- Workflow / Stage / Project concurrency limits not full

Where:

- Machine `online` is the infrastructure health check
- Provider `availability_state == available` is the check for whether the concrete adapter entry is executable
- Both must be satisfied; either one missing blocks scheduling

Allocation strategy (in orchestrator Tick):

```go
// internal/orchestrator/scheduler.go
func (s *Scheduler) resolveExecutionTarget(ctx context.Context, wf *workflow.Workflow) (*agent.Agent, *provider.AgentProvider, *machine.Machine, error) {
    ag, err := s.agentRepo.Get(ctx, wf.AgentID)
    if err != nil || !ag.IsEnabled {
        return nil, nil, nil, ErrAgentUnavailable
    }

    p, err := s.providerRepo.Get(ctx, ag.ProviderID)
    if err != nil {
        return nil, nil, nil, ErrProviderUnavailable
    }

    m, err := s.machineRepo.Get(ctx, p.MachineID)
    if err != nil || m.Status != machine.StatusOnline {
        return nil, nil, nil, ErrMachineUnavailable
    }

    if p.AvailabilityState != provider.AvailabilityAvailable {
        return nil, nil, nil, ErrProviderUnavailable
    }

    if s.providerSemaphore.Active(p.ID) >= p.MaxParallelRuns {
        return nil, nil, nil, ErrProviderBusy
    }
    return ag, p, m, nil
}
```

If the same Workflow needs both GPU and CPU variants in the future, or machine failover, model this through explicit creation of multiple Agent/Workflow bindings, or future addition of a “candidate Agent set” capability; the current PRD no longer uses `required_machine_labels` as the default scheduling entry.

### 25.6 SSH Agent Runner

When a Workflow is bound to a Provider on a remote Machine, the Worker completes the entire ticket execution flow on the remote host via SSH: create ticket workspace → **remote git clone** → write Prompt → start the Agent CLI corresponding to that Provider.

**Repo strategy: remote git clone (not rsync, no shared storage dependency).** Each remote machine independently clones the repository into its own local workspace. Reasons: remote machines may be on different networks, shared storage is unreliable; git clone guarantees a clean code state each time; Agent executing git push on remote does not require sending files back to the control plane.

**GitHub repository authentication convention:**

- OpenASE uniformly uses the platform-hosted `GH_TOKEN` as the outbound credential for GitHub repositories.
- Remote shell `git clone / fetch / push` must consume this `GH_TOKEN` through controlled environment injection; users must not be required to run `gh auth login` manually on remote hosts.
- The local `go-git` path also must explicitly consume the same `GH_TOKEN`, ensuring behavior is consistent between local and remote.
- GitHub repository URLs should uniformly prefer `https://github.com/...git` to avoid making SSH key login state an implicit platform prerequisite.

```go
// internal/orchestrator/runtime_runner.go — Remote execution (conceptual sketch)
func (w *Worker) runOnRemote(ctx context.Context, m *machine.Machine, p *provider.AgentProvider, t *ticket.Ticket, harness *Harness) error {
    sshClient, err := w.sshPool.Get(ctx, m)
    if err != nil {
        return fmt.Errorf("ssh connect to %s: %w", m.Name, err)
    }

    workDir := fmt.Sprintf("%s/%s/%s/%s", m.WorkspaceRoot, t.OrganizationSlug, t.ProjectSlug, t.Identifier)

    // 1. Create the ticket workspace + remote git clone of all involved repos
    repoScopes, _ := w.repoScopeRepo.ListByTicket(ctx, t.ID)
    var cloneCmds []string
    cloneCmds = append(cloneCmds, fmt.Sprintf("mkdir -p %s", workDir))
    for _, scope := range repoScopes {
        repo, _ := w.repoRepo.Get(ctx, scope.RepoID)
        repoDir := fmt.Sprintf("%s/%s", workDir, repo.Name)
        // clone if it does not exist, otherwise fetch + checkout (for retry scenarios)
        cloneCmds = append(cloneCmds, fmt.Sprintf(
            `if [ -d "%s/.git" ]; then cd %s && git fetch origin && git checkout -B agent/%s origin/%s; else git clone --depth 50 %s %s && cd %s && git checkout -B agent/%s; fi`,
            repoDir, repoDir, t.Identifier, repo.DefaultBranch,
            repo.RepositoryURL, repoDir, repoDir, t.Identifier,
        ))
    }
    session, _ := sshClient.NewSession()
    session.Run(strings.Join(cloneCmds, " && "))

    // 2. Write rendered Harness Prompt to remote
    prompt := harness.Render(w.buildTemplateData(t))
    promptPath := fmt.Sprintf("%s/.harness-prompt.md", workDir)
    w.sshWriteFile(sshClient, promptPath, prompt)

    // 3. Inject skills into remote Agent CLI skills directory
    //    scp skill files from control plane to remote
    for _, skillName := range harness.Skills {
        localSkill := filepath.Join(w.projectSkillsDir, skillName)
        remoteSkill := fmt.Sprintf("%s/.claude/skills/%s", workDir, skillName)
        w.sshCopyDir(sshClient, localSkill, remoteSkill)
    }

    // 4. Start Agent CLI, stream stdout back through SSH session to control plane
    execSession, _ := sshClient.NewSession()
    for _, env := range m.EnvVars {
        parts := strings.SplitN(env, "=", 2)
        execSession.Setenv(parts[0], parts[1])
    }
    // Inject platform API environment variables so Agent on remote can call control-plane API
    execSession.Setenv("OPENASE_API_URL", w.platformAPIURL)
    execSession.Setenv("OPENASE_AGENT_TOKEN", w.agentToken)

    cmd := fmt.Sprintf("cd %s && %s -p \"$(cat .harness-prompt.md)\" --output-format stream-json --allowedTools \"Bash,Read,Edit,Write,Glob,Grep\" --max-turns %d",
        workDir, p.CLICommand, harness.MaxTurns)

    stdout, _ := execSession.StdoutPipe()
    execSession.Start(cmd)

    // 5. Parse remote NDJSON stream (exact same event parsing logic as local)
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        event, _ := parseAgentEvent(scanner.Bytes())
        w.handleEvent(ctx, t, event)
    }

    return execSession.Wait()
}
```

**Key design point: remote execution and local execution share the same `AgentAdapter` interface.** The only difference is subprocess launch mode: local uses `os/exec`, remote uses SSH session. The adapter layer does not perceive Machine—Machine is an orchestrator-layer concept.

**Remote Agent can also call the platform API:** the control-plane `OPENASE_API_URL` and `OPENASE_AGENT_TOKEN` are injected into the remote environment via SSH variables. The remote Agent calls the control-plane API via HTTP (create subtickets, update projects, and so on)—the control plane port only needs to be reachable from remote machines.

### 25.7 Machine Monitor (Pyramid-Frequency Monitoring)

Monitoring uses a **pyramid strategy**: the lightest checks run most frequently, the heaviest checks have the longest interval. This reduces SSH overhead while keeping critical state timely.

**Monitoring levels (top to bottom: frequency decreases, cost increases):**

```
           ▲ Frequency
           │
 Level 1   │  ████████████████████████  Every 15 seconds
 Network+SSH│  ICMP ping + SSH handshake
           │
 Level 2   │  ████████████████          Every 60 seconds
 System     │  CPU/memory/disk (single SSH command)
 Resources  │
           │
 Level 3   │  ████████                  Every 5 minutes
 GPU Status │  nvidia-smi (for GPU-labeled machines)
           │
 Level 4   │  ████                      Every 30 minutes
 Agent Env  │  CLI availability+version+login state
           │
 Level 5   │  ██                        Every 6 hours / manual
 Full Audit │  Git credentials+GitHub CLI+egress network
           │
           └──────────────────────────→ Cost
```

**Checks per level:**

| Level | Frequency | SSH count | Check items | Failure behavior |
|------|------|---------|--------|---------|
| **L1: Network Reachability** | 15s | 0 (ICMP) + 1 (SSH handshake) | ping reachability + SSH port connectivity + SSH authentication success | 3 consecutive failures → `status=offline`, stop dispatching, notify alert |
| **L2: System Resources** | 60s | 1 | CPU usage, memory available, disk available | Disk < 5GB → `status=degraded`; memory < 10% → alert |
| **L3: GPU Status** | 5min | 1 | nvidia-smi memory/utilization (only `gpu`-labeled machines) | Full GPU memory usage → stop dispatching GPU tickets to this machine |
| **L4: Agent Environment** | 30min | 1 | Agent CLI installation status, version, whether logged in | Update corresponding Provider availability; set Machine to `degraded` if needed, but not directly to `offline` |
| **L5: Full Audit** | 6h / manual | 1 | Git credentials, gh CLI, git config, egress network (curl test) | Record report only, do not automatically change status |

#### 25.7.1 Machine State Machine

The Machine state machine is centered on **infrastructure-level failures**, not Provider execution ability:

```text
maintenance --(operator resume + next successful L1)--> online
maintenance --(operator resume + failed L1)-----------> offline

online --(3 consecutive L1 failures)---------------------------> offline
online --(L2 triggers scheduling-blocking resource issue)------------> degraded
online --(operator set maintenance)--------------------> maintenance

degraded --(3 consecutive L1 failures)-------------------------> offline
degraded --(L2/L3 recovered to safe range)---------------------> online
degraded --(operator set maintenance)------------------> maintenance

offline --(next successful L1 and no scheduling-blocking issue)---------> online
offline --(operator set maintenance)-------------------> maintenance
```

**State sources:**

- The only automatic source of `offline` is 3 consecutive L1 failures
- The only automatic source of `degraded` is “machine still reachable, but resource/environment has issues”
- The only source of `maintenance` is manual operation

**Special constraints:**

- L4/L5 failures default the Machine to `degraded`, not `offline`
- The `local` machine can be marked `degraded`, but should not skip L4/L5 semantic checks just because it is “local and does not require SSH”

#### 25.7.2 Provider Availability State Machine

Provider availability is a **derived result of Machine Monitor L4**, not a config-time input:

```text
unknown --(first successful L4 and all conditions met)---------------> available
unknown --(first successful L4 but conditions not met)-----------------> unavailable

available --(bound machine != online)-----------------> unavailable
available --(L4 explicitly detects cli_missing/not_logged_in/...)--> unavailable
available --(L4 snapshot expired)----------------------------> stale

unavailable --(later successful L4 and all conditions met)-----------> available
unavailable --(L4 snapshot expired)--------------------------> stale

stale --(later successful L4 and all conditions met)-----------------> available
stale --(later successful L4 but conditions not met)-------------------> unavailable
```

**Single source of truth for `provider.available = true`:**

- Most recent trusted L4 Agent Environment check
- And the bound Machine’s real-time `status`

The following signals **cannot individually imply** `provider.available = true`:

- `cli_command` is non-empty
- command name appears in PATH
- machine is still `online`
- a previous run had previously succeeded

**L4 determination must at least cover:**

- installed CLI for the corresponding adapter
- readable version
- authentication state is ready
- remote workspace / required env vars / startup path requirements are met

**Scheduling semantics:**

- `availability_state == available`: scheduling allowed
- `unknown / unavailable / stale`: scheduling forbidden

#### 25.7.3 Frontend Display and API Contract

The frontend and API must separate Machine status from Provider availability:

- Machine display: `Online / Degraded / Offline / Maintenance`
- Provider display: `Available / Unavailable / Unknown / Stale`
- Do not render Machine `online` as Provider “available”
- Do not render `cli_command` being on PATH as Provider “available”

Provider list/detail APIs should return:

- `availability_state`
- `available`
- `availability_checked_at`
- `availability_reason`

Where `available` is only a convenience boolean field; the frontend should prefer `availability_state`.

**L4 detailed check script (single SSH command):**

```bash
# Execute through one SSH session to minimize connection overhead
echo '{'

# Claude Code
echo '"claude_code":{'
if command -v claude >/dev/null 2>&1; then
  echo '"installed":true,'
  echo '"version":"'$(claude --version 2>/dev/null || echo "unknown")'",'
  echo '"auth_status":"'$(claude auth status --text 2>/dev/null | grep -q "Logged in" && echo "logged_in" || echo "not_logged_in")'"'
else
  echo '"installed":false'
fi
echo '},'

# Codex
echo '"codex":{'
if command -v codex >/dev/null 2>&1; then
  echo '"installed":true,'
  echo '"version":"'$(codex --version 2>/dev/null || echo "unknown")'"'
else
  echo '"installed":false'
fi
echo '},'

# Gemini CLI
echo '"gemini":{'
if command -v gemini >/dev/null 2>&1; then
  echo '"installed":true'
else
  echo '"installed":false'
fi
echo '}'

echo '}'
```

**L5 full audit (on-demand or every 6 hours):**

```bash
echo '{'

# Git configuration
echo '"git":{'
echo '"installed":'$(command -v git >/dev/null && echo true || echo false)','
echo '"user_name":"'$(git config --global user.name 2>/dev/null)'",'
echo '"user_email":"'$(git config --global user.email 2>/dev/null)'"'
echo '},'

# GitHub CLI
echo '"gh_cli":{'
if command -v gh >/dev/null 2>&1; then
  echo '"installed":true,'
  echo '"auth_status":"'$(gh auth status 2>&1 | grep -q "Logged in" && echo "logged_in" || echo "not_logged_in")'"'
else
  echo '"installed":false'
fi
echo '},'

# Network egress test
echo '"network":{'
echo '"github_reachable":'$(curl -s --max-time 5 https://api.github.com >/dev/null && echo true || echo false)','
echo '"pypi_reachable":'$(curl -s --max-time 5 https://pypi.org >/dev/null && echo true || echo false)','
echo '"npm_reachable":'$(curl -s --max-time 5 https://registry.npmjs.org >/dev/null && echo true || echo false)
echo '}'

echo '}'
```

**Additional notes:**

- `gh_cli.auth_status` only indicates GitHub CLI login state on the machine; it is observability data, not the source of truth for platform GitHub outbound authentication.
- L5 audit must also output a separate `github_token_probe`:
  - `configured`: whether platform-hosted `GH_TOKEN` is configured
  - `valid`: whether the token is valid
  - `permissions`: parsed permission snapshot / scope
  - `repo_access`: access probe result for target repositories
  - `checked_at`: most recent probe time
- UI and scheduler should display both `gh_cli.auth_status` and `github_token_probe`; the latter is the real determinant for GitHub clone / issue / PR availability.

### Environment Provisioner — fix machines with an Agent

When L4/L5 checks find environment issues (such as Claude Code not installed or missing Git credentials), an **Environment Provisioner Agent** can automatically repair them. This Agent connects to the target machine via SSH and executes predefined environment-setup skills:

```yaml
# Built-in Harness: roles/env-provisioner.md
---
status:
  pickup: "Environment repair"
  finish: "Environment ready"
skills:
  - openase-platform
  - install-claude-code     # Skill to install Claude Code
  - install-codex           # Skill to install Codex
  - setup-git               # Skill to configure Git credentials
  - setup-gh-cli            # Skill to install and configure GitHub CLI
---

# Environment Provisioner

You are responsible for configuring the Agent runtime environment on remote machines.

## Target machine

{{ machine.name }} ({{ machine.host }})

## Detected issues

{{ ticket.description }}

## Available fix skills

- install-claude-code: install and log in to Claude Code
- install-codex: install Codex CLI
- setup-git: configure git user.name / user.email + credentials
- setup-gh-cli: install gh CLI and authenticate

Please repair the environment using the corresponding skill for each detected issue.
```

Machine Monitor detects environment issues → automatically create an “environment repair” ticket (bound to the target machine) → Environment Provisioner Agent takes over → SSH into target machine to run skills → repair complete → next L4 check verifies.

This Agent is also connected to global Ephemeral Chat (Chapter 31), so users can say in the UI for a machine, “Please install Claude Code for me,” and the AI assistant directly triggers Environment Provisioner skills to handle it.

### 25.8 Agent Cross-Machine Access

During execution, an Agent may need to access other machines (for example, copy training results from a GPU machine to a storage machine). This is implemented by injecting machine information into the Harness Prompt:

New Harness template variables:

```
{{ machine.name }}           — current execution machine name
{{ machine.host }}           — current execution machine address
{{ machine.description }}    — current execution machine description

{{ range .AccessibleMachines }}
- {{ repo.name }} ({{ machine.host }}): {{ machine.description }}
  labels: {{ repo.labels | join(", ") }}
  SSH: ssh {{ machine.ssh_user }}@{{ machine.host }}
{{ end }}
```

Rendered snippet the Agent sees in Prompt:

```
## Execution environment

Current machine: gpu-01 (10.0.1.10)
  Description: NVIDIA A100 × 4, 256GB RAM, CUDA 12.2, used for model training
  Workspace: /home/openase/.openase/workspace/acme/research/ASE-42/

## Other accessible machines

- storage (10.0.1.20): Data storage server, 16TB NVMe, NFS share /data
  SSH: ssh openase@10.0.1.20
- dev-01 (10.0.1.30): General development machine, 64GB RAM
  SSH: ssh openase@10.0.1.30

You can use SSH to access the above machines for file transfer or command execution.
```

**Security boundary**: Machines the Agent can SSH to are determined by the `project.accessible_machines` configuration (allowlist). Machines not in the allowlist will not have their information injected by Harness, and the Agent will not have matching SSH keys.

Project table adds a field:

| Field | Type | Description |
|------|------|------|
| accessible_machine_ids | UUID[] | Whitelisted list of machine IDs the Agent can access |

### 25.9 SSH Connection Pool

Frequent connection setup/teardown overhead for SSH is high. The orchestrator maintains an SSH pool:

```go
// infra/ssh/pool.go
type Pool struct {
    mu    sync.Mutex
    conns map[string]*ssh.Client  // key: machine_id
    cfg   map[string]SSHConfig    // connection config
}

func (p *Pool) Get(ctx context.Context, m *machine.Machine) (*ssh.Client, error) {
    p.mu.Lock()
    defer p.mu.Unlock()

    // Reuse an existing connection
    if client, ok := p.conns[m.ID]; ok {
        // Check if the connection is still alive
        _, _, err := client.SendRequest("keepalive@openase", true, nil)
        if err == nil {
            return client, nil
        }
        // Connection is dead, remove it
        client.Close()
        delete(p.conns, m.ID)
    }

    // Create a new connection
    key, _ := os.ReadFile(filepath.Join("~/.openase", m.SSHKeyPath))
    signer, _ := ssh.ParsePrivateKey(key)
    client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", m.Host, m.Port), &ssh.ClientConfig{
        User:            m.SSHUser,
        Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
        HostKeyCallback: ssh.InsecureIgnoreHostKey(),  // TODO: use known_hosts in production
        Timeout:         10 * time.Second,
    })
    if err != nil {
        return nil, err
    }
    p.conns[m.ID] = client
    return client, nil
}
```

### 25.10 Product-level onboarding / machine management in Web UI

After OpenASE starts and enters the Web UI, a machine management page can be provided in settings (Settings → Machines):

```
┌─────────────────────────────────────────────┐
│ Machines                                  [Add machine]  │
├─────────────────────────────────────────────┤
│ ● local                                    │
│   Self · Online · CPU 32 cores 45% · Memory 32GB   │
│                                             │
│ ● gpu-01                                   │
│   10.0.1.10 · Online · GPU A100×4 · Memory 256GB  │
│   Labels: gpu, a100, cuda-12                   │
│                                             │
│ ○ gpu-02                                   │
│   10.0.1.11 · Offline (last online: 2 hours ago) │
│   Labels: gpu, h100                            │
│                                             │
│ ● storage                                  │
│   10.0.1.20 · Online · Disk 16TB (12TB available)    │
│   Labels: storage, nfs                         │
└─────────────────────────────────────────────┘
```

Flow for adding a new machine:

1. Fill in Host, SSH User, upload or specify SSH Key path
2. Click “Test connection” → OpenASE SSHes to run `whoami && uname -a` to verify reachability
3. Automatically collect resource information (CPU, memory, GPU, installed Agent CLI)
4. User fills in Name, Description, Labels
5. Save → SSH key saved to `~/.openase/keys/` (permissions 0600)

### 25.11 CLI Commands

```bash
openase machine list                     # list all machines and status
openase machine add gpu-01 \
  --host 10.0.1.10 \
  --user openase \
  --key ~/.ssh/gpu-01.pem \
  --labels gpu,a100,cuda-12 \
  --description "Training server, A100×4"  # add machine
openase machine test gpu-01              # test SSH connectivity + collect resources
openase machine remove gpu-01            # remove machine
openase machine status                   # live resource status for all machines
openase provider add codex-gpu01 \
  --machine gpu-01 \
  --adapter codex-app-server \
  --cmd /usr/local/bin/codex             # register a Provider on gpu-01

openase agent create training-codex \
  --project research \
  --provider codex-gpu01                 # create Agent definition bound to that Provider

openase workflow update training \
  --agent training-codex                 # Workflow binds to that Agent, machine is then determined
```

### 25.12 Impact on Existing Architecture

**Minimal impact**—multi-machine support is additive and does not change single-machine behavior:

| Component | Impact |
|------|------|
| Domain / Core Types | Add machine-related types, parsing, and pure logic in `internal/domain/catalog` |
| Service / Use-Case | `internal/service/catalog` adds orchestration and probing logic for Machine binding Providers; `internal/ticket` no longer requires manual machine selection |
| Orchestrator | `internal/orchestrator` adds `resolveExecutionTarget`, `runOnRemote`, `MachineMonitor` |
| Infrastructure | Add `internal/infra/ssh/` (connection pool + command execution wrapper) |
| Adapter layer | **Unchanged**. In remote execution, adapter is unaware of Machine; it only sees stdin/stdout pipe, local is os/exec, remote is SSH session |
| Interface / Entry | `internal/httpapi` and Web UI add Machine CRUD, plus machine-aware configuration entry for Provider/Agent selection |
| Hook | lifecycle Hook runs on remote machine (script executed in SSH session) |
| Database | Add `machines` table; add `machine_id` to `agent_providers`; redundantly record `machine_id` in `agent_runs` for audit; add `accessible_machine_ids` to `projects` |

**When only `local` exists (default), user experience is exactly the same as if multi-machine did not exist.** But to provide a ground truth source for Provider availability, the system still runs a local Machine Monitor:

- no SSH connection pool needed
- L1 uses local reachability semantics
- L2-L5 use local shell/exec collection
- Scheduler still checks both `machine.status` and `provider.availability_state`

### 25.13 New API Endpoints

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/machines` | list machines (with live resource state) |
| POST | `/api/v1/orgs/:orgId/machines` | add machine |
| GET | `/api/v1/machines/:machineId` | machine detail |
| PATCH | `/api/v1/machines/:machineId` | update machine config |
| DELETE | `/api/v1/machines/:machineId` | remove machine |
| POST | `/api/v1/machines/:machineId/test` | test SSH connectivity + collect resources |
| GET | `/api/v1/machines/:machineId/resources` | fetch latest resource snapshot |

SSE adds event types: `machine.online`, `machine.offline`, `machine.degraded`, `machine.resources_updated`, `provider.available`, `provider.unavailable`, `provider.stale`

### 25.14 New observability Metrics

| Metric name | Type | Labels | Description |
|--------|------|------|------|
| `openase.machine.status` | Gauge | machine_name | Machine status (1=online, 0=offline, -1=degraded) |
| `openase.machine.cpu_usage` | Gauge | machine_name | CPU usage |
| `openase.machine.memory_available_gb` | Gauge | machine_name | Available memory |
| `openase.machine.gpu_utilization` | Gauge | machine_name, gpu_index | GPU utilization |
| `openase.machine.ssh_latency_ms` | Histogram | machine_name | SSH connection latency |
| `openase.machine.ssh_errors_total` | Counter | machine_name | Number of SSH connection errors |
| `openase.provider.availability` | Gauge | provider_name, machine_name, adapter_type | Provider availability (1=available, 0=unknown/stale, -1=unavailable) |
| `openase.provider.l4_age_seconds` | Gauge | provider_name, machine_name | Age since the most recent L4 availability check |
| `openase.ticket.machine_dispatch` | Counter | machine_name, workflow_type | Number of ticket dispatches to each machine |

---
## Chapter 26 Agent Role System and Intelligent Recruitment

### 26.1 Core Concept: Harness = role JD

In OpenASE, a Harness is not just a “work specification”—it is a **complete role definition**, analogous to a job description (JD) in HR. A Harness defines:

- **Who this role is** (identity, scope of responsibility)
- **How this role works** (workflow, methodology)
- **The role’s delivery standards** (acceptance outcomes, quality requirements)
- **What resources this role needs** (machine labels, tool permissions)

Different Harnesses are different “occupations.” The same Agent CLI (for example, Claude Code) can become different roles by mounting different Harnesses—like one person wearing different uniforms and holding different tool manuals can perform different jobs.

### 26.2 Built-in Role Library (Harness Marketplace)

OpenASE includes a built-in role library that provides ready-to-use Harness templates. Users can use them directly or fork and customize them.

**Software Engineering Roles:**

| Role | Harness name | Typical ticket | Core prompt characteristics |
|------|-------------|---------|----------------|
| Full-Stack Developer | `roles/fullstack-developer.md` | Feature development, bug fixes | Read requirements → write code → write tests → open PR |
| Front-end Engineer | `roles/frontend-engineer.md` | UI components, page development | Focus on accessibility, responsiveness, component reuse |
| Back-end Engineer | `roles/backend-engineer.md` | API development, data modeling | Focus on performance, security, data consistency |
| QA Engineer | `roles/qa-engineer.md` | Write test cases, regression testing | Analyze code paths → write unit/integration/E2E tests → coverage report |
| DevOps Engineer | `roles/devops-engineer.md` | CI/CD, deployment, infrastructure | Write Dockerfiles, configure pipelines, run deployments |
| Security Engineer | `roles/security-engineer.md` | Security audit, vulnerability fixes | Scan vulnerabilities → write PoC → generate report → create remediation ticket |
| Technical Writer | `roles/technical-writer.md` | API docs, user guides | Read code changes → compare existing docs → update docs → open PR |
| Code Reviewer | `roles/code-reviewer.md` | PR review, code quality | Review PR → check style/performance/security → submit change request or approve |

**Product and Research Roles:**

| Role | Harness name | Typical ticket | Core prompt characteristics |
|------|-------------|---------|----------------|
| Product Manager | `roles/product-manager.md` | Requirements analysis, PRD writing | Research market → analyze competitors → write requirement document → split into sub tickets |
| Market Analyst | `roles/market-analyst.md` | Competitive analysis, industry trends | Search industry reports → analyze competitor features → generate research report |
| Research Idea Miner | `roles/research-ideation.md` | Paper research, direction exploration | Retrieve latest papers → analyze research trends → propose experimental hypotheses → output idea report |
| Experiment Runner | `roles/experiment-runner.md` | Experiment design, code implementation | Read idea report → design experiment → write experiment code → run → record results |
| Report Writer | `roles/report-writer.md` | Experiment reports, first draft of papers | Read experiment results → organize structure → write report/paper → generate charts |
| Data Analyst | `roles/data-analyst.md` | Data cleaning, visualization | Read datasets → clean → statistical analysis → generate visualization report |

**Role Harness example — Research Idea Mining:**

````markdown
---
agent:
  max_turns: 30
  timeout_minutes: 90
  max_budget_usd: 10.00
hooks:
  on_complete:
    - cmd: "test -f idea-report.md"  # Must produce a report file
      on_failure: block
execution_target:
  agent_binding_decides_machine: true
---

# Research Idea Mining

You are a professional research assistant skilled in literature review and exploration of research directions.

## ticket Context

- ticket: {{ ticket.identifier }} — {{ ticket.title }}
- Research area: {{ ticket.description }}

{{ range .ExternalLinks }}
- References/resources: [{{ link.title }}]({{ link.url }})
{{ end }}

## Workflow

1. **Domain understanding**: read the ticket description, understand the research direction and constraints
2. **Literature search**:
   - Search for related papers from the last 12 months (arXiv, Google Scholar, Semantic Scholar)
   - Focus on high-citation, top conference/journal papers
   - Record each paper’s core contributions and limitations
3. **Trend analysis**:
   - Identify 3-5 active sub-directions in this field
   - Analyze which directions are rising and which are saturated
4. **Gap identification**:
   - Find unsolved problems in existing work
   - Identify opportunities for cross-innovation between different directions
5. **Idea generation**:
   - Propose 3-5 concrete feasible research ideas
   - Each idea includes: hypothesis, method overview, expected contribution, preliminary experimental design, required resources
6. **Output report**:
   - Generate `idea-report.md` with a complete literature review + idea list
   - Add feasibility score (1-5) and novelty score (1-5) for each idea

## Acceptance criteria

- Cite at least 15 relevant papers
- Propose at least 3 feasible research ideas
- Each idea has a clear hypothesis and experimental design
- Report structure is clear and can be used directly as lab meeting material
````

### 26.3 Composition and collaboration of roles

The real power of roles lies in composition. A project can “hire” multiple roles at the same time, and collaborate through ticket dependencies:

**Scenario 1: MVP development team**

```
Product Manager ──(outputs PRD)──→ Full-Stack Developer ──(submits PR)──→ Code Reviewer
                           │                               │
                           ├──(outputs code)──→ QA Engineer────→ review passed
                           │
                           └──(outputs code)──→ Technical Writer ──→ Docs PR
```

Corresponding tickets:

```
ASE-1: [Product Manager] Write user registration PRD
ASE-2: [Full-Stack Developer] Implement user registration feature (blocks: ASE-1)
ASE-3: [QA Engineer] Write user registration test cases (blocks: ASE-2)
ASE-4: [Technical Writer] Update API documentation (blocks: ASE-2)
ASE-5: [Code Reviewer] Review user registration PR (blocks: ASE-2)
```

**Scenario 2: Research project**

```
Idea Mining ──(outputs idea report)──→ Experiment Runner ──(outputs experiment results)──→ Report Writer
     │                                   │
     └──→ Market Analyst                  └──→ Data Analyst
          (industry context)                        (data visualization)
```

**Scenario 3: Role evolution from MVP to continuous iteration**

| Stage | Required roles | Non-required roles |
|------|-----------|------------|
| Idea → POC | Research idea mining, Full-Stack Developer | — |
| POC → MVP | Full-Stack Developer, QA Engineer | Research idea mining (task completed) |
| MVP → Launch | DevOps Engineer, Security Engineer | — |
| Continuous iteration | Full-Stack Developer, QA Engineer, Code Reviewer, Technical Writer | DevOps (as needed) |
| Expansion stage | + Market Analyst, Product Manager | — |

### 26.4 Agent recruitment recommender (HR Advisor)

OpenASE has a built-in “HR Advisor” that recommends which roles should be “recruited” based on current project status.

**Inputs:**

- Project description, status (`Backlog` / `Planned` / `In Progress` / `Completed` / `Canceled` / `Archived`)
- Distribution of existing tickets and workflow families (such as `coding` / `test` / `docs` / `deploy` / `planning`)
- Distribution of ticket statuses and queue pressure in each status (for example `Backlog`, `To Test`, `To Doc`)
- Existing role/workflow lists, plus each workflow’s raw type label, derived family, and `pickup` / `finish` status binding
- Recent activity trends (merge rate dropping? low test coverage? outdated docs?)

**Outputs:**

- List of recommended roles and reasons
- Corresponding Harness template for each role (one-click activation)
- Recommended workforce allocation (number of developers + number of QA + number of docs)
- Each recommendation includes the suggested user-visible workflow type label and internal workflow family

**Recommendation coverage contract:**

- The HR Advisor must maintain an explicit “role recommendation support matrix” that distinguishes:
  - `supported_now`
  - `intentionally_unsupported`
  - `planned_not_yet_implemented`
- This matrix must cover all built-in roles and cannot omit roles implicitly by “missing rules.”
- The following roles should be automatically recommended at minimum: `dispatcher`, `fullstack-developer`, `frontend-engineer`, `backend-engineer`, `qa-engineer`, `devops-engineer`, `security-engineer`, `technical-writer`, `code-reviewer`, `product-manager`, `research-ideation`, `experiment-runner`, `report-writer`, `env-provisioner`, `harness-optimizer`.
- `market-analyst` may be explicitly marked as `intentionally_unsupported` because it depends on external market signals rather than internal project execution signals.
- `data-analyst` may be explicitly marked as `planned_not_yet_implemented` until snapshot provides dataset / metric / analysis artifact signals.
- Recommendation rationale must be tied to observability signals, such as status stage, lane queue pressure, workflow pickup/finish binding, failure retries, documentation drift, and research process stage.
- The HR Advisor must not directly depend on raw string equality checks like `workflow.type == "coding"`; it must derive families through an explicit rule classifier based on role slug, type label aliases, workflow name, status semantics, and skill/Harness signals.
- When multiple independent lane gaps correspond to the same role family, HR Advisor must keep multiple recommendations instead of collapsing them into a single `RoleSlug`.

**Example recommendation logic:**

```go
func (h *HRAdvisor) Recommend(ctx context.Context, project *project.Project) []RoleRecommendation {
    var recs []RoleRecommendation
    stats := h.getProjectStats(ctx, project.ID)

    // Rule 1: There are coding tickets but no test tickets -> recommend QA engineer.
    if stats.CodingTickets > 5 && stats.TestTickets == 0 {
        recs = append(recs, RoleRecommendation{
            Role:     "qa-engineer",
            Reason:   fmt.Sprintf("The project already has %d coding tickets, but there are no test tickets yet. Recommend recruiting a QA engineer to ensure code quality.", stats.CodingTickets),
            Priority: "high",
        })
    }

    // Rule 2: PRs were merged but docs were not updated -> recommend technical writer.
    if stats.MergedPRsWithoutDocUpdate > 3 {
        recs = append(recs, RoleRecommendation{
            Role:     "technical-writer",
            Reason:   fmt.Sprintf("The documentation was not synchronized after the last %d merged PRs. Recommend recruiting a technical writer.", stats.MergedPRsWithoutDocUpdate),
            Priority: "medium",
        })
    }

    // Rule 3: Project is in In Progress but still has no Agent -> recommend full-stack developer.
    if project.Status == "In Progress" && stats.TotalAgents == 0 {
        recs = append(recs, RoleRecommendation{
            Role:   "fullstack-developer",
            Reason: "The project is already in In Progress but has no Agent yet. Recommend recruiting a full-stack developer to start implementation.",
        })
    }

    // Rule 4: Security-related issues exist but no security workflow -> recommend security engineer.
    if stats.SecurityRelatedIssues > 0 && !stats.HasSecurityWorkflow {
        recs = append(recs, RoleRecommendation{
            Role:   "security-engineer",
            Reason: fmt.Sprintf("Found %d security-related Issues, but the project has not configured a security workflow.", stats.SecurityRelatedIssues),
        })
    }

    // Rule 5: Research label -> recommend research roles.
    if project.HasLabel("research") {
        if stats.TotalTickets == 0 {
            recs = append(recs, RoleRecommendation{
                Role:   "research-ideation",
                Reason: "The research project has just started; recommend recruiting the Idea Mining role first for literature review and direction exploration.",
            })
        }
    }

    // Rule 6: A status lane is congested but no workflow picks up the lane -> recommend filling the corresponding lane.
    if stats.StatusQueue["To Test"] >= 2 && !stats.HasPickupWorkflow("To Test") {
        recs = append(recs, RoleRecommendation{
            Role:   "qa-engineer",
            Reason: fmt.Sprintf("There are %d tickets in the To Test queue, but no workflow picks up this lane. Recommend adding a QA workflow to handle the To Test lane.", stats.StatusQueue["To Test"]),
        })
    }

    return recs
}
```

Additional implementation constraints:

- Recommendation logic should prioritize stable semantics: project status type, ticket status `stage`, workflow binding relationships; only fall back to status-name keywords when distinguishing specific lane capability is needed.
- Automatic recommendations for `frontend-engineer` / `backend-engineer` / `devops-engineer` / `code-reviewer` should come from explicit lane pressure, not by collapsing all implementation needs into `fullstack-developer`.
- Automatic recommendations for `env-provisioner` and `harness-optimizer` should be derived from execution degradation signals such as “retry pause / failure burst / workflow stall,” not from requiring users to discover problems manually first.

**Presentation in Web UI:**

```
┌─────────────────────────────────────────────────────────┐
│ 🤖 Agent Recruitment Recommendations   [View All]      │
├─────────────────────────────────────────────────────────┤
│                                                         │
│ 🔴 High Priority                                        │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ QA Engineer (qa-engineer)                          │ │
│ │ The project has 12 completed coding tickets, but no   │ │
│ │ testing tickets yet.                                  │ │
│ │ Recommend recruiting QA engineer to ensure code quality.│ │
│ │                                                     │ │
│ │ [View Role Harness]  [Activate with One Click]  [Later]│ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
│ 🟡 Medium Priority                                      │
│                                                         │
│ ┌─────────────────────────────────────────────────────┐ │
│ │ Technical Writer (technical-writer)                  │ │
│ │ Documentation was not updated after the last 5 PRs     │ │
│ │ merged.                                              │ │
│ │                                                     │ │
│ │ [View Role Harness]  [Activate with One Click]  [Later]│ │
│ └─────────────────────────────────────────────────────┘ │
│                                                         │
└─────────────────────────────────────────────────────────┘
```

**What “activate with one click” does:**

1. Copies the Harness template from the role library into the project workflow control plane store
2. Creates the corresponding workflow record (linking to the new Harness version)
3. Registers a new Agent definition and binds this workflow to that Agent
4. Optional: automatically creates the first ticket (for example, QA engineer automatically creates a “write test cases for existing code” ticket)

### 26.5 Custom roles

Users can create their own roles—the essence is writing a Harness document version. Web UI provides a role editor:

```
┌──────────────────────────────────────────────────┐
│ Create New Role                                 │
├──────────────────────────────────────────────────┤
│ Role name: [Database DBA                        ]│
│ Description: [Handles database performance       ]│
│            [optimization, migration, backups]    │
│ Tags:      [database] [postgresql] [performance]│
│ Machine requirements: [high-memory]             │
│                                                   │
│ Harness editor:                                  │
│ ┌───────────────────────────────────────────────┐ │
│ │ ---                                           │ │
│ │ agent:                                        │ │
│ │   max_turns: 15                               │ │
│ │ hooks:                                        │ │
│ │   on_complete:                                │ │
│ │     - cmd: "pg_dump --version"                │ │
│ │ ---                                           │ │
│ │                                               │ │
│ │ # Database DBA                               │ │
│ │                                               │ │
│ │ You are a database administration expert...    │ │
│ └───────────────────────────────────────────────┘ │
│                                                   │
│                    [Save to Project]  [Publish to Role Library] │
└──────────────────────────────────────────────────┘
```

“Publish to role library” lets users share their roles with the community (similar to the GitHub Actions Marketplace model).

### 26.6 Mapping to existing concepts

This feature does not require new entities—it is a **semantic upgrade** of existing concepts:

| Existing concept | Semantic in role system | Entity change |
|---------|----------------|---------|
| Workflow | The role’s “job definition” | Adds `role_name` field (such as `fullstack-developer`) |
| Harness | The role’s “job manual” (JD + SOP) | Unchanged, content is richer |
| Agent | The role’s “assigned employee” | Unchanged |
| Ticket | The role’s “work item” | Unchanged |
| ScheduledJob | The role’s “periodic responsibility” | Unchanged |
| on_complete Hook | The role’s “acceptance criteria” | Unchanged |

New fields added to the Workflow table:

| Field | Type | Description |
|------|------|------|
| role_name | String | Role identifier (such as `fullstack-developer`, `qa-engineer`), used for recommendation engine matching |
| role_description | Text | Role summary (one-line description displayed in UI) |
| role_icon | String | Role icon identifier (such as `code`, `shield`, `flask`, corresponding to Lucide icon names) |
| is_builtin | Boolean | Whether it is a built-in role (built-in roles cannot be deleted but can be forked and customized) |

### 26.7 Impact on PRD core narrative

The role system upgrades OpenASE from a “ticket automation platform” to an **“AI engineering team management platform.”** Users are no longer “assigning tasks to an Agent,” but “assembling a virtual team and assigning responsibilities to different roles.”

Benefits of this conceptual shift:

- **Lower cognitive overhead**: users do not need to understand Workflow, Harness, and Hook technical concepts; they only need to think “what roles does my project need”
- **Natural expansion path**: projects only need developers at early stages, then gradually recruit QA, security, docs, product manager, and other roles as the project grows
- **Anchor for community ecosystem**: the role library is the most natural shared community content—“This is my ML paper reviewer role, and it works very well”

---
## Chapter 27 Agent Autonomous Closed Loop (Platform API for Agents)

### 27.1 From "Passive Execution" to "Proactive Operation"

In the current design, Agents are purely executors—receiving tickets, writing code, and opening PRs. But real software engineering is far more than writing code: a senior engineer will discover during development that new repositories need to be split, follow-up tickets need to be created, project documentation needs to be updated, and CI scheduled jobs need to be adjusted. If an Agent cannot do these things, it is always just a "senior typist".

**The core idea of a closed loop is that Agents can not only consume platform tickets, but also operate the platform itself in reverse.** OpenASE’s REST API treats Agents and humans equally. An Agent uses the Platform API Token injected in the workspace to call the OpenASE API and perform controlled platform operations.

```
                    ┌─────────────────────────────┐
                    │         OpenASE Platform      │
                    │                               │
        Create ticket ──→ │  Ticket ──→ Agent executes ──→ │ ──→ PR
        Configure project ──→ │     ↑                    │   │
        Manage Repo ─────→│     │    Agent reverse-call │   │
                    │     │    Platform API      │   │
                    │     └────────────────────┘   │
                    │                               │
                    │  Create sub-ticket, register new │
                    │  Repo, update project description,│
                    │  configure scheduled jobs       │
                    └─────────────────────────────┘
```

### 27.2 Agent Platform API (Controlled Autonomy Capability)

Agents in a workspace can access Platform API through `openase` CLI or direct HTTP calls. The orchestrator injects these environment variables when starting an Agent:

| Environment Variable | Value | Description |
|---------|-----|------|
| `OPENASE_API_URL` | `http://localhost:19836/api/v1` | Platform API endpoint |
| `OPENASE_AGENT_TOKEN` | `ase_agent_xxx...` | Agent-specific short-lived Token (automatically expires after ticket completion) |
| `OPENASE_PROJECT_ID` | UUID | Current project ID |
| `OPENASE_TICKET_ID` | UUID | Current ticket ID |

Platform instructions visible in the Harness Prompt:

```
## Platform Operation Capabilities

You can operate the platform via openase CLI (preinstalled in the workspace):

  openase ticket create --title "..." --workflow coding    # create a sub-ticket
  openase ticket link ASE-42 --url https://github.com/... # link external Issue
  openase project update --description "..."               # update project description
  openase project add-repo --name "new-service" --url "..." # register new repository
  openase scheduled-job create --name "..." --cron "..."    # create a scheduled job
  openase project update-status "In Progress"              # update project status

Or via HTTP API:
  curl -H "Authorization: Bearer $OPENASE_AGENT_TOKEN" \
       $OPENASE_API_URL/projects/$OPENASE_PROJECT_ID/tickets \
       -d '{"title": "...", "workflow_id": "..."}'

All operations are recorded in the activity log and marked as initiated by you (Agent).
```

#### 27.2.1 CLI / HTTP Isomorphic Principle (adopt `gh` style, instead of introducing GraphQL first)

OpenASE CLI should not become another semantic source. **HTTP + OpenAPI is the only source of truth; CLI is just an endpoint projection of the same contract.**

Design goals:

1. **Reachability first**: Any exposed HTTP API must have a corresponding CLI path. There should be no capability gap where "HTTP exists but CLI does not."
2. **Isomorphism over sugar coating**: CLI parameter names, request field names, and return field names should align directly with the HTTP contract as much as possible; user experience improvements can be layered on top but cannot replace the real interface shape.
3. **OpenAPI single source of truth**: After updating an HTTP handler, update `api/openapi.json` first; CLI typed commands, help text, and parameter mappings should be generated from or validated against OpenAPI as much as possible.
4. **Clear authentication layering**:
   - Control plane CLI: for human operators, using user-mode authentication.
   - Agent Platform CLI: for Agents in workspace, default reads `OPENASE_AGENT_TOKEN`, capability constrained by Harness scope.
5. **Model streaming interfaces separately**: SSE / chat / watch / stream endpoints should not be forced into CRUD resource commands; they should be exposed as separate watch/stream/chat subcommands.

**Conclusion: adopt a two-layer GitHub CLI-like structure, not introducing GraphQL first.**

- The first layer is raw passthrough, similar to `gh api`
- The second layer is typed resource commands, similar to `gh issue` / `gh pr`

GraphQL is not a prerequisite at this stage. OpenASE’s current real interface is REST + OpenAPI, so the top priority is to keep CLI isomorphic with the REST contract, instead of introducing a parallel GraphQL schema truth source.

#### 27.2.2 CLI Layered Design

CLI must be split into two layers:

**A. Raw API Layer (guaranteed 100% reachability)**

```bash
openase api GET /api/v1/tickets/$OPENASE_TICKET_ID
openase api POST /api/v1/projects/$OPENASE_PROJECT_ID/tickets \
  -f title="Add integration test coverage" \
  -f workflow_id="..."
openase api PATCH /api/v1/tickets/$OPENASE_TICKET_ID/comments/$COMMENT_ID \
  -f body="$(cat workpad.md)"
```

Requirements:

- `openase api` must allow specifying HTTP method + path, and support:
  - `-f key=value` to assemble JSON body
  - `--input <file>` to submit raw JSON body directly
  - `--query key=value` to append query string
  - `--header key:value` to append extra headers
- Output JSON as-is by default; allow `--jq`, `--json`, `--template` for post-processing.
- This layer is the fallback egress for all HTTP capabilities. Even if typed commands are not fully covered yet, as long as HTTP exists, CLI must remain reachable.

**B. Typed Resource Layer (high-frequency, usability, discoverability)**

```bash
openase ticket list
openase ticket get 550e8400-e29b-41d4-a716-446655440000
openase ticket comment list 550e8400-e29b-41d4-a716-446655440000
openase ticket comment update 550e8400-e29b-41d4-a716-446655440000 550e8400-e29b-41d4-a716-446655440001 --body-file /tmp/comment.md
openase workflow create ...
openase scheduled-job update ...
```

Requirements:

- Group by resource: `ticket`, `project`, `workflow`, `scheduled-job`, `machine`, `provider`, `agent`, `skill`, etc.
- For each typed command, parameter names should reuse HTTP field names as much as possible, e.g.:
  - `status_name`
  - `workflow_id`
  - `external_ref`
- Typed commands are only human-readable wrappers for raw API and must not build a private semantic layer that deviates from HTTP.
- Typed commands’ help text, required params, body schema, defaults, and error messages should preferably be generated from or validated against OpenAPI.

#### 27.2.2.1 CLI Help / Discoverability Constraints

CLI `--help` is not decoration but a discoverable projection of the interface contract. Requirements are:

1. **Use Cobra default help template** without creating a separate global renderer; what needs consistency is unified generation strategy for each command’s `Long` / `Example`.
2. **OpenAPI typed CRUD commands**:
   - `Long` should be generated by a unified builder;
   - At minimum, explain: command purpose, positional argument source, UUID semantics, and source constraints for flag/body/query;
   - `watch/stream` commands must also reuse the same generation logic, not only a single `Short`.
3. **Agent Platform commands**:
   - Do not reuse OpenAPI CRUD help builder;
   - Must use separate platform help builder to consistently explain fallback logic for `OPENASE_API_URL`, `OPENASE_AGENT_TOKEN`, `OPENASE_PROJECT_ID`, and `OPENASE_TICKET_ID`;
   - Must clarify the semantics of “current project / current ticket,” and priority between positional args, flags, and environment variables.
4. **High-risk commands must have explicit examples**. At minimum include:
   - `openase api ...`
   - `openase watch ...`
   - `ticket update`
   - `ticket report-usage`
   - `ticket comment update`
   - `project add-repo`
5. **Help text must explain ID semantics**:
   - Unless a command explicitly declares support for a human-readable identifier, `*Id` positional parameters and `OPENASE_*_ID` environment variables are interpreted as UUIDs;
   - Project-local identifiers like `ASE-2` must not be mixed into UUID-style parameters such as `ticketId`.
6. **Help text must reflect actual execution semantics**:
   - `watch/stream` must clearly state that the connection remains open until user interruption;
   - `report-usage` must clearly state it is incremental reporting, not overwrite of totals;
   - Rules combining `api --input` and fields must be visible in help, rather than failing only at runtime.

#### 27.2.3 `cmd/openase` Top-Level Namespace Constraints

Top-level `cmd/openase` commands must support all at once:

- **Control plane typed commands**: humans operating platform resources directly
- **Agent Platform typed commands**: Agents writing back current project/current ticket within workspace
- **General raw API**: `openase api ...`

Recommended structure:

```bash
openase api ...

openase ticket ...
openase project ...
openase workflow ...
openase scheduled-job ...
openase machine ...
openase provider ...
openase agent ...
openase skill ...

openase watch ...
openase stream ...
openase chat ...
```

Where:

- `openase api ...` is protocol-level egress and must remain generic.
- `openase ticket ...` and other typed commands are the high-frequency resource entry points.
- `watch/stream/chat` are streaming/session-style entry points and are not part of CRUD resource trees.

#### 27.2.4 Output and Script Friendliness

To make CLI suitable for both humans and Agent / shell / CI usage, requirements are unified:

- Default to JSON output and do not rename fields inconsistently with HTTP fields.
- Support:
  - `--jq '<expr>'`
  - `--json field1,field2,...`
  - `--template '<go-template-or-equivalent>'`
- For typed commands, `--json` only performs field trimming and must not rewrite original field semantics.
- Error output must preserve HTTP status, error code, and message for script-friendly judgment.
- CLI help must serve both humans and script authors: humans need fast understanding through `Long/Example`, script authors need explicit parameter source, environment variable fallbacks, and field constraints visible in help.

#### 27.2.5 Workpad / Comment CLI + Skill Layer Constraints

Ticket comments are not an edge capability; they are part of the Workflow closed loop. CLI must natively support:

```bash
openase ticket comment list
openase ticket comment create
openase ticket comment update
openase ticket comment delete
openase ticket comment revisions
```

Here, Workpad upsert is no longer exposed as a separate CLI subcommand; it is provided by the runtime-injected `openase-platform` skill helper, for example:

```bash
./.agent/skills/openase-platform/scripts/upsert_workpad.sh --body-file /tmp/workpad.md
```

This helper must implement idempotent upsert:

- Find the comment titled `## Workpad` under the current ticket
- If it exists, update that same comment
- If it does not exist, create that comment
- If body lacks the heading, automatically add `## Workpad`

CLI should retain only the underlying `comment list/create/update/...` HTTP isomorphic capability; Workpad semantics belongs to the skill-layer helper, not CLI surface.

#### 27.2.6 OpenAPI-Driven Implementation Constraints

The implementation should not long-term keep the pattern of “handwritten HTTP handler, handwritten CLI, both drifting apart.” The constraints are:

1. Once HTTP contract changes, OpenAPI must be updated first.
2. CLI typed commands must at least align:
   - parameters with OpenAPI schema
   - body fields with OpenAPI schema
   - routes and methods with OpenAPI
3. CI must add checks:
   - OpenAPI updated but CLI generation / snapshot not updated → fail
   - CLI references a non-existent route/field → fail
4. For newly added interfaces not yet covered by typed commands, `openase api` must be immediately available to avoid CLI lagging behind capability coverage.

### 27.3 Platform Operations Agents Can Execute

| Operation | API Endpoint | Typical Scenario | Permission Level |
|------|---------|---------|---------|
| **Create sub-ticket** | `POST /tickets` | Bug discovered during development → create bugfix ticket; security scan finds vulnerability → create remediation ticket | Allowed by default |
| **Link external reference** | `POST /tickets/:id/links` | Auto-link created PRs, related discovered issues | Allowed by default |
| **Update current ticket description** | `PATCH /tickets/:id` | Add contextual information discovered during execution | Allowed by default |
| **Maintain current ticket comments / Workpad** | `GET/POST/PATCH /tickets/:id/comments...` | Maintain `## Workpad`, record progress, blockers, and verification results | Allowed by default (current ticket only) |
| **Update project description** | `PATCH /projects/:id` | Update project README/description after product research | Requires authorization |
| **Update project status** | `POST /projects/:id/transition` | For example move project from `Planned` to `In Progress`, or mark `In Progress` as `Completed` | Requires authorization |
| **Register new repository** | `POST /projects/:id/repos` | Register new microservice repo to platform after architecture-driven creation | Requires authorization |
| **Create scheduled job** | `POST /scheduled-jobs` | DevOps role configures cron for automated deployment/security scans | Requires authorization |
| **Update scheduled job** | `PATCH /scheduled-jobs/:id` | Adjust cron frequency or ticket template | Requires authorization |
| **Create workflow** | `POST /workflows` | Create corresponding workflow when defining new role | Requires authorization |
| **List tickets** | `GET /tickets` | Understand project-wide progress to avoid duplication | Allowed by default |
| **Get machine status** | `GET /machines` | Choose the best machine for compute tasks | Allowed by default |

### 27.4 Permission Model: Harness-Level Scope Control

Not all Agents should have the same platform operation permissions. A coding Agent can create sub-tickets but should not change project status; a product manager role can update project description but should not create scheduled jobs.

Permissions are declared in Harness YAML frontmatter (whitelist mode):

```yaml
---
# Declare this role's platform operation permissions in Harness
platform_access:
  # Default: only creating sub-tickets and linking references is allowed (minimum permissions)
  allowed:
    - "tickets.create"        # create sub-tickets
    - "tickets.update.self"   # update current ticket
    - "tickets.link"          # link external references
    - "tickets.list"          # list tickets
    - "machines.list"         # query machine status

  # Requires explicit declaration in Harness to open:
  # - "projects.update"       # update project information
  # - "projects.add_repo"     # register new repository
  # - "projects.transition"   # update project status
  # - "scheduled_jobs.create" # create scheduled jobs
  # - "scheduled_jobs.update" # update scheduled jobs
  # - "workflows.create"      # create workflow
---
```

**A product manager Harness might be configured like this:**

```yaml
platform_access:
  allowed:
    - "tickets.create"
    - "tickets.update.self"
    - "tickets.link"
    - "tickets.list"
    - "projects.update"        # can update project description
    - "projects.transition"    # can advance project status
```

**DevOps role Harness:**

```yaml
platform_access:
  allowed:
    - "tickets.create"
    - "tickets.update.self"
    - "tickets.link"
    - "tickets.list"
    - "projects.add_repo"       # can register new repository
    - "scheduled_jobs.create"   # can create scheduled jobs
    - "scheduled_jobs.update"   # can modify scheduled jobs
    - "machines.list"
```

**The orchestrator generates a scoped Agent Token when starting an Agent based on Harness `platform_access`.** The API layer validates token scope, and unauthorized operations return 403 directly.

### 27.5 Typical Closed-Loop Scenarios

**Scenario One: Automatically split tickets during development**

```
[User] Creates ticket ASE-10: "Implement User System (Sign-up/Login/Permission Management)"

[Full-Stack Developer Agent] Claims ASE-10, and after analysis determines it is too large, splits it via Platform API:
  → openase ticket create --title "Implement user registration API" --parent ASE-10 --workflow coding
  → openase ticket create --title "Implement user login API" --parent ASE-10 --workflow coding
  → openase ticket create --title "Implement RBAC permission model" --parent ASE-10 --workflow coding
  → openase ticket create --title "Write user system tests" --parent ASE-10 --workflow test

[Agent] Updates ASE-10 description, adding split rationale and architecture design
[Agent] Starts processing the first sub-ticket ASE-11
```

**Scenario Two: Agent creates a new repository and registers it on the platform**

```
[User] Creates ticket ASE-20: "Split notification module into an independent microservice"

[Backend Engineer Agent] Claims ASE-20:
  1. Analyzes notification module in existing codebase
  2. Creates a new repo on GitHub:
     → gh repo create acme/notification-service --public
  3. Migrates notification code to the new repo, submits initial code
  4. Registers the new repo to the project via Platform API:
     → openase project add-repo --name "notification-service" \
         --url "https://github.com/acme/notification-service" \
         --labels "go,backend,notification"
  5. Deletes notification module code from original repository and replaces it with SDK calls
  6. Creates PRs for both repositories
  7. Creates follow-up tickets:
     → openase ticket create --title "Configure CI/CD for notification-service" --workflow devops
```

**Scenario Three: Security engineer triggers remediation flow automatically after finding issues**

```
[Security Engineer Agent] Executes security scan ticket ASE-30:
  1. Scans code and finds 3 critical vulnerabilities
  2. Generates security report
  3. Automatically creates remediation tickets via Platform API:
     → openase ticket create --title "Fix SQL injection: auth/login.go:42" \
         --priority urgent --workflow coding
     → openase ticket create --title "Remove hardcoded API Key: config/secrets.go" \
         --priority urgent --workflow coding
     → openase ticket create --title "Upgrade lodash to 4.17.21" \
         --priority high --workflow coding
  4. Creates recurring security scan scheduled job:
     → openase scheduled-job create --name "weekly-security-scan" \
         --cron "0 9 * * 1" --workflow security
```

**Scenario Four: Product manager role-driven project evolution**

```
[Product Manager Agent] Executes ticket ASE-40: "Research competitors and plan v2.0 features"
  1. Searches competitor information and analyzes feature gaps
  2. Writes v2.0 PRD document
  3. Updates project description:
     → openase project update --description "$(cat v2-prd.md)"
  4. Splits into tickets according to PRD:
     → openase ticket create --title "v2.0: Real-time collaborative editing" --type feature --priority high
     → openase ticket create --title "v2.0: Mobile adaptation" --type feature --priority medium
     → openase ticket create --title "v2.0: Performance optimization (first paint < 2s)" --type refactor --priority high
  5. Advances project status:
     → openase project update-status "In Progress"
```

**Scenario Five: Self-optimization of refine-harness meta-workflow**

```
[Harness Optimization Agent] Executes ticket ASE-50: "Optimize coding Harness"
  1. Queries execution history of the latest 20 coding tickets:
     → openase ticket list --workflow coding --status done --limit 20
  2. Analyzes failure patterns (which on_complete Hooks fail often, which tickets time out)
  3. Updates coding Harness body via Platform API:
     - Add "run existing tests before making code changes" step
     - Adjust scope boundary rules
  4. Platform generates new Harness version; subsequent new runtimes automatically use the new version
  5. Creates validation ticket:
     → openase ticket create --title "Validate optimized coding Harness" --workflow test
```

### 27.6 Security Boundaries

Agent autonomy must have clear security boundaries:

**Defense Line One: Token Scope Limitation**

Agent Token includes only permissions declared in `platform_access.allowed` in Harness. API layer validates Token scope for each request; unauthorized actions return 403 directly. Token expires automatically after ticket completion.

**Defense Line Two: Rate Limiting**

| Operation Type | Rate Limit | Rationale |
|---------|---------|------|
| Create ticket | 20 per ticket lifecycle | Prevent Agent from indefinitely splitting sub-tickets |
| Register repository | 5 per ticket lifecycle | Prevent creation of many unnecessary repositories |
| Create scheduled job | 3 per ticket lifecycle | Prevent configuration of excessive cron jobs |
| Update project information | 10 per ticket lifecycle | Prevent frequent rewrites |

**Defense Line Three: Explicit Authorization**

High-risk operations must be explicitly whitelisted in Harness:

```yaml
platform_access:
  allowed:
    - "projects.add_repo"
```

Platform operations not explicitly authorized are refused; no intermediate state such as “pause and wait for approval” is supported.

**Defense Line Four: ActivityEvent Full Audit**

All platform operations initiated by Agents are recorded to ActivityEvent, with `created_by` marked as `agent:{agent_name}`, and `metadata` containing the full request parameters. Humans can trace all Agent platform operations in the activity timeline.

**Defense Line Five: Operate Only Current Project**

Agent Token scope is limited to current `project_id` and cannot operate across projects. An Agent from one project cannot create tickets for or modify another project’s configuration.

### 27.7 Architectural Impact

| Component | Change |
|------|------|
| `internal/httpapi` | Add Agent Token validation logic: parse scope, validate `project_id` boundaries, rate limiting |
| `internal/orchestrator` Worker | Generate Agent Token and inject environment variables when starting Agent |
| `internal/agentplatform` / provider contracts | Authentication and token validation must distinguish User Token and Agent Token |
| Harness Rendering | Inject `OPENASE_API_URL`, `OPENASE_AGENT_TOKEN`, and related environment variables |
| `cmd/openase` / CLI | Form two-layer structure of `openase api` + typed resource commands; Agent CLI defaults to reading `OPENASE_AGENT_TOKEN`, control plane CLI and HTTP/OpenAPI remain isomorphic |
| ActivityEvent | `created_by` supports both `user:xxx` and `agent:xxx` formats |
| Database | Add `agent_tokens` table (`token_hash`, `agent_id`, `ticket_id`, `scopes`, `expires_at`) |

### 27.8 Closed-Loop Overview

```
Human                          OpenASE Platform                        Agent
  │                              │                                │
  ├── Create project ─────────────→   │                                │
  ├── Configure role (Harness) ───→   │                                │
  ├── Create ticket ───────────────→   │                                │
  │                              │                                │
  │                              ├── orchestrator dispatches ──────→   │
  │                              │                                │
  │                              │   ┌──── Agent executes work ─────┐   │
  │                              │   │ Write code, run tests, open PRs │   │
  │                              │   │                              │   │
  │                              │   │ Also reverse-operates platform: │   │
  │                              │ ←─┤ Create sub-ticket            │   │
  │                              │ ←─┤ Register new Repo            │   │
  │                              │ ←─┤ Update project description   │   │
  │                              │ ←─┤ Configure scheduled jobs     │   │
  │                              │ ←─┤ Link external Issue          │   │
  │                              │   └──────────────────────────────┘   │
  │                              │                                │
  │   ← SSE live updates ────────    │                                │
  │   ← Approval requests ──────    │  ← Hook validation ────────────    │
  │                              │                                │
  ├── Approval / feedback ───────→  │                                │
  │                              │                                │
  │                              ├── status transition → done ─────→  │
  │                              │                                │
  │   ← New ticket notification ──    │  (sub-tickets created by Agent enter queue) │
  │                              ├── orchestrator dispatches next ticket ───→   │
  │                              │                    ...continues looping   │
```

This is the true closed loop: Agents are not only consumers of work, but also producers of work and operators of the platform. Humans evolve from supervising Agents to setting strategy (via Harness permission configuration), then observing the system self-operating.
## Chapter 28 External Issue Sync (Not Implemented for Now)

### 28.1 Decision

The current version **does not implement** external Issue sync capability. This includes but is not limited to:

- Does not automatically create tickets from GitHub Issues / GitLab Issues / Jira / Linear
- Does not write ticket status, comments, labels, or PR results back to these external systems
- Does not provide inbound webhook receive endpoints for external Issue systems
- Does not take external Issue / PR / CI status as input to the OpenASE state machine

### 28.2 Current Alternative Approach

If users need to connect external systems with OpenASE, only the following explicit, low-coupling methods are currently used:

- Manually create tickets via UI / API / Scheduled Job
- Manually save the external Issue or PR URL / `external_id` in `TicketExternalLink`
- Manually, or explicitly written by an Agent, write `pull_request_url` in `TicketRepoScope`

These links are only used as contextual references and navigation entry points; they do not participate in automatic synchronization, nor do they drive state progression.

### 28.3 Future Expansion Scope

If external Issue synchronization is reintroduced in the future, the following prerequisites must be met:

- It must be relaunched as an independent incremental capability and should not implicitly restore the old Webhook / Connector design
- Do not implicitly change the current ticket state machine principle of "explicit progression"
- `pull_request_url` in RepoScope is still primarily a reference field; if any external status caching is to be introduced, it must be defined separately in a new version PRD

---
## Chapter 29 Custom Ticket Status and Kanban Columns

### 29.1 Problem: Hardcoded Status Is Not Enough

The current PRD has changed ticket status to fully custom (`TicketStatus` entity), with no hardcoded enums. But workflows differ greatly across teams and projects:

| Scenario | Required states | Pain points of fixed 7 states |
|------|-----------|----------------|
| Research project | idea → literature_review → experiment → writing → submitted → revision → accepted | `in_progress` has to carry 4 completely different phases |
| Operations team | reported → triaging → investigating → mitigating → resolved → postmortem | No concept of `triaging` and `postmortem` |
| Design collaboration | brief → wireframe → mockup → review → handoff → implemented | `handoff` is before done but is not `in_review` |
| Content production | draft → editing → fact_check → legal_review → scheduled → published | Multi-round review stages cannot be expressed |

### 29.2 Design: Custom Status + Structured Stage + Workflow Pickup-driven Dispatch

`Custom Status` remains the only kanban state dimension that is transactable, sortable, rate-limited, and displayable. But the platform additionally introduces a structured `stage` for each `TicketStatus` to express lifecycle semantics. In other words:

- `status` is responsible for team naming, kanban column order, color, icon, and WIP limiting
- `stage` is responsible for unified backend judgment of whether the status is backlog, unstarted, started, completed, or canceled

`stage` is a small stable enum, does not directly render as the kanban column title, and does not replace `Custom Status`. Users edit “which `stage` a given status corresponds to”, rather than editing a ticket’s own `stage` directly.

```text
Ticket.status_id  ─────→ TicketStatus(name="QA Review", stage="started")
                                     │
                                     ├─ UI Display: "QA Review"
                                     └─ Runtime semantics: started
```

The platform defines the following stages:

| Stage | Semantics | Is terminal |
|------|------|------|
| `backlog` | Backlog that has not entered an execution entry point yet | No |
| `unstarted` | Ready but not started yet | No |
| `started` | Execution has started, including development / testing / review and other in-progress phases | No |
| `completed` | Completed in the positive direction | Yes |
| `canceled` | Actively canceled / abandoned | Yes |

The design goal of `stage` is not to restrict user dragging and dropping, and not to restrict which statuses a workflow can bind to. It is to remove backend magic dependency on status names like `"Done"`. Dependency release, terminal-state judgment, and dashboard statistics must all be based on `stage`, not inferred from status names.

```
Research project kanban:

  [💡 Idea] → [📋 To Investigate] → [📚 Literature Review] → [🧪 In Experiment] → [✍️ Writing] → [📤 Submitted] → [✅ Accepted]

Workflow: research-ideation
  pickup: "To Investigate"    ← Orchestrator scans this column
  finish: "Literature Review"  ← After Agent completes, move ticket to this column (user drags to next column after review)

Workflow: experiment-runner
  pickup: "In Experiment"    ← After user reviews research result and drags to "In Experiment" → Orchestrator takes over
  finish: "Writing"

Workflow: report-writer
  pickup: "Writing"    ← After user confirms experiment result and drags to "Writing" → Orchestrator takes over
  finish: "Submitted"
```

**Each column still retains the team’s own business semantics; platform lifecycle semantics are provided uniformly by stage.** The same `"To Test"` column can be a pickup (entry) for a test workflow, and can also be a finish (exit) for a coding workflow; but at system level it still must map to a stable stage such as `started` or `completed`.

### 29.3 Data Model

**TicketStatus entity (new):**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Belongs to project |
| name | String | Status name (e.g., `"Literature Review"`, `"In Experiment"`), unique within project |
| stage | Enum | Lifecycle phase: `backlog` / `unstarted` / `started` / `completed` / `canceled` |
| color | String | Hex color (e.g., `#3B82F6`), for kanban display |
| icon | String | Icon identifier (Lucide icon name, such as `flask`, `pen`) |
| position | Integer | Kanban column sort position |
| max_active_runs | Integer (nullable) | Maximum number of active `AgentRun` allowed in this status at the same time; null means no limit |
| is_default | Boolean | Whether this is the default status for new tickets |
| description | String | Status description (optional, hover tooltip) |

**Ticket table status field change:**

```
Before: status Enum (hardcoded 7 statuses)
After: status_id FK → TicketStatus.id
```

Workflow table new fields (replacing previous Phase mapping):

```
pickup_status_ids   NonEmptySet<FK → TicketStatus.id>   -- Orchestrator scans tickets in these statuses
finish_status_ids   NonEmptySet<FK → TicketStatus.id>   -- After Agent completes, tickets may land in these statuses
```

Orchestrator query semantics change to: `WHERE status_id IN workflow.pickup_status_ids AND current_run_id IS NULL AND retry_paused = false AND (next_retry_at IS NULL OR next_retry_at <= now())`. After hitting candidate tickets, it must also check whether the matched pickup status itself has `max_active_runs` configured and whether it is full. There is no fail status, no hardcoded state machine; on Agent error, retry with backoff.

Structured constraints:

- `pickup_status_ids / finish_status_ids` can bind to any status within the same project; each workflow defines its own “which statuses can be claimed” and “which statuses are allowed on completion”
- Dependency `blocks` release condition checks only whether the blocker ticket’s current status `stage` is terminal
- Project default status should not be in `completed` or `canceled`, to avoid creating new tickets directly into terminal columns

### 29.4 Default Status Template

When each new project is created, a default set of Custom Status is auto-generated. Users can directly use, edit, or delete them and fully customize afterwards:

| Default Custom Status | Stage | Color | Description | is_default |
|-------------------|------|------|------|-----------|
| Backlog | `backlog` | Gray | Backlog | ✅ |
| Todo | `unstarted` | Blue | Ready and waiting | |
| In Progress | `started` | Yellow | In progress | |
| In Review | `started` | Purple | Under review | |
| Done | `completed` | Green | Completed | |
| Cancelled | `canceled` | Gray | Canceled | |

Note: there is no `"Failed"` column. After a ticket errors, it stays in the pickup column (for example `"Todo"`) waiting for retry. The frontend highlights it in red based on `consecutive_errors > 0` and shows `"Retrying (Nth time)"`. What users see is not `"Failed"`, but `"still trying"`.

Built-in coding workflow initial configuration: `pickup_status_ids = ["Todo"]`, `finish_status_ids = ["Done"]`. After users modify kanban columns, they only need to update the workflow status set configuration accordingly.

If a team wants “at any moment only 1 Agent can process tickets from a particular pickup column,” they can directly set that status’s `max_active_runs` to `1`. For example, setting `Todo` `max_active_runs` to `1` makes all workflows hitting `Todo` share this single column-level concurrency limit; other tickets that also reach this pickup column will wait until the previous `AgentRun` completes before being claimed.

### 29.5 Status Transition Rules

**Manual user operations** — Drag-and-drop between any Custom Status is allowed, with no state machine blocking. If a user wants to drag a ticket from `"Completed"` back to `"In Progress"`? Yes. If they want to jump directly from `"Idea"` to `"Submitted"`? Also yes. OpenASE does not judge business validity — that is the team’s own process governance.

But manual dragging does not change the `stage` definition; `stage` is an attribute of status configuration itself, not a temporary tag attached to a ticket. In other words, dragging a ticket to a status is essentially switching it to “the stage that this status corresponds to.”

**Automatic orchestration trigger** — `status_id` is automatically changed only at two moments:

1. **Agent completion**
   - If `workflow.finish_status_ids` contains exactly 1 value → automatically set to that status
   - If `workflow.finish_status_ids` contains multiple values → Agent must explicitly choose one of them; the platform must reject target statuses outside the set
2. **Agent error** → clear `current_run_id`, then retry with exponential backoff (no fail status)

**Dependency release rules** — Release condition for `A blocks B` is:

- The `TicketStatus.stage` corresponding to A’s current `status_id` is in `{completed, canceled}`, or
- A has explicitly written `completed_at`

The system must no longer infer dependency release by guessing whether the status name equals strings like `"Done"`, `"Cancelled"`.

**Hook does not care about specific status, only orchestrator lifecycle events (claim / start / progress / complete / done / error / cancel).**

**What happens when a ticket is dragged into a workflow pickup status:** orchestrator scans it on next tick → executes `on_claim` Hook → assigns Agent → starts execution. If the user drags the ticket away during Agent execution (leaves pickup status), the orchestrator does nothing — the Agent continues executing until completion. Only when the user clicks `"Cancel"` does `on_cancel` trigger to stop the Agent.

### 29.6 Kanban View

Custom status maps directly to kanban columns. Users can:

- Drag columns to change order
- Add/delete/rename columns
- Change column color and icon
- See ticket count for that status above each column
- Move a ticket from one column to another = status change

```
┌─────────────────────────────────────────────────────────────────────┐
│ Research Project AlphaFold-Next                        [+ Add Column] [Settings] │
├──────────┬──────────┬──────────┬──────────┬──────────┬─────────────┤
│ 💡 Idea  │ 📋 To Investigate │ 📚 Literature Review │ 🧪 In Experiment │ ✍️ Writing │ ✅ Accepted │
│ (3)      │ (2)      │ (1)      │ (2)      │ (0)      │ (5)         │
├──────────┼──────────┼──────────┼──────────┼──────────┼─────────────┤
│ ASE-15   │ ASE-20   │ ASE-18   │ ASE-12   │          │ ASE-1       │
│ ASE-16   │ ASE-21   │          │ ASE-14   │          │ ASE-3       │
│ ASE-17   │          │          │          │          │ ASE-5       │
│          │          │          │          │          │ ASE-8       │
│          │          │          │          │          │ ASE-10      │
└──────────┴──────────┴──────────┴──────────┴──────────┴─────────────┘
```

**Kanban rendering order contract:**

- Kanban columns are rendered directly from left to right by `TicketStatus.position`
- The ordered `statuses` list returned by `/api/v1/projects/:projectId/statuses` is the main kanban rendering contract; the frontend should no longer add extra grouping layers on its own
- If a column configures `max_active_runs`, frontend may show `active_runs / max_active_runs` in the column header
- Kanban column titles always show status names; `stage` appears only in settings page, debug info, rule validation, and automation semantics, and does not replace user-defined column names

### 29.7 Harness and Hook compatibility

Harness and Hook do not care about specific Custom Status names — they care only about orchestrator lifecycle events:

```yaml
# In Harness
hooks:
  on_claim:     # Triggered when orchestrator takes over a ticket (regardless of which status the ticket is in)
  on_complete:  # Triggered when Agent declares completion
```

```text
# Harness prompt template variable
{{ ticket.status }}     → "Literature Review" (Custom Status name, shown to Agent)
```

Agent sees meaningful Custom Status names in the prompt (`"Literature Review"` is much more useful than `"in_progress"`). Orchestrator binds scheduling via `pickup_status_ids`, and uses `stage` for terminal and dependency judgment, rather than inferring semantics from status names.

### 29.8 External synchronization does not participate in status mapping

This version does not perform external Issue/PR status synchronization, so Custom Status is driven only by explicit internal status transitions in OpenASE:

- Humans modify status in UI/API
- Agents explicitly modify status via Platform API
- Scheduled Job / Dispatcher explicitly writes status when creating or routing

Status values in external systems such as `open / closed / merged / failed` do not participate in `TicketStatus` mapping.

### 29.9 API Endpoints

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/projects/:projectId/statuses` | List all Custom Statuses for a project (ordered by `position`, including `stage`) |
| POST | `/api/v1/projects/:projectId/statuses` | Create Custom Status `{name, stage, color, icon, position, max_active_runs?}` |
| PATCH | `/api/v1/statuses/:statusId` | Update Custom Status (name, stage, color, icon, ordering, concurrency cap) |
| DELETE | `/api/v1/statuses/:statusId` | Delete Custom Status (migrate tickets in this status to project default status or to caller-specified replacement) |
| POST | `/api/v1/projects/:projectId/statuses/reorder` | Batch update ordering `{status_ids: [...]}` |
| POST | `/api/v1/projects/:projectId/statuses/reset` | Reset to default template |

### 29.10 Database index updates

```sql
-- Orchestrator queries by status_id + current_run_id
-- Each workflow has a different pickup_status_ids set, so do not hardcode specific values in index
CREATE INDEX idx_tickets_dispatch ON tickets (project_id, status_id, current_run_id, priority, created_at);

-- Render kanban by position / count column capacity
CREATE INDEX idx_statuses_position ON ticket_statuses (project_id, position);

-- Group kanban by status_id
CREATE INDEX idx_tickets_board ON tickets (project_id, status_id);
```

### 29.11 Impact on existing chapters

| Chapter | Impact |
|------|------|
| Chapter 6 Domain model | `Ticket.status` changed to `status_id` FK; add `TicketStatus.stage`; add `Workflow.pickup_status_ids / finish_status_ids` |
| Chapter 7 (Rewritten) | No more hardcoded state machine; orchestrator dispatches based on `workflow.pickup_status_ids`, `stage` is only for terminal/dependency semantics |
| Chapter 8 Hook | Trigger points no longer reference specific status names, only orchestrator lifecycle events |
| Chapter 10 Orchestrator | Dispatch query changed to `WHERE status_id IN workflow.pickup_status_ids AND current_run_id IS NULL`, with additional check for matched status concurrency limit; dependency release changed to stage-based |
| Chapter 16 Pseudocode | Status judgment changed to `contains(wf.PickupStatusIDs, ticket.StatusID)` |
| Chapter 18 API | Ticket response includes nested `status: {id, name, color, icon, stage}` object |
| Chapter 20 Database index | Index fields changed from status enum to status_id FK |
| Chapter 28 External sync | This version does not do external status mapping; Custom Status is only explicitly advanced inside OpenASE |

Core simplification: **There is no hardcoded status transition graph, but there is structured Stage semantics.** Orchestrator only matches `status_id ∈ pickup_status_ids`; status transitions are still driven by users (kanban drag and drop) and Agents (moving to one of the allowed finish statuses on completion). Column order is determined by `TicketStatus.position`, and concurrency caps are expressed directly by `TicketStatus.max_active_runs`. `Stage` only uniformly answers “whether this status is terminal”, and how to interpret dependency and statistics.

### 29.12 Implementation and migration plan

The rollout is recommended in four phases to avoid breaking Workflow, scheduler, and frontend all at once:

1. **Schema and domain model**
   - Add non-null `stage` field to `ticket_statuses`
   - Default template directly writes Stage mapping:
     - `Backlog -> backlog`
     - `Todo -> unstarted`
     - `In Progress / In Review -> started`
     - `Done -> completed`
     - `Cancelled -> canceled`
   - HTTP / OpenAPI / frontend status models all include `stage`

2. **Backend semantic switch**
   - Change dependency release logic to only look at blocker `status.stage` terminality
   - Workspace / dashboard / active ticket statistics all switched to stage-based terminal checks
   - Remove any fallback checks based on `"Done"` or `"Cancelled"` names

3. **Constraints and control plane**
   - TicketStatus create/update APIs support editing stage
   - Workflow create/update validation:
     - `pickup_status_ids / finish_status_ids` only require non-empty, existing refs, and belonging to current project
   - Status settings page adds Stage selector and clearly indicates “stage controls lifecycle semantics”

4. **Data migration and validation**
   - Migrate old project statuses: prioritize mapping by default template names; infer remaining statuses using workflow finish bindings
   - For historical statuses that cannot be safely inferred, output migration warning requiring manual confirmation
   - Cover with integration tests:
     - blocker release after `completed/canceled`
     - custom terminal names no longer depend on `"Done"` string
     - workflow finish/pickup stage constraints take effect

---
## Chapter 30 Harness Template Variable Dictionary and Editor

### 30.1 Template Syntax

Harness Markdown body content supports variable substitution with **Jinja2 syntax**. The reason to choose Jinja2 over Go `text/template` is that Jinja2 is more user-friendly for non-programmers (`{{ ticket.title }}` is more intuitive), it supports filters (`{{ ticket.description | truncate(500) }}`), and conditional/loop syntax is more natural.

The backend uses a Go Jinja2-compatible library (such as `github.com/nikolalohinski/gonja`) to render templates.

```jinja
{# Variable substitution #}
You are handling ticket {{ ticket.identifier }}: {{ ticket.title }}

{# Conditional #}
{% if attempt > 1 %}
This is attempt {{ attempt }}. Continue from the current workspace state.
{% endif %}

{# Loop #}
{% for repo in repos %}
- {{ repo.name }} ({{ repo.labels | join(", ") }}): {{ repo.path }}
{% endfor %}

{# Filter #}
{{ ticket.description | default("No description") | truncate(1000) }}
```

### 30.2 Complete Variable Dictionary

The following is the complete list of all variables available in Harness templates. In the front-end Harness editor, they are shown in the sidebar "variable dictionary" panel, and clicking a variable name automatically inserts it at the cursor position.

**Ticket (ticket)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `ticket.id` | string | Ticket UUID | `550e8400-e29b-41d4-a716-446655440000` |
| `ticket.identifier` | string | Human-readable identifier | `ASE-42` |
| `ticket.title` | string | Ticket title | `Fix login form validation` |
| `ticket.description` | string | Ticket description (Markdown) | `The login form doesn't validate...` |
| `ticket.status` | string | Current custom status name | `In Progress` |
| `ticket.priority` | string | Priority | `high` |
| `ticket.type` | string | Ticket type | `bugfix` |
| `ticket.created_by` | string | Creator | `user:gary` or `agent:claude-01` |
| `ticket.created_at` | string | Creation time (ISO 8601) | `2026-03-19T10:30:00Z` |
| `ticket.attempt_count` | int | Current attempt count | `1` |
| `ticket.max_attempts` | int | Maximum attempt count | `3` |
| `ticket.budget_usd` | float | Budget cap (USD) | `5.00` |
| `ticket.external_ref` | string | External reference | `octocat/repo#42` |
| `ticket.parent_identifier` | string | Parent ticket identifier (if sub-issue) | `ASE-30` |
| `ticket.url` | string | Link to ticket in OpenASE Web UI | `http://localhost:19836/tickets/ASE-42` |

**Ticket external links (ticket.links)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `ticket.links` | list | External link list | — |
| `ticket.links[].type` | string | Link type | `github_issue` |
| `ticket.links[].url` | string | External URL | `https://github.com/acme/backend/issues/42` |
| `ticket.links[].title` | string | External title | `Login validation broken on Safari` |
| `ticket.links[].status` | string | External status | `open` |
| `ticket.links[].relation` | string | Relation | `resolves` |

**Ticket dependencies (ticket.dependencies)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `ticket.dependencies` | list | Dependency list | — |
| `ticket.dependencies[].identifier` | string | Dependent ticket identifier | `ASE-30` |
| `ticket.dependencies[].title` | string | Dependent ticket title | `Design user auth schema` |
| `ticket.dependencies[].type` | string | Dependency type | `blocks` or `sub_issue` |
| `ticket.dependencies[].status` | string | Current status of dependent ticket | `Done` |
| `ticket.dependencies[].stage` | string | Stage corresponding to dependent ticket status | `completed` |

**Ticket status semantics (ticket.status)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `ticket.status` | string | Current custom status name | `In Review` |
| `ticket.status_stage` | string | Stage corresponding to current custom status | `started` |

**Project (project)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `project.id` | string | Project UUID | — |
| `project.name` | string | Project name | `awesome-saas` |
| `project.slug` | string | URL identifier | `awesome-saas` |
| `project.description` | string | Project description | `A SaaS platform for...` |
| `project.status` | string | Project status; canonical values are `Backlog` / `Planned` / `In Progress` / `Completed` / `Canceled` / `Archived` | `In Progress` |
**Repos (repos)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `repos` | list | Repositories involved in this ticket (TicketRepoScope) | — |
| `repos[].name` | string | Repository alias | `backend` |
| `repos[].url` | string | Git repository URL | `https://github.com/acme/backend` |
| `repos[].path` | string | Local path in workspace | `/home/openase/.openase/workspace/acme/payments/ASE-42/backend` |
| `repos[].branch` | string | Current working branch | `agent/ASE-42` |
| `repos[].default_branch` | string | Default branch | `main` |
| `repos[].labels` | list | Repository labels | `["go", "backend", "api"]` |
| `all_repos` | list | All repositories under project (not only those involved in this ticket) | Same structure as repos |

**Agent (agent)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `agent.id` | string | Agent UUID | — |
| `agent.name` | string | Agent name | `claude-01` |
| `agent.provider` | string | Provider name | `Claude Code` |
| `agent.adapter_type` | string | Adapter type | `claude-code-cli` |
| `agent.model` | string | Model name | `claude-sonnet-4-6` |
| `agent.total_tickets_completed` | int | Total completed tickets historically | `47` |

**Machine (machine)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `machine.name` | string | Current execution machine name | `gpu-01` |
| `machine.host` | string | Machine address | `10.0.1.10` |
| `machine.description` | string | Machine description | `NVIDIA A100 × 4, 256GB RAM` |
| `machine.labels` | list | Machine labels | `["gpu", "a100", "cuda-12"]` |
| `machine.workspace_root` | string | Remote ticket workspace root directory | `/home/openase/.openase/workspace` |
| `accessible_machines` | list | List of other accessible machines | — |
| `accessible_machines[].name` | string | Machine name | `storage` |
| `accessible_machines[].host` | string | Address | `10.0.1.20` |
| `accessible_machines[].description` | string | Description | `Data storage, 16TB NVMe` |
| `accessible_machines[].labels` | list | Labels | `["storage", "nfs"]` |
| `accessible_machines[].ssh_user` | string | SSH username | `openase` |

**Execution context (context)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `attempt` | int | Current attempt number (starting from 1) | `1` |
| `max_attempts` | int | Maximum attempt count | `3` |
| `workspace` | string | Workspace root path | `/home/openase/.openase/workspace/acme/payments/ASE-42` |
| `timestamp` | string | Current time (ISO 8601) | `2026-03-19T10:30:00Z` |
| `openase_version` | string | OpenASE version | `0.3.1` |

**Workflow configuration (workflow)**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `workflow.name` | string | Workflow name | `coding` |
| `workflow.type` | string | User-visible workflow type label (raw value) | `Backend Engineer` |
| `workflow.family` | string | Stable workflow family derived from platform | `coding` |
| `workflow.role_name` | string | Role name | `fullstack-developer` |
| `workflow.pickup_statuses` | string[] | List of pickup status names | `["Todo"]` |
| `workflow.finish_statuses` | string[] | List of finish status names | `["Done"]` |

**Platform API (platform) — for agent autonomous closed-loop use**

| Variable | Type | Description | Example |
|------|------|------|--------|
| `platform.api_url` | string | Platform API address | `http://localhost:19836/api/v1` |
| `platform.agent_token` | string | Agent-specific short-lived token | `ase_agent_xxx...` |
| `platform.project_id` | string | Current project ID | UUID |
| `platform.ticket_id` | string | Current ticket ID | UUID |

### 30.3 Built-in Filters

| Filter | Description | Example |
|--------|------|------|
| `default(value)` | Use default value when variable is empty | `{{ ticket.description \| default("No description") }}` |
| `truncate(length)` | Truncate to a specified length | `{{ ticket.description \| truncate(500) }}` |
| `join(sep)` | Join list into a string | `{{ repos[0].labels \| join(", ") }}` |
| `upper` / `lower` | Case conversion | `{{ ticket.priority \| upper }}` |
| `length` | Get list length | `{{ repos \| length }} repositories` |
| `first` / `last` | Get first/last item in list | `{{ repos \| first }}` |
| `sort` | Sort | `{{ ticket.links \| sort(attribute="type") }}` |
| `selectattr(attr, value)` | Filter list by attribute | `{{ repos \| selectattr("labels", "backend") }}` |
| `map(attribute)` | Extract an attribute from list items | `{{ repos \| map(attribute="name") \| join(", ") }}` |
| `tojson` | Output as JSON string | `{{ repos \| tojson }}` |
| `markdown_escape` | Escape Markdown special characters | `{{ ticket.title \| markdown_escape }}` |

### 30.4 Front-end Harness Editor

In the Workflow management page of the Web UI, the Harness editor is the core interactive component:

```
┌──────────────────────────────────────────────────────────────────────┐
│ Edit Harness: coding.md                      [Preview] [Save] [History]   │
├──────────────────────────────────────┬───────────────────────────────┤
│                                      │ 📖 Variable Dictionary [Search] │
│  ---                                 │                               │
│  status:                             │ ▼ Ticket (ticket)              │
│    pickup: "Todo"                    │   ticket.identifier    ASE-42 │
│    finish: "Done"                    │   ticket.title         String │
│  agent:                              │   ticket.description   String │
│    max_turns: 20                     │   ticket.status        String │
│  hooks:                              │   ticket.priority      String │
│    on_complete:                      │   ticket.type          String │
│      - cmd: "make test"             │   ticket.budget_usd    Float  │
│  ---                                 │   ticket.links[]       List   │
│                                      │   ticket.dependencies[] List  │
│  # Coding Workflow                   │                               │
│                                      │ ▼ Repos (repos)                │
│  You are handling ticket             │   repos[].name         String │
│  {{ ticket.identifier }}             │   repos[].path         String │
│  ~~~~~~~~~~~~~~~~~~~~~~~~            │   repos[].url          String │
│  (Blue highlight, hover shows current│   repos[].branch       String │
│  value)                              │   repos[].labels[]     List   │
│                                      │                               │
│  {% if attempt > 1 %}                │ ▼ Agent (agent)                │
│  This is attempt {{ attempt }}        │   agent.name           String │
│  {% endif %}                         │   agent.provider       String │
│                                      │   agent.model          String │
│  ## Involved repositories            │                               │
│                                      │ ▼ Machine (machine)             │
│  {% for repo in repos %}             │   machine.name         String │
│  - **{{ repo.name }}**:              │   machine.host         String │
│    {{ repo.path }}                   │   machine.description  String │
│  {% endfor %}                        │   accessible_machines[] List  │
│                                      │                               │
│                                      │ ▼ Context (context)             │
│                                      │   attempt              Int    │
│                                      │   workspace            String │
│                                      │   timestamp            String │
│                                      │                               │
│                                      │ ▼ Platform API (platform)       │
│                                      │   platform.api_url     String │
│                                      │   platform.agent_token String │
│                                      │                               │
│                                      │ ▼ Filters                       │
│                                      │   default(val)                  │
│                                      │   truncate(len)                 │
│                                      │   join(sep)                    │
│                                      │   ...                          │
└──────────────────────────────────────┴───────────────────────────────┘
```

**Editor features:**

| Feature | Implementation |
|------|------|
| **Syntax highlighting** | `{{ }}` variable tags highlighted in blue, `{% %}` control tags in green, `{# #}` comments in gray |
| **Hover preview** | Hover over `{{ ticket.identifier }}` → tooltip shows "Current value: ASE-42 (actual value from latest ticket)" |
| **Autocomplete** | Type `{{ t` → suggestions pop up: `ticket.identifier`, `ticket.title`, `timestamp`, etc. |
| **Variable dictionary panel** | Fixed right panel grouping all available variables; click a variable name → inserts `{{ variable_name }}` at the cursor position |
| **YAML Frontmatter validation** | Real-time YAML syntax validation, highlights error lines in red |
| **Live preview** | Click "Preview" → renders the template with real data from the latest ticket so the Agent can see the actual prompt |
| **Diff view** | Click "History" → shows Git log, compare Diff between two selected versions |
| **Undefined variable detection** | If a variable not in the dictionary is used in the template → editor emits a yellow warning |
| **Quick snippets** | Quickly insert common template snippets (e.g., "standard workflow", "work boundaries", "acceptance criteria") |

**Data source for live preview:**

```go
// For preview rendering, use the latest completed ticket for this Workflow
func (h *HarnessEditor) PreviewData(ctx context.Context, workflowID string) TemplateData {
    // Find the most recent completed ticket
    recent, err := h.ticketRepo.FindLatestByWorkflow(ctx, workflowID)
    if err != nil {
        // No historical ticket -> use mock data
        return mockTemplateData()
    }
    return buildTemplateData(recent)
}
```

### 30.5 Template validation

When saving Harness, the backend performs validation to ensure the template can be rendered safely:

```go
func validateHarness(content string) []ValidationError {
    var errors []ValidationError

    normalized := normalizeHarnessNewlines(content)

    // 1. Legacy frontmatter is not allowed
    if strings.HasPrefix(strings.TrimSpace(normalized), "---") {
        errors = append(errors, ValidationError{
            Line:    1,
            Message: "Harness content must be pure Markdown/Gonja body text. YAML frontmatter is no longer supported.",
        })
    }

    // 2. Jinja2 template syntax validation (no rendering, only syntax checks)
    _, err := gonja.FromString(normalized)
    if err != nil {
        errors = append(errors, ValidationError{Message: "Template syntax error: " + err.Error()})
    }

    // 3. Variable reference check: ensure all variables used in the template exist in the dictionary
    usedVars := extractVariables(normalized) // Parse variable names from all {{ xxx }}
    for _, v := range usedVars {
        if !isKnownVariable(v) {
            errors = append(errors, ValidationError{
                Message: fmt.Sprintf("Unknown variable '%s', please check the variable dictionary", v),
                Level:   "warning",  // Warning instead of error (save allowed, but user is reminded)
            })
        }
    }

    return errors
}
```

### 30.6 API endpoints

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/harness/variables` | Get the complete variable dictionary (JSON format, including groups, types, descriptions, sample values) |
| POST | `/api/v1/harness/validate` | Validate Harness content (syntax + variable references) |
| POST | `/api/v1/harness/preview` | Render template preview with real data using `{workflow_id, content}` |
| GET | `/api/v1/harness/snippets` | Get a list of common template snippets |

Variable dictionary API response format:

```json
{
  "groups": [
    {
      "name": "Ticket",
      "prefix": "ticket",
      "variables": [
        {
          "key": "ticket.identifier",
          "type": "string",
          "description": "Readable identifier",
          "example": "ASE-42",
          "insertText": "{{ ticket.identifier }}"
        },
        {
          "key": "ticket.links",
          "type": "list",
          "description": "External links list",
          "children": [
            {"key": "type", "type": "string", "example": "github_issue"},
            {"key": "url", "type": "string", "example": "https://..."},
            {"key": "title", "type": "string"},
            {"key": "status", "type": "string"}
          ],
          "insertText": "{% for link in ticket.links %}\n- [{{ link.type }}] {{ link.title }}: {{ link.url }}\n{% endfor %}"
        }
      ]
    }
  ],
  "filters": [
    {"name": "default", "args": "value", "description": "Use default when variable is empty", "example": "{{ ticket.description | default(\"No description\") }}"},
    {"name": "truncate", "args": "length", "description": "Truncate to the specified length"},
    {"name": "join", "args": "separator", "description": "Join a list into a string"}
  ]
}
```

---
## Chapter 31 Embedded AI Assistant (Ephemeral Chat / Project Conversation)

### 31.1 Positioning

OpenASE’s core is still a ticket-driven orchestration system, but in daily use there are two types of lightweight AI interactions that **do not require creating a ticket**:

- Ephemeral assistance
  - Ask AI to write/edit a Prompt while editing Harness
  - Ask on the ticket details page, “Why did ASE-42 fail?”
  - Ask AI to generate an expression next to the cron input field
- Ongoing project-level conversation
  - Keep asking beside the board, “Where is this project blocked now?”
  - Discuss over multiple turns how to split requirements and which sub-tickets to do first
  - Encounter CLI tool approval mid-flow, then continue the same conversation after human handling

Both scenarios are **not processed through the orchestrator, do not trigger a lifecycle Hook, and do not consume the ticket state machine**, but their lifecycles differ:

- `harness_editor`, `command_palette`, `cron_input`, and similar entry points belong to **Ephemeral Chat**
  - Temporary conversation
  - Ends when closed
  - Transcript is not persisted by default
- `project_sidebar` and `ticket_detail(ticket-focused)` belong to **Project Conversation**
  - Project-level ongoing conversation
  - Transcript is persisted
  - Allows continuing the previous conversation after UI is closed
  - Allows waiting within a turn for Codex native tool approval / user input, then continuing the same turn

So this chapter no longer treats every embedded AI entry as the same kind of ephemeral interaction. Instead, it defines a unified Direct Chat subsystem with two modes: Ephemeral Chat and Project Conversation.

### 31.2 Architecture: Reuse Agent Adapter, bypass orchestrator

Direct Chat still reuses the existing Agent adapter, but it is neither an orchestrator worker nor a ticket runtime. It is hosted directly by `openase serve` to serve real-time browser interaction.

```
┌─────────────────────────────────────────────────────────────┐
│ Frontend                                                  │
│                                                           │
│  Harness AI / Cron AI ─────────────────┐ │                 │
│  Project Sidebar Ask AI ─────────────┐ │ │                 │
│  Ticket Drawer AI(ticket-focused) ──┐ │ │                 │
│                                     ▼ ▼                 │
│                        Chat API + SSE / watch stream        │
└──────────────────────────────┬────────────────────────────┘
                               │
┌──────────────────────────────┼─────────────────────────────┐
│ openase serve                ▼                             │
│                                                           │
│   ┌──────────────────────────────┐                        │
│   │ Direct Chat Service          │                        │
│   │ - context injection         │                        │
│   │ - provider selection         │                        │
│   │ - transcript persistence     │                        │
│   │ - interrupt / approval bridge│                        │
│   └──────────────┬───────────────┘                        │
│                  │                                          │
│      ┌───────────┴───────────┐                              │
│      │ Live Runtime Registry │                              │
│      │ manages browser chat sessions only                  │
│      └───────────┬───────────┘                              │
│                  │                                          │
│          local process / remote SSH                           │
│                  │                                          │
│        Claude / Codex / Gemini Adapter                          │
│                                                           │
│   Not through: orchestrator, Hook, ticket state machine         │
└─────────────────────────────────────────────────────────────┘
```

Core principles:

- `serve` owns browser sessions, SSE/watch stream, and the conversation runtime registry
- CLI processes for each provider still run on the machine pointed to by `provider.machine_id`
- Local provider uses `os/exec`; remote provider uses SSH / sidecar session. Direct Chat reuses the same machine boundary as the orchestrator instead of inventing a new “browser-side CLI”
- Direct Chat can have a lightweight runtime registry, but it is not a worker and does not participate in ticket claim/retry/health reconciliation
- `project_sidebar` introduces persisted transcript and interrupt recovery; other entry points remain lightweight

At the current implementation stage, the real product contract for editor-side AI surfaces still needs to be clarified:

- `Project AI` (`project_sidebar`) is still the mainline machine-aware Direct Chat; the goal is to execute on the machine where the provider is running
- `Harness AI` is currently a **local-only** editor-side Ephemeral Chat
  - Only allow selecting Codex / Claude / Gemini providers bound to the OpenASE host machine where the service runs
  - Remote-machine providers must be treated as unsupported in both UI and API until editor-side machine-aware execution is fully implemented
- `Skill AI` is currently not a universal Ephemeral Chat; it is the skill refinement loop in Chapter 34
  - Phase 1 contract is **local Codex-only**
  - Claude / Gemini providers and remote Codex provider are not in current support scope

Therefore, `ephemeral_chat` cannot be treated as a single source of truth for all editor-side AI surfaces; `Harness AI` and `Skill AI` must each expose independent, testable provider eligibility contracts.

Current recommended CLI run modes:

- Claude Code: `claude -p --verbose --output-format stream-json`
- Codex: reuse `codex app-server` session, using `thread/start` + `turn/start` + notification
- Gemini CLI: use Gemini’s non-interactive streaming output mode

A new Claude Code requirement: when using `-p/--print` with `--output-format stream-json`, you must also add `--verbose`.

### 31.2.1 Two-Mode Chat Model

| Dimension | Ephemeral Chat | Project Conversation |
|------|----------------|----------------------|
| Typical entry points | Harness AI, Cron AI | `project_sidebar`, `ticket_detail(ticket-focused)` |
| Transcript | Not persisted by default | Persisted |
| After UI close | Session ends | Conversation retained after UI close |
| Resume method | `session_id` within the same live session | Resume transcript by `conversation_id`; continue on same thread if live runtime exists, otherwise rebuild by resume strategy |
| Approval | User confirmation only for `action_proposal` | `platform_command_proposal` (compatibility with `action_proposal` during migration) + Codex native tool approval / user input interrupt |
| API | Single-turn SSE is sufficient | Requires conversation + turn + stream + interrupt response interfaces |

### 31.2.2 Multiple CLI Providers

Direct Chat cannot bind to a single CLI only. The project should allow users to explicitly choose the Agent CLI provider used by the current conversation, supporting at least:

- Claude Code
- OpenAI Codex
- Gemini CLI

Design principles:

- Provider selection is a **conversation-level** configuration and does not affect the project default ticket Agent provider
- A single project can have multiple Direct Chat providers at the same time
- Frontend defaults to project default provider; if the default does not support Direct Chat, it falls back to the first available chat-capable provider
- Each provider still uses its own native adapter and is not forced into a single unified protocol
- Switching providers must start a new conversation and cannot reuse a provider-native session/thread from the previous provider

Backend contract requirements:

- Both conversation creation and turn start requests allow explicit `provider_id`
- If `provider_id` is provided, backend must validate that the provider belongs to the organization of the current project and supports Direct Chat
- If `provider_id` is not provided, backend resolves it as “project default provider -> first available chat-capable provider”
- Provider control plane response must explicitly expose `capabilities.ephemeral_chat`
  - `state`: `available` / `unavailable` / `unsupported`
  - `reason`: Provide a structured reason when provider cannot be used for Direct Chat; for providers that support it but are temporarily unavailable, prefer reusing the availability reason
- Frontend selector and backend default/fallback resolution must both be based on this explicit capability, not just runtime.Supports or implicit adapter checks

### 31.2.3 Where the CLI Should Run

Direct Chat CLI runtime location must satisfy: **“Hosted by `serve`, but running under the project.”

Specific rules:

- Process ownership is with `openase serve`
  - Because browser connection, SSE/watch stream, interrupt response, and conversation recovery all happen on the API side
- Execution target is `provider.machine_id`
  - The machine bound to the provider is where the chat CLI starts
  - Do not treat a remote machine’s `local_path` as the local directory of the API process machine
- `cwd` is neither the current OpenASE service process directory nor the repository’s own repo root; it should be a **project-level chat workspace**

Define `ProjectConversationWorkspace`:

- Path is under the target machine’s project scope, for example:
  - `<machine.workspace_root>/<org>/<project>/.openase/chat/`
- Purpose:
  - Stable `cwd` for Codex / Claude / Gemini
  - Holds `.codex/` and `.agent/` skills and wrapper scripts
  - Holds repo link / manifest so project-level conversations can read context across repos
  - Holds non-authoritative runtime temp files and diagnostics

Why not use a specific repo working copy’s `repo_path` directly as `cwd`:

- A multi-repo project loses project-level perspective
- A single repo working copy covers only one repository and cannot assume a natural common parent directory that expresses the whole project
- The semantics of `project_sidebar` is “project conversation,” not “repo-specific conversation”

Therefore, the recommended approach is:

- Project Conversation always enters `ProjectConversationWorkspace`
- The workspace exposes currently visible repo checkouts via symlink / manifest
- For repo-level precise editing, users should explicitly create a ticket or enter Harness / ticket detail scenarios

### 31.3 Conversation Context Injection

Direct Chat’s value is that it knows project context. The backend automatically injects related information into system / developer prompts before calling the Agent CLI.

Injection rules:

- `harness_editor`
  - Current Harness content
  - Template variable dictionary
  - Diff output constraints in Harness edit mode
- `project_sidebar`
  - Project name, description, status overview
  - Ticket statistics, recent activity
  - Visible repo list and default branch
  - Current provider and cost/budget description
  - Rolling summary of current conversation
- `ticket_detail(ticket-focused)`
  - Reuse Project Conversation
  - Inject full ticket capsule: title, description, status, priority, attempt/retry, dependencies, repo scope / PR, recent activity, Hook history, assigned agent / current run / target machine summary
  - Append `OPENASE_TICKET_ID` to runtime env
- `command_palette` / `cron_input`
  - Minimal project context
  - Current form field values

Common constraints:

- AI must not claim that platform write operations were already executed unless it receives a confirmed execution result event
- `project_sidebar` / ticket-focused Project Conversation platform write operations should prefer `platform_command_proposal`; other direct chat entry points can still use `action_proposal`
- During migration, backend must remain compatible with legacy `action_proposal` until Project Conversation migration is complete
- When recovering Project Conversation, Turn 1 does not need verbatim replay of full history; prioritize injecting rolling summary + recent entries

### 31.4 Platform Operations and Approval

Direct Chat has two completely different kinds of “approval/confirmation,” and they must be strictly separated:

- **Platform write operation confirmation**
  - `Project Conversation` should primarily propose via `platform_command_proposal`; other entry points or migration fallback can still use `action_proposal`
  - Platform API executes only after user confirmation
- **CLI native tool approval / user input**
  - Initiated by provider during the current turn
  - For example, Codex command execution approval, file change approval, requestUserInput
  - Continue the same turn after handling

These two mechanisms must not be mixed.

### 31.4.1 Platform operations: Project Conversation prioritizes `platform_command_proposal`

When users in `Project Conversation` request platform write operations such as creating/updating tickets, adding project progress, adjusting workflows, or updating project configuration, AI must primarily output structured `platform_command_proposal` and wait for user confirmation. This protocol must be a restricted command DSL rather than raw REST path + body.

```json
{
  "type": "platform_command_proposal",
  "commands": [
    {
      "command": "ticket.create",
      "args": {
        "project": "ASE",
        "title": "Implement user registration API",
        "parent_ticket": "ASE-42"
      }
    },
    {
      "command": "ticket.create",
      "args": {
        "project": "ASE",
        "title": "Implement user login API"
      }
    },
    {
      "command": "ticket.create",
      "args": {
        "project": "ASE",
        "title": "Implement RBAC permission model"
      }
    }
  ],
  "summary": "Create 3 subtickets"
}
```

Initial restricted command set:

- `project_update.create`
- `ticket.update`
- `ticket.create`

Command arguments may reference human-readable identifiers, for example:

- `project`: project name / slug / UUID
- `ticket`: ticket identifier / UUID
- `status`: status name / UUID
- `workflow`: workflow name / UUID

Backend must complete parsing before execution:

1. Parse proposal JSON into typed domain command
2. Resolve human-readable references like `project` / `ticket` / `status` / `workflow` into UUIDs
3. Call existing service-layer mutations to execute, rather than replaying handwritten internal HTTP requests
4. Write structured per-command execution results back into conversation transcript

Execution flow:

1. AI outputs `platform_command_proposal`
2. Frontend renders confirmation UI
3. User clicks confirm
4. Backend parses commands, resolves references, executes existing service mutation, and writes structured results back as conversation entries
5. Actual platform write operations continue through standard API, audit, and ActivityEvent

It is recommended that backend executes the proposal instead of having the frontend call business APIs one by one. Reasons:

- Execution state is not lost on page refresh
- Execution results can be stably written back to conversation transcript
- Audit boundaries are clearer

Compatibility requirements:

- Continue accepting legacy `action_proposal` during migration
- New prompt / UI / executor default to `platform_command_proposal`
- If a reference cannot be uniquely resolved, AI should ask for clarification first, rather than guessing UUIDs or REST fields

### 31.4.2 Codex Native Tool Approval and User Input

Project Conversation should continue using **Codex native tool approval** instead of inventing a new abstraction in OpenASE and flattening provider semantics.

OpenASE requirements for Codex:

- Must support and forward the following interrupts:
  - `item/commandExecution/requestApproval`
  - `item/fileChange/requestApproval`
  - `item/tool/requestUserInput`
- For these interrupts, OpenASE should only standardize the outer envelope and preserve provider-native semantics:
  - `interrupt_id`
  - `provider`
  - `provider_request_id`
  - `kind`
  - `payload`
  - `options`
- UI must display Codex native decision options
  - for example, approve once / approve for session / deny
  - must not hardcode the product layer to only a single “Approve” button

Project Conversation turn interrupt flow:

1. `turn/start`
2. Codex runs to a point requiring approval / user input
3. Codex sends request
4. OpenASE persists the request as `pending_interrupt`
5. SSE / watch stream emits `interrupt_requested` to frontend
6. Frontend shows approval / input UI
7. User submits decision / answer
8. OpenASE calls provider-native response interface
9. Same turn continues until `turn/completed` / `turn/failed`

Note:

- This type of interrupt is a pause point in conversation runtime, not a replacement for ticket state approval in section 7.5
- Platform write operations must still go through `platform_command_proposal` or migration-period compatible `action_proposal`; they cannot be allowed through Codex tool approval alone

### 31.5 Frontend Entry Points

| Entry location | Mode | Injected context | Typical question |
|---------|------|------------|---------|
| Harness sidebar | Ephemeral Chat | Current Harness content + variable dictionary | “Help me optimize the scope-of-work description,” “Add a recovery hint when retrying” |
| Board right-side drawer | Project Conversation | Project info + ticket statistics + recent activity + conversation summary | “How is the project progressing,” “Which tickets are blocked,” “Continue our breakdown proposal” |
| Ticket detail bottom area | Project Conversation (ticket-focused) | Full ticket capsule + `OPENASE_TICKET_ID` + persistent conversation workspace | “Why did it fail,” “Split this into subtickets,” “What did the Agent do” |
| Global command palette | Ephemeral Chat | Basic project info | “Help me configure a daily 9 AM security scan cron,” “How to add a new Repo” |
| Scheduled task configuration page | Ephemeral Chat | Current cron expression | “Monday, Wednesday, Friday at 10 AM” → AI generates `0 10 * * 1,3,5` |

All entry points must include a provider selector consistent with their real runtime contract:

- `Project AI` continues to allow switching the current conversation provider among Claude Code / Codex / Gemini CLI
- `Harness AI` only shows local providers that can actually execute on this surface
- `Skill AI` only shows locally supported Codex providers currently supported by the skill refinement flow in Chapter 34

Additional interaction constraints:

- `project_sidebar` uses a regular chat experience and no longer uses a fixed turn cap as transcript UX
- `ticket_detail` no longer uses a separate ephemeral `Ticket AI`; it must open or reuse Project AI ticket-focused conversation and keep ticket-native inline affordance
- `project_sidebar` may open multiple conversation tabs in the same UI session; Project AI live transport must converge to a single multiplexed SSE built by `project_id`, then dispatch by `conversation_id` to each tab, rather than creating one stream per tab
- `project_sidebar` must clearly communicate Project AI workspace isolation semantics: isolation unit is `conversation_id`, not browser tab; the same `conversation_id` across browser tabs sees the same conversation workspace / branch
- One assistant turn in `project_sidebar` corresponds to one mutable transcript block; streaming deltas must merge into the current assistant reply, and chunk boundaries must not appear as separate bubbles
- When a turn is paused by a Codex interrupt, frontend should insert an interrupt card into transcript rather than making the user think the turn is complete
- `project_sidebar` must persistently display the current conversation workspace dirty summary, clearly labeling this as the OpenASE-managed Project AI workspace, not the user’s local checkout. Summary should at minimum cover repo-level branch/path/file-count/`+/-` totals, plus file-level status/`+/-`
- At any moment, the same `conversation_id` allows at most one active turn; multi-tab concurrency is only across different conversations
- Provider switching creates a new conversation and does not reuse the previous provider-native thread

### 31.6 AI Assistance Details in Harness Editor

Harness editor remains the highest-frequency scenario, but it remains Ephemeral Chat and does not introduce persisted transcript or tool interrupt recovery.

```text
┌──────────────────────────────────────┬───────────────────────────────┐
│ Harness Editor                        │ 💬 AI Assistant              │
│                                       │                              │
│  ---                                  │ Help me improve the “Scope of   │
│  status:                              │ Work” section and add         │
│    pickup: "Todo"                     │ test-file protection rules     │
│  ---                                  │                              │
│                                       │ AI returns structured diff     │
│                                       │ [Apply to Editor] [Keep Editing] │
└──────────────────────────────────────┴───────────────────────────────┘
```

Requirements:

- When user requests Harness edits, return structured `diff` first
- Clicking “Apply to Editor” should apply the patch directly to editor content and support undo
- Normal Harness suggestions should avoid outputting `action_proposal`

### 31.7 Conversation Management and Persistence

#### 31.7.1 Two Session Strategies

| Feature | Ephemeral Chat | Project Conversation |
|------|----------------|----------------------|
| Lifecycle | Ends after user closes sidebar / switches page | Transcript persisted; live runtime can be reclaimed when idle or budget is exceeded |
| Transcript | Not persisted by default | Persisted |
| Resume entry | Resume within same live session | Resume via `conversation_id` |
| Concurrency | At most 1 ephemeral session per user at the same time | Conversation stream is a request-owned interactive stream; automatic recovery brings up at most 1 active conversation, and each `conversation_id` has at most 1 active turn at once |
| Cost control | Lightweight default budget (for example, 10 turns / $2.00) | Transcript and live runtime are decoupled; when budget is hit, close live runtime but do not delete conversation |

#### 31.7.2 Project Conversation Persistence Model

Project Conversation needs a minimal persistence model:

| Entity | Key fields | Meaning |
|------|---------|------|
| `chat_conversations` | `id`, `project_id`, `user_id`, `source`, `provider_id`, `status`, `provider_thread_id`, `last_turn_id`, `rolling_summary`, `last_activity_at` | Stable conversation-level primary key; frontend only recognizes `conversation_id` |
| `project_conversation_principals` | `id`, `conversation_id`, `project_id`, `name`, `status`, `runtime_state`, `current_session_id`, `current_workspace_path`, `current_run_id`, `last_heartbeat_at`, `current_step_status`, `current_step_summary`, `current_step_changed_at` | First-class runtime identity of Project Conversation; has live runtime, workspace, session, and token boundaries |
| `project_conversation_runs` | `id`, `conversation_principal_id`, `conversation_id`, `project_id`, `turn_id`, `status`, `started_at`, `terminal_at`, `last_heartbeat_at`, `current_step_status`, `current_step_summary`, `input_tokens`, `output_tokens`, `reasoning_tokens`, `total_tokens`, `cost_usd` | Run record for each conversation turn; used for recovery, state display, and cost attribution |
| `project_conversation_step_events` | `id`, `conversation_run_id`, `conversation_principal_id`, `conversation_id`, `event_type`, `payload_json`, `created_at` | Step-level event stream for conversation runtime |
| `project_conversation_trace_events` | `id`, `conversation_run_id`, `conversation_principal_id`, `conversation_id`, `event_type`, `payload_json`, `created_at` | Trace/output/token/interrupt event stream for conversation runtime |
| `chat_turns` | `id`, `conversation_id`, `turn_index`, `provider_turn_id`, `status`, `started_at`, `completed_at` | One user message corresponds to one turn |
| `chat_entries` | `id`, `conversation_id`, `turn_id`, `seq`, `kind`, `payload_json` | Append-only transcript covering text / diff / platform_command_proposal / action_proposal / interrupt / result |
| `chat_pending_interrupts` | `id`, `conversation_id`, `turn_id`, `provider_request_id`, `kind`, `payload_json`, `status`, `resolved_at` | Persisted Codex-native approval / user-input interruptions |

Field semantics:

- `conversation_id` is OpenASE’s stable ID and is exposed to the frontend
- Each persisted `chat_conversation` must correspond to and only one `project_conversation_principal`
- `ProjectConversationPrincipal` is the runtime identity for Project AI; it is not a hidden Agent row in `agents` and cannot be inferred from synthetic names such as `project-ai-<conversation-id>`
- `project_conversation_principals.name` is an explicitly persisted principal name, used only for observability, audit, and token claim; it is not a compatibility convention for deriving identity from `agents` scan
- `provider_thread_id` is a provider-native recovery anchor and is stored only on the server
- `last_turn_id` is used only for provider-level diagnostics and recovery, not as a frontend primary key
- `current_session_id` / `current_workspace_path` / `current_run_id` / `last_heartbeat_at` / `current_step_*` are first-class states for live runtime recovery and UI observability, and must not be hidden only in transcript payload
- Frontend localStorage may cache only the recent tab set and active tab under current `(project_id, provider_id)`; authoritative data remains from backend conversation/list API, and only one active tab should auto-open on recovery

#### 31.7.2.1 Boundary Between Project Conversation Principal and Ticket Agent

`ProjectConversationPrincipal` and `Ticket Agent` are different domain concepts:

- `Ticket Agent` belongs to orchestration domain, binding `agent_id`, ticket current run, and scheduler / workflow dispatch semantics
- `ProjectConversationPrincipal` belongs to direct chat domain, binding `conversation_id`, conversation workspace, interrupt / reconnect / transcript recovery semantics
- Both may share runtime / token / workspace infrastructure, but neither can masquerade as the other
- New Project Conversation runtime creation path must no longer create synthetic `agents` rows as a compatibility implementation; migration code may only be used to take over legacy conversation data

#### 31.7.2.2 Principal-aware Platform Token

Project Conversation is not read-only chat. It must obtain an auditable, authorized platform token, but token semantics must be principal-aware, not agent-only.

Unified token claim should include at least:

- `principal_kind`
- `principal_id`
- `principal_name`
- `project_id`
- `ticket_id` (optional)
- `conversation_id` (optional)
- `scopes`
- `expires_at`

Where:

- `ticket_agent` tokens continue to be used for orchestration Agent runs and keep existing effective capabilities
- `project_conversation` tokens are for Project AI runtime and cannot impersonate ticket runtime by forging `ticket_id`
- `project_conversation` ticket mutation must go through project-scoped ticket routes and requires a dedicated `tickets.update` capability
- `tickets.update.self` remains a current-ticket-only capability and must not grant `project_conversation` access to ticket-runtime-only endpoints
- API authorization must check both scope and `principal_kind`
- Any ticket-runtime-only endpoint must explicitly reject `project_conversation` principal
- Workspace injection, skill injection, and platform API injection must take effect based on this principal-aware token

#### 31.7.3 Recovery Strategy

Recovery order:

1. If live runtime is still running
   - Continue directly on the same provider thread
2. If live runtime is not running but provider supports durable resume
   - Use `provider_thread_id` for provider-native resume
3. If provider-native resume is unavailable
   - Create a new thread
   - Inject `rolling_summary + recent entries + current project context`
   - Recover from the OpenASE perspective instead of discarding transcript

Frontend recovery constraints:

- `project_sidebar` may recover multiple persisted conversation tabs under the same project, but only one Project AI multiplexed SSE may be retained within a single page, regardless of how many tabs are restored
- The active tab should hydrate transcript immediately after restore; inactive tabs during restore may only patch runtime status, latest summary, unread markers, and `needsHydration`, and reconcile transcript when user switches back to that tab
- If a `conversation_id` in localStorage no longer exists, is unauthorized, or provider mismatches, remove only that invalid tab without affecting other tabs’ recovery
- Workspace diff summary must be reread after restore, turn completion, and live runtime reset / reconnect; the truth source is each repo’s actual git state in the conversation workspace, not streaming tool-call inference

Therefore Project Conversation is a model of **conversation persistence with recoverable live runtime**, not “session_id permanently equals a live process.”

### 31.8 API Endpoints

#### 31.8.1 Ephemeral Chat API

Ephemeral Chat keeps a lightweight single-turn API:

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/chat` | Start one round of Ephemeral Chat with SSE streaming |
| DELETE | `/api/v1/chat/:sessionId` | Close current ephemeral live session |

Request body:

```json
{
  "message": "Help me improve the scope-of-work description",
  "source": "harness_editor",
  "provider_id": "uuid",
  "context": {
    "project_id": "uuid",
    "workflow_id": "uuid",
    "ticket_id": "uuid"
  },
  "session_id": null
}
```

#### 31.8.2 Project Conversation API

`project_sidebar` must use conversation-based APIs instead of continuing to reuse the “POST and hang a single SSE to done” pattern.

| Method | Path | Description |
|------|------|------|
| POST | `/api/v1/chat/conversations` | Create a new project conversation |
| GET | `/api/v1/chat/conversations` | Query the current user’s conversation list or current active conversation under a project/provider/source |
| GET | `/api/v1/chat/conversations/:conversationId` | Get conversation metadata |
| GET | `/api/v1/chat/conversations/:conversationId/entries` | Fetch historical transcript |
| GET | `/api/v1/chat/conversations/:conversationId/workspace-diff` | Read current conversation workspace repo/file diff summary |
| POST | `/api/v1/chat/conversations/:conversationId/turns` | Send one user turn |
| GET | `/api/v1/chat/conversations/:conversationId/stream` | Watch real-time events for current conversation |
| POST | `/api/v1/chat/conversations/:conversationId/interrupts/:interruptId/respond` | Submit decision / answer for Codex interrupt |
| POST | `/api/v1/chat/conversations/:conversationId/action-proposals/:entryId/execute` | Confirm and execute action proposal |
| DELETE | `/api/v1/chat/conversations/:conversationId/runtime` | Close live runtime while preserving conversation and transcript |

Create conversation request example:

```json
{
  "source": "project_sidebar",
  "provider_id": "uuid",
  "context": {
    "project_id": "uuid"
  }
}
```

Send turn request example:

```json
{
  "message": "Continue our breakdown discussion about ASE-42"
}
```

Interrupt response example:

```json
{
  "decision": "approve_once",
  "answer": null
}
```

Where:

- `decision` values must follow provider-native options; OpenASE only performs outer-layer mapping
- For `requestUserInput`, `answer` may be structured JSON and is not limited to single-line text

#### 31.8.3 Stream Events

Project Conversation stream must support at least these events:

```text
event: session
data: {"conversation_id":"conv-xxx","runtime_state":"ready"}

event: message
data: {"type":"text","content":"Okay, let’s continue."}

event: message
data: {"type":"diff","file":"harness content","hunks":[...]}

event: message
data: {"type":"platform_command_proposal","summary":"Create 3 subtickets","commands":[...]}

event: interrupt_requested
data: {
  "interrupt_id":"intr-xxx",
  "provider":"codex",
  "kind":"command_execution_approval",
  "options":[{"id":"approve_once","label":"Approve Once"},{"id":"approve_for_session","label":"Approve this Session"},{"id":"deny","label":"Deny"}],
  "payload": {...}
}

event: interrupt_resolved
data: {"interrupt_id":"intr-xxx","decision":"approve_once"}

event: turn_done
data: {"conversation_id":"conv-xxx","turn_id":"turn-xxx","cost_usd":0.03}

event: error
data: {"message":"codex chat turn failed"}
```

Constraints:

- `project_sidebar` one assistant turn corresponds to one assistant transcript block
- Provider deltas can only be merged into the current block, and chunk boundaries must not appear as multiple bubbles
- `interrupt_requested` is not `turn_done`

Besides transcript stream, backend must also keep separate runtime observability:

- `ProjectConversationRun` records current turn/run status, cost, token usage, and terminal information
- `ProjectConversationStepEvent` records state transitions of step progression, interrupt, resume, done, and error
- `ProjectConversationTraceEvent` records model output, trace, tool / token / session events
- UI recovery and debugging should first read typed runtime state rather than reverse-parse transcript payload

#### 31.8.4 Audit Semantics for Proposal Execution

Platform write operations in Project Conversation default to `platform_command_proposal` + human confirmation; compatibility with `action_proposal` remains during migration. Final audit semantics must explicitly distinguish origin:

- For platform changes executed after user confirms a proposal in conversation UI, audit actor should be `user:<id> via project-conversation:<conversation_id>`
- If certain `project_conversation` direct mutation scopes are opened in the future, allowed scope, principal attribution, and test coverage must be defined separately; user confirmation semantics cannot be silently reused
- Do not silently attribute conversation-initiated changes to synthetic ticket agent
- Whether invoking service-layer mutation directly or replaying internal API during migration, the above audited origin must be explicitly written into mutation payload or equivalent audit field

### 31.9 Difference from Ticket Agent

| Dimension | Ticket Agent (orchestrator-driven) | Direct Chat (user-driven) |
|------|--------------------------|-----------------------------|
| Trigger method | Automatically dispatched by orchestrator Tick | Actively started by user in UI |
| Lifecycle | Ticket from pickup to finish | Ephemeral Chat is one-off; Project Conversation transcript persists |
| Context source | Harness Prompt + ticket description | Page / project / conversation context injected by backend |
| State management | Ticket status transition, attempt_count | Does not drive ticket state machine |
| Hook | on_claim / on_complete / on_done | None |
| Cost tracking | Recorded in ticket `cost_amount` | Recorded in project-level direct chat / conversation cost |
| Runtime identity | `Ticket Agent` / `agent_id` | `ProjectConversationPrincipal` / `conversation_id` |
| Platform operations | `ticket_agent` principal token executes directly | Default `platform_command_proposal` with user confirmation before execution; migration compatibility with `action_proposal`; token principal is `project_conversation` principal |
| Tool approval | Orchestrator run is default unattended | Project Conversation supports Codex native interrupt |
| Persistence | Layered persistence with `AgentTraceEvent + AgentStepEvent + ActivityEvent` | Ephemeral Chat not persisted by default; Project Conversation uses `ProjectConversationPrincipal + Run + StepEvent + TraceEvent + transcript` layered persistence |
| Concurrency | Multiple Agent definitions can execute tickets in parallel; same Agent definition may drive multiple AgentRuns concurrently when concurrency limit allows | Live runtime concurrency is limited by user/project/provider dimensions |
## Chapter 32 Dispatcher: Auto-assignment with Workflow

### 32.1 Design philosophy: assignment is a kind of work

In the earlier PRD, auto-assignment was two boolean switches on Project configuration (`auto_assign_workflow`, `auto_assign_agent`), and the logic was hard-coded inside the orchestrator. This violates OpenASE’s core principle—**all work is a ticket, and all behavior is defined by a Workflow.**

The correct approach: **the assignment itself is a Workflow.** A Dispatcher role Agent picks up tickets in Backlog, reads requirements, decides who should do the work, and assigns to the right role through the Platform API. No new mechanism is needed—fully reuse the existing role system + autonomous closed loop + Platform API.

### 32.2 Dispatcher Workflow

````markdown
---
status:
  pickup: "Backlog"          # Dispatcher listens to the Backlog column
  finish: "Backlog"          # After assignment, the ticket remains in Backlog (status is changed, so in practice it is no longer in Backlog)
agent:
  max_turns: 5               # Assignment does not require many rounds
  timeout_minutes: 5
  max_budget_usd: 0.50       # Assignment is inexpensive
platform_access:
  allowed:
    - "tickets.update.self"  # Update ticket status (move to another column)
    - "tickets.create"       # Can split into subtickets
    - "tickets.list"         # View other tickets to understand context
    - "tickets.link"         # Link related tickets
    - "machines.list"        # View machine resources
---

# Dispatcher — Ticket Dispatcher

You are the ticket dispatcher of the project. Your only responsibility is: **to assess tickets in the Backlog, determine which role should handle it, and assign it to the correct status column.**

## Current ticket

- Identifier: {{ ticket.identifier }}
- Title: {{ ticket.title }}
- Description: {{ ticket.description | default("No description") }}
- Priority: {{ ticket.priority }}
- Type: {{ ticket.type }}

{% if ticket.links | length > 0 %}
## Linked information
{% for link in ticket.links %}
- [{{ link.type }}] {{ link.title }}: {{ link.url }}
{% endfor %}
{% endif %}

## Available roles and corresponding destination columns

Below are the roles (Workflows) configured in the project and their pickup statuses:

{% for wf in project.workflows %}
- **{{ wf.role_name }}** ({{ wf.name }}): pickup status = "{{ wf.pickup_status }}"
  {{ wf.role_description }}
{% endfor %}

## Available machines

{% for m in project.machines %}
- **{{ m.name }}** ({{ m.host }}): {{ m.description }}
  Tags: {{ m.labels | join(", ") }}
{% endfor %}

## Assignment process

1. **Understand the requirement**: carefully read the ticket title, description, and linked information
2. **Select a role**: match the requirement type with the most suitable role
   - Feature development/bug fix → fullstack-developer (pickup: "In Development")
   - Testing-related → qa-engineer (pickup: "In Testing")
   - Documentation updates → technical-writer (pickup: "Pending Writing")
   - Security issues → security-engineer (pickup: "Pending Scan")
   - Deployment-related → devops-engineer (pickup: "Ready for Deployment")
   - Requirement unclear → do not assign; add a comment explaining missing information
3. **Determine whether splitting is needed**: if the ticket is too large (involves multiple modules/roles), split it into subtickets
4. **Confirm that the target Workflow is bound to the correct Agent**: for example, a GPU training Workflow should be bound to a Provider on a GPU machine, not manually select a machine at assignment time
5. **Execute assignment**: call Platform API to change the ticket status to the target role’s pickup status

## Assignment operations

```bash
# Assign to fullstack developer
openase ticket update ASE-{{ ticket.identifier }} --status "In Development"

# Assign to testing engineer
openase ticket update ASE-{{ ticket.identifier }} --status "In Testing"

# Split into subtickets
openase ticket create --title "Subtask 1" --parent {{ ticket.identifier }} --status "In Development"
openase ticket create --title "Subtask 2" --parent {{ ticket.identifier }} --status "In Testing"
```

## Assignment principles

- When requirements are unclear, **do not force assignment**; add a comment on the ticket saying what needs to be supplemented
- A ticket should be assigned to only one role. If multiple roles are involved, split into subtickets
- Tickets with urgent priority are assigned first
- If all suitable roles are full, comment on the ticket explaining the reason for waiting
````

### 32.3 Workflow diagram

```
User creates ticket → Backlog column
                  │
                  ▼
         Dispatcher Agent picks up
         (pickup: "Backlog")
                  │
                  ├── Assess requirement
                  │
                  ├── Requirement clear ──→ Call Platform API to change status
                  │                     │
                  │               ┌─────┼──────┬──────────┐
                  │               ▼     ▼      ▼          ▼
                  │           "In Development" "In Testing" "Ready for Deployment" "Pending Research"
                  │               │     │      │          │
                  │               ▼     ▼      ▼          ▼
                  │           coding  test   deploy   research
                  │           Agent   Agent  Agent    Agent
                  │           picks up picks up picks up picks up
                  │
                  ├── Requirement too large ──→ Split subtickets, each to corresponding column
                  │
                  └── Requirement unclear ──→ Add comment, keep ticket in Backlog
```

### 32.4 New template variables

Dispatcher needs to know which Workflows and machines exist in the project, which requires adding template variables:

| Variable | Type | Description |
|------|------|------|
| `project.workflows` | list | List of all active Workflows in the project |
| `project.workflows[].name` | string | Workflow name |
| `project.workflows[].type` | string | Workflow type |
| `project.workflows[].role_name` | string | Role name |
| `project.workflows[].role_description` | string | Role description |
| `project.workflows[].pickup_status` | string | Pickup status name |
| `project.workflows[].pickup_statuses` | list | Structured list of pickup statuses; each item includes at least `id / name / stage / color`, preventing information loss when multiple statuses are bound |
| `project.workflows[].finish_status` | string | Finish status name (for quick human readability) |
| `project.workflows[].finish_statuses` | list | Structured list of finish statuses; each item includes at least `id / name / stage / color` |
| `project.workflows[].max_concurrent` | int | Maximum concurrency |
| `project.workflows[].current_active` | int | Number of currently running tickets; defined as count of `tickets.current_run_id = agent_runs.id AND agent_runs.workflow_id = workflow.id`, not derived from ActivityEvent |
| `project.machines` | list | List of available machines in the project |
| `project.machines[].name` | string | Machine name |
| `project.machines[].host` | string | Address |
| `project.machines[].description` | string | Description |
| `project.machines[].labels` | list | Labels |
| `project.machines[].status` | string | Machine status |
| `project.machines[].resources` | object | Latest persisted resource snapshot (CPU/memory/GPU/agent environment, etc.); if the machine has not finished probing, this is an empty object `{}` |
| `project.statuses` | list | All Custom Status columns of the project |
| `project.statuses[].id` | string | Status UUID |
| `project.statuses[].name` | string | Status name |
| `project.statuses[].stage` | string | Stage corresponding to the status |
| `project.statuses[].color` | string | Color |

### 32.5 Remove hardcoded switches on Project

The two boolean fields `auto_assign_workflow` and `auto_assign_agent` on the Project table can now be removed. Whether auto-assignment is enabled or disabled is: **whether there is a Dispatcher Workflow configured with pickup "Backlog".** If present, auto-assignment runs; if absent, it does not—this is fully user-controlled, with no special toggle needed.

### 32.6 Advantages of Dispatcher

Compared with hardcoded auto-assignment logic, implementing Dispatcher with Workflow has several clear advantages:

**Customizable**: Different projects can have completely different assignment strategies. A Dispatcher in a research project may focus on experiment resource allocation, while one in a product project may focus on priority ordering. Only the Harness prompt needs modification, no code changes.

**Auditable**: All assignment actions by the Dispatcher Agent are recorded in ActivityEvent (`created_by: agent:dispatcher-01`), so you can trace why ASE-42 was assigned to security-engineer instead of fullstack-developer.

**Iterative**: The Dispatcher Harness is version-managed by the platform like any other role; you can view history, compare versions, and roll back. The refine-harness meta workflow can analyze Dispatcher’s assignment history and optimize assignment strategy.

**Closed-loop consistency**: No new mechanism is introduced; it fully reuses the existing role + Workflow + Platform API + lifecycle Hook system. The Dispatcher is simply another role—except that its "output" is ticket status changes rather than code.

### 32.7 Position in the role library

Dispatcher is added to the role library as a built-in role:

| Role | Harness | pickup | Special note |
|------|---------|--------|--------------|
| Dispatcher | `roles/dispatcher.md` | Backlog | The only role whose "output is a status change rather than code" |

The rules for the HR Advisor recommendation engine also need to be updated:

```go
// Rule: More than 10 tickets backlog and no Dispatcher → recommend
if stats.BacklogCount > 10 && !stats.HasDispatcherWorkflow {
    recs = append(recs, RoleRecommendation{
        Role:   "dispatcher",
        Reason: fmt.Sprintf("There are %d tickets in Backlog awaiting assignment. It is recommended to hire a dispatcher to enable automatic assignment.", stats.BacklogCount),
    })
}
```

---
## Chapter 33 Unified Notification System

### 33.1 Design: Event Subscriptions + Channel Adapters

The notification system consists of two parts: **subscription rules** (which events trigger notifications) and **channel adapters** (where notifications are sent). Users can freely combine “event × channel”.

```text
Event Source              Rule Matching                Channel Adapter
                              │
ticket.status_changed ──→ "Has new ticket in "In Development" column?"──→ Telegram
agent.completed ──────→ Agent has completed?────────→ WeCom
hook.failed ──────────→ Hook has failed?────────────→ Slack + Email
machine.offline ──────→ Machine went offline?───────→ WeCom
```

### 33.2 NotificationChannel Entity

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| organization_id | FK | Owning organization |
| name | String | Channel name (for example, "Ops Telegram Group", "R&D WeCom Group") |
| type | String | Channel type identifier (see adapter list) |
| config | JSONB (encrypted) | Channel configuration (`token`, `webhook_url`, etc.; stored encrypted) |
| is_enabled | Boolean | Enabled |

### 33.3 Channel Adapter

Channel adapters implement the same Go interface:

```go
// domain/notification/channel.go
type ChannelAdapter interface {
    Type() string
    Send(ctx context.Context, cfg ChannelConfig, msg Message) error
    Validate(ctx context.Context, cfg ChannelConfig) error  // Test connection
}

type Message struct {
    Title    string            // Notification title
    Body     string            // Notification body (Markdown)
    Level    string            // info / warning / error
    Link     string            // Related OpenASE page URL
    Metadata map[string]string // Extra fields (may be used by channel-specific rendering)
}
```

**Built-in Adapters:**

| Channel | type identifier | Config fields | Implementation |
|------|----------|---------|---------|
| Slack | `slack` | `webhook_url` | Incoming Webhook POST |
| Telegram | `telegram` | `bot_token`, `chat_id` | Bot API `sendMessage` |
| WeCom | `wecom` | `webhook_key` | Group bot Webhook |
| Feishu | `feishu` | `webhook_url`, `secret` | Custom bot Webhook |
| Discord | `discord` | `webhook_url` | Discord Webhook |
| Email | `email` | `smtp_host`, `smtp_port`, `from`, `to[]`, `username`, `password` | SMTP |
| Generic Webhook | `webhook` | `url`, `secret`, `headers` | POST JSON + HMAC signature |

**The universal Webhook is a universal output.** Any system that can receive HTTP POST can be used as a notification channel.

**WeCom Configuration Example:**

```yaml
# Configure via Web UI or CLI
channels:
  - name: "R&D Group"
    type: wecom
    config:
      webhook_key: "${WECOM_WEBHOOK_KEY}"  # Group robot key
```

**Telegram Configuration Example:**

```yaml
channels:
  - name: "Gary's notification Bot"
    type: telegram
    config:
      bot_token: "${TELEGRAM_BOT_TOKEN}"
      chat_id: "123456789"
```

### 33.4 NotificationRule Entity (Subscription Rules)

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning project (can also be empty, indicating an organization-level rule) |
| name | String | Rule name (for example, "Notify Telegram when new ticket enters In Development") |
| event_type | String | Event type listened to (see event list) |
| filter | JSONB | Filter condition (optional, exact match) |
| channel_id | FK | Which channel to send notifications to |
| template | Text | Message template (Jinja2, optional, with default template) |
| is_enabled | Boolean | Enabled |

**filter example:**

The `filter` key must come from structured field names in event payload / metadata, and must not depend on fuzzy matching against `message` text.

```json
// Notify only when a ticket enters the "In Development" column
{"event_type": "ticket.status_changed", "filter": {"to_status_name": "In Development"}}

// Notify only when an urgent-priority ticket is created
{"event_type": "ticket.created", "filter": {"priority": "urgent"}}

// Notify only when Hook in a specific workflow fails
{"event_type": "hook.failed", "filter": {"workflow_type": "coding"}}

// Notify when machine goes offline
{"event_type": "machine.offline", "filter": {"machine_name": "gpu-01"}}
```

### 33.5 Subscribable Event List

`NotificationRule` `event_type` has two categories:

1. **Activity-derived events**: Reuse canonical values of `ActivityEvent.event_type` directly  
2. **Notification-only alert events**: Used only for alerts/notifications, not required to be written to `ActivityEvent`

**Rules:**

- As long as a notification event has the same semantics as an `ActivityEvent`, it must reuse the same canonical name; introducing notification-only aliases is prohibited.
- Therefore, aliases such as `ticket.assigned`, `ticket.retry`, and `agent.error` are no longer used here when they overlap in activity stream semantics but use different names.

| Event Type | Category | Trigger Condition | Default Message Template |
|---------|------|---------|------------|
| `ticket.created` | Activity-derived | Ticket created | "📋 New ticket {{ ticket.identifier }}: {{ ticket.title }}" |
| `ticket.status_changed` | Activity-derived | Ticket status changed | "🔄 {{ ticket.identifier }} status changed to {{ to_status_name }}" |
| `ticket.completed` | Activity-derived | Ticket completed (reaches finish state) | "✅ {{ ticket.identifier }} has been completed" |
| `ticket.cancelled` | Activity-derived | Ticket cancelled | "🚫 {{ ticket.identifier }} has been cancelled" |
| `ticket.retry_scheduled` | Activity-derived | Platform has scheduled the next backoff retry | "🔄 {{ ticket.identifier }} will retry at {{ next_retry_at }} (attempt {{ attempt_count }})" |
| `ticket.retry_paused` | Activity-derived | Ticket retry is paused and waiting for manual handling | "⏸️ Retry for {{ ticket.identifier }} has been paused: {{ pause_reason }}" |
| `ticket.budget_exhausted` | Activity-derived | Budget exhausted, retry paused | "💰 {{ ticket.identifier }} budget exhausted (${ { cost_amount } }/${ { budget_usd } }), retry paused" |
| `agent.claimed` | Activity-derived | Agent claimed ticket | "🤖 {{ agent.name }} claimed {{ ticket.identifier }}" |
| `agent.failed` | Activity-derived | Agent execution failed | "🚨 Agent {{ agent.name }} failed executing {{ ticket.identifier }}: {{ error }}" |
| `hook.failed` | Activity-derived | Hook execution failed | "🔧 {{ hook_name }} for {{ ticket.identifier }} failed: {{ error }}" |
| `hook.passed` | Activity-derived | Hook execution passed | "✅ {{ hook_name }} for {{ ticket.identifier }} passed" |
| `pr.linked` | Activity-derived | RepoScope recorded PR link | "📝 {{ ticket.identifier }} linked PR: {{ pull_request_url }}" |
| `ticket.stalled` | Notification-only | Agent stall or long no heartbeat | "⚠️ Agent for {{ ticket.identifier }} is unresponsive" |
| `ticket.error_rate_high` | Notification-only | 3+ consecutive failures | "🔴 {{ ticket.identifier }} failed {{ consecutive_errors }} times in a row, retrying after backoff {{ backoff }}" |
| `machine.offline` | Notification-only | Machine offline | "🔴 Machine {{ machine.name }} is offline" |
| `machine.online` | Notification-only | Machine online | "🟢 Machine {{ machine.name }} is back online" |
| `machine.degraded` | Notification-only | Machine resource alert | "⚠️ Machine {{ machine.name }} disk free {{ disk_free_gb }}GB" |
| `budget.threshold` | Cost reaches threshold | "💰 Project {{ project.name }} has consumed ${{ cost_usd }}" |

### 33.6 Message Templates

Each subscription rule can define a custom message template (Jinja2 syntax), or use the default template. Template variables are tied to the event type:

```jinja
{# Custom template example: send a detailed summary when ticket completes #}
🎉 **{{ ticket.identifier }}** completed

**Title**: {{ ticket.title }}
**Role**: {{ workflow.role_name }}
**Agent**: {{ agent.name }}
**Duration**: {{ duration_minutes }} minutes
**Cost**: ${{ cost_usd }}

{% if pr_urls | length > 0 %}
**PR**:
{% for url in pr_urls %}
- {{ url }}
{% endfor %}
{% endif %}

[View details]({{ ticket.url }})
```

### 33.7 Notification Engine

The notification engine listens to the EventProvider event stream, matches subscription rules, and calls channel adapters to send:

```go
// infra/notification/engine.go
func (e *NotificationEngine) Run(ctx context.Context) {
    events, _ := e.eventBus.Subscribe(ctx, "ticket.events", "agent.events",
        "hook.events", "machine.events")

    for event := range events {
        // Find matching subscription rules
        rules, _ := e.ruleRepo.FindMatching(ctx, event.Type, event.ProjectID)

        for _, rule := range rules {
            // Check filter conditions
            if !rule.MatchesFilter(event) {
                continue
            }

            // Render message template
            msg := e.renderMessage(rule, event)

            // Send via channel adapter
            channel, _ := e.channelRepo.Get(ctx, rule.ChannelID)
            adapter := e.adapterRegistry.Get(channel.Type)

            if err := adapter.Send(ctx, channel.Config, msg); err != nil {
                e.logger.Warn("notification send failed",
                    "channel", channel.Name, "event", event.Type, "err", err)
                // No retry — notifications are best-effort and do not block the main flow
            }

            e.metrics.Counter("openase.notification.sent_total",
                provider.Tags{"channel_type": channel.Type, "event_type": event.Type}).Inc()
        }
    }
}
```

**Key design decision: notifications are “best-effort” (fire-and-forget), not blocking any main flow.** On send failure, only log and record metrics; no retries, no impact on ticket status. Notifications are supplemental, not a critical path.

### 33.8 Notifications in Workflow Hooks

In addition to configuring notifications via subscription rules, users can also call notifications directly inside Workflow hooks — the two approaches are complementary:

```yaml
# Send notifications via hook in Harness (more flexible, can execute any logic)
hooks:
  on_done:
    - cmd: 'curl -X POST "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/sendMessage" -d "chat_id=${TELEGRAM_CHAT_ID}&text=Ticket ${OPENASE_TICKET_IDENTIFIER} is completed"'
      on_failure: ignore  # Notification failure does not block
```

The difference between the two approaches:

| Dimension | NotificationRule | Direct call in Hook |
|------|---------------------------|---------------|
| Configuration | Web UI / API | Harness YAML |
| Flexibility | Standard events + filter conditions | Fully customized (any shell command) |
| Version control | Stored in database | Stored with Harness in Git |
| Suitable scenarios | General notifications ("notify when all tickets complete") | Workflow-specific notifications ("notify security team when security scan is done") |

### 33.9 Web UI

Settings → Notification page:

```text
┌────────────────────────────────────────────────────────────────────┐
│ Notifications                                                      │
├────────────────────────────────────────────────────────────────────┤
│                                                                    │
│ 📡 Notification Channels                                  [Add Channel] │
│                                                                    │
│ ┌───────────────────────────────────────────────────────────────┐  │
│ │ 🔵 Telegram: Gary's Notification Bot                           │  │
│ │    chat_id: 123456789 · Enabled                  [Test] [Edit]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ 🟢 WeCom: R&D Group                                              │  │
│ │    webhook · Enabled                              [Test] [Edit]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ 🔴 Slack: #openase-alerts                                       │  │
│ │    webhook · Disabled                             [Test] [Edit]  │  │
│ └───────────────────────────────────────────────────────────────┘  │
│                                                                    │
│ 📋 Notification Rules                                        [Add Rule] │
│                                                                    │
│ ┌───────────────────────────────────────────────────────────────┐  │
│ │ New ticket in "In Development" column → Telegram                 │  │
│ │   Event: ticket.status_changed                                     │  │
│ │   Filter: new_status = "In Development"                             │  │
│ │   Channel: Gary's Notification Bot                [Edit] [Delete]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ Ticket failed → WeCom                                              │  │
│ │   Event: ticket.error                                             │  │
│ │   Filter: none (all failures)                                      │  │
│ │   Channel: R&D Group                              [Edit] [Delete]  │  │
│ ├───────────────────────────────────────────────────────────────┤  │
│ │ GPU machine offline → Telegram + WeCom                              │  │
│ │   Event: machine.offline                                            │  │
│ │   Filter: machine_name contains "gpu"                                │  │
│ │   Channel: Gary's Notification Bot, R&D Group      [Edit] [Delete]  │  │
│ └───────────────────────────────────────────────────────────────┘  │
│                                                                    │
└────────────────────────────────────────────────────────────────────┘
```

### 33.10 API Endpoints

**Channel Management:**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/orgs/:orgId/channels` | List notification channels |
| POST | `/api/v1/orgs/:orgId/channels` | Create channel |
| PATCH | `/api/v1/channels/:channelId` | Update channel |
| DELETE | `/api/v1/channels/:channelId` | Delete channel |
| POST | `/api/v1/channels/:channelId/test` | Send test message |

**Notification Rule Management:**

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/projects/:projectId/notification-rules` | List notification rules |
| POST | `/api/v1/projects/:projectId/notification-rules` | Create rule |
| PATCH | `/api/v1/notification-rules/:ruleId` | Update rule |
| DELETE | `/api/v1/notification-rules/:ruleId` | Delete rule |

### 33.11 Impact on Provider Layer

The `NotifyProvider` interface in the earlier PRD was too simplified (only one `Send` method). After upgrading to a full notification system, `NotifyProvider` becomes an internal implementation detail of the notification engine; what is exposed externally is `NotificationEngine`, which internally manages channel adapter registration and subscription rule matching.

Provider matrix update:

| Provider | Before | After |
|----------|------|------|
| NotifyProvider | LogNotifier / SlackNotifier / WebhookNotifier | Removed, replaced by NotificationEngine + ChannelAdapter registry |
## Chapter 34 Skills Lifecycle Management

### 34.1 What is a Skill

A Skill is a native capability-extension unit of Agent CLI (such as Claude Code, Codex). In OpenASE, a Skill is correctly modeled not as a single `SKILL.md` text file, but as a **named directory-style bundle**:

- An entry file `SKILL.md` must exist
- It can include sibling files such as `scripts/`, `references/`, `assets/`, `agents/openai.yaml`
- Stable relative path relationships exist among files within the bundle
- At runtime, it must be materialized as-is by directory into the Agent CLI skills directory

Typical structure is:

```text
skill-name/
├── SKILL.md
├── agents/
│   └── openai.yaml
├── scripts/
│   └── *.sh / *.py / ...
├── references/
│   └── *.md / *.json / ...
└── assets/
    └── Any templates, images, and sample files used by the skill
```

For example:

- `commit` skill: teaches the Agent how to write conventional commit messages
- `land` skill: teaches the Agent how to safely merge PRs (rebase, squash, CI checks)
- `openase-platform` skill: teaches the Agent how to call OpenASE Platform APIs (create ticket, register Repo, etc.)
- `review-code` skill: teaches the Agent how to do code review (style, performance, security checks)
- `deploy-openase` skill: in addition to `SKILL.md`, may include `scripts/redeploy_local.sh`

To an Agent, a Skill is “what you can do”; a Harness is “what role you are.” **Binding Skills to a Workflow means which capability packages this role has in this run.**

### 34.2 Latest Design and Deprecation Notes

OpenASE now uses a **DB authoritative source + runtime materialize** model:

- Skill bundle metadata, file lists, versions, enabled state, and binding relations are stored in the control plane
- If an Agent or user wants to modify a Skill, it must call the Platform API
- When creating a new runtime, the platform materializes the current version of the Skill bundle into the Agent CLI skills directory in the workspace
- Running runtimes do not drift automatically due to background Skill updates; only new runtimes use the latest version by default, or explicit refresh/restart

> **Deprecated design**: In legacy design, Skills were stored in the project repository under `.openase/skills/` and synchronized bi-directionally between repo and workspace through `skills refresh` / `skills harvest`. This design is now deprecated. Repo branch and working tree state no longer determine the authoritative skill content.

### 34.3 Storage Model

A Skill is persisted by the platform control plane, and the version unit is the “whole directory bundle,” not a single Markdown text.

#### 34.3.1 Domain objects

- `Skill`
  - Stable identity and lifecycle object within the project
- `SkillVersion`
  - A bundle snapshot of a release
- `SkillFile`
  - A single file entry in a version
- `WorkflowSkillBinding`
  - The binding relationship between Workflow and Skill
- `RuntimeSkillSnapshot`
  - The set of SkillVersions consumed by a specific AgentRun

#### 34.3.2 Database model

**Skill**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| project_id | FK | Owning project |
| name | String | Skill name, unique within the project |
| description | String | Display description |
| is_builtin | Boolean | Whether it is a built-in Skill |
| is_enabled | Boolean | Whether enabled |
| created_by | String | Creator |
| archived_at | DateTime | Soft-delete / archive time |
| created_at | DateTime | Creation time |
| updated_at | DateTime | Last update time |

**SkillVersion**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| skill_id | FK | Owning Skill |
| version | Integer | Monotonically increasing version number |
| bundle_hash | String | Canonicalized hash for the whole bundle (not single-file hash) |
| manifest_json | JSONB | Canonicalized file list, entrypoint info, bundle metadata |
| size_bytes | BigInt | Total size of the whole bundle |
| file_count | Integer | Number of files in the bundle |
| created_by | String | Submitter of this version |
| created_at | DateTime | Version creation time |

**SkillVersionFile**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| skill_version_id | FK | Owning SkillVersion |
| path | String | Relative path within bundle; case preserved, normalized for comparison |
| file_kind | Enum | `entrypoint` / `metadata` / `script` / `reference` / `asset` |
| media_type | String | MIME / text type hint |
| encoding | Enum | `utf8` / `base64` / `binary` |
| is_executable | Boolean | Whether executable bit should be set after materialize |
| size_bytes | BigInt | File size |
| sha256 | String | Single-file content hash |
| content_blob_id | FK | Points to content blob |
| created_at | DateTime | Creation time |

**SkillBlob**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| sha256 | String | Content hash, unique |
| size_bytes | BigInt | Raw size |
| compression | Enum | `none` / `gzip` |
| content_bytes | BYTEA | Compressed or raw bytes |
| created_at | DateTime | Creation time |

**WorkflowSkillBinding**

| Field | Type | Description |
|------|------|------|
| id | UUID | Primary key |
| workflow_id | FK | Bound Workflow |
| skill_id | FK | Bound Skill |
| required_version_id | FK (nullable) | Optional pinned version; empty means follow current published version |
| rollout_mode | Enum | `follow_current` / `pin_version` |
| created_at | DateTime | Binding time |

#### 34.3.3 Why this set of tables

- `skills` stores stable identity and lifecycle state
- `skill_versions` stores bundle-level version snapshots
- `skill_version_files` stores directory path structure and file attributes
- `skill_blobs` stores actual bytes safely and deduplicated
- `workflow_skill_bindings` expresses only binding relationships, not duplicated bundle content

This guarantees:

- Correct version unit is the entire skill directory
- Database storage preserves path relationships and executable flags
- No need to force binary files into `SKILL.md`
- Materialize can reliably restore the original directory structure
- Agent run only needs to record `skill_version_ids` to fully replay the capability package for that run

### 34.4 Binding to Workflow

The binding relationship between Skill and Workflow is persisted separately in the platform control plane and `skills:` is no longer the authoritative source in Harness YAML.

Key rules:

- When editing bindings on the Workflow configuration page, call the platform binding API
- Harness body can render a “current bound Skills list” for Agent reading, but that is read-only projection, not the true config source
- If a specific Workflow needs to bind a pinned Skill version, use `required_version_id`

### 34.5 Runtime projection

The authoritative source of Skills is in DB, but Agent CLI still consumes a file directory. Therefore the platform must perform materialize when runtime starts:

1. Parse the Workflow version used by this AgentRun
2. Read the enabled Skill set currently bound to that Workflow
3. Fetch the corresponding bundle versions of these Skills and their file lists
4. Create a directory for each Skill and write files one by one into the Agent CLI skills directory of the current workspace:
   - Claude: `.claude/skills/<skill>/...`
   - Codex: `.codex/skills/<skill>/...`
   - Others: `.agent/skills/<skill>/...`
5. Record the `skill_version_ids` used in this run in `agent_runs` or associated snapshot table

Key rules:

- The runtime directory is only a projection, not a single source of truth
- Deleting the runtime directory does not lose Skill content
- Materialize may only write inside the target root; any path escape must fail
- Materialize should restore only allowlisted permission bits, such as `0644` for regular files and `0755` for executable scripts
- Whether the repo workspace already exists does not affect Skill viewing, editing, enabling, or binding; it only affects repository-related execution

### 34.6 Built-in `openase-platform` Skill

This is the most important Skill. It teaches the Agent how to operate OpenASE through the Platform API and is a core capability for Agent autonomous closed-loop operation.

Built-in Skills are initialized into the control plane during installation by the platform and are no longer copied into project repositories. When a new runtime starts, the platform materializes it to the workspace just like other Skills.

### 34.7 How to store safely

Skill bundle storage must follow the principle of “parse first, then persist; normalize first, then write to DB.”

#### 34.7.1 Ingestion flow

1. API receives a skill bundle request
   - UI/CLI can upload zip/tar.gz
   - External API can also submit normalized `files[]` structure
2. At the **import boundary**, unpack and parse into `ParsedSkillBundle`
3. Perform bundle-level validation to produce `ValidatedSkillBundle`
4. Generate normalized manifest, per-file sha256, and whole bundle `bundle_hash`
5. In one database transaction, write:
   - `skill_versions`
   - `skill_blobs`
   - `skill_version_files`
   - Update `skills.current_version_id` when necessary
6. New SkillVersion becomes visible to later runtimes only after transaction commit

#### 34.7.2 Secure storage rules

- **Path safety**
  - All file paths must be relative
  - Forbid absolute paths, empty paths, `.`, `..`, path traversal, duplicate canonicalized paths
  - Forbid Windows/Unix mixed escape tricks (such as backslashes, drive-letter prefixes)
- **File type safety**
  - Only regular files are accepted
  - Symlinks, hardlinks, device files, sockets, FIFOs are forbidden
  - External links and permission poisoning in archives are forbidden
- **Size limits**
  - Restrict per-file size, total bundle size, total file count
  - `SKILL.md` and reference documents must be within readable limits
  - Binary assets are allowed, but with stricter size thresholds
- **Encoding safety**
  - `SKILL.md` must be UTF-8 text
  - If marked as text, `agents/openai.yaml`, `references/*`, `scripts/*` must be parseable
  - Binary content is uniformly stored as blobs and not carried in string columns
- **Permission safety**
  - Do not trust raw `mode` on upload
  - DB stores only a canonicalized `is_executable` boolean
  - Materialize maps permissions to safe mode bits at the platform layer
- **Content addressing**
  - `skill_blobs.sha256` is unique, enabling deduplication
  - `skill_version_files.sha256` and `bundle_hash` are used for auditing and snapshot replay
- **Transaction consistency**
  - Manifest, file rows, and blob references of the same SkillVersion must be committed in one transaction
  - “Partial state” where version row is created but files are not fully written is forbidden

### 34.8 Where validation logic should live

Follow the Parse, Not Validate principle. Validation must be concentrated at system boundaries, not scattered as condition checks throughout business logic paths.

#### Boundary 1: HTTP / CLI / Agent Platform API input boundary

Responsibilities:

- Authentication, authorization, project ownership checks
- Request envelope parsing
- Rejecting missing required fields, format errors, and upload size over limits
- Converting archive upload to raw file stream or `RawSkillBundle`

This layer does not perform business persistence and does not directly determine Workflow binding strategy.

#### Boundary 2: Skill import parse boundary (core boundary)

Responsibilities:

- Parse `RawSkillBundle` into `ValidatedSkillBundle`
- Normalize and freeze paths
- Validate directory structure, required files, frontmatter, script paths, file counts and sizes
- Compute manifest, per-file hash, bundle hash

The domain object produced by this layer is already a “safe-to-persist skill bundle.” Subsequent services only consume this domain object.

Suggested domain types:

- `SkillName`
- `SkillBundlePath`
- `SkillBundleFile`
- `SkillBundleManifest`
- `ValidatedSkillBundle`

#### Boundary 3: Service / Use-Case layer

Responsibilities:

- Handle “create skill,” “publish new version,” “bind workflow,” and “enable/disable”
- Accept only `ValidatedSkillBundle`
- Do not re-check path traversal, frontmatter, or archive structure input constraints
- Handle only business rules, such as “name unique within project” and “built-in skill cannot be deleted directly”

#### Boundary 4: Repository / DB layer

Responsibilities:

- Use unique indexes, foreign keys, and check constraints as a safeguard
- For example:
  - `skills(project_id, name)` unique
  - `skill_versions(skill_id, version)` unique
  - `skill_version_files(skill_version_id, path)` unique
  - `skill_blobs(sha256)` unique

Repository does not shoulder bundle parsing logic.

#### Boundary 5: Runtime materialize boundary

Responsibilities:

- Write validated, persisted SkillVersions into the workspace
- Re-check target path is within root before writing
- Do not re-run business validation, only file-write safety protection

### 34.9 Can an Agent create a new Skill

It can, but must go through Platform API, not write files directly under workspace or repo and then let the platform harvest afterward.

Correct flow:

1. During execution, the Agent summarizes a reusable pattern
2. The Agent calls `create skill` or `update skill` platform capability and submits a bundle (at minimum containing `SKILL.md`)
3. The platform parses the bundle at the import boundary, validates, creates a new version, records audit logs
4. If needed, Agent or user then calls `bind skill to workflow`
5. New runtimes use the new version by default; current runtimes must explicitly refresh/restart if they should use it

### 34.10 Refresh semantics for running runtimes

`refresh` no longer means “rescan the `.openase/skills/` directory in the repo.”

The new meaning is:

- `refresh current runtime skills`
  - Re-read the **latest published bundle version** of skills currently bound to this runtime from the control plane
  - Overwrite the current runtime’s Agent CLI skills directory
  - Requires explicit user/platform-triggered action

Default behavior remains:

- New runtimes automatically take the latest version
- Running runtimes do not auto-refresh

### 34.11 Skill Editor fix-and-verify feedback loop

A Skill Editor cannot provide only “unverified suggestions.” For a draft skill bundle, the platform must provide an independent **fix-and-verify refinement loop** separate from project conversation / ticket workspace.

#### 34.11.1 Independent session

- Each `Fix and verify` creates an independent skill refinement session
- Session maintains:
  - selected Provider (Phase 1 supports Codex first)
  - an isolated temporary workspace
  - multiple retry turns on the same workspace
  - explicit runtime session closure and cleanup logic
- This is not a variant of project conversation; do not reuse ticket workspace or workflow run workspace

Suggested directory:

- `~/.openase/skill-tests/<project>/<skill>/<session>/workspace`

#### 34.11.2 Draft bundle is the source of truth

- The Skill Editor’s current draft bundle is the source of truth for refinement input
- The platform must accept full draft `files[]`, not only read the latest published version
- Before launching Codex, the platform first materializes this draft bundle to:
  - `.codex/skills/<skill>/...`
- The temporary workspace is only a projection and does not automatically overwrite control plane data

#### 34.11.3 Internal execution loop

The platform executes in this order:

1. Read editor draft bundle
2. materialize to skill test workspace
3. Start Codex session
4. Codex directly edits bundle files and runs verification commands
5. On failure, platform starts bounded retry within the same session/workspace
6. Continue until:
   - `verified`
   - `blocked`
   - `unverified` (only when runtime-backed verification is fundamentally not possible)

Key rules:

- A result cannot be marked `verified` without successful actual runtime-backed verification
- `verified` must be accompanied by final candidate bundle
- `blocked` must include a clear failure reason and verification evidence summary
- When refinement session closes or resets, platform must clean up temporary workspace and provider session

#### 34.11.4 UI return model

Skill Editor must expose this loop as a single action:

- Primary button: `Fix and verify`
- Running states must show at least:
  - `editing`
  - `testing`
  - `retrying`
  - `verified`
  - `blocked`
- Return result must include at least:
  - `status`
  - `workspace_path`
  - provider thread / turn identifiers (if available)
  - transcript / transcript summary
  - command output summary
  - final candidate files / bundle diff
  - `failure_reason`

Only a `verified` result may be one-click applied back to the current draft bundle.

### 34.11 Management operations

| Operation | Description |
|------|------|
| Create | Create Skill bundle via Web UI / CLI / Agent Platform API |
| Update | Submit a new Skill bundle to create a new version |
| Bind | Bind Skill to Workflow |
| Unbind | Unbind Skill from Workflow |
| Disable | Keep Skill but stop injecting it into new runtimes |
| Enable | Resume injection |
| Delete / archive | Skill can no longer be bound; existing running runtimes are unaffected |
| Publish to current runtime | Explicitly refresh Skill projection of current runtime |

### 34.12 Built-in Skill library

OpenASE comes with a set of built-in Skills, written into the control plane during initialization:

| Skill | Description | Bound to by default |
|-------|------|-----------|
| `openase-platform` | Platform operation capabilities (create ticket, register Repo, etc.) | Most workflows |
| `commit` | Conventional Commit standard | coding, testing |
| `push` | Git push standard (including force-push protection) | coding |
| `pull` | Git pull + rebase flow | coding |
| `create-pr` | PR creation standard (title format, description template, label) | coding |
| `land` | PR merge flow (CI checks, squash, cleanup) | coding |
| `review-code` | Code review standard | code-reviewer |
| `write-test` | Test writing standard (naming, coverage, mock strategy) | qa-engineer |
| `security-scan` | Security scan workflow (OWASP Top 10 checklist) | security-engineer |

### 34.13 Web UI — Skill management page

```
┌──────────────────────────────────────────────────────────────────┐
│ Skills Management                                 [Create Skill]  │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│ 🔍 Search Skills...                      [All] [Built-in] [Custom] [Disabled]  │
│                                                                  │
│ ┌────────────────────────────────────────────────────────────┐   │
│ │ 📦 openase-platform                          Built-in · Enabled  │   │
│ │    Platform operation capabilities (create ticket, register Repo, etc.)  │   │
│ │    Bound to: coding, testing, dispatcher, security (All)     │   │
│ │                                        [View] [Unbind]         │   │
│ ├────────────────────────────────────────────────────────────┤   │
│ │ 📦 commit                                    Built-in · Enabled  │   │
│ │    Conventional Commit standard                               │   │
│ │    Bound to: coding, testing                                 │   │
│ │                                  [View] [Bind More] [Unbind] │   │
│ ├────────────────────────────────────────────────────────────┤   │
│ │ 🤖 deploy-docker                           Custom (ASE-42) · Enabled   │   │
│ │    Standard Docker deployment flow                           │   │
│ │    Bound to: devops                                          │   │
│ │    Creator: agent:claude-01 via ASE-42                       │   │
│ │                          [View] [Edit] [Bind More] [Disable]   │   │
│ ├────────────────────────────────────────────────────────────┤   │
│ │ ⚠️ legacy-deploy                           Custom · Disabled    │   │
│ │    Old deployment flow (replaced by deploy-docker)             │   │
│ │    Bound to: (None)                                          │   │
│ │                                      [Enable] [Delete]         │   │
│ └────────────────────────────────────────────────────────────┘   │
│                                                                  │
│ Binding panel — Drag Skill to Harness                                │
│ ┌──────────────┬──────────────┬──────────────┬──────────────┐   │
│ │ coding       │ testing      │ dispatcher   │ devops       │   │
│ │ ├ platform   │ ├ platform   │ ├ platform   │ ├ platform   │   │
│ │ ├ commit     │ ├ commit     │ └             │ ├ commit     │   │
│ │ ├ push       │ ├ write-test │               │ ├ deploy-    │   │
│ │ ├ pull       │ └             │               │ │  docker   │   │
│ │ ├ create-pr  │              │               │ └             │   │
│ │ └ land       │              │               │              │   │
│ └──────────────┴──────────────┴──────────────┴──────────────┘   │
└──────────────────────────────────────────────────────────────────┘
```

The Skill detail page must display:

- Current version and version history
- bundle file tree
- `SKILL.md` preview
- whether `scripts/`, `references/`, `assets/` exist
- which Workflows it is bound to
- whether builtin / enabled

### 34.14 API endpoints

| Method | Path | Description |
|------|------|------|
| GET | `/api/v1/projects/:projectId/skills` | List skills (including binding status) |
| POST | `/api/v1/projects/:projectId/skills` | Create skill bundle (`multipart archive` or `files[]`) |
| GET | `/api/v1/skills/:skillId` | Get skill details (including current version manifest) |
| GET | `/api/v1/skills/:skillId/files` | Get current version file tree |
| GET | `/api/v1/skills/:skillId/files/*path` | Get individual file content |
| PUT | `/api/v1/skills/:skillId` | Update skill bundle, create new version |
| DELETE | `/api/v1/skills/:skillId` | Delete skill |
| POST | `/api/v1/skills/:skillId/enable` | Enable skill |
| POST | `/api/v1/skills/:skillId/disable` | Disable skill |
| POST | `/api/v1/skills/:skillId/bind` | Bind to workflow `{workflow_ids: [...]}` |
| POST | `/api/v1/skills/:skillId/unbind` | Unbind from workflow `{workflow_ids: [...]}` |
| POST | `/api/v1/agent-runs/:runId/skills/refresh` | Explicitly refresh the current runtime skill projection |
| GET | `/api/v1/skills/:skillId/history` | Get skill version history |
| GET | `/api/v1/skills/:skillId/versions/:versionId/files` | Get historical version file tree |

UI may upload zip/tar.gz, but the platform must parse the archive into a normalized file set at the import boundary before persistence.

### 34.15 CLI import semantics

CLI must support “directly provide a local directory path and import it to a specified project as a Skill bundle.”

Recommended commands:

```bash
openase skill import --project <project-id> --path ./path/to/skill-dir
openase skill import <project-id> ./path/to/skill-dir
```

Key rules:

- `--path` / positional argument must point to a local directory, not a single file
- CLI traverses the directory locally, performs basic file type checks, packages a normalized bundle, and uploads it
- Server API **does not accept** requests like “please read `/tmp/foo` directory on the server”
- CLI import defaults directory name as candidate skill name, but still uses `name` from `SKILL.md` frontmatter as authoritative
- If directory name, frontmatter `name`, and explicit CLI `--name` conflict, parse by “explicit arg > frontmatter > directory name” and require a consistent final result; inconsistent input fails
- CLI does not follow symlinks and does not upload socket, FIFO, device files
- CLI can choose two upload protocols:
  - upload zip/tar.gz archive
  - upload normalized `files[]`

Recommended minimum CLI subcommand set:

- `openase skill list <project-id>`
- `openase skill import <project-id> <dir>`
- `openase skill export <skill-id> --output ./dir`
- `openase skill bind <skill-id> --workflow <workflow-id>`
- `openase skill unbind <skill-id> --workflow <workflow-id>`
- `openase skill enable <skill-id>`
- `openase skill disable <skill-id>`

Additional notes:

- `import` is “import from local directory into platform control plane”
- `export` is “export a SkillVersion back to a local directory”
- Neither changes the principle that “platform DB is the single source of truth”

### 34.16 Skill validation rules (Fail Fast)

When writing Skill bundle, validate strictly; malformed requests are rejected directly and do not affect other Skills or normal Agent operation:

| Check item | Failure behavior |
|--------|---------|
| Skill name is empty or invalid | `400 Bad Request` |
| Missing root `SKILL.md` | `400 Bad Request` |
| `SKILL.md` is not UTF-8 text | `400 Bad Request` |
| Syntax error in `SKILL.md` frontmatter | `400 Bad Request` |
| frontmatter `name` does not match Skill name | `400 Bad Request` |
| Contains absolute path / `..` / duplicate canonicalized path | `400 Bad Request` |
| Contains symlink / device / FIFO / socket | `400 Bad Request` |
| Single file size exceeds limit | `400 Bad Request` |
| Bundle total size or file count exceeds limit | `400 Bad Request` |
| `agents/openai.yaml` exists but has invalid format | `400 Bad Request` |
| Bundle hash does not match manifest | `400 Bad Request` |

### 34.17 Impact on architecture

| Component | Change |
|------|------|
| Domain | Add `domain/skill/` (`SkillName`, `SkillBundlePath`, `ValidatedSkillBundle`, entity, repository, service) |
| Orchestrator Worker | Execute Skill bundle materialize before starting Agent; no longer perform repo harvest |
| Infrastructure | Add `infra/skill/` (archive import, blob storage, runtime materialize, format validation) |
| API | Add Skill CRUD + bind/unbind + runtime refresh endpoints |
| CLI | Add `openase skill` subcommands (list/create/enable/disable/bind) |
| Web UI | Add Skill management page under Settings |
| Database | Add `skills`, `skill_versions`, `skill_version_files`, `skill_blobs`, `workflow_skill_bindings` |
| Runtime | `.claude/skills` / `.codex/skills` in workspace become pure projections |
| Deprecations | Remove repo `.openase/skills` as source of truth; remove harvest semantics; remove “directory scan sync index” model |

### 34.18 Skills lifecycle overview

```
        Create / Update bundle
   ┌────────────┬────────────┐
   ▼            ▼            ▼
 Web UI        CLI        Agent Platform API
   │            │            │
   └────────────┼────────────┘
                ▼
      Parse RawSkillBundle at import boundary
                ▼
      Produce ValidatedSkillBundle
                ▼
   Transactionally write SkillVersion / Files / Blobs
                │
                ▼
     Bind / Enable / Publish / Pin version
                │
                ▼
     New runtime materializes that bundle version
                │
                ├── Default: only new runtime takes effect
                │
                └── Explicit: refresh current runtime
```

---
## Chapter 35 Scenario Validation

This chapter validates the current architecture across three complete real-world scenarios step by step. Each step marks the dependent PRD chapter, and unsupported items are labeled as GAP.

### 35.1 Scenario 1: Newcomer MVP — From Idea to Demo

**Persona**: Individual developer with an app idea who does not want to write much code and wants an Agent to quickly produce a demo.

**Step 1: First User Touchpoint**

```text
User downloads the openase binary → ./openase setup (or first run ./openase up)
→ Completes PostgreSQL / auth mode / runtime selection in terminal
→ setup detects that Claude Code is installed and writes to `~/.openase/*`
→ Start local OpenASE and enter the default workspace
→ Auto-generate default board columns: Backlog, Todo, In Progress, Done
```

- Dependency: Chapter 14 Onboarding, Chapter 29 Custom Status (default template)
- **Fully supported** ✅

**Step 2: User Chats Idea with AI in Sidebar**

```text
User (in the board page sidebar): "I want to make a Todo App with categories and reminders, using a Go backend + SvelteKit frontend"

AI assistant (Ephemeral Chat):
  "Great idea! Based on your needs, I suggest:
   1. Create two repositories: backend (Go) and frontend (SvelteKit)
   2. Hire two roles: Design Assistant (responsible for architecture design) + Coding Assistant (responsible for implementation)
   3. Have the design assistant output the architecture plan first, then let the coding assistant implement by modules

   [One-click project configuration]"

User clicks → AI calls Platform API via action_proposal:
  → Register backend repo + frontend repo
  → Activate architect role (Harness: roles/architect.md, pickup: "Backlog")
  → Activate coding role (Harness: roles/fullstack-developer.md, pickup: "Todo")
  → Create the first ticket: "Design technical architecture for Todo App", place it in Backlog
```

- Dependency: Chapter 31 Ephemeral Chat (AI assistant), Chapter 27 Agent Autonomous Loop (action_proposal), Chapter 26 Role System (role library)
- **Fully supported** ✅

**Step 3: Design Assistant Takes Over**

```text
Orchestrator Tick:
  → architect Workflow pickup = "Backlog"
  → Finds ticket "Design technical architecture for Todo App" in Backlog
  → on_claim lifecycle Hook (clone two empty repos)
  → Claude Code starts, injecting architect Harness

Design Assistant work:
  1. Analyze requirements, design technical architecture (API routes, data model, frontend page structure)
  2. Output architecture document to docs/architecture.md in backend repo
  3. Create coding sub-tickets via Platform API:
     → "Implement Todo CRUD API" (workflow: coding, status: "Todo")
     → "Implement frontend Todo list page" (workflow: coding, status: "Todo")
     → "Implement frontend category management page" (workflow: coding, status: "Todo")
     → "Implement reminder feature" (workflow: coding, status: "Todo")
  4. Design assistant completes → ticket moves to Done
```

- Dependency: Chapter 7 Orchestrator pickup rules, Chapter 27 Agent creates sub-tickets, Chapter 34 Skills (openase-platform skill)
- **Fully supported** ✅

**Step 4: Coding Assistant Parallel Development**

```text
Orchestrator Tick:
  → coding Workflow pickup = "Todo"
  → Finds 4 coding tickets in Todo
  → max_concurrent = 2, dispatches the first 2 (by priority)
  → on_claim lifecycle Hook (pull code, install dependencies)
  → Two Claude Code instances execute in parallel

Coding Assistant work:
  → Directly develop on main branch (no PR flow in newcomer mode)
  → on_complete Hook: "make test" (run if tests exist, skip if not)
  → Completion → ticket moves to Done
  → Orchestrator dispatches next batch
```

- Dependency: Chapter 10 Orchestrator concurrency control, Chapter 8 Hook (on_complete can be configured as warn and non-blocking)
- Note: Direct development on main branch = `git.branch_pattern` in Harness is not configured or configured as main
- **Fully supported** ✅

**Step 5: User Sees Results**

```text
User sees on board:
  Backlog (0) | Todo (2) | In Progress (2) | Done (2)

SSE real-time pushes Agent progress → User sees code being written
After all tickets are Done → User runs git pull and checks the result locally
```

- Dependency: Chapter 29 Custom Status board, SSE pushes
- **Fully supported** ✅

**Scenario 1 GAP analysis:** No architecture gaps. All steps are covered by existing chapters. The only configuration detail to note is simplified Harness settings in newcomer mode (no PR process, no approvals, tolerant hooks).

---

### 35.2 Scenario 2: Veteran Team Continuous Iteration — Self-governed, Controlled Pipeline

**Persona**: A 3–5 person team, product already launched, needs continuous iteration, high code quality requirements.

**Board configuration:**

```text
Backlog → Design → Design Review → Todo → In Progress → Code Review → Agent Review → CI/CD → Staging → Production → Done
```

- Dependency: Chapter 29 Custom Status (fully custom columns)
- **Fully supported** ✅

**Role configuration:**

| Role | Harness | pickup | finish | Description |
|------|---------|--------|--------|-------------|
| Dispatcher | dispatcher.md | Backlog | — | Evaluate requirements and assign to Design or Todo |
| Design Assistant | architect.md | Design | Design Review | Output design plans and other items for human checkpoint |
| Coding Assistant | fullstack-dev.md | Todo | Code Review | Develop features on feature branch, open PR |
| Review Assistant | code-reviewer.md | Agent Review | CI/CD | Automated code review |
| DevOps | devops.md | CI/CD | Staging | Deploy to staging environment |
| Security Engineer | security-engineer.md | Todo | Done | Scheduled full scan + incremental scan after PR merge |
| Market Analyst | market-analyst.md | — | — | Triggered by scheduled jobs, not via board |

- Dependency: Chapter 26 Role System, Chapter 32 Dispatcher, Chapter 7 Workflow pickup/finish
- **Fully supported** ✅

**Pipeline operation:**

```text
1. User writes a Backlog ticket: "Support user avatar upload"

2. Dispatcher takes over (pickup: Backlog)
   → Determines design is needed → changes status to "Design"
   → Platform API: openase ticket update --status "Design"

3. Design assistant takes over (pickup: Design)
   → Output design plan (API design, storage plan, frontend interaction draft)
   → Completion → status becomes "Design Review"

4. Human checkpoints in the "Design Review" column
   → Plan is OK → manually drag to "Todo"
   → Plan is not acceptable → drag back to "Design", add comments

5. Coding assistant takes over (pickup: Todo)
   → Multi-repo development (backend + frontend)
   → create feature branch: agent/ASE-42
   → on_complete Hook: make lint && make test && make typecheck
   → All hooks pass → open PR → status becomes "Code Review"

6. Human review at "Code Review" column
   → Rejected → drag back to "Todo", agent continues at next Tick (attempt + 1)
   → Approved → drag to "Agent Review"

7. Review assistant takes over (pickup: Agent Review)
   → Automatically check coding style, performance, security
   → Pass → status becomes "CI/CD"

8. DevOps assistant takes over (pickup: CI/CD)
   → Deploy to staging
   → on_complete Hook: curl staging health check
   → Pass → status becomes "Staging"

9. Human validates staging environment
   → OK → drag to "Production"
   → DevOps assistant takes over again → deploy to production → Done

10. Notification system pushes key events to corporate WeChat throughout
```

- Dependency: Chapter 7 pickup/finish, Chapter 8 Hook, Chapter 12 GitHub integration (PR), Chapter 25 multi-machine (staging/production deployment), Chapter 33 notification
- **Fully supported** ✅

**Market Analyst assistant (scheduled trigger):**

```text
ScheduledJob:
  cron: "0 9 * * *"  (every day at 9:00 AM)
  workflow: market-analyst
  ticket_template:
    title: "Daily market research - {{ date }}"
    description: "Collect competitor updates, industry news, user feedback"
    status: "Backlog"

Market Analyst assistant executes:
  → Search competitor updates, industry news
  → Generate research report
  → If feasible options exist → create tickets to Backlog via Platform API
  → Notify humans for checkpoint
```

- Dependency: Chapter 6 ScheduledJob, Chapter 27 Agent autonomy (ticket creation), Chapter 33 notification
- **Fully supported** ✅

**Security scanning assistant (dual mode: scheduled + event-driven):**

```text
# Mode 1: Scheduled full scan (every Monday at 9:00 AM)
ScheduledJob:
  cron: "0 9 * * 1"
  workflow: security-engineer
  ticket_template:
    title: "Weekly security scan - {{ date }}"
    description: "Full code security audit + dependency vulnerability checks"
    status: "Todo"

# Mode 2: Incremental scan after PR merge (triggered by Notification Rule)
# When ticket reaches Done and workflow_type=coding, automatically create security scan ticket
NotificationRule:
  event: ticket.completed
  filter: { workflow_type: "coding" }
  action: Use on_done Hook to auto-create security scan ticket

# Configure in coding Harness on_done Hook:
hooks:
  on_done:
    - cmd: |
        openase ticket create \
          --title "Security scan: code changes for ${OPENASE_TICKET_IDENTIFIER}" \
          --workflow security \
          --status "Todo" \
          --parent ${OPENASE_TICKET_IDENTIFIER}
      on_failure: ignore

Security scan assistant executes:
  → Scan code changes (incremental) or full repository
  → Check dependency vulnerabilities (CVE)
  → Check hardcoded secrets, injection risks, XSS, etc.
  → Generate security report
  → If high-risk vulnerability found → create urgent fix ticket to Backlog via Platform API
  → Notify security channel (Telegram / WeChat Work)

Security scanner role configuration:
  Role: security-engineer
  Harness: roles/security-engineer.md
  pickup: "Todo"
  finish: "Done"
  Skills: openase-platform, security-scan
  Notification: hook.failed → WeChat Work #security-alerts
```

- Dependency: Chapter 6 ScheduledJob (scheduled trigger), Chapter 8 Hook (on_done creates scan ticket), Chapter 26 role (security-engineer), Chapter 27 Agent autonomy (create fix ticket), Chapter 33 notification (security alerts)
- **Fully supported** ✅

**Scenario 2 GAP analysis:**

There is a small GAP: **Automatic re-entry mechanism to pickup after PR rejection**. In the current design, when a human drags a ticket back to "Todo", the coding assistant’s pickup is "Todo", and the next Tick will reassign it; however, `current_run_id` may not have been cleared yet (the previous AgentRun has finished but the field may not be reset).

Fix: The worker must clear `current_run_id` after Agent finishes (finish or fail). It should also be cleared when a ticket is manually dragged back by a human. This is already designed in Chapter 7 Section 7.4 (`t.CurrentRunID = ""`), but it must be ensured that manual drag-and-drop by humans also triggers clearing.

→ Must add: **Any `status_id` change (whether agent-automatic or human-manual) should clear `current_run_id`.** This ensures a ticket returning to any pickup column can be reassigned.

---

### 35.3 Scenario 3: Research Task — Mass Idea Generation and Validation

**Persona**: Research team with a GPU cluster, needing bulk exploration of research directions.

**Board configuration:**

```text
Idea Pool → Pending Research → Researching → Pending Experiment → Experimenting → Success → Fail → Pending Report → Reporting → Human Review → Published
```

**Machine configuration:**

| Machine | Labels | Purpose |
|---------|--------|---------|
| local | — | control plane + literature research |
| gpu-01 | gpu, a100 | model training experiments |
| gpu-02 | gpu, h100 | large model experiments |
| storage | storage, nfs | data storage |

**Role configuration:**

| Role | pickup | finish | Execution binding | Description |
|------|--------|--------|-------------------|-------------|
| Idea Producer Assistant | — | — | Bound to scheduled Agent on local | Triggered by scheduled task, outputs to "Idea Pool" |
| Dispatcher | Idea Pool | — | Bound to scheduling Agent on local | Evaluate idea feasibility, route to "Pending Research" or directly to "Pending Experiment" |
| Literature Research Assistant | Pending Research | Pending Experiment | Bound to research Agent on local | Search papers, analyze feasibility |
| Experiment Validation Assistant | Pending Experiment | Success or Fail | Bound to experimentation Agent on `gpu-01` or `gpu-02` | Write experiment code, run experiments |
| Report Writing Assistant | Pending Report | Human Review | Bound to reporting Agent on local | Write polished report |

**Operational flow:**

```text
1. Scheduled task: Idea Producer assistant runs every day at 9:00
   → ScheduledJob: cron "0 9 * * *", workflow: idea-producer
   → Idea assistant searches latest arXiv papers, analyzes trends
   → Uses Platform API to create tickets in batch to "Idea Pool":
     → "Explore: Replacing Transformer attention with Mamba"
     → "Explore: Multimodal CoT applications in code generation"
     → "Explore: LoRA fine-tuning effects on low-resource language translation"

2. Dispatcher takes over (pickup: "Idea Pool")
   → Evaluates feasibility and resource needs of each idea
   → High feasibility → change status to "Pending Research" or directly "Pending Experiment"
   → Low feasibility → add comments with reasons, keep in Idea Pool

3. Literature Research assistant takes over (pickup: "Pending Research")
   → Deep search of 15–20 related papers
   → Output literature review + experiment design plan
   → Completion → status becomes "Pending Experiment"

4. Experiment Validation assistant takes over (pickup: "Pending Experiment")
   → Workflow is bound to `experiment-runner-gpu` Agent
   → That Agent binds to Claude Code Provider on `gpu-01`
   → Orchestrator resolves binding and SSHes to gpu-01 to launch Claude Code
   → Write experiment code and run training
   → Access storage machine to store data via accessible_machines

   After experiment completion, Agent judges result:
   → Significant effect → Platform API sets status to "Success"
   → Poor effect → Platform API sets status to "Fail"

   → Successful ticket: Agent automatically creates report ticket to "Pending Report"
     openase ticket create --title "Report: Mamba Attention Experiment Results" \
       --status "Pending Report" --parent ASE-42

5. Report Writing assistant takes over (pickup: "Pending Report")
   → Read experiment results (pull data from storage machine)
   → Write report: charts, data analysis, conclusions
   → Output report.md or report.pdf
   → Completion → status becomes "Human Review"

6. Human review
   → Review report quality and research value
   → Approved → drag to "Published"
   → Needs changes → drag back to "Pending Report" + add revision comments

7. Notification system:
   → New ticket in "Success" column → Telegram notification (Experiment succeeded!)
   → New ticket in "Human Review" column → Telegram notification (Report pending review)
   → GPU machine offline → WeChat Work alert
```

- Dependency: Chapter 6 ScheduledJob, Chapter 25 multi-machine SSH, Chapter 26 Role System, Chapter 27 Agent autonomy (create ticket + status update), Chapter 32 Dispatcher, Chapter 33 notification, Chapter 34 Skills
- **Fully supported** ✅

**Batch concurrency:**

```text
There are 10 ideas in Idea Pool
Dispatcher assigns quickly (5 seconds × 10 = 50 seconds per idea set)

gpu-01 and gpu-02 each run 1 experiment (max_concurrent=1 per machine)
local machine runs 3 literature research tasks (max_concurrent=3)

→ 10 ideas processed in parallel, all results within about 2–3 hours
→ Human sees a batch of experiment reports to review the next day
```

- Dependency: Chapter 10 Orchestrator concurrency control, Chapter 25 machine-level concurrency
- **Fully supported** ✅

**Scenario 3 GAP analysis:**

A small GAP: **The experiment validation assistant needs to dynamically choose finish status as "Success" or "Fail" based on experiment results.** If finish has only one value, the orchestrator always moves to that single status and cannot represent this branching completion semantics.

Proposed fix: `workflow.finish_status_ids` is required and can contain multiple values. When the set length is `1`, the orchestrator automatically moves to that status; when the set length is greater than `1`, the Agent must explicitly choose one target status from the allowed set via `openase ticket update --status ...`. The platform must reject statuses outside the set.

→ Need to add to Chapter 7: **When there are multiple finish options, the Agent chooses from the allowed set; when there is a single finish, the orchestrator auto-completes.**

---

### 35.4 GAP Summary and Fixes

| GAP | Affected scenarios | Fix approach | Related chapters |
|-----|--------------------|--------------|------------------|
| Clear `current_run_id` when humans manually drag a ticket | Scenario 2 (PR rejection and re-claim) | Clear `current_run_id` on any `status_id` change | Chapter 7 Section 7.4 |
| Finish state needs dynamic selection (agent-driven) | Scenario 3 (different states for experiment success/failure) | When `status.finish` is empty, orchestrator does not auto-move, Agent sets status via Platform API | Chapter 7 Section 7.2 |

Both GAPs are small config-level patches and do not involve architectural changes.

### 35.5 Architecture Validation Conclusion

The three scenarios cover the vast majority of OpenASE modules:

```text
                     Newcomer MVP   Veteran Iteration   Research Task
Onboarding (14)        ✅
Ephemeral Chat (31)    ✅        ✅
Custom Status (29)     ✅        ✅         ✅
Role system (26)       ✅        ✅         ✅
Dispatcher (32)        ✅        ✅         ✅
Pickup/Finish (7)      ✅        ✅         ✅
Hook system (8)        ✅        ✅         ✅
Orchestrator (10)      ✅        ✅         ✅
Agent adapter (11)     ✅        ✅         ✅
Multi Repo (6.4)       ✅        ✅         ✅
Agent autonomy (27)    ✅        ✅         ✅
Skills (34)            ✅        ✅         ✅
GitHub integration (12)                 ✅
Approval (7.5)                      ✅         ✅
Multi-machine SSH (25)                             ✅
Scheduled jobs (6.10)                ✅         ✅
Notification system (33)             ✅         ✅
External sync (28, not implemented)  —
Observability (9)                     ✅         ✅
```

**Architecture completeness assessment: The current 35-chapter design can fully support all three scenarios; only two configuration-level micro adjustments are needed (no new entities or modules are introduced).**
## Chapter 36 Task Breakdown and Dependency Graph

This is the full PRD-based decomposition (Chapter 36) of all features, with each one labeled by ID, dependencies, and estimated effort. `→` means “must be completed first,” and `~` means “recommended to do first but not blocking.”

### 36.1 Global Dependency Graph

```
Layer 0 (base layer, no dependencies)
  F01 Go scaffolding
  F02 ent schema + migration
  F03 SvelteKit scaffolding + go:embed
  F71 lefthook pre-commit configuration → F01
  F72 golangci-lint + depguard architecture guard → F71
  F73 SvelteKit ESLint + Prettier + svelte-check → F03, F71

Layer 1 (core CRUD, depends on Layer 0)
  F04 Org/Project CRUD → F01, F02
  F05 TicketStatus custom status → F02
  F06 Ticket CRUD + dependencies → F04, F05
  F07 ProjectRepo multi-repo → F04
  F08 TicketRepoScope → F06, F07
  F09 Workflow + Harness version storage → F04
  F10 AgentProvider + Agent registration → F04

Layer 2 (orchestrator, depends on Layer 1)
  F11 Scheduler dispatch loop (Tick) → F06, F09, F10
  F12 Worker + Agent CLI subprocess management → F11
  F13 Claude Code CLI adapter → F12
  F14 Codex adapter → F12
  F15 Ticket lifecycle Hook (on_claim/on_complete/on_done/on_error) → F11, F12
  F16 Workflow Hook (on_activate/on_reload) → F09
  F17 Pickup/Finish state-driven → F05, F11
  F18 Exponential backoff retry + budget pause → F11
  F19 Stall detection + HealthChecker → F12
  F20 Harness Jinja2 rendering + variable injection → F09, F06

Layer 3 (real-time + frontend, depends on Layers 1-2)
  F21 EventProvider (ChannelBus + PGNotifyBus) → F01
  F22 SSE Hub (fan-out broadcasting) → F21
  F23 Canonical project event bus endpoint (`/projects/:projectId/events/stream`) → F22
  F24 Web UI Kanban page (custom status columns + drag-and-drop) → F03, F05, F06, F23
  F25 Web UI ticket detail page → F24
  F26 Web UI Agent console → F03, F10, F23
  F27 Web UI Workflow management + Harness editor → F03, F09

Layer 4 (onboarding, depends on Layers 2-3)
  F28 terminal-first local setup (database, auth, runtime bootstrap) → F04, F10
  F29 current-user service management (`systemd --user` on Linux, `launchd` on macOS, shared `openase up/down/restart/logs`) → F01
  F30 openase up startup flow (enter setup when no config; update service when config exists) → F28, F29
  F31 openase doctor environment diagnostics → F01, F10
  F32 Progressive unlock hints → F24

Layer 5 (Git integration, depends on Layer 2)
  F33 Multi-repo collaborative workspace (clone + branch) → F07, F08, F12
  F34 RepoScope PR link binding and display → F08, F25

Layer 6 (agent autonomy + roles, depends on Layers 2-5)
  F37 Agent Platform API (governed autonomy) → F06, F07, F12
  F38 Agent Token scope permission control → F37
  F39 openase-platform built-in Skill → F37
  F40 Skills lifecycle management (inject/harvest/bind) → F09, F12, F39
  F41 Role library (built-in Harness templates) → F09, F40
  F42 Dispatcher Workflow (automatic assignment) → F37, F41
  F43 HR Advisor recommendation engine → F41, F06

Layer 7 (notifications, depends on Layers 3-6)
  F44 NotificationChannel + ChannelAdapter (Telegram/WeCom/Slack/...) → F21
  F45 NotificationRule subscription rules → F44
  F46 Notification management UI → F03, F44, F45

Layer 8 (advanced features, depends on Layers 6-7)
  F51 Ephemeral Chat (embedded AI assistant) → F13, F06
  F52 Harness editor AI assistance (side-panel chat) → F51, F27
  F53 Harness variable dictionary API + editor autocomplete → F20, F27
  F56 ScheduledJob timed tasks → F06, F09
  F57 TicketExternalLink (multi-Issue association) → F06
  F58 Refine-Harness meta-workflow → F42, F40

Layer 9 (multi-machine, depends on Layer 2)
  F59 Machine entity + SSH connection pool → F04
  F60 Machine Monitor L1-L3 (network/resources/GPU) → F59
  F61 Machine Monitor L4-L5 (agent environment/full audit) → F60
  F62 SSH Agent Runner (remote clone + launch) → F12, F59, F33
  F63 ticket binds machine + automatic label matching → F59, F11
  F64 Agent cross-machine access (Harness injection) → F59, F20
  F65 Environment Provisioner Agent → F62, F41, F61
  F66 Machine management UI → F03, F59, F60

Layer 10 (observability + later governance)
  F67 OTel TraceProvider implementation → F01
  F68 OTel MetricsProvider implementation → F01
  F69 in-memory Metrics + Web UI dashboard → F68, F03
  F70 cost tracking (token consumption + budget) → F12, F68
  F74 Conventional Commits validation → F71

Layer 11 (enterprise + open ecosystem)
  F75 Gemini CLI adapter → F12
  F76 OIDC authentication → F04
  F77 Team / Member / Role / Permission → F04, F76
  F79 Open API + Go SDK + TypeScript SDK → F06
  F80 custom Adapter plugin system → F12
  F81 upgrade + automatic migration mechanism → F02
```

### 36.2 Task List (Topologically Sorted by Dependencies)

**Layer 0 — Foundation + engineering baseline (parallel, Weeks 1-2, F71-F73 prioritized in Week 1)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F01 | Go project scaffolding: cobra CLI + Echo routing + viper configuration + slog logging | 3d | None | 5 |
| F02 | ent Schema full definition + atlas migration + database indexes | 5d | None | 6, 20 |
| F03 | SvelteKit scaffolding + Tailwind + shadcn-svelte + `go:embed` integration | 3d | None | 5 |
| F71 | lefthook setup + Makefile | 1d | F01 | 15 |
| F72 | golangci-lint strict configuration + depguard architecture guard | 1d | F71 | 15 |
| F73 | SvelteKit ESLint + Prettier + svelte-check | 1d | F03, F71 | 15 |

**Layer 1 — Core CRUD (Weeks 2-4)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F04 | Org / Project CRUD (API + frontend pages) | 3d | F01, F02 | 6, 18 |
| F05 | TicketStatus custom status CRUD + default template generation | 3d | F02 | 29 |
| F06 | Ticket CRUD + dependencies (blocks / sub-issue) | 5d | F04, F05 | 6, 18 |
| F07 | ProjectRepo multi-repo CRUD | 2d | F04 | 6 |
| F08 | TicketRepoScope (ticket binds multiple repos) | 2d | F06, F07 | 6 |
| F09 | Workflow CRUD + Harness version storage + runtime materialize | 5d | F04 | 6, 7 |
| F10 | AgentProvider + Agent registration + automatic PATH detection | 3d | F04 | 6 |

**Layer 2 — Orchestrator (Weeks 4-7, core path)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F11 | Scheduler dispatch loop (Tick mode + pickup matching) | 5d | F06, F09, F10 | 7, 10 |
| F12 | Worker + Agent CLI subprocess management (os/exec + context) | 5d | F11 | 8, 10 |
| F13 | **Claude Code CLI adapter** (NDJSON stream + multi-turn + --resume) | 5d | F12 | 11 |
| F14 | Codex adapter (JSON-RPC over stdio) | 3d | F12 | 11 |
| F15 | Ticket Hook execution engine (shell subprocess + environment variable injection + on_failure policy) | 4d | F11, F12 | 8 |
| F16 | Workflow Hook (on_activate / on_reload) | 2d | F09 | 8 |
| F17 | Pickup/Finish state-driven + current_run_id clear rule | 2d | F05, F11 | 7 |
| F18 | Exponential backoff retry + retry_paused + budget_exhausted pause | 3d | F11 | 7, 19 |
| F19 | Stall detection + HealthChecker | 2d | F12 | 10, 19 |
| F20 | Harness Jinja2 rendering (gonja) + full variable dictionary injection | 3d | F09, F06 | 30 |

**Layer 3 — Real-time push + basic frontend (Weeks 5-8, partially parallel with Layer 2)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F21 | EventProvider interface + ChannelBus + PGNotifyBus implementation | 3d | F01 | 5 |
| F22 | SSE Hub (fan-out broadcasting + connection register/unregister) | 3d | F21 | 16 |
| F23 | Canonical project event bus HTTP endpoint (tickets / agents / hooks / activity / ticket runs) | 2d | F22 | 18 |
| F24 | **Web UI Kanban page** (custom status columns + drag-and-drop + SSE real-time updates) | 8d | F03, F05, F06, F23 | 13, 29 |
| F25 | Web UI ticket detail page (multi-repo PR link list + activity log + Hook history) | 5d | F24 | 13 |
| F26 | Web UI Agent console (status + live output stream + heartbeat) | 3d | F03, F10, F23 | 13 |
| F27 | Web UI Workflow management + Harness editor (syntax highlighting + YAML validation) | 5d | F03, F09 | 13, 30 |

**Layer 4 — Onboarding (Weeks 7-9)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F28 | **terminal-first local setup** (database preparation/check + CLI checks + auth/runtime selection + default control plane data initialization; keep legacy web compatibility entrypoint) | 5d | F04, F10 | 14 |
| F29 | current-user service management (`systemd --user` on Linux, `launchd` on macOS, shared `openase up/down/restart/logs`) | 3d | F01 | 14 |
| F30 | `openase up` startup flow (detect config → enter setup if absent / update services if present) | 2d | F28, F29 | 14 |
| F31 | openase doctor environment diagnosis | 2d | F01, F10 | 14 |
| F32 | Progressive unlock hints (milestone detection + UI banner) | 2d | F24 | 14 |

**Layer 5 — Git integration (Weeks 8-10)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F33 | **Multi-repo collaborative workspace** (clone + feature branch + branch naming convention) | 4d | F07, F08, F12 | 6, 25 |
| F34 | RepoScope PR link binding and display (manual/API/platform operations) | 3d | F08, F25 | 12, 16 |

**Layer 6 — Agent autonomy + role system (Weeks 9-12)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F37 | **Agent Platform API** (environment variable injection + API routing + token generation) | 4d | F06, F07, F12 | 27 |
| F38 | Agent Token scope permission control (Harness platform_access allowlist) | 3d | F37 | 27 |
| F39 | openase-platform built-in Skill (SKILL.md + openase CLI wrapper) | 2d | F37 | 34 |
| F40 | **Skills lifecycle** (inject into Agent CLI directory + harvest + bind/unbind + refresh) | 5d | F09, F12, F39 | 34 |
| F41 | Role library (14 built-in Harness templates + Web UI Skill management page) | 5d | F09, F40 | 26, 34 |
| F42 | **Dispatcher Workflow** (Backlog automatic role assignment) | 3d | F37, F41 | 32 |
| F43 | HR Advisor recommendation engine (rule analysis + UI recommendation panel) | 3d | F41, F06 | 26 |

**Layer 7 — Notifications (Weeks 11-14)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F44 | NotificationChannel + ChannelAdapter (Telegram / WeCom / Slack / Email / Webhook) | 5d | F21 | 33 |
| F45 | NotificationRule subscription rules (event matching + filter + Jinja2 message templates) | 3d | F44 | 33 |
| F46 | Notification management UI (channel configuration + rule configuration + test send) | 3d | F03, F44, F45 | 33 |

**Layer 8 — Advanced features (Weeks 12-16)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F51 | **Ephemeral Chat** (embedded AI assistant + context injection + action_proposal) | 5d | F13, F06 | 31 |
| F52 | Harness editor AI assistance (side-panel chat + diff application) | 3d | F51, F27 | 31 |
| F53 | Harness variable dictionary API + editor autocomplete + live preview | 3d | F20, F27 | 30 |
| F56 | ScheduledJob timed tasks (robfig/cron + ticket templates + UI) | 3d | F06, F09 | 6 |
| F57 | TicketExternalLink (multi-Issue association + API + UI) | 2d | F06 | 6 |
| F58 | Refine-Harness meta-workflow (history analysis + automatic harness optimization) | 3d | F42, F40 | 26 |

**Layer 9 — Multi-machine (Weeks 14-18)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F59 | Machine entity + SSH connection pool | 4d | F04 | 25 |
| F60 | Machine Monitor L1-L3 (ping / resources / GPU, tiered frequency) | 4d | F59 | 25 |
| F61 | Machine Monitor L4-L5 (agent environment / full audit) | 3d | F60 | 25 |
| F62 | **SSH Agent Runner** (remote git clone + Skill injection + start Agent CLI) | 5d | F12, F59, F33 | 25 |
| F63 | Provider binds Machine + Workflow binds Agent scheduling path | 2d | F59, F11 | 25 |
| F64 | Agent cross-machine access (Harness injection accessible_machines) | 2d | F59, F20 | 25 |
| F65 | Environment Provisioner Agent (SSH + preconfigured Skill to repair environment) | 3d | F62, F41, F61 | 25 |
| F66 | Machine management UI | 3d | F03, F59, F60 | 25 |

**Layer 10 — Observability + later governance (can be inserted at any time, mainly completed later)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F67 | OTel TraceProvider implementation (Span creation + export + request tracing middleware) | 3d | F01 | 9 |
| F68 | OTel MetricsProvider implementation (Counter / Histogram / Gauge + export) | 3d | F01 | 9 |
| F69 | In-memory Metrics + Web UI dashboard (ticket throughput / cost / Agent utilization) | 4d | F68, F03 | 9, 13 |
| F70 | Cost tracking (token consumption records + budget alerts + automatic circuit breaker) | 3d | F12, F68 | 9 |
| F74 | Conventional Commits validation (commit-msg hook) | 0.5d | F71 | 15 |

**Layer 11 — Enterprise + open ecosystem (Weeks 18-26)**

| ID | Task | Effort | Dependencies | PRD section |
|----|------|--------|--------------|--------------|
| F75 | Gemini CLI adapter | 3d | F12 | 11 |
| F76 | OIDC authentication (Provider implementation + terminal setup prompts) | 4d | F04 | 5, 14 |
| F77 | Team / Member / Role / Permission (RBAC) | 8d | F04, F76 | — |
| F79 | Open API docs + Go SDK + TypeScript SDK | 5d | F06 | — |
| F80 | Custom Adapter plugin system | 5d | F12 | — |
| F81 | Upgrade + automatic migration mechanism (atlas versioned migration) | 3d | F02 | 23 |

### 36.3 Critical Path

The longest dependency chain determines the shortest delivery time:

```
F01 → F02 → F04 → F06 → F11 → F12 → F13 → F33 → F62
 3d    5d    3d    5d    5d    5d    5d    4d    5d  = 40 workdays ≈ 8 weeks

Parallel paths:
F03 → F24 → F28 → F30 (frontend + onboarding = 18d)
F21 → F22 → F23 (SSE = 8d)
F71 → F72 and F03 → F73 (engineering baseline = 2-3d, prioritized in Week 1, then continuously supported throughout)
```

**Phase 1 milestone (Week 8):** F01-F32 all complete, a user can `openase setup` / `openase up` → terminal setup → create ticket → Claude Code Agent auto-coding → Kanban real-time update → ticket complete.

**Phase 2 milestone (Week 14):** Add F33-F46, enabling Git integration + agent autonomy + role library + Dispatcher + notifications.

**Phase 3 milestone (Week 18):** Add F51-F66, enabling Ephemeral Chat + timed tasks + multi-machine + approval.

**Phase 4 milestone (Week 26):** Add F67-F70 and F74-F81, enabling observability + enterprise + open API; F71-F73 were completed early as engineering baseline.

### 36.4 Parallel Development Recommendations

Assume a team of 3:

| Developer | Week 1-4 | Week 5-8 | Week 9-12 | Week 13-16 |
|-----------|----------|----------|-----------|------------|
| **Backend A** | F01, F02, F71, F72, F04, F06 | F11, F12, F13, F15 | F37, F38, F40, F42 | F59, F62, F63 |
| **Backend B** | F09, F10, F05, F07 | F14, F17, F18, F19, F20, F21 | F34, F44, F45 | F56, F54 |
| **Frontend** | F03, F73, F74, F22 exploration | F22, F23, F24, F25, F26, F27 | F28, F30, F31, F32, F41 | F46, F51, F52, F53, F55 |

---

*— END OF DOCUMENT —*
