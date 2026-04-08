#!/bin/sh
set -eu

fail() {
  printf 'openase-entrypoint: %s\n' "$*" >&2
  exit 1
}

require_env() {
  name="$1"
  eval "value=\${$name-}"
  [ -n "${value}" ] || fail "required environment variable ${name} is missing"
}

validate_port() {
  case "$1" in
    ''|*[!0-9]*)
      fail "OPENASE_SERVER_PORT must be an integer between 1 and 65535"
      ;;
  esac
  if [ "$1" -lt 1 ] || [ "$1" -gt 65535 ]; then
    fail "OPENASE_SERVER_PORT must be an integer between 1 and 65535"
  fi
}

validate_auth_mode() {
  case "$1" in
    disabled)
      ;;
    oidc)
      require_env OPENASE_AUTH_OIDC_ISSUER_URL
      require_env OPENASE_AUTH_OIDC_CLIENT_ID
      require_env OPENASE_AUTH_OIDC_CLIENT_SECRET
      require_env OPENASE_AUTH_OIDC_REDIRECT_URL
      ;;
    *)
      fail "OPENASE_AUTH_MODE must be either disabled or oidc"
      ;;
  esac
}

if [ "$#" -gt 0 ]; then
  exec "$@"
fi

export HOME="${HOME:-/var/lib/openase}"
export CODEX_CONFIG_SOURCE_PATH="${CODEX_CONFIG_SOURCE_PATH:-/run/coolify-secrets/codex-config.toml}"
export OPENASE_SERVER_HOST="${OPENASE_SERVER_HOST:-0.0.0.0}"
export OPENASE_SERVER_PORT="${OPENASE_SERVER_PORT:-40023}"
export OPENASE_ORCHESTRATOR_TICK_INTERVAL="${OPENASE_ORCHESTRATOR_TICK_INTERVAL:-5s}"
export OPENASE_EVENT_DRIVER="${OPENASE_EVENT_DRIVER:-auto}"
export OPENASE_LOG_LEVEL="${OPENASE_LOG_LEVEL:-info}"
export OPENASE_LOG_FORMAT="${OPENASE_LOG_FORMAT:-json}"
export OPENASE_AUTH_MODE="${OPENASE_AUTH_MODE:-disabled}"
export OPENASE_SERVER_MODE=all-in-one

require_env OPENASE_DATABASE_DSN
validate_port "$OPENASE_SERVER_PORT"
validate_auth_mode "$OPENASE_AUTH_MODE"

case "$OPENASE_LOG_FORMAT" in
  text|json)
    ;;
  *)
    fail "OPENASE_LOG_FORMAT must be either text or json"
    ;;
esac

mkdir -p \
  "$HOME/.codex" \
  "$HOME/.openase" \
  "$HOME/.openase/chat" \
  "$HOME/.openase/logs" \
  "$HOME/.openase/projects" \
  "$HOME/.openase/workspace"

[ -w "$HOME/.codex" ] || fail "${HOME}/.codex is not writable"
[ -w "$HOME/.openase" ] || fail "${HOME}/.openase is not writable"

if [ -f "$CODEX_CONFIG_SOURCE_PATH" ]; then
  cp "$CODEX_CONFIG_SOURCE_PATH" "$HOME/.codex/config.toml"
  chmod 600 "$HOME/.codex/config.toml"
fi

printf 'openase-entrypoint: starting openase all-in-one on %s:%s\n' \
  "$OPENASE_SERVER_HOST" "$OPENASE_SERVER_PORT" >&2

exec /usr/local/bin/openase all-in-one
