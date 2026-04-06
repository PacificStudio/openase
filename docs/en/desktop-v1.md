# OpenASE Desktop v1 Guide

OpenASE Desktop v1 is a desktop shell around the existing local OpenASE runtime. It does **not** replace the current Go backend, SvelteKit frontend, or PostgreSQL schema. The desktop host is responsible for:

- launching `openase all-in-one` on `127.0.0.1` with a dynamic port
- polling `/healthz` and `/api/v1/healthz` before the UI opens
- surfacing startup, timeout, and unexpected-exit errors with log and data-directory actions
- exposing logs, restart, and data-directory entrypoints from the Electron shell
- packaging the Electron shell and the OpenASE binary together for macOS and Linux

## Scope And Non-goals

Desktop v1 intentionally keeps the current product shape:

- PostgreSQL stays required.
- The existing Web UI stays the primary UI.
- `openase all-in-one` stays the local service entrypoint.
- Desktop v1 does not add SQLite, auto-update, deep links, or a production embedded PostgreSQL runtime.

## Runtime Model

### Production mode

1. The desktop shell resolves the OpenASE binary from the packaged app resources.
2. The shell starts `openase all-in-one --host 127.0.0.1 --port <dynamic-port> --config <resolved-config>`.
3. The shell waits for both readiness endpoints.
4. The BrowserWindow opens `http://127.0.0.1:<dynamic-port>`.

### Development mode

1. Start the existing Vite dev server from `web/`.
2. The Electron shell still launches the Go service locally.
3. The BrowserWindow points at the Vite dev server.
4. The Vite proxy continues forwarding `/api/*` and SSE traffic to the local Go service.

## Directory And Path Contract

Desktop v1 keeps desktop-host state separate from existing OpenASE data.

| Purpose | Path |
| --- | --- |
| desktop shell source | `desktop/` |
| desktop host user data | Electron `app.getPath("userData")` or `OPENASE_DESKTOP_USER_DATA_DIR` |
| desktop host logs | Electron `app.getPath("logs")` or `OPENASE_DESKTOP_LOGS_DIR` |
| OpenASE home/data directory | `~/.openase` or `OPENASE_DESKTOP_OPENASE_HOME` |
| OpenASE config path | `~/.openase/config.yaml` or `OPENASE_DESKTOP_OPENASE_CONFIG` |
| OpenASE service stdout log | `~/.openase/logs/desktop-service.log` |
| OpenASE service stderr log | `~/.openase/logs/desktop-service.stderr.log` |

Compatibility rule: desktop v1 reuses the existing `~/.openase` data layout instead of creating a second copy of workspaces, database configuration, or service data.

## PostgreSQL Strategy For v1

Desktop v1 supports the same PostgreSQL preparation modes as the CLI and source-build flow:

- manual PostgreSQL connection
- Docker PostgreSQL prepared through `openase setup`

Required operator expectation:

- desktop startup expects a valid config file before the shell can launch OpenASE successfully
- that config must point at a reachable PostgreSQL instance
- the shell does not try to mutate your database configuration during startup

Recommended first-run preparation:

```bash
make build-web
./bin/openase setup
```

After setup creates `~/.openase/config.yaml`, the desktop shell can reuse that config.

### Managed local PostgreSQL follow-up direction

Desktop v1 only reserves the extension path. It does not ship a production managed PostgreSQL runtime yet.

The intended follow-up path is:

1. wrap the existing `internal/testutil/pgtest` / `embedded-postgres` assets in a dedicated desktop provider boundary
2. let the desktop shell provision and supervise a local PostgreSQL process under an explicit `managed-local-postgres` mode
3. keep the current OpenASE application schema unchanged while swapping only the database provisioning path
4. preserve the current manual and Docker modes as explicit alternatives

That future work should stay outside the current desktop shell lifecycle and should land behind a separate runtime/provider contract rather than leaking provisioning logic into business code.

## Commands

Run commands from the repo root.

### Install dependencies

```bash
make desktop-install
```

### Install the desktop Playwright browser

```bash
make desktop-install-browsers
```

### Development mode

Before the first desktop dev run, build the local OpenASE binary once:

```bash
make build
make desktop-install
cd desktop
OPENASE_DESKTOP_OPENASE_BIN=../bin/openase pnpm run dev
```

The dev script starts:

- `pnpm --dir ../web dev --host 127.0.0.1 --port 4174`
- the Electron shell with `OPENASE_DESKTOP_DEV_SERVER_URL=http://127.0.0.1:4174`

### Desktop unit and integration tests

```bash
make desktop-test
```

### Full desktop validation

```bash
make desktop-validate
```

This runs:

- desktop unit/integration tests
- desktop Electron E2E
- desktop package smoke

### Build the desktop bundle

```bash
make desktop-build
```

This compiles the packaged OpenASE binary into `desktop/.bundle/bin/openase` and copies the config template into `desktop/.bundle/config/`.

### Create desktop distributables

```bash
make desktop-package
```

Current package targets:

- macOS: `dmg`, `zip`
- Linux: `AppImage`, `deb`

Windows is intentionally out of scope for v1 because the current validation, packaging, and support matrix focus on macOS and Linux first.

### Package smoke

```bash
make desktop-package-smoke
```

The smoke step builds the unpacked desktop app and verifies that the packaged resources contain:

- the bundled OpenASE binary
- the config template
- the desktop bundle manifest

## Test Layers

Desktop v1 validation is deliberately layered instead of relying on manual checks only.

### Existing repo baselines retained

- `make test-backend-coverage`
- `make web-validate`
- existing service black-box scripts under `scripts/dev/`

### Desktop-specific coverage

- unit: port allocation, command assembly, health timeout, single instance, directory resolution
- integration: service lifecycle, restart behavior, window/controller flow, unexpected exit handling
- E2E: launch the Electron shell, reach the hosted page, verify API connectivity, verify the error page when config is missing
- package smoke: build unpacked app resources and verify packaged contents

## Local Packaging Checklist

Use this checklist when validating an installable build on macOS or Linux.

1. `make desktop-package`
2. Install or open the generated app package.
3. Confirm the app starts only one instance.
4. Confirm the shell shows a startup page while readiness checks are in progress.
5. Confirm the main UI loads from `127.0.0.1:<dynamic-port>`.
6. Confirm the menu actions open logs and data directories.
7. Confirm `Restart Local Service` replaces the child process cleanly.
8. Confirm app exit does not leave an orphan OpenASE process behind.
9. Confirm a second launch reuses the existing `~/.openase` config and data.

## CI Integration

The repository CI now includes a dedicated `Desktop Checks` job whenever desktop, Go, web, build, or workflow paths change in ways that can affect desktop packaging or lifecycle behavior.

That job runs:

- `make desktop-install`
- `make desktop-install-browsers`
- `make desktop-validate`

## Troubleshooting

### Config missing

If the desktop shell reports that `~/.openase/config.yaml` is missing, run:

```bash
./bin/openase setup
```

### Database not reachable

Desktop v1 does not hide PostgreSQL errors. If the service fails after config resolution, open the logs directory from the error page and verify that the configured DSN is reachable.

### Wrong binary during local development

Set `OPENASE_DESKTOP_OPENASE_BIN=../bin/openase` before `pnpm run dev` so the shell launches the repo-local binary you just built.
