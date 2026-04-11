#!/usr/bin/env bash
set -euo pipefail

api_url="${OPENASE_API_URL:-http://127.0.0.1:19836/api/v1}"
session_file="${OPENASE_HUMAN_SESSION_FILE:-$HOME/.openase/human-session.json}"
repo_root="$(git rev-parse --show-toplevel)"

if command -v openase >/dev/null 2>&1; then
  openase_cmd=(openase)
elif [[ -x "$repo_root/bin/openase" ]]; then
  openase_cmd=("$repo_root/bin/openase")
else
  echo "openase binary not found in PATH or $repo_root/bin/openase" >&2
  exit 1
fi

echo "[1/4] health"
curl -fsS "${api_url%/api/v1}/healthz"
printf '\n'

echo "[2/4] session file"
if [[ -f "$session_file" ]]; then
  ls -l "$session_file"
  sed -n '1,120p' "$session_file"
else
  echo "missing: $session_file"
fi

echo "[3/4] current auth session"
"${openase_cmd[@]}" auth session --api-url "$api_url"

echo "[4/4] governable session inventory"
if ! "${openase_cmd[@]}" auth sessions list --api-url "$api_url"; then
  echo "protected CLI auth is not ready; try: ${openase_cmd[*]} auth bootstrap login --control-plane-url ${api_url%/api/v1}" >&2
fi
