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

LINT_VERSION="${GOLANGCI_LINT_VERSION:-v2.11.0}"
NEW_FROM_REV="${OPENASE_LINT_NEW_FROM_REV:-}"

args=(run)
if [[ -n "${NEW_FROM_REV}" ]]; then
  args+=(--new-from-rev "${NEW_FROM_REV}")
fi
if [[ "$#" -gt 0 ]]; then
  args+=("$@")
else
  args+=("./...")
fi

lint_cmd=(go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${LINT_VERSION}" "${args[@]}")
retryable_patterns=(
  "proxy.golang.org"
  "Client.Timeout exceeded"
  "TLS handshake timeout"
  "connection reset by peer"
  "EOF"
)

attempt=1
max_attempts=3
while (( attempt <= max_attempts )); do
  output="$("${lint_cmd[@]}" 2>&1)" && {
    printf '%s\n' "${output}"
    exit 0
  }

  should_retry=0
  for pattern in "${retryable_patterns[@]}"; do
    if [[ "${output}" == *"${pattern}"* ]]; then
      should_retry=1
      break
    fi
  done

  printf '%s\n' "${output}" >&2
  if (( should_retry == 0 || attempt == max_attempts )); then
    exit 1
  fi

  printf 'lint bootstrap failed with a transient network error; retrying (%d/%d)\n' "${attempt}" "${max_attempts}" >&2
  sleep $(( attempt * 2 ))
  ((attempt++))
done
