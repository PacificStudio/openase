# OpenASE Source Build And Startup Guide

This guide covers the current repository state for building OpenASE from source, running first-time setup, and starting the platform in development or on a single host.

## What You Need

- Go `1.26.1` on `PATH`
- PostgreSQL reachable from the machine that will run OpenASE
- `git`
- A local checkout of the primary repository you want OpenASE to manage
- Optional: `pnpm` via `corepack pnpm` when you modify files under `web/`
- Optional: `codex`, `claude`, or `gemini` on `PATH` if you want setup to seed detected agent providers

If `go` is not already on `PATH`, this workspace commonly uses one of these paths:

```bash
export PATH=$PWD/.tooling/go/bin:/home/yuzhong/.local/go1.26.1/bin:$PATH
```

## 1. Clone The Repository

```bash
git clone git@github.com:BetterAndBetterII/openase.git
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
corepack pnpm --dir web run api:generate
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

`make build` compiles the Go binary against whatever is currently present under `internal/webui/static/`. In a fresh checkout that means the tracked placeholder only, so the root UI will return a 503 build hint until you regenerate `web/`.

If you intentionally want to refresh the embedded frontend without using `make build-web`, run:

```bash
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run api:generate
corepack pnpm --dir web run build
go build -o ./bin/openase ./cmd/openase
```

The frontend build and Go build are one release unit. `vite build` refreshes files under `internal/webui/static/`, but any already-built or already-running `openase` binary continues serving the older embedded bundle until you rebuild the binary too. If browser stack traces mention chunk names that do not exist under `internal/webui/static/_app/immutable/`, first assume an old binary or cached immutable assets, rebuild `./cmd/openase`, and then hard refresh the page.

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

Start from the sample config if you want a repo-local config file:

```bash
cp openase.example.yaml ./openase.yaml
```

Update `database.dsn` to a reachable PostgreSQL instance. The minimum working value looks like:

```yaml
database:
  dsn: postgres://openase:openase@localhost:5432/openase?sslmode=disable
```

Config lookup order for most commands is:

1. `--config <path>`
2. `./openase.yaml`, `./openase.yml`, `./openase.json`, `./openase.toml`
3. `~/.openase/config.yaml`
4. `~/.openase/openase.yaml`, `~/.openase/openase.yml`, `~/.openase/openase.json`, `~/.openase/openase.toml`
5. `OPENASE_*` environment variables and built-in defaults

Useful overrides:

```bash
export OPENASE_DATABASE_DSN=postgres://openase:openase@localhost:5432/openase?sslmode=disable
export OPENASE_SERVER_PORT=19836
export OPENASE_ORCHESTRATOR_TICK_INTERVAL=2s
export OPENASE_LOG_FORMAT=json
```

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

Launch the setup wizard:

```bash
./bin/openase setup --host 127.0.0.1 --port 19836
```

Then open `http://127.0.0.1:19836/setup` if the browser does not open automatically.

The wizard currently asks for:

- workspace mode: `personal`, `team`, or `enterprise`
- PostgreSQL host, port, database, user, password, and SSL mode
- primary project name
- primary repo path: must already exist locally and contain `.git/`
- primary repo URL: optional
- default branch
- detected agent CLIs to seed as providers

Successful setup does all of the following:

- tests database connectivity and migrates the schema
- writes `~/.openase/config.yaml`
- writes `~/.openase/openase.yaml` for legacy config discovery
- writes `~/.openase/.env` with the generated platform auth token
- creates `~/.openase/logs/` and `~/.openase/workspaces/`
- scaffolds `.openase/` assets inside the selected primary repo

The primary repo scaffold currently includes:

- `.openase/harnesses/coding.md`
- `.openase/harnesses/roles/*.md`
- `.openase/skills/*/SKILL.md`
- `.openase/bin/openase`

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

After setup has created a config file, you can let OpenASE install a per-user background service:

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

On supported platforms this uses the repo's user-service abstraction for the local platform. The service reads `~/.openase/.env` and writes logs under `~/.openase/logs/`.

## 7. Validate The Installation

Recommended validation sequence after build or doc-driven startup changes:

```bash
./bin/openase version
./bin/openase doctor --config ~/.openase/config.yaml
./bin/openase serve --help
./bin/openase orchestrate --help
./bin/openase all-in-one --help
```

`doctor` checks config loading, git availability, detected agent CLIs, PostgreSQL reachability, `~/.openase/` layout, harness files, and referenced hook scripts.

For a live instance, also verify the HTTP health endpoints directly:

```bash
curl -fsS http://127.0.0.1:19836/healthz
curl -fsS http://127.0.0.1:19836/api/v1/healthz
```

## 8. Common Operational Notes

- `make build-web` is the safe source-build path because it regenerates the embedded UI before compiling the Go binary.
- Rebuild `web/` before compiling if you changed the Svelte app, otherwise the binary will still embed the old frontend output.
- `make build` only compiles the Go binary against the current contents of `internal/webui/static/`; with only the tracked placeholder present, the root UI will serve a 503 guidance response until you rebuild `web/`.
- `setup` requires the primary repo path to be a real Git repository. A plain directory is rejected.
- `up` should be run from a compiled binary path you intend to keep, because the managed service stores the executable path it was installed with.
- `serve`, `orchestrate`, and `all-in-one` all accept `--config`, and `serve` / `all-in-one` also accept host and port overrides.
- If `all-in-one` fails with `bind: address already in use`, inspect the current listener with `lsof -nP -iTCP:<port> -sTCP:LISTEN`.
- When running from a local `.env`, keep the file permissions tight, for example `chmod 600 ~/.openase/.env`.
- Local log files are typically kept under `~/.openase/logs/`.
