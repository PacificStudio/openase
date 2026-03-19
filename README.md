# OpenASE

Issue-driven automated software engineering platform.

## Scaffold Status

This repository now includes the Layer 0 runtime scaffold described in `OpenASE-PRD.md` Chapter 5:

- `cobra` CLI entrypoint with `serve`, `orchestrate`, `all-in-one`, and `version`
- `Echo` HTTP server with baseline health routes
- `viper` configuration loading from defaults, env vars, and an optional config file
- `slog` structured logging with text or JSON output

## Quick Start

```bash
go run ./cmd/openase version
go run ./cmd/openase serve
go run ./cmd/openase all-in-one --tick-interval 2s
```

Default configuration can be copied from [`openase.example.yaml`](./openase.example.yaml). You can also configure the process via environment variables such as:

```bash
export OPENASE_SERVER_PORT=41000
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s
export OPENASE_LOG_FORMAT=json
```

## Routes

- `GET /`
- `GET /healthz`
- `GET /api/v1/healthz`
