# Workspace Notes

## Environment

- This workspace may not have `go` or `gofmt` on `PATH`. If Go validation is needed, use the local toolchain under `.tooling/go/bin/` inside the workspace, for example `PATH=$PWD/.tooling/go/bin:$PATH go test ./...`.
