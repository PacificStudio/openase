#!/usr/bin/env bash
set -euo pipefail

mode="all-in-one"
host="127.0.0.1"
port="19836"
env_file="${HOME}/.openase/.env"

usage() {
  cat <<'EOF'
Usage:
  redeploy_local.sh [--mode all-in-one|serve|orchestrate] [--host 127.0.0.1] [--port 19836] [--env-file ~/.openase/.env]

Builds web assets and the Go binary, stops existing repo-local openase
processes, starts the requested mode, and verifies startup.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --mode)
      mode="${2:-}"
      shift 2
      ;;
    --host)
      host="${2:-}"
      shift 2
      ;;
    --port)
      port="${2:-}"
      shift 2
      ;;
    --env-file)
      env_file="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "$mode" in
  all-in-one|serve|orchestrate)
    ;;
  *)
    echo "unsupported mode: $mode" >&2
    exit 1
    ;;
esac

repo_root="$(git rev-parse --show-toplevel)"
cd "$repo_root"

go_path="$repo_root/.tooling/go/bin:/home/yuzhong/.local/go1.26.1/bin:$PATH"
stdout_log="${HOME}/.openase/logs/openase-local.stdout.log"
stderr_log="${HOME}/.openase/logs/openase-local.stderr.log"
repo_binary="${repo_root}/bin/openase"

mkdir -p "${HOME}/.openase/logs"

repo_local_openase_pids() {
  local pid exe
  for pid in $(ps -o pid= -C openase 2>/dev/null); do
    exe="$(readlink -f "/proc/${pid}/exe" 2>/dev/null || true)"
    exe="${exe% (deleted)}"
    if [[ "$exe" == "$repo_binary" ]]; then
      printf '%s\n' "$pid"
    fi
  done
}

repo_local_listener_pid() {
  local pid exe
  if ! command -v lsof >/dev/null 2>&1; then
    return 1
  fi
  while read -r pid; do
    [[ -n "$pid" ]] || continue
    exe="$(readlink -f "/proc/${pid}/exe" 2>/dev/null || true)"
    exe="${exe% (deleted)}"
    if [[ "$exe" == "$repo_binary" ]]; then
      printf '%s\n' "$pid"
      return 0
    fi
  done < <(lsof -tiTCP:"$port" -sTCP:LISTEN 2>/dev/null || true)
  return 1
}

echo "[1/5] build web assets"
corepack pnpm --dir web install --frozen-lockfile
corepack pnpm --dir web run build

echo "[2/5] build backend binary"
PATH="$go_path" go build -o ./bin/openase ./cmd/openase

echo "[3/5] stop existing repo-local openase processes"
mapfile -t existing_pids < <(repo_local_openase_pids)
if [[ ${#existing_pids[@]} -gt 0 ]]; then
  kill -9 "${existing_pids[@]}"
fi

if [[ ! -f "$env_file" ]]; then
  echo "env file not found: $env_file" >&2
  exit 1
fi

echo "[4/5] start $mode"
set -a
# shellcheck disable=SC1090
source "$env_file"
set +a

: >"$stdout_log"
: >"$stderr_log"

args=("$mode")
if [[ "$mode" != "orchestrate" ]]; then
  args+=("--host" "$host" "--port" "$port")
fi

setsid ./bin/openase "${args[@]}" >"$stdout_log" 2>"$stderr_log" </dev/null &
pid="$!"

echo "[5/5] verify startup"
if [[ "$mode" == "serve" || "$mode" == "all-in-one" ]]; then
  for _ in $(seq 1 20); do
    listener_pid="$(repo_local_listener_pid || true)"
    if [[ -n "${listener_pid:-}" ]] && curl -fsS "http://${host}:${port}/healthz" >/dev/null 2>&1; then
      printf 'MODE=%s\nPID=%s\nURL=http://%s:%s\n' "$mode" "$listener_pid" "$host" "$port"
      exit 0
    fi
    sleep 1
  done
else
  for _ in $(seq 1 10); do
    mapfile -t current_pids < <(repo_local_openase_pids)
    if [[ ${#current_pids[@]} -gt 0 ]]; then
      printf 'MODE=%s\nPID=%s\n' "$mode" "${current_pids[-1]}"
      exit 0
    fi
    sleep 1
  done
fi

echo "startup_check_failed" >&2
echo "STDOUT" >&2
cat "$stdout_log" >&2 || true
echo "STDERR" >&2
cat "$stderr_log" >&2 || true
exit 1
