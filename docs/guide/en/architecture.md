# Module Architecture

## How Modules Collaborate

```
Settings ──→ Define statuses, connect repositories
  │
  ├─→ Machines ──→ Register execution environments
  │
  ├─→ Agents ──→ Register AI executors
  │
  ├─→ Skills ──→ Create reusable skill packs
  │     │
  │     ▼
  ├─→ Workflows ──→ Define execution templates (bind Agent + Skills + status triggers)
  │     │
  │     ▼
  ├─→ Tickets ──→ Create tickets (link to Workflow)
  │     │                    │
  │     │         ┌──────────┘
  │     ▼         ▼
  │   Scheduled Jobs ──→ Auto-create tickets on schedule
  │
  ├─→ Activity ──→ Auto-record all events
  │
  └─→ Updates ──→ Manually publish project progress
```

## Typical Workflow

A complete work cycle follows these layers:

### 1. Infrastructure Layer (One-time Setup)

```
Settings → Configure statuses and repositories
Machines → Register execution environments
Agents   → Register AI Providers
```

### 2. Template Layer (Create as Needed)

```
Skills    → Define reusable skill packs
Workflows → Create execution templates, bind agents, skills, and status triggers
```

### 3. Execution Layer (Daily Use)

```
Tickets        → Manually create tickets to trigger agent execution
Scheduled Jobs → Auto-create tickets on a timer
```

### 4. Observation Layer (Ongoing Monitoring)

```
Activity → View system events in real-time
Updates  → Manually record project progress
```

## Data Flow

```
                    ┌─────────────┐
                    │  Scheduled  │
                    │    Jobs     │
                    └──────┬──────┘
                           │ auto-create
                           ▼
┌──────────┐      ┌─────────────┐      ┌─────────────┐
│   User   │─────→│   Ticket    │─────→│  Workflow    │
└──────────┘ create└──────┬──────┘ link  └──────┬──────┘
                           │                     │ includes
                           │ claim               ▼
                           ▼            ┌─────────────┐
                    ┌─────────────┐     │   Skills    │
                    │   Agent     │◄────┘
                    └──────┬──────┘ invoke
                           │
                           │ execute on
                           ▼
                    ┌─────────────┐
                    │  Machine    │
                    └──────┬──────┘
                           │
                           │ generate events
                           ▼
                    ┌─────────────┐
                    │  Activity   │
                    └─────────────┘
```

## Remote Runtime v1 Boundaries

Remote machine execution now has a single runtime plane:

- local machines stay on `local_process`
- remote machines execute over websocket only
- `ws_listener` means the control plane dials the machine's advertised listener directly
- `ws_reverse` means `openase machine-agent run` keeps a reverse websocket session open and carries runtime envelopes through that channel
- SSH bootstrap and SSH diagnostics remain outside the execution plane as helper-only operations

That separation matters operationally:

- machine topology decides who dials whom
- websocket runtime contract decides how commands, processes, and artifacts flow
- SSH helper commands repair or bootstrap remote access, but ticket execution does not fall back to SSH
