#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck source=/dev/null
source "${ROOT_DIR}/scripts/release/openase-install.sh"

fail() {
  return 1
}

assert_eq() {
  local actual="$1"
  local expected="$2"
  local message="$3"
  if [[ "$actual" != "$expected" ]]; then
    printf 'assertion failed: %s\nexpected: %s\nactual:   %s\n' "$message" "$expected" "$actual" >&2
    exit 1
  fi
}

assert_fails() {
  local message="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    printf 'assertion failed: %s\n' "$message" >&2
    exit 1
  fi
}

run_with_uname() {
  local sysname="$1"
  local machine="$2"
  local mode="$3"

  uname() {
    if [[ "${1:-}" == "-s" ]]; then
      printf '%s\n' "$sysname"
      return
    fi
    if [[ "${1:-}" == "-m" ]]; then
      printf '%s\n' "$machine"
      return
    fi
    printf '%s\n' "$sysname"
  }

  case "$mode" in
    os) detect_os ;;
    arch) detect_arch ;;
    *) printf 'unsupported test mode %s\n' "$mode" >&2; return 1 ;;
  esac
}

assert_eq "$(parse_release_tag_from_url 'https://github.com/pacificstudio/openase/releases/tag/v1.2.3')" "v1.2.3" "parse tag from release URL"
assert_eq "$(json_escape $'line1\n"quoted"')" 'line1\n\"quoted\"' "escape JSON special characters"
assert_eq "$(run_with_uname Linux x86_64 os)" "linux" "linux OS detection"
assert_eq "$(run_with_uname Darwin arm64 os)" "darwin" "darwin OS detection"
assert_eq "$(run_with_uname Linux x86_64 arch)" "amd64" "amd64 arch detection"
assert_eq "$(run_with_uname Linux aarch64 arch)" "arm64" "arm64 arch detection"
assert_fails "unsupported OS should fail" run_with_uname FreeBSD amd64 os
assert_fails "unsupported arch should fail" run_with_uname Linux ppc64le arch

printf 'release install script tests passed\n'
