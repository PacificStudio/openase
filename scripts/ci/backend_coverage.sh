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
ORIGINAL_HOME="${HOME:-}"
ORIGINAL_XDG_CACHE_HOME="${XDG_CACHE_HOME:-}"
ORIGINAL_GOPATH="$("${GO_BIN}" env GOPATH)"
ORIGINAL_GOMODCACHE="$("${GO_BIN}" env GOMODCACHE)"
ORIGINAL_GOCACHE="$("${GO_BIN}" env GOCACHE)"
GO_TEST_PROGRESS_MODE="${OPENASE_GO_TEST_PROGRESS_MODE:-}"

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
  if [[ "${GITHUB_ACTIONS:-}" == "true" ]]; then
    GO_TEST_PROGRESS_MODE="json"
  else
    GO_TEST_PROGRESS_MODE="plain"
  fi
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

run_backend_full_suite() {
  printf 'Running backend full test suite...\n'
  run_go_test \
    -count=1 \
    -timeout="${GO_TEST_TIMEOUT}" \
    -parallel=1 \
    "${backend_packages[@]}"
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
    -covermode=atomic \
    -coverpkg="${backend_coverpkg}" \
    -coverprofile="${backend_profile}" \
    "${backend_packages[@]}"

  backend_pct="$(extract_total_pct "${backend_profile}")"
else
  printf '\nSkipping backend full-code coverage metric (set OPENASE_ENABLE_FULL_BACKEND_COVERAGE=true to enable).\n'
fi

printf '\nRunning domain/core coverage gate...\n'
run_go_test \
  -count=1 \
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
