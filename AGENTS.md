# Workspace Notes

## Environment

- This workspace may not have `go` or `gofmt` on `PATH`. If Go validation is needed, first try the workspace-local toolchain under `.tooling/go/bin/`, for example `PATH=$PWD/.tooling/go/bin:$PATH go test ./...`.
- If `.tooling/go/bin` is unavailable, a verified fallback toolchain is installed at `/home/yuzhong/.local/go1.26.1/bin/go`; prepend `PATH=/home/yuzhong/.local/go1.26.1/bin:$PATH` for builds, tests, and `gofmt`.
- This workspace may not have repo-local git author identity configured. If `git commit` fails with `Author identity unknown`, set local config with `git config user.name "Codex"` and `git config user.email "codex@openai.com"` before retrying; this matches existing repository history.
