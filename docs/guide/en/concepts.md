# Core Concepts

## The Mental Model

OpenASE lets you manage a team of AI agents the same way an engineering manager manages a team of developers — except your "developers" are AI, and you scale by writing better instructions, not by hiring more people.

The key insight: **you are the manager, not the fixer.** When an agent does poor work, the answer is not to jump in and patch the code yourself — it is to improve the Harness (the agent's instructions) so that *future tickets* benefit too. That is how you scale from supervising one agent to coordinating many of them in parallel.

```
You (the manager)
 │
 │  write better instructions, not better code
 ▼
Harness  ──→  Agent  ──→  Code changes, PRs, test results
```

## The Four Core Concepts

You only need to understand four things to use OpenASE:

```
┌─────────────────────────────────────────────────┐
│                                                 │
│   Ticket  ────→  Workflow  ────→  Agent         │
│   "what"         "how"           "who"          │
│                                     │           │
│                                     ▼           │
│                                  Machine        │
│                                  "where"        │
│                                                 │
└─────────────────────────────────────────────────┘
```

| Concept | One-liner | Analogy |
|---------|-----------|---------|
| **Ticket** | A task to be done | A Jira ticket you'd assign to a developer |
| **Workflow** | Instructions for how to do it | The role description + onboarding doc you give a new hire |
| **Agent** | The AI that does the work | A developer on your team |
| **Machine** | The environment where work happens | The developer's laptop / dev server |

That is the core model. Everything else — Skills, Scheduled Jobs, Activity, and Project AI — builds on top of these four.

### Ticket — "What needs to be done"

A ticket describes a unit of work: fix a bug, build a feature, write tests. You write it the same way you would write a task for a human developer — clear context, expected outcome, relevant code locations.

The ticket does not say *how* to do the work. That is the Workflow's job.

### Workflow — "How to do it"

A Workflow is the bridge between a ticket and an agent. It contains:

- A **Harness**: a Markdown document that tells the agent its role, approach, and acceptance criteria
- **Status bindings**: which ticket statuses cause the workflow to pick work up, and which statuses it can finish into
- **Configuration**: concurrency limits, timeouts, retry behavior

The Harness is where you invest your time. A vague Harness ("please complete this task") produces vague results. A specific Harness ("you are a backend engineer; read the failing test, trace the root cause, fix it, run the test suite, open a PR") produces more consistent results.

### Agent — "Who does it"

An Agent is a registered AI provider (Claude Code, Codex, Gemini CLI) that executes work. You register it once, bind it to workflows, and it automatically picks up matching tickets.

You can run **multiple agents in parallel**, each bound to different workflows — one for coding, one for testing, one for review. That is where the leverage appears: not one AI doing everything sequentially, but a small team of specialists working at the same time.

### Machine — "Where it runs"

A Machine is the execution environment — a server, container, or your local machine where the agent actually runs commands, reads code, and writes files. Register at least one before you expect agents to execute tickets automatically.

## Building on Top: Skills

Once you have the four core concepts working, **Skills** let you extract reusable capabilities:

| Without Skills | With Skills |
|----------------|-------------|
| Copy-paste the same Git workflow instructions into every Harness | Create a "Git PR" skill once, bind it to any workflow |
| Each workflow reinvents deployment steps | Create a "Deploy" skill once and reuse it across deployment workflows |

Skills are optional, but they become valuable as your workflow count grows.

## Project AI: Your Planning Partner

You do not have to write every ticket from scratch. **[Project AI](./project-ai.md)** is the persistent synchronous assistant in the project sidebar — talk to it like a colleague to clarify requirements, inspect code in an isolated workspace, and draft or create tickets.

Typical flow:

```
Fuzzy idea  ──→  Chat with Project AI  ──→  Well-structured tickets
                                              │
                                              ▼
                                        Ticket Agents execute
```

Project AI is strongest as the planning and exploration layer: analyzing requirements, inspecting code, reviewing diffs, and helping you author tickets, Harnesses, or Skills. Ticket Agents handle the queued asynchronous execution pipeline. Depending on the project's **Project AI Platform** access settings, Project AI can also write back to project state such as tickets, workflows, skills, or updates.

## The Core Loop: Manage Through Harnesses

This is the most important idea in OpenASE:

```
 ┌──────────────────────────────────────────────────────┐
 │                                                      │
 │    Create ticket (manually or via Project AI)        │
 │        │                                             │
 │        ▼                                             │
 │    Agent executes automatically                      │
 │        │                                             │
 │        ├── Good result?  ──→  Move on to next ticket │
 │        │                                             │
 │        └── Bad result?   ──→  Improve the Harness    │
 │                                     │                │
 │                                     │                │
 │              ALL future tickets benefit              │
 │                                                      │
 └──────────────────────────────────────────────────────┘
```

**Do not** manually fix the agent's output and move on. That fixes one ticket but wastes the lesson. Instead:

1. Look at what went wrong
2. Update the Harness with clearer instructions, better constraints, or explicit steps
3. Let the agent re-run (or handle the next similar ticket correctly)

This is how you go from "babysitting one AI" to "managing a fleet." The Harness is your leverage — every improvement multiplies across future executions.

### Anti-patterns to avoid

| Anti-pattern | Why it hurts | What to do instead |
|---|---|---|
| Manually fixing agent output every time | You become the bottleneck; agents never improve | Improve the Harness so the agent gets it right next time |
| One giant Workflow for everything | Hard to debug, hard to parallelize | Split into focused workflows: coding, testing, review, deploy |
| Skipping the Harness ("just do the ticket") | The agent guesses your standards and gets them wrong | Write a clear Harness — it pays back on every ticket |
| Not setting concurrency / timeouts | Agents pile up or run forever | Set `MaxConcurrent` and `TimeoutMinutes` in every workflow |

## Walk-through: From Zero to Parallel AI Team

Here is a concrete example — you have a Python backend and want AI agents to handle bug fixes and tests in parallel.

### 1. One-time setup (Infrastructure)

- **Settings**: create statuses such as `To Do → In Progress → In Review → Done`, connect your GitHub repo
- **Machines**: register your dev server (or use the local machine)
- **Agents**: register two agents — for example "Claude-Coder" and "Claude-Tester"

### 2. Define how work gets done (Templates)

- **Workflow "Bug Fix"**: bind to Claude-Coder, write a Harness:
  ```markdown
  # Role
  You are a backend engineer working on a Python project.

  # Task
  Fix the bug described in this ticket:
  - Title: {{ ticket.title }}
  - Description: {{ ticket.description }}

  # Steps
  1. Read the relevant code and reproduce the issue
  2. Identify the root cause
  3. Write a fix with a regression test
  4. Run the full test suite
  5. Open a PR with a clear description

  # Constraints
  - Do not change public API signatures without noting it in the PR
  - All tests must pass before opening the PR
  ```
- **Workflow "Test Suite"**: bind to Claude-Tester, with a Harness focused on writing and running tests
- **Skills**: bind a shared "Git PR" skill to both workflows so they follow the same branching conventions

### 3. Daily use (Execution)

- Create tickets — write them manually or tell [Project AI](./project-ai.md) what you need and let it draft or create them for you
- Each ticket triggers its workflow's agent **automatically** when it reaches a pickup status
- Both agents work **in parallel** on their respective tickets
- Watch progress in **Activity**, review PRs when they land

### 4. Continuous improvement

- Claude-Coder keeps forgetting to run `ruff check`? Add it to the Harness under Constraints.
- Claude-Tester writes tests that miss edge cases? Add explicit edge-case expectations to its Harness.
- Every Harness improvement benefits future tickets — you are managing a system, not doing the work yourself.

## Concept Map

For reference, here is how everything fits together:

```
Settings ── define statuses, connect repos
   │
   ├── Machine ── register where agents run
   │
   ├── Agent ── register AI providers
   │
   ├── Skill ── create reusable capability packs (optional)
   │      │
   │      ▼
   ├── Workflow ── bind Agent + Harness + Skills + status triggers
   │      │
   │      ▼
   ├── Ticket ── create tasks, link to Workflow → agent executes automatically
   │      │
   │      ▼
   ├── Scheduled Job ── auto-create tickets on a timer (optional)
   │
   ├── Activity ── real-time execution logs
   │
   ├── Updates ── manual project announcements
   │
   └── Project AI ── persistent assistant for planning, workspace inspection, and ticket drafting
```

## Next Steps

- [Quick Start](./startup.md) — set up and run your first ticket in 5 steps
- [Project AI](./project-ai.md) — your interactive assistant for planning and operating project state
- [Workflows](./workflows.md) — deep-dive into Harness writing
- [Skills](./skills.md) — extract reusable patterns from your workflows
