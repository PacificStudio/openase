# Workspace Notes

## 需求文档

需求文档位于 OpenASE-PRD.md

## Environment

- This workspace may not have `go` or `gofmt` on `PATH`. If Go validation is needed, first try the workspace-local toolchain under `.tooling/go/bin/`, for example `PATH=$PWD/.tooling/go/bin:$PATH go test ./...`.
- If `.tooling/go/bin` is unavailable, a verified fallback toolchain is installed at `/home/yuzhong/.local/go1.26.1/bin/go`; prepend `PATH=/home/yuzhong/.local/go1.26.1/bin:$PATH` for builds, tests, and `gofmt`.
- The frontend toolchain may also require a newer Node than `/usr/bin/node` provides. If Svelte/Vite startup fails on `node:util.styleText`, use `PATH=/home/yuzhong/.nvm/versions/node/v22.22.1/bin:$PATH` for `vitest`, `prettier`, and other `web/` checks.
- This workspace may not have repo-local git author identity configured. If `git commit` fails with `Author identity unknown`, set local config with `git config user.name "Codex"` and `git config user.email "codex@openai.com"` before retrying; this matches existing repository history.
- In embedded-postgres backed tests, creating `ProjectRepo.labels` or `Agent.capabilities` via direct Ent builders currently serializes `TEXT[]` as JSON-like literals and fails with `pq: malformed array literal`. Prefer API/service paths that already normalize array input, or avoid asserting on those fields in direct Ent setup until the repository-side array encoding is fixed.
- Fresh empty databases can currently race schema migration when `openase all-in-one` starts `serve` and `orchestrate` concurrently. For local validation against a brand-new DB, pre-run one migration pass first (for example by opening the DB once through `internal/runtime/database.Open`) before launching `all-in-one`.
- `internal/webui/static/.keep` must remain tracked on clean checkouts. CI jobs like `make openapi-check` compile `internal/webui/ui.go` without first building frontend assets, and deleting the placeholder causes `go:embed all:static: no matching files found`.
- Local redeploy cleanup must identify repo-local `openase` processes by resolved `/proc/<pid>/exe`, not only by argv. Older runs may survive as `/home/yuzhong/workspace/openase/bin/openase (deleted)` and keep serving stale code on `127.0.0.1:19836` until explicitly killed.
- Tests that touch `~/.openase/...` must isolate `HOME` per test run, preferably with `t.TempDir()` or a test-name-derived temp home. When parallel execution exposes cross-test contamination, fix the isolation boundary first instead of lowering concurrency to hide the problem.
- `scripts/ci/backend_coverage.sh` resets `HOME` to a temp directory. For embedded-postgres backed test suites under that script, export `OPENASE_PGTEST_SHARED_ROOT=/home/yuzhong/.cache/openase/pgtest` so `internal/testutil/pgtest` reuses the already extracted binaries instead of a cold temp-home cache that may miss the sibling `postgres` binary next to `initdb`.
