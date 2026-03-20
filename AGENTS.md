# Workspace Notes

## Environment

- This workspace may not have `go` or `gofmt` on `PATH`. If Go validation is needed, first try the workspace-local toolchain under `.tooling/go/bin/`, for example `PATH=$PWD/.tooling/go/bin:$PATH go test ./...`.
- If `.tooling/go/bin` is unavailable, a verified fallback toolchain is installed at `/home/yuzhong/.local/go1.26.1/bin/go`; prepend `PATH=/home/yuzhong/.local/go1.26.1/bin:$PATH` for builds, tests, and `gofmt`.
- This workspace may not have repo-local git author identity configured. If `git commit` fails with `Author identity unknown`, set local config with `git config user.name "Codex"` and `git config user.email "codex@openai.com"` before retrying; this matches existing repository history.
- In embedded-postgres backed tests, creating `ProjectRepo.labels` or `Agent.capabilities` via direct Ent builders currently serializes `TEXT[]` as JSON-like literals and fails with `pq: malformed array literal`. Prefer API/service paths that already normalize array input, or avoid asserting on those fields in direct Ent setup until the repository-side array encoding is fixed.
- `go test ./...` may fail in embedded-postgres-backed packages with `process already listening on port <port>` even when the changed code compiles. Treat that as an environment/runtime port-allocation issue first, not an immediate signal that the edited package logic regressed.
- Fresh empty databases can currently race schema migration when `openase all-in-one` starts `serve` and `orchestrate` concurrently. For local validation against a brand-new DB, pre-run one migration pass first (for example by opening the DB once through `internal/runtime/database.Open`) before launching `all-in-one`.
