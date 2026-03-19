# OpenASE

Issue-driven automated software engineering platform.

## Scaffold Status

This repository now includes the Layer 0 base slice described in `OpenASE-PRD.md` Chapter 5:

- `cobra` CLI entrypoint with `serve`, `orchestrate`, `all-in-one`, and `version`
- managed service commands: `up`, `down`, `restart`, and `logs`
- `Echo` HTTP server with baseline health routes and an embedded frontend mount
- `viper` configuration loading from defaults, env vars, and an optional config file
- `slog` structured logging with text or JSON output
- `web/`: SvelteKit 2 + Tailwind CSS 4 + shadcn-svelte source components
- `internal/webui/static/`: prerendered frontend assets embedded into the Go binary via `go:embed`

## Build

```bash
npm --prefix web install
npm --prefix web run build
go build ./cmd/openase
```

## Quick Start

```bash
go run ./cmd/openase version
go run ./cmd/openase serve
go run ./cmd/openase all-in-one --tick-interval 2s
go run ./cmd/openase up --config ./openase.example.yaml
```

Default configuration can be copied from [`openase.example.yaml`](./openase.example.yaml). You can also configure the process via environment variables such as:

```bash
export OPENASE_SERVER_PORT=41000
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s
export OPENASE_LOG_FORMAT=json
```

## Managed Service Lifecycle

`openase up` installs or updates a per-user service definition that launches `openase all-in-one` from the current binary. On Linux it writes `~/.config/systemd/user/openase.service`; on macOS it writes `~/Library/LaunchAgents/com.openase.plist`.

Use the CLI wrappers to manage the background service without remembering platform-specific commands:

```bash
openase up --config ~/.openase/openase.yaml
openase down
openase restart
openase logs --lines 100
```

## Routes

- `GET /` serves the embedded SvelteKit UI
- `GET /healthz`
- `GET /api/v1/healthz`

If you modify the web app, rebuild the embedded assets before compiling or running the Go binary:

```bash
npm --prefix web run build
go run ./cmd/openase serve
```
