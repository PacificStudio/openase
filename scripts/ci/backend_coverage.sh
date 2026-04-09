#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TEST_ENV_WRAPPER="${ROOT_DIR}/scripts/ci/with_clean_openase_test_env.sh"

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

cd "${ROOT_DIR}"

GO_BIN="${GO_BIN:-go}"
GO_TEST_TIMEOUT="${OPENASE_GO_TEST_TIMEOUT:-20m}"
ENABLE_FULL_BACKEND_COVERAGE="${OPENASE_ENABLE_FULL_BACKEND_COVERAGE:-false}"
BACKEND_COVERAGE_MIN="${OPENASE_BACKEND_COVERAGE_MIN:-75.0}"
DOMAIN_COVERAGE_MIN="${OPENASE_DOMAIN_COVERAGE_MIN:-100.0}"
GO_TEST_HEARTBEAT_INTERVAL_SECONDS_RAW="${OPENASE_GO_TEST_HEARTBEAT_INTERVAL_SECONDS:-}"
GO_TEST_FAILURE_CONTEXT_LINES="${OPENASE_GO_TEST_FAILURE_CONTEXT_LINES:-30}"
GO_TEST_FAILURE_MAX_LINES="${OPENASE_GO_TEST_FAILURE_MAX_LINES:-160}"
ORIGINAL_HOME="${HOME:-}"
ORIGINAL_XDG_CACHE_HOME="${XDG_CACHE_HOME:-}"
ORIGINAL_GOPATH="$("${GO_BIN}" env GOPATH)"
ORIGINAL_GOMODCACHE="$("${GO_BIN}" env GOMODCACHE)"
ORIGINAL_GOCACHE="$("${GO_BIN}" env GOCACHE)"
GO_TEST_PROGRESS_MODE="${OPENASE_GO_TEST_PROGRESS_MODE:-}"

GO_TEST_HEARTBEAT_INTERVAL_SECONDS=0
if [[ -n "${GO_TEST_HEARTBEAT_INTERVAL_SECONDS_RAW}" ]]; then
  if [[ ! "${GO_TEST_HEARTBEAT_INTERVAL_SECONDS_RAW}" =~ ^[0-9]+$ ]]; then
    printf 'OPENASE_GO_TEST_HEARTBEAT_INTERVAL_SECONDS must be a non-negative integer, got %s\n' "${GO_TEST_HEARTBEAT_INTERVAL_SECONDS_RAW}" >&2
    exit 1
  fi
  GO_TEST_HEARTBEAT_INTERVAL_SECONDS="${GO_TEST_HEARTBEAT_INTERVAL_SECONDS_RAW}"
fi

if [[ ! "${GO_TEST_FAILURE_CONTEXT_LINES}" =~ ^[0-9]+$ ]]; then
  printf 'OPENASE_GO_TEST_FAILURE_CONTEXT_LINES must be a non-negative integer, got %s\n' "${GO_TEST_FAILURE_CONTEXT_LINES}" >&2
  exit 1
fi

if [[ ! "${GO_TEST_FAILURE_MAX_LINES}" =~ ^[1-9][0-9]*$ ]]; then
  printf 'OPENASE_GO_TEST_FAILURE_MAX_LINES must be a positive integer, got %s\n' "${GO_TEST_FAILURE_MAX_LINES}" >&2
  exit 1
fi

tmp_dir="$(mktemp -d)"
backend_profile="${tmp_dir}/backend.out"
domain_profile="${tmp_dir}/domain.out"
tmp_home="${tmp_dir}/home"

cleanup() {
  rm -rf "${tmp_dir}"
}

trap cleanup EXIT

mkdir -p "${tmp_home}"
export HOME="${tmp_home}"
export GOPATH="${ORIGINAL_GOPATH}"
export GOMODCACHE="${ORIGINAL_GOMODCACHE}"
export GOCACHE="${ORIGINAL_GOCACHE}"

if [[ -z "${OPENASE_PGTEST_SHARED_ROOT:-}" ]]; then
  shared_cache_root="${ORIGINAL_XDG_CACHE_HOME:-}"
  if [[ -z "${shared_cache_root}" && -n "${ORIGINAL_HOME}" ]]; then
    shared_cache_root="${ORIGINAL_HOME}/.cache"
  fi
  if [[ -n "${shared_cache_root}" ]]; then
    export OPENASE_PGTEST_SHARED_ROOT="${shared_cache_root}/openase/pgtest"
  fi
fi

if [[ -z "${GO_TEST_PROGRESS_MODE}" ]]; then
  GO_TEST_PROGRESS_MODE="plain"
fi

GO_TEST_PACKAGE_PARALLEL="${OPENASE_GO_TEST_PACKAGE_PARALLEL:-}"
if [[ -z "${GO_TEST_PACKAGE_PARALLEL}" && "${GITHUB_ACTIONS:-}" == "true" ]]; then
  # CI runners can OOM or terminate `go test ./...` when too many package
  # binaries and embedded Postgres instances overlap. Keep local defaults
  # unchanged, but cap the GitHub Actions package fan-out more aggressively.
  GO_TEST_PACKAGE_PARALLEL=2
fi

go_test_parallel_args=()
if [[ -n "${GO_TEST_PACKAGE_PARALLEL}" ]]; then
  go_test_parallel_args+=("-p=${GO_TEST_PACKAGE_PARALLEL}")
fi

mapfile -t domain_packages < <("${GO_BIN}" list ./internal/domain/... ./internal/types/...)
mapfile -t backend_packages < <("${GO_BIN}" list ./internal/... ./cmd/openase)

domain_coverpkg="$(IFS=,; printf '%s' "${domain_packages[*]}")"

extract_total_pct() {
  local profile="$1"
  "${GO_BIN}" tool cover -func="${profile}" | awk '/^total:/{print $3}'
}

assert_threshold() {
  local label="$1"
  local actual_pct="$2"
  local minimum_pct="$3"
  python3 - "$label" "$actual_pct" "$minimum_pct" <<'PY'
import math
import sys

label, actual_raw, minimum_raw = sys.argv[1:]
actual = float(actual_raw.rstrip('%'))
minimum = float(minimum_raw.rstrip('%'))

if actual + 1e-9 < minimum:
    raise SystemExit(f"{label} coverage {actual:.1f}% is below required threshold {minimum:.1f}%")
PY
}

run_go_test() {
  if [[ "${GO_TEST_PROGRESS_MODE}" == "json" ]]; then
    "${TEST_ENV_WRAPPER}" "${GO_BIN}" test -json "$@"
    return
  fi

  "${TEST_ENV_WRAPPER}" "${GO_BIN}" test "$@"
}

print_failure_excerpt() {
  local output_file="$1"
  python3 - "$output_file" "${GO_TEST_FAILURE_CONTEXT_LINES}" "${GO_TEST_FAILURE_MAX_LINES}" <<'PY'
import re
import sys
from pathlib import Path

path = Path(sys.argv[1])
context = int(sys.argv[2])
max_lines = int(sys.argv[3])
lines = path.read_text(errors="replace").splitlines()

if not lines:
    print("[backend-test-output] no captured output")
    raise SystemExit(0)

markers = (
    re.compile(r"--- FAIL:"),
    re.compile(r"^FAIL\b"),
    re.compile(r"\bpanic:", re.IGNORECASE),
    re.compile(r"\bfatal\b", re.IGNORECASE),
    re.compile(r"\berror\b", re.IGNORECASE),
    re.compile(r"\bexception\b", re.IGNORECASE),
    re.compile(r"\bassert\b", re.IGNORECASE),
    re.compile(r"timed out", re.IGNORECASE),
)

marker_index = None
for idx in range(len(lines) - 1, -1, -1):
    line = lines[idx]
    if any(pattern.search(line) for pattern in markers):
        marker_index = idx
        break

if marker_index is None:
    start = max(0, len(lines) - max_lines)
    excerpt = lines[start:]
else:
    start = max(0, marker_index - context)
    end = min(len(lines), marker_index + context + 1)
    excerpt = lines[start:end]
    if len(excerpt) > max_lines:
        excerpt = excerpt[-max_lines:]

print("[backend-test-output] showing captured context from the failing go test run")
for line in excerpt:
    print(line)
PY
}

run_go_test_quiet_success() {
  local output_file="${tmp_dir}/go-test-$RANDOM.log"
  local test_pid=""
  local heartbeat_pid=""
  local status=0

  # Preserve the explicit JSON mode override for ad hoc CI debugging sessions.
  if [[ "${GO_TEST_PROGRESS_MODE}" == "json" ]]; then
    if run_go_test "$@" 2>&1 | tee "${output_file}"; then
      return 0
    fi

    return 1
  fi

  run_go_test "$@" >"${output_file}" 2>&1 &
  test_pid=$!

  if (( GO_TEST_HEARTBEAT_INTERVAL_SECONDS > 0 )); then
    (
      while kill -0 "${test_pid}" 2>/dev/null; do
        sleep "${GO_TEST_HEARTBEAT_INTERVAL_SECONDS}"
        if ! kill -0 "${test_pid}" 2>/dev/null; then
          exit 0
        fi

        now="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
        printf '[backend-coverage-heartbeat] %s go test still running\n' "${now}"
      done
    ) &
    heartbeat_pid=$!
  fi

  if wait "${test_pid}"; then
    if [[ -n "${heartbeat_pid}" ]]; then
      kill "${heartbeat_pid}" 2>/dev/null || true
      wait "${heartbeat_pid}" 2>/dev/null || true
    fi
    return 0
  fi

  status=$?
  if [[ -n "${heartbeat_pid}" ]]; then
    kill "${heartbeat_pid}" 2>/dev/null || true
    wait "${heartbeat_pid}" 2>/dev/null || true
  fi

  print_failure_excerpt "${output_file}" >&2
  return "${status}"
}

run_go_test_ci_visible() {
  run_go_test_quiet_success "$@"
}

run_backend_full_suite() {
  printf 'Running backend full test suite...\n'
  run_go_test_ci_visible \
    -count=1 \
    -timeout="${GO_TEST_TIMEOUT}" \
    -parallel=1 \
    "${go_test_parallel_args[@]}" \
    "${backend_packages[@]}"
  printf 'Backend full test suite passed.\n'
}

enable_full_backend_coverage() {
  case "${ENABLE_FULL_BACKEND_COVERAGE}" in
    1|true|TRUE|yes|YES|on|ON)
      return 0
      ;;
    0|false|FALSE|no|NO|off|OFF|'')
      return 1
      ;;
  esac

  printf 'OPENASE_ENABLE_FULL_BACKEND_COVERAGE must be one of true/false, got %s\n' "${ENABLE_FULL_BACKEND_COVERAGE}" >&2
  exit 1
}
run_backend_full_suite

backend_pct=""
if enable_full_backend_coverage; then
  backend_coverpkg="$(IFS=,; printf '%s' "${backend_packages[*]}")"

  printf '\nRunning backend full-code coverage...\n'
  run_go_test \
    -count=1 \
    -timeout="${GO_TEST_TIMEOUT}" \
    -parallel=1 \
    "${go_test_parallel_args[@]}" \
    -covermode=atomic \
    -coverpkg="${backend_coverpkg}" \
    -coverprofile="${backend_profile}" \
    "${backend_packages[@]}"

  backend_pct="$(extract_total_pct "${backend_profile}")"
else
  printf '\nSkipping backend full-code coverage metric (set OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true to enable).\n'
fi

printf '\nRunning domain/core coverage gate...\n'
run_go_test_ci_visible \
  -count=1 \
  "${go_test_parallel_args[@]}" \
  -covermode=atomic \
  -coverpkg="${domain_coverpkg}" \
  -coverprofile="${domain_profile}" \
  "${domain_packages[@]}"

domain_pct="$(extract_total_pct "${domain_profile}")"

printf '\nBackend coverage summary:\n'
if [[ -n "${backend_pct}" ]]; then
  printf '  overall backend: %s (required >= %s)\n' "${backend_pct}" "${BACKEND_COVERAGE_MIN}%"
else
  printf '  overall backend: skipped by default (set OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true to require >= %s)\n' "${BACKEND_COVERAGE_MIN}%"
fi
printf '  domain/core:     %s (required >= %s)\n' "${domain_pct}" "${DOMAIN_COVERAGE_MIN}%"

if [[ -n "${backend_pct}" ]]; then
  assert_threshold "overall backend" "${backend_pct}" "${BACKEND_COVERAGE_MIN}%"
fi
assert_threshold "domain/core" "${domain_pct}" "${DOMAIN_COVERAGE_MIN}%"
