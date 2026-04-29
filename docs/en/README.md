# OpenASE Technical Docs — Index

This directory holds **technical contracts, rollout guides, and developer-facing references**. For end-user product docs (tickets, workflows, machines, etc.), see [`../guide/en/`](../guide/en/index.md).

## Getting Started

| Document | When to read it |
|----------|-----------------|
| [Source Build & Startup Guide](./source-build-and-run.md) | Building OpenASE from source, first-run setup, run modes, managed user service, validation |
| [Configuration Reference](./configuration.md) | Environment variables, config lookup order, authentication settings |
| [CLI Reference](./cli-reference.md) | `openase` resource commands, raw API escape hatch, live streams, agent platform env |
| [Development Guide](./development.md) | Repository layout, build/lint/test/openapi commands |

## Identity & Access Management

These three docs are layered — read top-down:

| Document | Role |
|----------|------|
| [IAM Dual-Mode Contract](./iam-dual-mode-contract.md) | Product-level contract for the two auth modes (`disabled`, `oidc`). Start here for the mental model. |
| [IAM Admin Boundaries](./iam-admin-boundaries.md) | Which surface owns which IAM control (Settings → Security vs Admin Console). |
| [OIDC & RBAC Setup](./human-auth-oidc-rbac.md) | Step-by-step OIDC provider setup, operator checklist, and RBAC configuration. |

## Remote Runtime & Transport

| Document | Role |
|----------|------|
| [WebSocket Runtime Contract](./websocket-runtime-contract.md) | Wire contract, message model, versioning, error taxonomy shared by `ws_listener` and `ws_reverse`. |
| [Remote Runtime v1 Rollout](./remote-websocket-rollout.md) | Topology selection, migration, daemon install, SSH helper diagnostics, operator guidance. |

## Agent CLI Adaptation

| Document | Role |
|----------|------|
| [Claude Code Stream Protocol](./claude-code-stream-protocol.md) | How OpenASE consumes Claude Code's streaming output. |
| [Gemini CLI Adaptation Guide](./gemini-cli-adaptation-guide.md) | Integration notes for the Gemini CLI adapter. |
| [Provider Reasoning Effort Matrix](./provider-reasoning-effort-matrix.md) | Supported reasoning-effort knobs per provider. |

## Operations

| Document | Role |
|----------|------|
| [Observability Checklist](./observability-checklist.md) | Metrics, logs, activity stream, webhook ingestion hygiene. |
| [Desktop v1](./desktop-v1.md) | Electron desktop shell workflow, packaging, test layers, PostgreSQL v1 desktop strategy. |
