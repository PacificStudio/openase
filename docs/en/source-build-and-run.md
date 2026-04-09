# OpenASE Source Build And Startup Guide

This guide covers the current repository state for building OpenASE from source, running first-time setup, and starting the platform in development or on a single host.

For Remote Runtime v1 topology selection, migration, daemon install, and troubleshooting, see [`docs/remote-websocket-rollout.md`](./remote-websocket-rollout.md).

For the Electron desktop shell workflow, packaging commands, test layers, and PostgreSQL v1 desktop strategy, see [`docs/en/desktop-v1.md`](./desktop-v1.md).

## What You Need

- Go `1.26.1` on `PATH`
- Node.js `22 LTS` or `24 LTS` plus `corepack pnpm` if you will run `make build-web` or modify files under `web/`
- PostgreSQL reachable from the machine that will run OpenASE, or Docker if you want setup to start a local PostgreSQL for you
- `git`
- Optional: `codex`, `claude`, or `gemini` on `PATH` if you want setup to seed detected agent providers

Avoid odd-numbered non-LTS Node releases such as `23.x`. The current frontend dependency set includes `engines` constraints that can cause `pnpm` to reject versions like `v23.11.1`. For predictable builds, prefer Node `22.12+` on the `22.x` line or a supported `24.x` release.

If `go` is not already on `PATH`, this workspace commonly uses one of these paths:

```bash
export PATH=$PWD/.tooling/go/bin:$HOME/.local/go1.26.1/bin:$PATH
```

## 1. Clone The Repository

```bash
git clone https://github.com/PacificStudio/openase.git
cd openase
```

If you already use GitHub SSH keys on this machine, the equivalent SSH clone is:

```bash
git clone git@github.com:PacificStudio/openase.git
cd openase
```

## 2. Build The Binary

Build the embedded frontend and Go binary together from the repo root:

```bash
make build-web
```

The equivalent explicit commands are:

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

`make build-web` only rebuilds frontend assets and then compiles the Go binary. It does **not** run `make openapi-generate` and does not refresh `api/openapi.json` or `web/src/lib/api/generated/openapi.d.ts` for you.

`make build` compiles the Go binary against whatever is currently present under `internal/webui/static/`. In a fresh checkout that means the tracked placeholder only, so the root UI will return a 503 build hint until you regenerate `web/`.

If you intentionally want to refresh the embedded frontend without using `make build-web`, run:

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

If you changed backend API contracts, or you want the committed OpenAPI artifacts refreshed before building, run this separately first:

```bash
make openapi-generate
```

The frontend build and Go build are one release unit. `vite build` refreshes files under `internal/webui/static/`, but any already-built or already-running `openase` binary continues serving the older embedded bundle until you rebuild the binary too. If browser stack traces mention chunk names that do not exist under `internal/webui/static/_app/immutable/`, first assume an old binary or cached immutable assets, rebuild `./cmd/openase`, and then hard refresh the page.

## Remote Runtime Local Validation

Use the remote-runtime-specific checks from the repo root when you touch websocket execution, daemon setup, or SSH helper guidance:

```bash
scripts/ci/remote_transport_matrix.sh
```

That fast matrix is the default repo-side validation gate for Remote Runtime v1 changes. For Linux hosts with Docker Compose, the slower end-to-end harness exercises the real listener, reverse daemon, and SSH helper flows:

```bash
make remote-runtime-container
scripts/ci/remote_runtime_container_harness.sh listener
scripts/ci/remote_runtime_container_harness.sh reverse ssh
```

Use the fast matrix first, then the container harness when you need rollout-grade validation for daemon install, helper repair, or listener reachability.

## API Contract Generation

OpenASE now keeps the backend-exported OpenAPI contract and the frontend-generated TypeScript contract under version control.

Regenerate both artifacts from the repo root with:

```bash
make openapi-generate
```

The explicit commands are:

```bash
go run ./cmd/openase openapi generate --output api/openapi.json
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run api:generate
```

To verify that committed artifacts are up to date, run:

```bash
make openapi-check
```

CI runs the same diff check and fails the PR when `api/openapi.json` or `web/src/lib/api/generated/openapi.d.ts` is stale.

Use the compiled binary for service management commands such as `up`, `down`, `restart`, and `logs`. Those commands install or control a managed user service and should point at a stable executable path, not a temporary `go run` build artifact.

## 3. Prepare PostgreSQL

You can skip this section and let `openase setup` start a local Docker-backed PostgreSQL for you. If you already have PostgreSQL and want to configure it manually, the minimum working DSN looks like this:

```yaml
database:
  dsn: postgres://openase:openase@localhost:5432/openase?sslmode=disable
```

There is no third setup-managed database path. If your user account cannot access Docker and the machine does not already provide PostgreSQL, prepare PostgreSQL first and then use the manual connection path in setup.

For macOS local development, the recommended manual path is a native PostgreSQL install managed by Homebrew or Postgres.app. Use Docker-backed setup when you already rely on Docker Desktop or Colima and want OpenASE to manage the container lifecycle for you.

Homebrew example:

```bash
brew install postgresql@16
brew services start postgresql@16
psql postgres://localhost:5432/postgres?sslmode=disable -c "SELECT 1;"
```

If you prefer to manage config by hand instead of using setup, start from the sample config:

```bash
cp config.example.yaml ./config.yaml
```

Config lookup order for most commands is:

1. `--config <path>`
2. `./config.yaml`, `./config.yml`, `./config.json`, `./config.toml`
3. `~/.openase/config.yaml`, `~/.openase/config.yml`, `~/.openase/config.json`, `~/.openase/config.toml`
4. `OPENASE_*` environment variables and built-in defaults

Useful overrides:

```bash
export OPENASE_DATABASE_DSN=postgres://openase:openase@localhost:5432/openase?sslmode=disable
export OPENASE_SECURITY_CIPHER_SEED=shared-cluster-seed
export OPENASE_SERVER_PORT=19836
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s
export OPENASE_LOG_FORMAT=json
```

When you migrate an existing OpenASE database between environments with different PostgreSQL DSNs, set the same `OPENASE_SECURITY_CIPHER_SEED` everywhere. Otherwise OpenASE falls back to its legacy DSN-derived seed for GitHub credential encryption, and previously stored encrypted credentials will no longer decrypt across environments.

### Docker PostgreSQL Example On A Non-default Port

For local bring-up, a simple Docker-backed PostgreSQL instance on port `15432` looks like this:

```bash
docker run -d \
  --name openase-local-pg \
  --restart unless-stopped \
  -e POSTGRES_DB=openase_local \
  -e POSTGRES_USER=openase \
  -e POSTGRES_PASSWORD=change-me \
  -p 127.0.0.1:15432:5432 \
  -v openase_local_pgdata:/var/lib/postgresql/data \
  postgres:16-alpine
```

Then point OpenASE at it:

```bash
export OPENASE_DATABASE_DSN='postgres://openase:change-me@127.0.0.1:15432/openase_local?sslmode=disable'
```

If `docker` fails with `permission denied while trying to connect to the docker API`, the current login session may not have picked up the `docker` group yet. A practical workaround is:

```bash
sg docker -c 'docker ps'
```

## 4. Run First-Time Setup

Launch the interactive terminal setup:

```bash
./bin/openase setup
```

The default flow stays inside the terminal and does not open a browser. It walks through:

- choosing a database source:
  - start a local Docker PostgreSQL automatically
  - enter an existing PostgreSQL connection manually by supplying host, port, database, user, password, and `sslmode`
- validating the chosen database connection inside setup
- checking local CLI availability and version probes for `git`, `codex`, `claude`, and other built-in provider CLIs
- choosing the browser auth mode:
  - `disabled`
  - `oidc`, with in-flow prompts for issuer URL, client ID, client secret, redirect URL, scopes, and bootstrap admins
- choosing how OpenASE should run after setup:
  - config-only
  - install/update the current-user managed service: `systemd --user` on Linux, `launchd` on macOS
- writing `~/.openase/config.yaml`
- writing `~/.openase/.env` with the generated platform auth token
- creating `~/.openase/logs/` and `~/.openase/workspaces/`
- initializing the default local organization, project, ticket statuses, and detected provider seed data

Successful setup does all of the following:

- tests database connectivity and migrates the schema
- writes `~/.openase/config.yaml`
- writes `~/.openase/.env`
- creates `~/.openase/logs/` and `~/.openase/workspaces/`
- does not require a repo path, repo URL, default branch, or mode selection
- does not scaffold repo-local `.openase/` assets during setup
- can install the managed OpenASE service directly when the local platform supports `systemd --user` or `launchd`
- can write runnable OIDC browser-login settings directly into the generated config

When you choose Docker-backed PostgreSQL, setup uses predictable defaults:

- container: `openase-local-postgres`
- volume: `openase-local-postgres-data`
- database: `openase`
- user: `openase`
- host port: `127.0.0.1:15432`

Setup generates the PostgreSQL password automatically, validates the container-backed connection, and prints reuse / stop / remove commands after success.

If Docker is unavailable to your user account, setup does not fall back to another local database mode. In that case, bring your own PostgreSQL first and choose the manual connection path.

If you choose OIDC mode during setup, the flow points to the dual-mode IAM
contract ([`docs/en/iam-dual-mode-contract.md`](./iam-dual-mode-contract.md))
and the OIDC setup guide ([`docs/en/human-auth-oidc-rbac.md`](./human-auth-oidc-rbac.md)).
The runtime is intended for standard OIDC providers such as Auth0 or Azure
Entra ID.

## 5. Start OpenASE

### Single-process mode

This is the default path for a local bring-up:

```bash
./bin/openase all-in-one --config ~/.openase/config.yaml
```

By default the setup-generated config listens on `127.0.0.1:19836`, so the control plane is available at:

```text
http://127.0.0.1:19836
```

You can override the bind address or scheduler interval at startup:

```bash
./bin/openase all-in-one --config ~/.openase/config.yaml --host 0.0.0.0 --port 40023 --tick-interval 2s
```

### Env-only local mode

If you want local-only configuration to live in `~/.openase/.env` instead of a config file, export the variables into the current shell before starting OpenASE:

```bash
set -a
source ~/.openase/.env
set +a

./bin/openase all-in-one
```

Important: the CLI reads `OPENASE_*` environment variables, but it does not automatically source `~/.openase/.env` for an interactive shell launch. If you skip the `source` step, the process will start with defaults or fail because required values such as `OPENASE_DATABASE_DSN` are missing.

### Split-process mode

Run these when you want the API server and orchestrator as separate processes:

```bash
./bin/openase serve --config ~/.openase/config.yaml
./bin/openase orchestrate --config ~/.openase/config.yaml
```

## 6. Managed User Service

Setup can install the managed service inline when you choose the service runtime mode. You can also apply or refresh it later with:

```bash
./bin/openase up --config ~/.openase/config.yaml
./bin/openase logs --lines 100
./bin/openase restart
./bin/openase down
```

`up` runs setup only when no config file can be found. Otherwise it installs or updates a managed service that executes:

```text
openase all-in-one --config <resolved-config-path>
```

On supported platforms this uses the repo's user-service abstraction for the local platform. The service reads `~/.openase/.env`. On Linux it runs as a `systemd --user` unit and `openase logs` reads from `journalctl`; on macOS it runs as a `launchd` LaunchAgent and `openase logs` tails `~/.openase/logs/openase.stdout.log` plus `~/.openase/logs/openase.stderr.log`.

Important long-running deployment notes:

- The managed user service only runs OpenASE itself. It does not manage PostgreSQL for you.
- If you use an existing PostgreSQL instance, keep that database running separately.
- If setup created a Docker PostgreSQL container, that container is still separate from the OpenASE service.

Service manager details:

| Platform | Manager | Definition path | Inspect | Restart | Stop | Logs |
|----------|---------|-----------------|---------|---------|------|------|
| Linux | `systemd --user` | `~/.config/systemd/user/openase.service` | `systemctl --user status openase` | `systemctl --user restart openase` | `systemctl --user stop openase` | `journalctl --user -u openase -n 200 -f` |
| macOS | `launchd` | `~/Library/LaunchAgents/com.openase.plist` | `launchctl print gui/$(id -u)/com.openase \|\| launchctl print user/$(id -u)/com.openase` | `launchctl kickstart -k <target>` | `launchctl bootout <target>` | `tail -n 200 -f ~/.openase/logs/openase.stdout.log ~/.openase/logs/openase.stderr.log` |

On macOS, setup prefers the `gui/<uid>/com.openase` target and falls back to `user/<uid>/com.openase` when that is the active user domain. Use whichever target `launchctl print` succeeds with for `kickstart` and `bootout`.

On Linux servers where OpenASE should stay up after you log out, enable lingering for the user account:

```bash
loginctl enable-linger "$USER"
```

## 7. Validate The Installation

Recommended validation sequence after build or doc-driven startup changes:

```bash
./bin/openase version
./bin/openase doctor --config ~/.openase/config.yaml
./bin/openase serve --help
./bin/openase orchestrate --help
./bin/openase all-in-one --help
```

`doctor` checks config loading, local CLI availability, PostgreSQL reachability, and the `~/.openase/` layout produced by setup.

For a live instance, also verify the HTTP health endpoints directly:

```bash
curl -fsS http://127.0.0.1:19836/healthz
curl -fsS http://127.0.0.1:19836/api/v1/healthz
```

## 8. Common Operational Notes

- `make build-web` is the safe source-build path for refreshing embedded UI assets before compiling the Go binary, but it does not run `make openapi-generate`.
- Run `make openapi-generate` separately when backend API contracts changed or when you need to refresh the committed OpenAPI and TypeScript artifacts.
- Rebuild `web/` before compiling if you changed the Svelte app, otherwise the binary will still embed the old frontend output.
- `make build` only compiles the Go binary against the current contents of `internal/webui/static/`; with only the tracked placeholder present, the root UI will serve a 503 guidance response until you rebuild `web/`.
- If Docker-backed setup fails, check whether Docker is installed, the daemon is running, the selected port is free, and the container name is unused.
- If Docker is not installed or your user cannot access the Docker daemon, setup does not provide another local database fallback; use a pre-existing PostgreSQL instance instead.
- `up` should be run from a compiled binary path you intend to keep, because the managed service stores the executable path it was installed with.
- `serve`, `orchestrate`, and `all-in-one` all accept `--config`, and `serve` / `all-in-one` also accept host and port overrides.
- If `all-in-one` fails with `bind: address already in use`, inspect the current listener with `lsof -nP -iTCP:<port> -sTCP:LISTEN`.
- When running from a local `.env`, keep the file permissions tight, for example `chmod 600 ~/.openase/.env`.
- Local log files are typically kept under `~/.openase/logs/`.
