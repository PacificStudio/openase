#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -eq 0 ]]; then
  printf 'usage: %s <command> [args...]\n' "${0##*/}" >&2
  exit 1
fi

preserved_openase_env=(
  OPENASE_PGTEST_SHARED_ROOT
)

preserve_openase_env() {
  local name="$1"
  local preserved_name
  for preserved_name in "${preserved_openase_env[@]}"; do
    if [[ "${name}" == "${preserved_name}" ]]; then
      return 0
    fi
  done
  return 1
}

while IFS= read -r name; do
  case "${name}" in
    OPENASE_*)
      if ! preserve_openase_env "${name}"; then
        unset "${name}"
      fi
      ;;
  esac
done < <(compgen -e)

exec "$@"
