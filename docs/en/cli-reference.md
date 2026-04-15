# OpenASE CLI Reference

OpenASE follows a **GitHub-style dual-layer CLI contract**: high-level resource commands for common operations, plus a raw API escape hatch for anything not yet surfaced.

## Resource Commands

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

## Raw API Escape Hatch

```bash
openase api GET  /api/v1/projects/$PID/tickets --query status_name=Todo
openase api PATCH /api/v1/tickets/$TID --field status_id=$SID
```

## Live Streams

```bash
openase watch tickets $PROJECT_ID
```

## Output Formatting

```bash
--jq '<expr>'              # JQ filter
--json field1,field2       # Select fields
--template '{{...}}'       # Go template
```

Both `--kebab-case` and `--snake_case` flag spellings are accepted.

## Agent Platform Environment

Agent workers inherit these environment variables from the workspace wrapper:

| Variable | Purpose |
|----------|---------|
| `OPENASE_API_URL` | Platform API endpoint |
| `OPENASE_AGENT_TOKEN` | Agent authentication token |
| `OPENASE_PROJECT_ID` | Current project context |
| `OPENASE_TICKET_ID` | Current ticket context |
