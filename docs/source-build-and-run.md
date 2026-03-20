# OpenASE Source Build And Startup Guide

This guide covers the current repository state for building OpenASE from source, running first-time setup, and starting the platform in development or on a single host.

## What You Need

- Go `1.26.1` on `PATH`
- PostgreSQL reachable from the machine that will run OpenASE
- `git`
- A local checkout of the primary repository you want OpenASE to manage
- Optional: `npm` only when you modify files under `web/`
- Optional: `codex`, `claude`, or `gemini` on `PATH` if you want setup to seed detected agent providers

## 1. Clone The Repository

```bash
git clone git@github.com:BetterAndBetterII/openase.git
cd openase
```

## 2. Build The Binary

For normal source builds, the committed UI assets in `internal/webui/static/` are already embedded, so Go is enough:

```bash
go build -o ./bin/openase ./cmd/openase
```

Only rebuild the frontend when you changed files under `web/`:

```bash
npm --prefix web install
npm --prefix web run build
go build -o ./bin/openase ./cmd/openase
```

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

## 8. Common Operational Notes

- `go build ./cmd/openase` is enough for normal backend-only work because the embedded UI assets are already checked in.
- Rebuild `web/` before compiling if you changed the Svelte app, otherwise the binary will still embed the old frontend output.
- `setup` requires the primary repo path to be a real Git repository. A plain directory is rejected.
- `up` should be run from a compiled binary path you intend to keep, because the managed service stores the executable path it was installed with.
- `serve`, `orchestrate`, and `all-in-one` all accept `--config`, and `serve` / `all-in-one` also accept host and port overrides.
