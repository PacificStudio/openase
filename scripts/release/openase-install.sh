#!/usr/bin/env bash
set -euo pipefail

OPENASE_RELEASE_REPO="${OPENASE_RELEASE_REPO:-pacificstudio/openase}"
OPENASE_RELEASE_HOST="${OPENASE_RELEASE_HOST:-https://github.com}"
OPENASE_INSTALL_VERSION="${OPENASE_INSTALL_VERSION:-latest}"
OPENASE_INSTALL_DIR="${OPENASE_INSTALL_DIR:-$HOME/.local/bin}"
OPENASE_HOME_DIR="${OPENASE_HOME_DIR:-$HOME/.openase}"
OPENASE_CONFIG_PATH="${OPENASE_CONFIG_PATH:-$OPENASE_HOME_DIR/config.yaml}"
OPENASE_INSTALL_SETUP_MODE="${OPENASE_INSTALL_SETUP_MODE:-docker}"
OPENASE_INSTALL_START_MODE="${OPENASE_INSTALL_START_MODE:-service}"
OPENASE_INSTALL_ALLOW_OVERWRITE="${OPENASE_INSTALL_ALLOW_OVERWRITE:-0}"
OPENASE_CONTROL_PLANE_URL="${OPENASE_CONTROL_PLANE_URL:-http://127.0.0.1:19836}"

log() {
  printf '[openase-install] %s\n' "$*"
}

warn() {
  printf '[openase-install] warning: %s\n' "$*" >&2
}

fail() {
  printf '[openase-install] error: %s\n' "$*" >&2
  exit 1
}

print_usage() {
  cat <<'USAGE'
Install the latest OpenASE release binary and optionally bootstrap/start it.

Environment variables:
  OPENASE_INSTALL_VERSION           Release tag to install, default: latest
  OPENASE_INSTALL_DIR               Binary install dir, default: ~/.local/bin
  OPENASE_HOME_DIR                  OpenASE state dir, default: ~/.openase
  OPENASE_CONFIG_PATH               Config path, default: ~/.openase/config.yaml
  OPENASE_INSTALL_SETUP_MODE        docker | manual | skip, default: docker
  OPENASE_INSTALL_START_MODE        service | foreground | skip, default: service
  OPENASE_INSTALL_ALLOW_OVERWRITE   1 to recreate config when it already exists
  OPENASE_CONTROL_PLANE_URL         Health/bootstrap base URL, default: http://127.0.0.1:19836

Manual database mode expects:
  OPENASE_DATABASE_HOST
  OPENASE_DATABASE_PORT
  OPENASE_DATABASE_NAME
  OPENASE_DATABASE_USER
  OPENASE_DATABASE_PASSWORD         Optional
  OPENASE_DATABASE_SSL_MODE         Optional, default: disable

Docker mode optional overrides:
  OPENASE_DOCKER_CONTAINER_NAME
  OPENASE_DOCKER_DATABASE_NAME
  OPENASE_DOCKER_DATABASE_USER
  OPENASE_DOCKER_PORT
  OPENASE_DOCKER_VOLUME_NAME
  OPENASE_DOCKER_IMAGE

Examples:
  curl -fsSL https://github.com/pacificstudio/openase/releases/latest/download/openase-install.sh | bash
  OPENASE_INSTALL_VERSION=v0.1.0 curl -fsSL https://github.com/pacificstudio/openase/releases/latest/download/openase-install.sh | bash
  OPENASE_INSTALL_SETUP_MODE=skip OPENASE_INSTALL_START_MODE=skip bash ./openase-install.sh
USAGE
}

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    fail "required command not found: $1"
  fi
}

normalize_bool() {
  case "${1:-0}" in
    1|true|TRUE|yes|YES|on|ON) printf '1\n' ;;
    0|false|FALSE|no|NO|off|OFF|'') printf '0\n' ;;
    *) fail "unsupported boolean value: $1" ;;
  esac
}

json_escape() {
  local value="${1-}"
  value=${value//\\/\\\\}
  value=${value//\"/\\\"}
  value=${value//$'\n'/\\n}
  value=${value//$'\r'/\\r}
  value=${value//$'\t'/\\t}
  printf '%s' "$value"
}

parse_release_tag_from_url() {
  local url="${1%/}"
  local tag="${url##*/}"
  if [[ -z "$tag" || "$tag" == "latest" ]]; then
    fail "could not resolve release tag from $1"
  fi
  printf '%s\n' "$tag"
}

curl_download() {
  local token="${OPENASE_GITHUB_TOKEN:-${GITHUB_TOKEN:-}}"
  if [[ -n "$token" ]]; then
    curl -fsSL -H "Authorization: Bearer $token" "$@"
    return
  fi
  curl -fsSL "$@"
}

resolve_release_tag() {
  if [[ "$OPENASE_INSTALL_VERSION" != "latest" ]]; then
    printf '%s\n' "$OPENASE_INSTALL_VERSION"
    return
  fi

  local latest_url
  latest_url="$(curl_download -o /dev/null -w '%{url_effective}' -L "${OPENASE_RELEASE_HOST}/${OPENASE_RELEASE_REPO}/releases/latest")" || \
    fail "failed to resolve the latest release tag from ${OPENASE_RELEASE_HOST}/${OPENASE_RELEASE_REPO}"
  parse_release_tag_from_url "$latest_url"
}

detect_os() {
  case "$(uname -s)" in
    Linux) printf 'linux\n' ;;
    Darwin) printf 'darwin\n' ;;
    *) fail "unsupported operating system: $(uname -s)" ;;
  esac
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) printf 'amd64\n' ;;
    arm64|aarch64) printf 'arm64\n' ;;
    *) fail "unsupported CPU architecture: $(uname -m)" ;;
  esac
}

checksum_file_sha256() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$1" | awk '{print $1}'
    return
  fi
  fail "sha256sum or shasum is required to verify release checksums"
}

copy_executable() {
  local source_path="$1"
  local target_path="$2"

  if command -v install >/dev/null 2>&1; then
    install -m 0755 "$source_path" "$target_path"
    return
  fi

  cp "$source_path" "$target_path"
  chmod 0755 "$target_path"
}

ensure_install_dir() {
  mkdir -p "$OPENASE_INSTALL_DIR"
}

verify_archive_checksum() {
  local archive_path="$1"
  local checksums_path="$2"
  local archive_name="${archive_path##*/}"
  local expected actual

  expected="$(awk -v target="$archive_name" '$2 == target {print $1}' "$checksums_path")"
  if [[ -z "$expected" ]]; then
    fail "checksum entry missing for $archive_name"
  fi

  actual="$(checksum_file_sha256 "$archive_path")"
  if [[ "$expected" != "$actual" ]]; then
    fail "checksum mismatch for $archive_name"
  fi
}

write_setup_request() {
  local destination="$1"
  local mode="$OPENASE_INSTALL_SETUP_MODE"
  local overwrite
  overwrite="$(normalize_bool "$OPENASE_INSTALL_ALLOW_OVERWRITE")"

  case "$mode" in
    docker)
      cat > "$destination" <<JSON
{
  "database": {
    "type": "docker",
    "docker": {
      "container_name": "$(json_escape "${OPENASE_DOCKER_CONTAINER_NAME:-openase-local-postgres}")",
      "database_name": "$(json_escape "${OPENASE_DOCKER_DATABASE_NAME:-openase}")",
      "user": "$(json_escape "${OPENASE_DOCKER_DATABASE_USER:-openase}")",
      "port": ${OPENASE_DOCKER_PORT:-15432},
      "volume_name": "$(json_escape "${OPENASE_DOCKER_VOLUME_NAME:-openase-local-postgres-data}")",
      "image": "$(json_escape "${OPENASE_DOCKER_IMAGE:-postgres:16-alpine}")"
    }
  },
  "allow_overwrite": ${overwrite}
}
JSON
      ;;
    manual)
      : "${OPENASE_DATABASE_HOST:?OPENASE_DATABASE_HOST is required when OPENASE_INSTALL_SETUP_MODE=manual}"
      : "${OPENASE_DATABASE_PORT:?OPENASE_DATABASE_PORT is required when OPENASE_INSTALL_SETUP_MODE=manual}"
      : "${OPENASE_DATABASE_NAME:?OPENASE_DATABASE_NAME is required when OPENASE_INSTALL_SETUP_MODE=manual}"
      : "${OPENASE_DATABASE_USER:?OPENASE_DATABASE_USER is required when OPENASE_INSTALL_SETUP_MODE=manual}"
      cat > "$destination" <<JSON
{
  "database": {
    "type": "manual",
    "manual": {
      "host": "$(json_escape "$OPENASE_DATABASE_HOST")",
      "port": ${OPENASE_DATABASE_PORT},
      "name": "$(json_escape "$OPENASE_DATABASE_NAME")",
      "user": "$(json_escape "$OPENASE_DATABASE_USER")",
      "password": "$(json_escape "${OPENASE_DATABASE_PASSWORD:-}")",
      "ssl_mode": "$(json_escape "${OPENASE_DATABASE_SSL_MODE:-disable}")"
    }
  },
  "allow_overwrite": ${overwrite}
}
JSON
      ;;
    skip)
      return 0
      ;;
    *)
      fail "unsupported OPENASE_INSTALL_SETUP_MODE: $mode"
      ;;
  esac
}

run_auto_setup() {
  local binary_path="$1"
  local request_path="$2"

  case "$OPENASE_INSTALL_SETUP_MODE" in
    docker)
      need_cmd docker
      ;;
    manual)
      :
      ;;
    skip)
      log "skipping OpenASE config bootstrap"
      return 0
      ;;
    *)
      fail "unsupported OPENASE_INSTALL_SETUP_MODE: $OPENASE_INSTALL_SETUP_MODE"
      ;;
  esac

  write_setup_request "$request_path"
  log "bootstrapping OpenASE config via setup desktop apply"
  "$binary_path" setup desktop apply --input "$request_path"
}

wait_for_health() {
  local url="$1"
  local attempt=0
  local max_attempts=30

  while (( attempt < max_attempts )); do
    if curl -fsS "$url" >/dev/null 2>&1; then
      return 0
    fi
    attempt=$((attempt + 1))
    sleep 1
  done

  return 1
}

print_bootstrap_link() {
  local binary_path="$1"
  if ! "$binary_path" auth bootstrap create-link --config "$OPENASE_CONFIG_PATH" --return-to / --format text; then
    warn "OpenASE started, but failed to generate a local bootstrap link"
  fi
}

start_openase() {
  local binary_path="$1"

  case "$OPENASE_INSTALL_START_MODE" in
    service)
      log "installing or refreshing the managed OpenASE user service"
      "$binary_path" up --config "$OPENASE_CONFIG_PATH"
      if wait_for_health "${OPENASE_CONTROL_PLANE_URL}/healthz"; then
        log "OpenASE is healthy at ${OPENASE_CONTROL_PLANE_URL}"
      else
        warn "service started but health check did not pass at ${OPENASE_CONTROL_PLANE_URL}/healthz yet"
      fi
      log "local bootstrap link:"
      print_bootstrap_link "$binary_path"
      ;;
    foreground)
      log "starting OpenASE in the foreground"
      exec "$binary_path" all-in-one --config "$OPENASE_CONFIG_PATH"
      ;;
    skip)
      log "skipping OpenASE start"
      ;;
    *)
      fail "unsupported OPENASE_INSTALL_START_MODE: $OPENASE_INSTALL_START_MODE"
      ;;
  esac
}

main() {
  if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    print_usage
    exit 0
  fi

  need_cmd curl
  need_cmd tar
  need_cmd mktemp
  need_cmd uname
  need_cmd awk

  local os_name arch_name release_tag package_basename archive_name tmp_dir archive_path checksums_path
  local download_base binary_path extracted_dir request_path

  os_name="$(detect_os)"
  arch_name="$(detect_arch)"
  release_tag="$(resolve_release_tag)"
  package_basename="openase_${release_tag}_${os_name}_${arch_name}"
  archive_name="${package_basename}.tar.gz"
  download_base="${OPENASE_RELEASE_HOST}/${OPENASE_RELEASE_REPO}/releases/download/${release_tag}"

  log "resolved release ${release_tag} for ${os_name}/${arch_name}"

  tmp_dir="$(mktemp -d)"
  trap "rm -rf -- $(printf '%q' "$tmp_dir")" EXIT
  archive_path="${tmp_dir}/${archive_name}"
  checksums_path="${tmp_dir}/checksums.txt"
  request_path="${tmp_dir}/setup-request.json"

  curl_download -o "$archive_path" "${download_base}/${archive_name}"
  curl_download -o "$checksums_path" "${download_base}/checksums.txt"
  verify_archive_checksum "$archive_path" "$checksums_path"

  tar -xzf "$archive_path" -C "$tmp_dir"
  extracted_dir="${tmp_dir}/${package_basename}"
  binary_path="${extracted_dir}/openase"
  if [[ ! -x "$binary_path" ]]; then
    fail "release archive did not contain an executable openase binary"
  fi

  ensure_install_dir
  copy_executable "$binary_path" "${OPENASE_INSTALL_DIR}/openase"
  binary_path="${OPENASE_INSTALL_DIR}/openase"

  log "installed ${binary_path}"
  "$binary_path" version

  if [[ -f "$OPENASE_CONFIG_PATH" && "$(normalize_bool "$OPENASE_INSTALL_ALLOW_OVERWRITE")" != "1" ]]; then
    log "existing config detected at ${OPENASE_CONFIG_PATH}; skipping setup"
  else
    run_auto_setup "$binary_path" "$request_path"
  fi

  if [[ "$OPENASE_INSTALL_START_MODE" != "skip" ]]; then
    if [[ ! -f "$OPENASE_CONFIG_PATH" ]]; then
      fail "missing config at ${OPENASE_CONFIG_PATH}; rerun with OPENASE_INSTALL_SETUP_MODE=docker or configure manual database settings"
    fi
  fi

  start_openase "$binary_path"

  if [[ ":${PATH}:" != *":${OPENASE_INSTALL_DIR}:"* ]]; then
    warn "${OPENASE_INSTALL_DIR} is not on PATH yet; add it if you want to run 'openase' directly in new shells"
  fi
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
