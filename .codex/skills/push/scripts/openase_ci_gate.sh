#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: openase_ci_gate.sh [--plan] [--base-rev <rev>]

Mirrors .github/workflows/ci.yml for the current branch:
- always runs make openapi-check
- runs frontend CI when web_changed=true
- runs backend and Go lint checks when go_changed=true
EOF
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PLAN_ONLY=false
BASE_REV=""
CREATED_LINT_PLACEHOLDER=false
LINT_PLACEHOLDER_PATH=""

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --plan)
      PLAN_ONLY=true
      shift
      ;;
    --base-rev)
      if [[ "$#" -lt 2 ]]; then
        printf 'Missing value for --base-rev\n' >&2
        usage >&2
        exit 1
      fi
      BASE_REV="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'Unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

cleanup() {
  if [[ "${CREATED_LINT_PLACEHOLDER}" == "true" && -n "${LINT_PLACEHOLDER_PATH}" ]]; then
    rm -f "${LINT_PLACEHOLDER_PATH}"
  fi
}

trap cleanup EXIT

run_cmd() {
  printf '+ %s\n' "$*"
  if [[ "${PLAN_ONLY}" == "false" ]]; then
    "$@"
  fi
}

resolve_base_rev() {
  if [[ -n "${BASE_REV}" ]]; then
    return
  fi

  if ! git rev-parse --verify origin/main >/dev/null 2>&1; then
    printf 'origin/main not found locally; fetching from origin\n'
    git fetch --no-tags origin main
  fi

  BASE_REV="$(git merge-base origin/main HEAD)"
}

ensure_lint_placeholder() {
  local placeholder_dir="${ROOT_DIR}/internal/webui/static"
  LINT_PLACEHOLDER_PATH="${placeholder_dir}/lint-placeholder.txt"

  mkdir -p "${placeholder_dir}"
  if [[ ! -e "${LINT_PLACEHOLDER_PATH}" ]]; then
    printf 'lint placeholder for embedded web assets\n' > "${LINT_PLACEHOLDER_PATH}"
    CREATED_LINT_PLACEHOLDER=true
  fi
}

cd "${ROOT_DIR}"
resolve_base_rev

mapfile -t changed_files < <(git diff --name-only "${BASE_REV}"...HEAD)

go_changed=false
web_changed=false

for file in "${changed_files[@]}"; do
  case "${file}" in
    .github/workflows/*)
      go_changed=true
      web_changed=true
      ;;
    *.go|go.mod|go.sum|Makefile|.golangci.yml|.golangci.yaml|api/*)
      go_changed=true
      ;;
    scripts/ci/*)
      go_changed=true
      ;;
    web/*)
      web_changed=true
      ;;
  esac
done

printf 'Base revision: %s\n' "${BASE_REV}"
if [[ "${#changed_files[@]}" -eq 0 ]]; then
  printf 'Changed files:\n  (none)\n'
else
  printf 'Changed files:\n'
  printf '  %s\n' "${changed_files[@]}"
fi
printf 'go_changed=%s\n' "${go_changed}"
printf 'web_changed=%s\n' "${web_changed}"

run_cmd make openapi-check

if [[ "${web_changed}" == "true" ]]; then
  run_cmd make web-install
  run_cmd corepack pnpm --dir web run ci
fi

if [[ "${go_changed}" == "true" ]]; then
  ensure_lint_placeholder
  run_cmd make check
  run_cmd make build
  run_cmd make LINT_BASE_REV="${BASE_REV}" lint
  run_cmd make lint-depguard
fi
