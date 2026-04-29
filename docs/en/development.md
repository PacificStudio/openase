# OpenASE Development Guide

This page documents repository layout, build commands, quality gates, and tests for contributors. For end-user installation and run modes, see [`source-build-and-run.md`](./source-build-and-run.md).

## Repository Layout

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

## Build Commands

```bash
make hooks-install        # Set up git hooks (lefthook)
make check                # Run formatting + backend coverage checks
make build-web            # Build frontend assets + Go binary (does not refresh OpenAPI artifacts)
make build                # Build Go binary only (uses existing frontend)
make run                  # Run API server in dev mode
make doctor               # Run local environment diagnostics
```

## Frontend Quality Gates

```bash
make web-format-check     # Prettier formatting
make web-lint             # ESLint checks
make web-check            # Svelte type checking
make web-validate         # All of the above
```

## OpenAPI Contract

```bash
make openapi-generate     # Regenerate api/openapi.json + TS types
make openapi-check        # Verify committed artifacts are up-to-date
```

## Testing

```bash
make test                        # Go test suite
make test-backend-coverage       # Full backend tests + coverage gate
make lint                        # Lint changes since origin/main
make lint-all                    # Full lint suite
```
