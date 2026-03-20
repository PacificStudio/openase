#!/bin/sh

set -eu

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

go_bin=""
if [ -n "${OPENASE_GO:-}" ]; then
	go_bin="$OPENASE_GO"
elif [ -x "$repo_root/.tooling/go/bin/go" ]; then
	go_bin="$repo_root/.tooling/go/bin/go"
elif command -v go >/dev/null 2>&1; then
	go_bin=$(command -v go)
else
	printf '%s\n' "Could not find a Go toolchain for lefthook. Set OPENASE_GO, install Go on PATH, or provide .tooling/go/bin/go." >&2
	exit 1
fi

gofmt_bin=""
if [ -n "${OPENASE_GOFMT:-}" ]; then
	gofmt_bin="$OPENASE_GOFMT"
elif [ -x "$repo_root/.tooling/go/bin/gofmt" ]; then
	gofmt_bin="$repo_root/.tooling/go/bin/gofmt"
elif command -v gofmt >/dev/null 2>&1; then
	gofmt_bin=$(command -v gofmt)
else
	gofmt_bin="$(dirname "$go_bin")/gofmt"
fi

export GO="$go_bin"
export GOFMT="$gofmt_bin"

if [ "$#" -gt 0 ] && [ "$1" = "run" ]; then
	shift
	exec "$go_bin" tool lefthook run --no-auto-install "$@"
fi

exec "$go_bin" tool lefthook "$@"
