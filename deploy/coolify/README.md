# OpenASE Coolify deployment

This is the env-driven, non-interactive container deployment path for OpenASE.

The container entrypoint starts `openase all-in-one` directly. It does **not** run
`openase setup`, does **not** require a TUI, and does **not** require manually entering
the container after boot.

## What this deployment expects

- An external PostgreSQL database.
- Environment variables supplied by Coolify or another container platform.
- A persistent volume mounted at `/var/lib/openase` so OpenASE can keep its local state.
- Optional OIDC settings if you are not using the default `disabled` auth mode.

This ticket does **not** add an in-container PostgreSQL service.

## Required environment variables

### Always required

- `OPENASE_DATABASE_DSN`: PostgreSQL DSN for the OpenASE control plane database.

Example:

```env
OPENASE_DATABASE_DSN=postgres://openase:secret@postgres.internal:5432/openase?sslmode=disable
```

### Common optional variables

- `OPENASE_SERVER_HOST`: defaults to `0.0.0.0` and should usually stay that way in containers.
- `OPENASE_SERVER_PORT`: defaults to `40023`.
- `OPENASE_PUBLISH_PORT`: host-side published port in the sample compose file; defaults to `40023`.
- `OPENASE_SECURITY_CIPHER_SEED`: optional shared encryption seed for GitHub credential storage; set the same value across local and Coolify deployments if they need to read the same encrypted rows.
- `OPENASE_ORCHESTRATOR_TICK_INTERVAL`: defaults to `5s`.
- `OPENASE_EVENT_DRIVER`: defaults to `auto`.
- `OPENASE_LOG_LEVEL`: defaults to `info`.
- `OPENASE_LOG_FORMAT`: defaults to `json`.
- `OPENASE_AUTH_MODE`: defaults to `disabled`; set to `oidc` when enabling browser login.

### Required when `OPENASE_AUTH_MODE=oidc`

- `OPENASE_AUTH_OIDC_ISSUER_URL`
- `OPENASE_AUTH_OIDC_CLIENT_ID`
- `OPENASE_AUTH_OIDC_CLIENT_SECRET`
- `OPENASE_AUTH_OIDC_REDIRECT_URL`

### Optional OIDC variables

- `OPENASE_AUTH_OIDC_SCOPES` (comma-separated; default `openid,profile,email,groups`)
- `OPENASE_AUTH_OIDC_ALLOWED_EMAIL_DOMAINS` (comma-separated)
- `OPENASE_AUTH_OIDC_BOOTSTRAP_ADMIN_EMAILS` (comma-separated)
- `OPENASE_AUTH_OIDC_SESSION_TTL` (default `8h`)
- `OPENASE_AUTH_OIDC_SESSION_IDLE_TTL` (default `30m`)

## Minimal startup example

Build the image from the repository root:

```bash
docker build -t openase:local .
```

Run it with env only:

```bash
docker run --rm \
  -p 40023:40023 \
  -e OPENASE_DATABASE_DSN='postgres://openase:secret@postgres.internal:5432/openase?sslmode=disable' \
  -e OPENASE_SECURITY_CIPHER_SEED='shared-cluster-seed' \
  -v openase-data:/var/lib/openase \
  openase:local
```

Health checks:

```bash
curl -fsS http://127.0.0.1:40023/healthz
curl -fsS http://127.0.0.1:40023/api/v1/healthz
```

## Coolify deployment

Use `deploy/coolify/docker-compose.yml` as the reference compose file.

Recommended flow in Coolify:

1. Create a new service with the repository connected.
2. Choose the Docker Compose deployment mode.
3. Point Coolify at `deploy/coolify/docker-compose.yml`.
4. Set at least `OPENASE_DATABASE_DSN` in the environment UI.
5. If the database was migrated from another environment and you need existing encrypted GitHub credentials to keep working, set the same `OPENASE_SECURITY_CIPHER_SEED` value used by the source environment.
6. Keep the persistent volume mounted at `/var/lib/openase`.
7. If using OIDC, set `OPENASE_AUTH_MODE=oidc` and add the four required OIDC env vars.
8. Expose the service on the same internal port as `OPENASE_SERVER_PORT`.

If your Coolify setup prefers a Dockerfile-based service instead of a compose service, you can use the repository root `Dockerfile` directly and copy the same environment variables from this document.

## Files in this deployment path

- `Dockerfile`: multi-stage image build for the web assets and Go binary.
- `.dockerignore`: keeps the build context focused.
- `deploy/coolify/entrypoint.sh`: validates env, prepares directories under `HOME`, and starts `openase all-in-one`.
- `deploy/coolify/docker-compose.yml`: Coolify-ready compose reference.

## Common failure modes

### `required environment variable OPENASE_DATABASE_DSN is missing`

The container entrypoint failed before startup. Add `OPENASE_DATABASE_DSN` in Coolify.

### `OPENASE_AUTH_MODE must be either disabled or oidc`

`OPENASE_AUTH_MODE` is set to an unsupported value. Use `disabled` or `oidc` only.

### OIDC mode fails before startup

When `OPENASE_AUTH_MODE=oidc`, the entrypoint requires:

- `OPENASE_AUTH_OIDC_ISSUER_URL`
- `OPENASE_AUTH_OIDC_CLIENT_ID`
- `OPENASE_AUTH_OIDC_CLIENT_SECRET`
- `OPENASE_AUTH_OIDC_REDIRECT_URL`

If any of them is missing, the container exits immediately instead of falling into a partial setup state.

### Database connection errors during boot

The entrypoint only checks that the DSN exists. If OpenASE then exits with a PostgreSQL connection or migration error:

- verify the hostname, port, username, password, database name, and `sslmode` in `OPENASE_DATABASE_DSN`
- confirm the PostgreSQL instance accepts connections from the Coolify host
- confirm the target database already exists

### GitHub credentials stopped decrypting after a database migration

OpenASE encrypts stored GitHub credentials with a seed derived from `OPENASE_SECURITY_CIPHER_SEED` when it is set, or from `OPENASE_DATABASE_DSN` for legacy setups. If you moved the same database between environments with different DSNs, set the same `OPENASE_SECURITY_CIPHER_SEED` value in every environment that must read those credentials.

### Health check stays unhealthy

Check the logs first. Typical causes are:

- the published port does not match `OPENASE_SERVER_PORT`
- reverse proxy or service port configuration in Coolify points to the wrong internal port
- the process exited due to missing OIDC or database settings

### Volume permission errors

The container runs as a non-root `openase` user and writes under `/var/lib/openase`.
If that path is mounted from the host, ensure the mount is writable for the container user.
