---
name: local-bootstrap-cli-auth-debug
description:
  Diagnose and repair OpenASE CLI access in local bootstrap mode. Use when
  protected `openase` commands fail because the CLI is missing a local human
  session, has a stale `~/.openase/human-session.json`, is missing CSRF state,
  or needs a fresh `openase auth bootstrap login`.
---

# Local Bootstrap CLI Auth Debug

## Use This When

- OpenASE is running in local bootstrap mode, not active OIDC.
- Protected CLI commands return `HUMAN_SESSION_REQUIRED`,
  `HUMAN_SESSION_INVALID`, or CSRF errors.
- You need to create, inspect, refresh, or clear the CLI human session used by
  typed `openase` commands.

## Key Facts

- CLI human session state lives at `~/.openase/human-session.json` by default.
- `openase auth bootstrap login` creates a fresh local bootstrap authorization,
  redeems it through `/api/v1/auth/local-bootstrap/redeem`, and stores the
  resulting session + CSRF for later commands.
- Typed and raw API commands automatically reuse that stored session unless a
  Bearer token is provided.
- Local bootstrap sessions are tied to the current host context; if the stored
  session goes stale, re-login from the same machine and shell.

## Standard Flow

1. Verify the service is healthy:
   - `curl -fsS http://127.0.0.1:19836/healthz`
2. Run the non-mutating diagnostic helper:
   - `.codex/skills/local-bootstrap-cli-auth-debug/scripts/check_local_cli_auth.sh`
3. If the helper reports no valid CLI human session, refresh it:
   - `openase auth bootstrap login`
4. Re-check the current principal:
   - `openase auth session`
5. Re-run the failing protected command.

## Fast Repair Commands

```sh
openase auth bootstrap login
openase auth session
openase auth sessions list
openase auth logout
openase auth bootstrap login
```

## Failure Patterns

- `HUMAN_SESSION_REQUIRED`
  - The CLI has no usable stored local human session.
  - Fix: run `openase auth bootstrap login`.
- `HUMAN_SESSION_INVALID`
  - The stored session file exists, but the cookie is expired, revoked, or no
    longer matches the current request context.
  - Fix: `openase auth logout` then `openase auth bootstrap login`.
- `CSRF_TOKEN_INVALID` or `CSRF_ORIGIN_FORBIDDEN`
  - The CLI session is present but the request is missing valid CSRF state or
    origin context.
  - Fix: refresh the stored session with `openase auth bootstrap login`, then
    retry from the same machine.
- `LOCAL_BOOTSTRAP_DISABLED`
  - The instance is no longer in local bootstrap mode.
  - Stop and switch to the OIDC/browser auth path instead of retrying this
    skill.

## Useful Overrides

- Alternate session file:
  - `--session-file /tmp/openase-human-session.json`
  - `OPENASE_HUMAN_SESSION_FILE=/tmp/openase-human-session.json`
- Manual session injection:
  - `OPENASE_HUMAN_SESSION_TOKEN=...`
  - `OPENASE_HUMAN_CSRF_TOKEN=...`
- Alternate API base:
  - `OPENASE_API_URL=http://127.0.0.1:19836/api/v1`

## Files

- Skill helper:
  - `.codex/skills/local-bootstrap-cli-auth-debug/scripts/check_local_cli_auth.sh`
- Default stored state:
  - `~/.openase/human-session.json`
