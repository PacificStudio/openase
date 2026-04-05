#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

if [ -x "$ROOT_DIR/.tooling/go/bin/go" ]; then
  export PATH="$ROOT_DIR/.tooling/go/bin:$PATH"
elif [ -x "$HOME/.local/go1.26.1/bin/go" ]; then
  export PATH="$HOME/.local/go1.26.1/bin:$PATH"
fi

declare -A CASE_PKGS=(
  [listener]="./internal/infra/machinetransport"
  [reverse]="./internal/infra/machinetransport"
  [ssh]="./internal/cli"
)

declare -A CASE_PATTERNS=(
  [listener]='TestWebsocketListenerRuntimeContainerE2E$'
  [reverse]='TestWebsocketReverseRuntimeContainerE2E$'
  [ssh]='TestMachineSSHHelperContainerE2E$'
)

declare -A CASE_LABELS=(
  [listener]='Direct-connect websocket runtime container e2e'
  [reverse]='Reverse-connect websocket runtime container e2e'
  [ssh]='SSH helper container e2e'
)

usage() {
  cat <<'EOF'
usage: scripts/ci/remote_runtime_container_harness.sh [listener] [reverse] [ssh]

Runs the slow local-only remote runtime container harness. With no arguments,
it runs all supported cases.

Artifacts are written under:
  .artifacts/remote-runtime-container

Environment overrides:
  OPENASE_REMOTE_RUNTIME_ARTIFACT_DIR
  OPENASE_TEST_REMOTE_RUNTIME_COMPOSE_FILE
  OPENASE_TEST_OPENASE_BINARY
EOF
}

if [ "${1:-}" = "--help" ] || [ "${1:-}" = "-h" ]; then
  usage
  exit 0
fi

ARTIFACT_DIR="${OPENASE_REMOTE_RUNTIME_ARTIFACT_DIR:-$ROOT_DIR/.artifacts/remote-runtime-container}"
COMPOSE_FILE="${OPENASE_TEST_REMOTE_RUNTIME_COMPOSE_FILE:-$ROOT_DIR/scripts/ci/remote_runtime_container.compose.yml}"
OPENASE_BINARY="${OPENASE_TEST_OPENASE_BINARY:-$ROOT_DIR/bin/openase}"

mkdir -p "$ARTIFACT_DIR"

if [ ! -x "$OPENASE_BINARY" ]; then
  printf 'OpenASE binary %s is missing or not executable; building it first.\n' "$OPENASE_BINARY"
  (
    cd "$ROOT_DIR"
    make build
  )
fi

export OPENASE_RUN_REMOTE_RUNTIME_CONTAINER_TESTS=1
export OPENASE_TEST_OPENASE_BINARY="$OPENASE_BINARY"
export OPENASE_TEST_REMOTE_RUNTIME_COMPOSE_FILE="$COMPOSE_FILE"
export OPENASE_REMOTE_RUNTIME_ARTIFACT_DIR="$ARTIFACT_DIR"

if [ "$#" -eq 0 ]; then
  set -- listener reverse ssh
fi

run_case() {
  local case_name="$1"
  local label="${CASE_LABELS[$case_name]:-}"
  local pkg="${CASE_PKGS[$case_name]:-}"
  local pattern="${CASE_PATTERNS[$case_name]:-}"
  local logfile="$ARTIFACT_DIR/${case_name}.go-test.log"

  if [ -z "$label" ] || [ -z "$pkg" ] || [ -z "$pattern" ]; then
    printf 'unknown case: %s\n' "$case_name" >&2
    usage >&2
    exit 1
  fi

  printf '\n== %s ==\n' "$label"
  (
    cd "$ROOT_DIR"
    go test "$pkg" -count=1 -run "$pattern"
  ) 2>&1 | tee "$logfile"
}

for case_name in "$@"; do
  run_case "$case_name"
done
