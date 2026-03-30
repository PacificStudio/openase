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
GO_TEST_GROUP_SIZE_RAW="${OPENASE_BACKEND_TEST_GROUP_SIZE:-}"
GO_TEST_GROUP_SELECTION_RAW="${OPENASE_BACKEND_TEST_GROUPS:-}"
ENABLE_FULL_BACKEND_COVERAGE="${OPENASE_ENABLE_FULL_BACKEND_COVERAGE:-false}"
BACKEND_COVERAGE_MIN="${OPENASE_BACKEND_COVERAGE_MIN:-75.0}"
DOMAIN_COVERAGE_MIN="${OPENASE_DOMAIN_COVERAGE_MIN:-100.0}"
ORIGINAL_GOPATH="$("${GO_BIN}" env GOPATH)"
ORIGINAL_GOMODCACHE="$("${GO_BIN}" env GOMODCACHE)"
ORIGINAL_GOCACHE="$("${GO_BIN}" env GOCACHE)"
GO_TEST_PROGRESS_MODE="${OPENASE_GO_TEST_PROGRESS_MODE:-}"

parse_optional_positive_int() {
  local name="$1"
  local raw="$2"

  if [[ -z "${raw}" ]]; then
    printf '0\n'
    return
  fi

  case "${raw}" in
    ''|*[!0-9]*)
      printf '%s must be a positive integer, got %s\n' "${name}" "${raw}" >&2
      exit 1
      ;;
  esac

  if (( raw <= 0 )); then
    printf '%s must be a positive integer, got %s\n' "${name}" "${raw}" >&2
    exit 1
  fi

  printf '%s\n' "${raw}"
}

GO_TEST_GROUP_SIZE="$(parse_optional_positive_int "OPENASE_BACKEND_TEST_GROUP_SIZE" "${GO_TEST_GROUP_SIZE_RAW}")"

declare -A SELECTED_BACKEND_GROUPS=()

parse_optional_group_selection() {
  local name="$1"
  local raw="$2"

  if [[ -z "${raw}" ]]; then
    return
  fi

  local segment=""
  local start=0
  local end=0
  local value=0

  IFS=',' read -r -a raw_segments <<< "${raw}"
  for segment in "${raw_segments[@]}"; do
    case "${segment}" in
      ''|*[!0-9-]*|-*|*-|*--*)
        printf '%s must be a comma-separated list of positive integers or ranges, got %s\n' "${name}" "${raw}" >&2
        exit 1
        ;;
      *-*)
        start="${segment%-*}"
        end="${segment#*-}"
        if (( start <= 0 || end <= 0 || start > end )); then
          printf '%s range must be ascending positive integers, got %s\n' "${name}" "${segment}" >&2
          exit 1
        fi
        for (( value = start; value <= end; value++ )); do
          SELECTED_BACKEND_GROUPS["${value}"]=1
        done
        ;;
      *)
        value="${segment}"
        if (( value <= 0 )); then
          printf '%s must use positive integers, got %s\n' "${name}" "${segment}" >&2
          exit 1
        fi
        SELECTED_BACKEND_GROUPS["${value}"]=1
        ;;
    esac
  done
}

validate_group_selection() {
  local total_groups="$1"

  if (( ${#SELECTED_BACKEND_GROUPS[@]} == 0 )); then
    return
  fi

  local selected_group=0
  for selected_group in "${!SELECTED_BACKEND_GROUPS[@]}"; do
    if (( selected_group > total_groups )); then
      printf 'OPENASE_BACKEND_TEST_GROUPS selected group %d, but only %d groups exist\n' "${selected_group}" "${total_groups}" >&2
      exit 1
    fi
  done
}

group_is_selected() {
  local group_index="$1"

  if (( ${#SELECTED_BACKEND_GROUPS[@]} == 0 )); then
    return 0
  fi

  [[ -n "${SELECTED_BACKEND_GROUPS["${group_index}"]:-}" ]]
}

parse_optional_group_selection "OPENASE_BACKEND_TEST_GROUPS" "${GO_TEST_GROUP_SELECTION_RAW}"

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
    "${GO_BIN}" test -json "$@"
    return
  fi

  "${GO_BIN}" test "$@"
}

run_backend_full_suite() {
  if (( GO_TEST_GROUP_SIZE == 0 )); then
    if (( ${#SELECTED_BACKEND_GROUPS[@]} > 0 )); then
      printf 'OPENASE_BACKEND_TEST_GROUPS requires OPENASE_BACKEND_TEST_GROUP_SIZE to be set\n' >&2
      exit 1
    fi

    printf 'Running backend full test suite...\n'
    run_go_test \
      -count=1 \
      -timeout="${GO_TEST_TIMEOUT}" \
      -p 1 \
      -parallel=1 \
      "${backend_packages[@]}"
    return
  fi

  local total_packages="${#backend_packages[@]}"
  local total_groups=$(( (total_packages + GO_TEST_GROUP_SIZE - 1) / GO_TEST_GROUP_SIZE ))
  local offset=0
  local group_index=1
  local selected_count=0

  validate_group_selection "${total_groups}"

  printf 'Running backend full test suite in %d groups (%d packages total)...\n' "${total_groups}" "${total_packages}"
  if (( ${#SELECTED_BACKEND_GROUPS[@]} > 0 )); then
    printf 'Selected backend test groups: %s\n' "${GO_TEST_GROUP_SELECTION_RAW}"
  fi

  while (( offset < total_packages )); do
    local -a group_packages=( "${backend_packages[@]:offset:GO_TEST_GROUP_SIZE}" )

    if ! group_is_selected "${group_index}"; then
      offset=$(( offset + GO_TEST_GROUP_SIZE ))
      group_index=$(( group_index + 1 ))
      continue
    fi

    selected_count=$(( selected_count + 1 ))

    printf '\nRunning backend test group %d/%d (%d packages):\n' "${group_index}" "${total_groups}" "${#group_packages[@]}"
    printf '  %s\n' "${group_packages[@]}"

    run_go_test \
      -count=1 \
      -timeout="${GO_TEST_TIMEOUT}" \
      -p 1 \
      -parallel=1 \
      "${group_packages[@]}"

    offset=$(( offset + GO_TEST_GROUP_SIZE ))
    group_index=$(( group_index + 1 ))
  done

  if (( ${#SELECTED_BACKEND_GROUPS[@]} > 0 && selected_count == 0 )); then
    printf 'OPENASE_BACKEND_TEST_GROUPS did not match any backend test groups\n' >&2
    exit 1
  fi
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
    -p 1 \
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
