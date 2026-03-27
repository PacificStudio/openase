#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

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
BACKEND_COVERAGE_MIN="${OPENASE_BACKEND_COVERAGE_MIN:-85.0}"
DOMAIN_COVERAGE_MIN="${OPENASE_DOMAIN_COVERAGE_MIN:-100.0}"
ORIGINAL_GOPATH="$("${GO_BIN}" env GOPATH)"
ORIGINAL_GOMODCACHE="$("${GO_BIN}" env GOMODCACHE)"
ORIGINAL_GOCACHE="$("${GO_BIN}" env GOCACHE)"

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

printf 'Running backend full-code coverage...\n'
"${GO_BIN}" test \
  -count=1 \
  -timeout="${GO_TEST_TIMEOUT}" \
  -p 1 \
  -parallel=1 \
  -coverprofile="${backend_profile}" \
  ./internal/... ./cmd/openase

printf '\nRunning domain/core coverage gate...\n'
"${GO_BIN}" test \
  -count=1 \
  -coverprofile="${domain_profile}" \
  ./internal/domain/... ./internal/types/...

backend_pct="$(extract_total_pct "${backend_profile}")"
domain_pct="$(extract_total_pct "${domain_profile}")"

printf '\nBackend coverage summary:\n'
printf '  overall backend: %s (required >= %s)\n' "${backend_pct}" "${BACKEND_COVERAGE_MIN}%"
printf '  domain/core:     %s (required >= %s)\n' "${domain_pct}" "${DOMAIN_COVERAGE_MIN}%"

assert_threshold "overall backend" "${backend_pct}" "${BACKEND_COVERAGE_MIN}%"
assert_threshold "domain/core" "${domain_pct}" "${DOMAIN_COVERAGE_MIN}%"
