# OpenASE Configuration Reference

This page documents environment variables, config-file lookup order, and authentication settings for OpenASE. For source build and startup steps, see [`source-build-and-run.md`](./source-build-and-run.md).

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `OPENASE_SERVER_PORT` | `19836` | HTTP server port |
| `OPENASE_DATABASE_DSN` | — | PostgreSQL connection string (**required**) |
| `OPENASE_SECURITY_CIPHER_SEED` | empty | Optional shared encryption seed for GitHub credential storage; set this explicitly when different environments must read the same encrypted records |
| `OPENASE_ORCHESTRATOR_TICK_INTERVAL` | `5s` | Orchestrator polling interval |
| `OPENASE_LOG_FORMAT` | `text` | Log format (`text` or `json`) |
| `OPENASE_LOG_LEVEL` | `info` | Log level |

`OPENASE_SECURITY_CIPHER_SEED` maps to `security.cipher_seed` in config files. If unset, OpenASE keeps the legacy behavior and derives the GitHub credential cipher seed from `database.dsn`.

## Config File Lookup Order

1. `--config <path>` flag
2. `./config.yaml` (or `.yml`, `.json`, `.toml`)
3. `~/.openase/config.yaml`
4. `OPENASE_*` environment variables + built-in defaults

## Authentication

- Fresh local installs use one-time **local bootstrap links** for browser authorization; they no longer expose anonymous admin access.
- **OIDC** remains the long-term browser auth path for shared, team, and networked deployments.
- If an active OIDC rollout breaks login, use `openase auth break-glass disable-oidc`, then re-enter through `openase auth bootstrap create-link --return-to /admin/auth --format text`.

OIDC supports standard providers: Auth0, Azure Entra ID, and any OpenID Connect compliant IdP.

See also:

- IAM Dual-Mode Contract — [English](./iam-dual-mode-contract.md) | [中文](../zh/iam-dual-mode-contract.md)
- OIDC & RBAC Guide — [English](./human-auth-oidc-rbac.md) | [中文](../zh/human-auth-oidc-rbac.md)
