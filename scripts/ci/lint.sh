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

go run "github.com/golangci/golangci-lint/v2/cmd/golangci-lint@${LINT_VERSION}" "${args[@]}"
