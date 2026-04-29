#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
release_tag="${OPENASE_INSTALL_TEST_TAG:-v9.9.9}"
install_path=""
pg_mode=""
work_root=""
server_pid=""
openase_pid=""
docker_container=""
health_port=""
fixture_port=""
home_dir=""
install_dir=""
openase_log=""
openase_err_log=""
openase_bin=""
package_dir=""
archive_name=""
archive_path=""
fixture_root=""
config_path=""
env_path=""
manual_db_dsn=""
manual_db_user=""
manual_db_name=""
manual_db_password=""

usage() {
  cat <<'USAGE'
Usage: install_matrix_smoke.sh --install-path <installer_script|source_build|release_binary> --pg-mode <docker|system>
USAGE
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --install-path)
      install_path="${2:-}"
      shift 2
      ;;
    --pg-mode)
      pg_mode="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'Unknown argument: %s\n' "$1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

case "$install_path" in
  installer_script|source_build|release_binary) ;;
  *)
    printf 'Unsupported --install-path: %s\n' "$install_path" >&2
    usage >&2
    exit 1
    ;;
esac

case "$pg_mode" in
  docker|system) ;;
  *)
    printf 'Unsupported --pg-mode: %s\n' "$pg_mode" >&2
    usage >&2
    exit 1
    ;;
esac

if [[ "$(uname -s)" != "Linux" ]]; then
  printf 'install_matrix_smoke.sh currently supports Linux only\n' >&2
  exit 1
fi

find_free_port() {
  python3 - <<'PY'
import socket
with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
}

cleanup() {
  status=$?
  set +e

  if [[ -n "$openase_pid" ]]; then
    kill "$openase_pid" 2>/dev/null || true
    wait "$openase_pid" 2>/dev/null || true
  fi

  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi

  if [[ -n "$docker_container" ]]; then
    docker rm -f "$docker_container" >/dev/null 2>&1 || true
  fi

  if [[ "$install_path" == "installer_script" && "$pg_mode" == "docker" ]]; then
    docker rm -f openase-local-postgres >/dev/null 2>&1 || true
    docker volume rm openase-local-postgres-data >/dev/null 2>&1 || true
  fi

  if [[ "$status" -ne 0 ]]; then
    printf '\n[install-matrix] failure context for %s / %s\n' "$install_path" "$pg_mode" >&2
    if [[ -n "$openase_log" && -f "$openase_log" ]]; then
      printf '\n--- openase stdout ---\n' >&2
      tail -n 120 "$openase_log" >&2 || true
    fi
    if [[ -n "$openase_err_log" && -f "$openase_err_log" ]]; then
      printf '\n--- openase stderr ---\n' >&2
      tail -n 120 "$openase_err_log" >&2 || true
    fi
    if [[ "$pg_mode" == "docker" ]]; then
      if [[ -n "$docker_container" ]]; then
        printf '\n--- docker logs (%s) ---\n' "$docker_container" >&2
        docker logs "$docker_container" >&2 || true
      elif docker ps -a --format '{{.Names}}' | grep -Fxq openase-local-postgres; then
        printf '\n--- docker logs (openase-local-postgres) ---\n' >&2
        docker logs openase-local-postgres >&2 || true
      fi
    fi
  fi

  if [[ -n "$work_root" ]]; then
    rm -rf "$work_root"
  fi

  exit "$status"
}
trap cleanup EXIT

map_os_arch() {
  case "$(uname -m)" in
    x86_64|amd64)
      goarch="amd64"
      ;;
    arm64|aarch64)
      goarch="arm64"
      ;;
    *)
      printf 'Unsupported architecture: %s\n' "$(uname -m)" >&2
      exit 1
      ;;
  esac
  goos="linux"
}

make_home_layout() {
  home_dir="$work_root/home"
  install_dir="$work_root/install/bin"
  mkdir -p "$home_dir/.openase" "$install_dir"
  chmod 700 "$home_dir/.openase"
  config_path="$home_dir/.openase/config.yaml"
  env_path="$home_dir/.openase/.env"
  openase_log="$work_root/openase.stdout.log"
  openase_err_log="$work_root/openase.stderr.log"
}

build_release_fixture() {
  fixture_root="$work_root/fixture"
  package_dir="$work_root/openase_${release_tag}_${goos}_${goarch}"
  archive_name="openase_${release_tag}_${goos}_${goarch}.tar.gz"
  archive_path="$fixture_root/releases/download/$release_tag/$archive_name"
  mkdir -p "$fixture_root/releases/download/$release_tag" "$fixture_root/scripts" "$package_dir"

  cp "$repo_root/scripts/install.sh" "$fixture_root/scripts/install.sh"
  chmod +x "$fixture_root/scripts/install.sh"

  make -C "$repo_root" build-web VERSION="$release_tag" OPENASE_BIN="$package_dir/openase"
  cp "$repo_root/README.md" "$repo_root/LICENSE" "$package_dir/"

  tar -C "$work_root" -czf "$archive_path" "$(basename "$package_dir")"
  (
    cd "$(dirname "$archive_path")"
    if command -v sha256sum >/dev/null 2>&1; then
      sha256sum "$archive_name" > checksums.txt
    else
      shasum -a 256 "$archive_name" > checksums.txt
    fi
  )
}

start_fixture_server() {
  fixture_port="$(find_free_port)"
  python3 - "$fixture_root" "$fixture_port" "$release_tag" >/dev/null 2>&1 <<'PY' &
import functools
import http.server
import socketserver
import sys

root, port, release_tag = sys.argv[1], int(sys.argv[2]), sys.argv[3]

class Handler(http.server.SimpleHTTPRequestHandler):
    def do_GET(self):  # noqa: N802
        if self.path == "/releases/latest":
            self.send_response(302)
            self.send_header("Location", f"/releases/tag/{release_tag}")
            self.end_headers()
            return
        if self.path == f"/releases/tag/{release_tag}":
            body = f"<html><body>{release_tag}</body></html>".encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        super().do_GET()

    def log_message(self, fmt, *args):  # noqa: A003
        return

with socketserver.TCPServer(("127.0.0.1", port), functools.partial(Handler, directory=root)) as httpd:
    httpd.serve_forever()
PY
  server_pid=$!
  sleep 1
}

setup_system_postgres() {
  manual_db_user="openase_ci_${install_path}"
  manual_db_name="openase_ci_${install_path}"
  manual_db_password="$(python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(24))
PY
)"

  export DEBIAN_FRONTEND=noninteractive
  sudo apt-get update -qq
  sudo apt-get install -y -qq postgresql postgresql-client
  sudo systemctl start postgresql || sudo service postgresql start

  sudo -u postgres psql -v ON_ERROR_STOP=1 <<SQL
DO \$\$ BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '${manual_db_user}') THEN
      CREATE ROLE ${manual_db_user} LOGIN PASSWORD '${manual_db_password}';
   ELSE
      ALTER ROLE ${manual_db_user} WITH LOGIN PASSWORD '${manual_db_password}';
   END IF;
END \$\$;
SQL
  if ! sudo -u postgres psql -tAc "SELECT 1 FROM pg_database WHERE datname = '${manual_db_name}'" | grep -q 1; then
    sudo -u postgres createdb -O "$manual_db_user" "$manual_db_name"
  fi
  manual_db_dsn="postgres://${manual_db_user}:${manual_db_password}@127.0.0.1:5432/${manual_db_name}?sslmode=disable"
  psql "$manual_db_dsn" -c 'SELECT 1;' >/dev/null
}

setup_docker_postgres() {
  local docker_port
  docker_port="$(find_free_port)"
  docker_container="openase-install-matrix-${install_path}-${docker_port}"
  manual_db_user="openase"
  manual_db_name="openase"
  manual_db_password="$(python3 - <<'PY'
import secrets
print(secrets.token_urlsafe(24))
PY
)"

  docker rm -f "$docker_container" >/dev/null 2>&1 || true
  docker run -d --name "$docker_container" \
    -e POSTGRES_DB="$manual_db_name" \
    -e POSTGRES_USER="$manual_db_user" \
    -e POSTGRES_PASSWORD="$manual_db_password" \
    -p "127.0.0.1:${docker_port}:5432" \
    postgres:16 >/dev/null

  for _ in $(seq 1 60); do
    if docker exec "$docker_container" pg_isready -U "$manual_db_user" -d "$manual_db_name" >/dev/null 2>&1; then
      manual_db_dsn="postgres://${manual_db_user}:${manual_db_password}@127.0.0.1:${docker_port}/${manual_db_name}?sslmode=disable"
      return
    fi
    sleep 2
  done

  printf 'Docker PostgreSQL did not become ready\n' >&2
  exit 1
}

write_manual_runtime_files() {
  cat > "$config_path" <<CFG
server:
  mode: all-in-one
  host: 127.0.0.1
  port: 19836
  read_timeout: 15s
  write_timeout: 15s
  shutdown_timeout: 10s
auth:
  mode: disabled
orchestrator:
  tick_interval: 5s
log:
  level: info
  format: text
CFG
  cat > "$env_path" <<ENV
OPENASE_DATABASE_DSN=${manual_db_dsn}
ENV
  chmod 600 "$config_path" "$env_path"
}

install_via_installer_script() {
  if [[ "$pg_mode" == "docker" ]]; then
    if docker ps -a --format '{{.Names}}' | grep -Fxq openase-local-postgres; then
      printf 'Refusing to reuse pre-existing docker container openase-local-postgres\\n' >&2
      exit 1
    fi
    if docker volume ls --format '{{.Name}}' | grep -Fxq openase-local-postgres-data; then
      printf 'Refusing to reuse pre-existing docker volume openase-local-postgres-data\\n' >&2
      exit 1
    fi
  fi
  build_release_fixture
  start_fixture_server
  HOME="$home_dir" curl -fsSL "http://127.0.0.1:${fixture_port}/scripts/install.sh" \
    | HOME="$home_dir" env OPENASE_INSTALL_RELEASES_BASE_URL="http://127.0.0.1:${fixture_port}/releases" \
      sh -s -- --version "$release_tag" --pg-mode "$pg_mode" --install-dir "$install_dir" --yes
  openase_bin="$install_dir/openase"
}

install_via_source_build() {
  case "$pg_mode" in
    docker) setup_docker_postgres ;;
    system) setup_system_postgres ;;
  esac
  write_manual_runtime_files
  make -C "$repo_root" build-web VERSION="$release_tag" OPENASE_BIN="$install_dir/openase"
  openase_bin="$install_dir/openase"
}

install_via_release_binary() {
  case "$pg_mode" in
    docker) setup_docker_postgres ;;
    system) setup_system_postgres ;;
  esac
  write_manual_runtime_files
  build_release_fixture
  start_fixture_server
  curl -fsSL "http://127.0.0.1:${fixture_port}/releases/download/${release_tag}/${archive_name}" -o "$work_root/${archive_name}"
  tar -C "$work_root" -xzf "$work_root/${archive_name}"
  openase_bin="$package_dir/openase"
}

wait_for_health() {
  local url1="http://127.0.0.1:${health_port}/healthz"
  local url2="http://127.0.0.1:${health_port}/api/v1/healthz"
  for _ in $(seq 1 90); do
    if curl -fsS "$url1" >/dev/null 2>&1 && curl -fsS "$url2" >/dev/null 2>&1; then
      return
    fi
    sleep 2
  done
  printf 'Timed out waiting for OpenASE health on port %s\n' "$health_port" >&2
  exit 1
}

start_openase() {
  health_port="$(find_free_port)"
  (
    set -a
    if [[ -f "$env_path" ]]; then
      . "$env_path"
    fi
    set +a
    HOME="$home_dir" "$openase_bin" all-in-one --config "$config_path" --host 127.0.0.1 --port "$health_port"
  ) >"$openase_log" 2>"$openase_err_log" &
  openase_pid=$!
  wait_for_health
}

verify_runtime() {
  "$openase_bin" version >/dev/null
  curl -fsS "http://127.0.0.1:${health_port}/healthz" >/dev/null
  curl -fsS "http://127.0.0.1:${health_port}/api/v1/healthz" >/dev/null
  printf 'PASS %s %s health_port=%s\n' "$install_path" "$pg_mode" "$health_port"
}

map_os_arch
work_root="$(mktemp -d)"
make_home_layout

case "$install_path" in
  installer_script)
    install_via_installer_script
    ;;
  source_build)
    install_via_source_build
    ;;
  release_binary)
    install_via_release_binary
    ;;
esac

start_openase
verify_runtime
