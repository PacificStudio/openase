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
