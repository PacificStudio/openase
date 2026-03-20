---
name: deploy-openase
description:
  Build and locally redeploy OpenASE from the current branch, including web
  static assets and the Go binary, then restart the local service and verify
  health. Use when asked to compile, deploy, restart, or locally upgrade the
  current checkout.
---

# Deploy OpenASE

## Goals

- Build the current checkout deterministically.
- Regenerate embedded web assets.
- Restart the local OpenASE process.
- Verify the new process is actually serving health checks.

## Related Skills

- `pull`: use this first if the user wants the branch updated from remote before
  deployment.

## Environment Assumptions

- Repo root is `/home/yuzhong/workspace/openase`.
- Local runtime env lives in `~/.openase/.env`.
- Preferred Go toolchain:
  - `$PWD/.tooling/go/bin`
  - fallback `/home/yuzhong/.local/go1.26.1/bin`

## Standard Flow

1. Confirm current branch and commit.
2. Build frontend assets:
   - `corepack pnpm --dir web install --frozen-lockfile`
   - `corepack pnpm --dir web run build`
3. Build backend binary:
   - `PATH=$PWD/.tooling/go/bin:/home/yuzhong/.local/go1.26.1/bin:$PATH go build -o ./bin/openase ./cmd/openase`
4. Stop old local `openase` processes started from this repo.
5. Source `~/.openase/.env`.
6. Start the requested mode with `setsid`.
7. Verify success:
   - `serve` / `all-in-one`: `curl http://127.0.0.1:19836/healthz`
   - `orchestrate`: process stays alive and logs show runtime ready

## Failure Handling

- Do not silently downgrade modes.
- If `all-in-one` fails, surface the exact logs and let the user decide whether
  a `serve`-only fallback is acceptable.
- If port binding fails, report the listener conflict explicitly.
- If startup crashes, include the relevant stdout/stderr tail in the summary.

## Commands

Use the helper script:

- `.codex/skills/deploy-openase/scripts/redeploy_local.sh`

Examples:

```sh
.codex/skills/deploy-openase/scripts/redeploy_local.sh --mode all-in-one
.codex/skills/deploy-openase/scripts/redeploy_local.sh --mode serve
```

## Notes

- `all-in-one` and `serve` both expose the panel on `http://127.0.0.1:19836`
  with the current local defaults.
- This skill is for local redeploy of the current checkout, not for git branch
  updates by itself.
