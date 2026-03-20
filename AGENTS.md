# Workspace Notes

## Environment

- This workspace may not have `go` or `gofmt` on `PATH`. If Go validation is needed, first try the workspace-local toolchain under `.tooling/go/bin/`, for example `PATH=$PWD/.tooling/go/bin:$PATH go test ./...`.
- If `.tooling/go/bin` is unavailable, a verified fallback toolchain is installed at `/home/yuzhong/.local/go1.26.1/bin/go`; prepend `PATH=/home/yuzhong/.local/go1.26.1/bin:$PATH` for builds, tests, and `gofmt`.
- This workspace may not have repo-local git author identity configured. If `git commit` fails with `Author identity unknown`, set local config with `git config user.name "Codex"` and `git config user.email "codex@openai.com"` before retrying; this matches existing repository history.
- In embedded-postgres backed tests, creating `ProjectRepo.labels` or `Agent.capabilities` via direct Ent builders currently serializes `TEXT[]` as JSON-like literals and fails with `pq: malformed array literal`. Prefer API/service paths that already normalize array input, or avoid asserting on those fields in direct Ent setup until the repository-side array encoding is fixed.
- Installing frontend dependencies under `web/node_modules` before running `go test ./...` will currently surface the vendored Go package under `web/node_modules/flatted/golang/pkg/flatted` and break backend checks. In CI and other clean environments, prefer building frontend assets in a separate step/job and pass only `internal/webui/static` into Go test/build steps.
