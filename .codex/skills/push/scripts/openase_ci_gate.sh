#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
Usage: openase_ci_gate.sh [--plan] [--base-rev <rev>]

Mirrors .github/workflows/ci.yml for the current branch:
- runs make openapi-check-ci when openapi_check_changed=true
- runs make frontend-api-audit-check when frontend_api_audit_changed=true
- runs frontend CI when web_changed=true
- runs backend formatting, tests, coverage gates, and Go lint checks when go_changed=true
EOF
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../../../.." && pwd)"
PLAN_ONLY=false
BASE_REV=""
CREATED_LINT_PLACEHOLDER=false
LINT_PLACEHOLDER_PATH=""

toolchain_paths=()
for candidate in \
  "${ROOT_DIR}/.tooling/go/bin" \
  "${OPENASE_GO_BIN_DIR:-}" \
  "${HOME}/.local/go1.26.1/bin"
do
  if [[ -n "${candidate}" && -d "${candidate}" ]]; then
    toolchain_paths+=("${candidate}")
  fi
done

if [[ "${#toolchain_paths[@]}" -gt 0 ]]; then
  PATH="$(IFS=:; printf '%s' "${toolchain_paths[*]}"):${PATH}"
fi

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
openapi_check_changed=false
frontend_api_audit_changed=false

for file in "${changed_files[@]}"; do
  case "${file}" in
    .github/workflows/*)
      go_changed=true
      web_changed=true
      openapi_check_changed=true
      ;;
    *.go|go.mod|go.sum|api/*)
      go_changed=true
      openapi_check_changed=true
      ;;
    Makefile)
      go_changed=true
      openapi_check_changed=true
      ;;
    scripts/ci/*)
      go_changed=true
      ;;
    web/*)
      web_changed=true
      frontend_api_audit_changed=true
      ;;
    scripts/dev/audit_frontend_api_usage.py|scripts/dev/frontend_api_audit_ignores.json)
      frontend_api_audit_changed=true
      ;;
    .golangci.yml|.golangci.yaml)
      go_changed=true
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
printf 'openapi_check_changed=%s\n' "${openapi_check_changed}"
printf 'frontend_api_audit_changed=%s\n' "${frontend_api_audit_changed}"

if [[ "${openapi_check_changed}" == "true" ]]; then
  run_cmd make openapi-check-ci
fi

if [[ "${frontend_api_audit_changed}" == "true" ]]; then
  run_cmd make frontend-api-audit-check
fi

if [[ "${web_changed}" == "true" ]]; then
  run_cmd make web-install
  run_cmd corepack pnpm --dir web run ci
fi

if [[ "${go_changed}" == "true" ]]; then
  ensure_lint_placeholder
  run_cmd env OPENASE_BACKEND_TEST_GROUP_SIZE=8 make check
  run_cmd make build
  run_cmd make LINT_BASE_REV="${BASE_REV}" lint
  run_cmd make lint-depguard
  run_cmd make lint-architecture
fi
