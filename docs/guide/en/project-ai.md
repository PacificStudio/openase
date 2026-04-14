# Project AI

## What Is This?

Project AI is the **persistent, interactive AI assistant** that lives in the project sidebar. While Ticket Agents work asynchronously on assigned tickets, Project AI is the AI you talk to directly — more like a senior engineer paired with you inside its own workspace.

Think of it this way:

| | Ticket Agents | Project AI |
|---|---|---|
| **Mode** | Asynchronous — tickets trigger execution | Synchronous — you converse in real time |
| **Trigger** | Ticket status + workflow pickup rules | You open a conversation and send a message |
| **Scope** | One ticket run at a time | Project-wide context plus one conversation workspace |
| **Best at** | Executing defined work | Exploring, planning, drafting, inspecting, and editing |

One important boundary: what Project AI is allowed to change inside a project depends on that project's **Project AI Platform** access settings. By default it gets the full project-conversation scope set, but admins can narrow that down.

## When to Use Project AI

- **Requirements are still fuzzy** — talk through ideas before committing to tickets
- **You need to draft or create tickets** — describe the work in plain language and turn it into structured tickets
- **You want to edit Harnesses or Skills** — iterate interactively instead of rewriting docs by hand
- **You need to inspect project state** — review ticket context, agent runs, activity, repo diffs, or machine health
- **You want to work directly in an isolated workspace** — browse files, search code, switch branches, and make focused edits

## Capabilities

### Read project state

- Ticket details, dependencies, recent activity, and hook history
- Workflow metadata and Harness content
- Skill bundles and workflow bindings
- Machine health, host, and connection summary
- Workspace branch state and diff summary for the current conversation

### Create and manage work

- **Draft or create tickets** from conversation when the project grants ticket-write access
- Update tickets and comments while you refine scope or record findings
- Read or publish project updates when project-level write access is enabled

### Author automation

- Review and edit Workflows and Harnesses
- Create, import, update, enable, disable, and bind Skills when those scopes are allowed
- Validate Harness edits before handing them back to Ticket Agents

### Work inside the conversation workspace

- Browse the repo tree, search paths, and preview files
- Edit and save files in the conversation's isolated workspace
- Inspect the git diff summary for the workspace
- Switch a workspace repo to another branch when the repo is clean
- Open a terminal session for deeper inspection or one-off commands

### Operate project state

- Update project metadata or repo bindings when the corresponding scopes are enabled
- Inspect agent state from ticket focus
- Interrupt the currently running ticket agent from ticket-focused context when an active run is in progress

## Isolated Workspaces

Each Project AI conversation tab gets its own **isolated workspace**. This means:

- Multiple tabs can work on different tasks simultaneously
- Each tab keeps its own branch and file state
- Workspace diffs are scoped to that one conversation, so reviewing changes stays predictable

This is the same isolation model that lets parallel agent runs avoid stepping on each other.

## Context-Aware Focus

Project AI can automatically carry structured focus from supported parts of the UI. Today there are **five** focus types:

| Focus | When it activates | What gets injected |
|-------|-------------------|-------------------|
| **Workflow** | Viewing a workflow editor | Workflow metadata, Harness path, selected area, unsaved-draft hint |
| **Skill** | Viewing a skill editor | Skill name, selected file, bound workflows, unsaved-draft hint |
| **Ticket** | Viewing a ticket | Ticket details, dependencies, recent activity, current run, assigned agent |
| **Machine** | Viewing a machine | Machine host, status, and health summary |
| **Workspace File** | Working in Project AI's workspace browser/editor | Repo path, file path, selection context, dirty draft state, working set |

You usually do not need to paste raw context into the chat first — the focus capsule already carries the most relevant state.

## Best Practices

### Use Project AI as the planning layer

The most effective pattern is:

1. **Start with a fuzzy idea** — discuss it with Project AI
2. **Converge on an approach** — let it inspect the codebase, compare options, and call out constraints
3. **Draft or create tickets** — capture the outcome as structured work
4. **Let Ticket Agents execute** — the asynchronous pipeline takes over from there

Project AI is best for exploration and decision-making; Ticket Agents are best for repeatable queued execution.

### Use Project AI to improve Harnesses

When a Ticket Agent produces poor results, open Project AI and:

1. Review the ticket context, run state, and diffs
2. Identify what the Harness was missing
3. Edit the Harness or related Skill with clearer instructions
4. Let the next ticket run benefit from that improvement

### Use workspace focus for precise edits

If you are already inside Project AI's workspace browser, focus a specific file or selection before asking for help. That gives the assistant the exact patch target, surrounding context, and dirty-draft state.

## Tips

- Project AI conversations are persisted, so you can resume earlier threads
- Different tabs can use different providers (for example Claude Code, Codex, or Gemini CLI)
- Project AI writes back to OpenASE only through the scopes allowed by the project's **Project AI Platform** settings
- Use multiple tabs when you want parallel investigations without sharing a workspace
