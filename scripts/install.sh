#!/bin/sh

set -eu

OPENASE_INSTALL_REPO=${OPENASE_INSTALL_REPO:-pacificstudio/openase}
OPENASE_INSTALL_RELEASES_BASE_URL=${OPENASE_INSTALL_RELEASES_BASE_URL:-https://github.com/${OPENASE_INSTALL_REPO}/releases}
OPENASE_INSTALL_VERSION=${OPENASE_INSTALL_VERSION:-}
OPENASE_INSTALL_INSTALL_DIR=${OPENASE_INSTALL_INSTALL_DIR:-}
OPENASE_INSTALL_PG_MODE=${OPENASE_INSTALL_PG_MODE:-}
OPENASE_INSTALL_YES=${OPENASE_INSTALL_YES:-0}
OPENASE_INSTALL_DRY_RUN=${OPENASE_INSTALL_DRY_RUN:-0}
OPENASE_INSTALL_ALLOW_OVERWRITE_CONFIG=${OPENASE_INSTALL_ALLOW_OVERWRITE_CONFIG:-0}
OPENASE_INSTALL_UNAME_S=${OPENASE_INSTALL_UNAME_S:-}
OPENASE_INSTALL_UNAME_M=${OPENASE_INSTALL_UNAME_M:-}

SCRIPT_NAME=install.sh
OPENASE_BIN_NAME=openase

say() {
  printf '%s\n' "$*"
}

info() {
  printf '==> %s\n' "$*"
}

warn() {
  printf 'warning: %s\n' "$*" >&2
}

die() {
  printf 'error: %s\n' "$*" >&2
  exit 1
}

usage() {
  cat <<'EOF'
OpenASE installer

Usage:
  install.sh [options]

Options:
  --version <tag>        Install a specific release tag, for example v0.4.0.
  --install-dir <dir>    Install the openase binary into the given directory.
  --pg-mode <mode>       PostgreSQL bootstrap mode: system, docker, or skip.
  --overwrite-config     Allow the installer to overwrite ~/.openase/config.yaml.
  -y, --yes              Accept the safe default at every prompt.
  --dry-run              Print the resolved plan and exit without making changes.
  -h, --help             Show this help text.

Environment overrides:
  OPENASE_INSTALL_RELEASES_BASE_URL
  OPENASE_INSTALL_VERSION
  OPENASE_INSTALL_INSTALL_DIR
  OPENASE_INSTALL_PG_MODE
  OPENASE_INSTALL_YES=1
  OPENASE_INSTALL_DRY_RUN=1
EOF
}

parse_args() {
  while [ "$#" -gt 0 ]; do
    case "$1" in
      --version)
        [ "$#" -ge 2 ] || die "--version requires a value"
        OPENASE_INSTALL_VERSION=$2
        shift 2
        ;;
      --install-dir)
        [ "$#" -ge 2 ] || die "--install-dir requires a value"
        OPENASE_INSTALL_INSTALL_DIR=$2
        shift 2
        ;;
      --pg-mode)
        [ "$#" -ge 2 ] || die "--pg-mode requires a value"
        OPENASE_INSTALL_PG_MODE=$2
        shift 2
        ;;
      --overwrite-config)
        OPENASE_INSTALL_ALLOW_OVERWRITE_CONFIG=1
        shift
        ;;
      -y|--yes)
        OPENASE_INSTALL_YES=1
        shift
        ;;
      --dry-run)
        OPENASE_INSTALL_DRY_RUN=1
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        die "unknown argument: $1"
        ;;
    esac
  done
}

open_prompt_fd() {
  if (: </dev/tty) 2>/dev/null; then
    exec 3</dev/tty
  else
    exec 3<&0
  fi
}

prompt_choice() {
  prompt_label=$1
  default_choice=$2
  option_count=$3
  option_1=${4:-}
  option_2=${5:-}
  option_3=${6:-}

  if [ "$OPENASE_INSTALL_YES" = "1" ]; then
    printf '%s' "$default_choice"
    return 0
  fi

  while :; do
    printf '\n' >&2
    printf '%s\n' "$prompt_label" >&2
    [ -n "$option_1" ] && printf '  1) %s\n' "$option_1" >&2
    [ -n "$option_2" ] && printf '  2) %s\n' "$option_2" >&2
    [ -n "$option_3" ] && printf '  3) %s\n' "$option_3" >&2
    printf 'Choose [default %s]: ' "$default_choice" >&2
    if ! IFS= read -r selection <&3; then
      selection=""
    fi
    [ -n "$selection" ] || selection=$default_choice
    case "$selection" in
      1|2|3)
        if [ "$selection" -le "$option_count" ] 2>/dev/null; then
          printf '%s' "$selection"
          return 0
        fi
        ;;
    esac
    warn "Please enter a number between 1 and ${option_count}, or press Enter for ${default_choice}."
  done
}

prompt_string() {
  prompt_label=$1
  default_value=${2:-}

  if [ "$OPENASE_INSTALL_YES" = "1" ]; then
    printf '%s' "$default_value"
    return 0
  fi

  while :; do
    if [ -n "$default_value" ]; then
      printf '%s [%s]: ' "$prompt_label" "$default_value" >&2
    else
      printf '%s: ' "$prompt_label" >&2
    fi
    if ! IFS= read -r value <&3; then
      value=""
    fi
    [ -n "$value" ] || value=$default_value
    if [ -n "$value" ]; then
      printf '%s' "$value"
      return 0
    fi
    warn "A value is required."
  done
}

need_command() {
  command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"
}

normalize_tag() {
  raw_tag=$(printf '%s' "$1" | tr -d '[:space:]')
  [ -n "$raw_tag" ] || die "Release version cannot be empty."
  case "$raw_tag" in
    v*) printf '%s' "$raw_tag" ;;
    *) printf 'v%s' "$raw_tag" ;;
  esac
}

detect_platform() {
  raw_os=$OPENASE_INSTALL_UNAME_S
  raw_arch=$OPENASE_INSTALL_UNAME_M

  [ -n "$raw_os" ] || raw_os=$(uname -s)
  [ -n "$raw_arch" ] || raw_arch=$(uname -m)

  lower_os=$(printf '%s' "$raw_os" | tr '[:upper:]' '[:lower:]')
  lower_arch=$(printf '%s' "$raw_arch" | tr '[:upper:]' '[:lower:]')

  case "$lower_os" in
    linux) OPENASE_OS=linux ;;
    darwin) OPENASE_OS=darwin ;;
    *)
      die "Unsupported OS: ${raw_os}. Supported platforms are linux and darwin."
      ;;
  esac

  case "$lower_arch" in
    x86_64|amd64) OPENASE_ARCH=amd64 ;;
    arm64|aarch64) OPENASE_ARCH=arm64 ;;
    *)
      die "Unsupported architecture: ${raw_arch}. Supported architectures are amd64 and arm64."
      ;;
  esac
}

resolve_latest_version() {
  latest_url=${OPENASE_INSTALL_RELEASES_BASE_URL%/}/latest
  resolved_url=$(curl -fsS -L -o /dev/null -w '%{url_effective}' "$latest_url") || {
    die "Could not resolve the latest OpenASE release from ${latest_url}. Try rerunning with --version <tag>."
  }
  latest_tag=$(basename "$resolved_url")
  case "$latest_tag" in
    v*) printf '%s' "$latest_tag" ;;
    *)
      die "Could not determine the latest OpenASE release tag from ${resolved_url}. Try rerunning with --version <tag>."
      ;;
  esac
}

dir_or_parent_writable() {
  target_dir=$1
  probe_dir=$target_dir
  while [ ! -d "$probe_dir" ]; do
    next_dir=$(dirname "$probe_dir")
    [ "$next_dir" != "$probe_dir" ] || break
    probe_dir=$next_dir
  done
  [ -w "$probe_dir" ]
}

path_contains_dir() {
  path_dir=$1
  old_ifs=$IFS
  IFS=:
  for entry in $PATH; do
    if [ "$entry" = "$path_dir" ]; then
      IFS=$old_ifs
      return 0
    fi
  done
  IFS=$old_ifs
  return 1
}

run_as_root() {
  if [ "$(id -u)" -eq 0 ]; then
    "$@"
    return 0
  fi
  command -v sudo >/dev/null 2>&1 || die "This step requires root privileges, but sudo is not available."
  sudo "$@"
}

run_as_postgres() {
  if [ "$SUPPORTED_PACKAGE_MANAGER" = "brew" ]; then
    psql postgres "$@"
    return 0
  fi

  if [ "$(id -u)" -eq 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
      sudo -u postgres psql postgres "$@"
      return 0
    fi
    die "Root access is available, but sudo is required to run PostgreSQL admin commands as the postgres user."
  fi

  command -v sudo >/dev/null 2>&1 || die "sudo is required to run PostgreSQL admin commands as the postgres user."
  sudo -u postgres psql postgres "$@"
}

detect_install_targets() {
  [ -n "${HOME:-}" ] || die "HOME is not set."

  SYSTEM_INSTALL_DIR=/usr/local/bin
  USER_INSTALL_DIR=${HOME}/.local/bin

  if dir_or_parent_writable "$SYSTEM_INSTALL_DIR"; then
    SYSTEM_INSTALL_STATE=writable
  elif [ "$(id -u)" -eq 0 ] || command -v sudo >/dev/null 2>&1; then
    SYSTEM_INSTALL_STATE=sudo
  else
    SYSTEM_INSTALL_STATE=unavailable
  fi

  if dir_or_parent_writable "$USER_INSTALL_DIR"; then
    USER_INSTALL_STATE=writable
  else
    USER_INSTALL_STATE=unavailable
  fi
}

detect_package_manager() {
  DETECTED_PACKAGE_MANAGERS=""
  SUPPORTED_PACKAGE_MANAGER=""

  append_pm() {
    if [ -n "$DETECTED_PACKAGE_MANAGERS" ]; then
      DETECTED_PACKAGE_MANAGERS="${DETECTED_PACKAGE_MANAGERS}, $1"
    else
      DETECTED_PACKAGE_MANAGERS=$1
    fi
  }

  if command -v brew >/dev/null 2>&1; then
    append_pm "brew"
    if [ "$OPENASE_OS" = "darwin" ] && [ -z "$SUPPORTED_PACKAGE_MANAGER" ]; then
      SUPPORTED_PACKAGE_MANAGER=brew
    fi
  fi

  if command -v apt-get >/dev/null 2>&1; then
    append_pm "apt-get"
    if [ "$OPENASE_OS" = "linux" ] && [ -z "$SUPPORTED_PACKAGE_MANAGER" ]; then
      SUPPORTED_PACKAGE_MANAGER=apt-get
    fi
  fi

  if command -v dnf >/dev/null 2>&1; then
    append_pm "dnf"
    if [ "$OPENASE_OS" = "linux" ] && [ -z "$SUPPORTED_PACKAGE_MANAGER" ]; then
      SUPPORTED_PACKAGE_MANAGER=dnf
    fi
  fi

  if command -v yum >/dev/null 2>&1; then
    append_pm "yum"
    if [ "$OPENASE_OS" = "linux" ] && [ -z "$SUPPORTED_PACKAGE_MANAGER" ]; then
      SUPPORTED_PACKAGE_MANAGER=yum
    fi
  fi

  if command -v zypper >/dev/null 2>&1; then
    append_pm "zypper"
  fi

  if [ -z "$DETECTED_PACKAGE_MANAGERS" ]; then
    DETECTED_PACKAGE_MANAGERS="none detected"
  fi
}

detect_docker() {
  DOCKER_USABLE=0
  if ! command -v docker >/dev/null 2>&1; then
    DOCKER_STATUS="not installed"
    return 0
  fi

  if docker info >/dev/null 2>&1; then
    DOCKER_USABLE=1
    DOCKER_STATUS="installed and usable"
  else
    DOCKER_STATUS="installed, but the daemon is unavailable or the current user cannot access it"
  fi
}

choose_release_tag() {
  latest_tag=$(resolve_latest_version)
  RESOLVED_LATEST_TAG=$latest_tag

  if [ -n "$OPENASE_INSTALL_VERSION" ]; then
    RELEASE_TAG=$(normalize_tag "$OPENASE_INSTALL_VERSION")
    return 0
  fi

  choice=$(prompt_choice "Choose an OpenASE release" 1 2 \
    "Latest release (${latest_tag})" \
    "Install a specific version")

  case "$choice" in
    1) RELEASE_TAG=$latest_tag ;;
    2) RELEASE_TAG=$(normalize_tag "$(prompt_string 'Release tag' "$latest_tag")") ;;
    *) die "Unexpected release selection: ${choice}" ;;
  esac
}

choose_install_dir() {
  if [ -n "$OPENASE_INSTALL_INSTALL_DIR" ]; then
    INSTALL_DIR=$OPENASE_INSTALL_INSTALL_DIR
  else
    install_option_count=0
    install_option_1=""
    install_option_2=""
    install_default=1

    if [ "$SYSTEM_INSTALL_STATE" != "unavailable" ]; then
      install_option_count=1
      if [ "$SYSTEM_INSTALL_STATE" = "writable" ]; then
        install_option_1="${SYSTEM_INSTALL_DIR} (system-level, writable now)"
      else
        install_option_1="${SYSTEM_INSTALL_DIR} (system-level, requires sudo)"
      fi
    fi

    if [ "$USER_INSTALL_STATE" != "unavailable" ]; then
      if [ "$install_option_count" -eq 0 ]; then
        install_option_count=1
        install_option_1="${USER_INSTALL_DIR} (user-level, writable now)"
        install_default=1
      else
        install_option_count=2
        install_option_2="${USER_INSTALL_DIR} (user-level, writable now)"
      fi
    fi

    if [ "$install_option_count" -eq 0 ]; then
      die "No writable install target is available. Create ${USER_INSTALL_DIR} or rerun with sudo access to ${SYSTEM_INSTALL_DIR}."
    fi

    if [ "$SYSTEM_INSTALL_STATE" != "writable" ] && [ "$USER_INSTALL_STATE" = "writable" ]; then
      if [ "$install_option_count" -eq 2 ]; then
        install_default=2
      else
        install_default=1
      fi
    fi

    if [ "$install_option_count" -eq 1 ]; then
      install_choice=1
    else
      install_choice=$(prompt_choice "Choose where to install ${OPENASE_BIN_NAME}" "$install_default" 2 "$install_option_1" "$install_option_2")
    fi

    case "$install_choice" in
      1)
        if [ "$SYSTEM_INSTALL_STATE" != "unavailable" ]; then
          INSTALL_DIR=$SYSTEM_INSTALL_DIR
        else
          INSTALL_DIR=$USER_INSTALL_DIR
        fi
        ;;
      2)
        INSTALL_DIR=$USER_INSTALL_DIR
        ;;
      *)
        die "Unexpected install target selection: ${install_choice}"
      ;;
    esac
  fi

  if dir_or_parent_writable "$INSTALL_DIR"; then
    INSTALL_WITH_SUDO=0
  elif [ "$(id -u)" -eq 0 ] || command -v sudo >/dev/null 2>&1; then
    INSTALL_WITH_SUDO=1
  else
    die "The install target ${INSTALL_DIR} is not writable. Choose a writable directory or rerun with sudo access."
  fi
}

choose_pg_mode() {
  CONFIG_PATH=${HOME}/.openase/config.yaml
  AUTO_SETUP_ALLOWED=1
  if [ -f "$CONFIG_PATH" ] && [ "$OPENASE_INSTALL_ALLOW_OVERWRITE_CONFIG" != "1" ]; then
    overwrite_choice=$(prompt_choice \
      "An existing OpenASE config was found at ${CONFIG_PATH}. How should the installer handle OpenASE setup?" \
      1 \
      2 \
      "Keep the existing config and skip automatic OpenASE setup" \
      "Overwrite the config with settings from this install")
    case "$overwrite_choice" in
      1)
        AUTO_SETUP_ALLOWED=0
        warn "The existing OpenASE config will be kept. PostgreSQL bootstrap can still run, but the installer will not rewrite ${CONFIG_PATH}."
        ;;
      2)
        OPENASE_INSTALL_ALLOW_OVERWRITE_CONFIG=1
        ;;
    esac
  fi

  if [ -n "$OPENASE_INSTALL_PG_MODE" ]; then
    PG_MODE=$OPENASE_INSTALL_PG_MODE
  else
    pg_option_count=0
    pg_default=1
    pg_label_1=""
    pg_label_2=""
    pg_label_3=""
    pg_value_1=""
    pg_value_2=""
    pg_value_3=""

    if [ -n "$SUPPORTED_PACKAGE_MANAGER" ]; then
      pg_option_count=1
      pg_value_1=system
      pg_label_1="Install PostgreSQL with ${SUPPORTED_PACKAGE_MANAGER} and configure OpenASE"
      if [ "$OPENASE_OS" = "darwin" ]; then
        pg_default=1
      fi
    fi

    if [ "$DOCKER_USABLE" = "1" ]; then
      if [ "$pg_option_count" -eq 0 ]; then
        pg_option_count=1
        pg_value_1=docker
        pg_label_1="Use Docker-backed PostgreSQL and configure OpenASE"
        pg_default=1
      else
        pg_option_count=2
        pg_value_2=docker
        pg_label_2="Use Docker-backed PostgreSQL and configure OpenASE"
        if [ "$OPENASE_OS" = "linux" ]; then
          pg_default=2
        fi
      fi
    fi

    case "$pg_option_count" in
      0)
        pg_option_count=1
        pg_value_1=skip
        pg_label_1="Skip PostgreSQL bootstrap for now"
        pg_default=1
        ;;
      1)
        pg_option_count=2
        pg_value_2=skip
        pg_label_2="Skip PostgreSQL bootstrap for now"
        ;;
      2)
        pg_option_count=3
        pg_value_3=skip
        pg_label_3="Skip PostgreSQL bootstrap for now"
        ;;
    esac

    pg_choice=$(prompt_choice "Choose how to handle PostgreSQL" "$pg_default" "$pg_option_count" "$pg_label_1" "$pg_label_2" "$pg_label_3")
    case "$pg_choice" in
      1) PG_MODE=$pg_value_1 ;;
      2) PG_MODE=$pg_value_2 ;;
      3) PG_MODE=$pg_value_3 ;;
      *) die "Unexpected PostgreSQL selection: ${pg_choice}" ;;
    esac
  fi

  case "$PG_MODE" in
    system)
      [ -n "$SUPPORTED_PACKAGE_MANAGER" ] || die "System-package PostgreSQL bootstrap is not available here. Install Docker or rerun with --pg-mode skip."
      ;;
    docker)
      [ "$DOCKER_USABLE" = "1" ] || die "Docker-backed PostgreSQL is not available here. Start Docker or rerun with --pg-mode system or --pg-mode skip."
      ;;
    skip)
      ;;
    *)
      die "Unsupported PostgreSQL mode: ${PG_MODE}. Expected system, docker, or skip."
      ;;
  esac
}

compute_sha256() {
  file_path=$1
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file_path" | awk '{print $1}'
    return 0
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$file_path" | awk '{print $1}'
    return 0
  fi
  die "Neither sha256sum nor shasum is available, so the release checksum cannot be verified."
}

download_release_assets() {
  ASSET_NAME="openase_${RELEASE_TAG}_${OPENASE_OS}_${OPENASE_ARCH}.tar.gz"
  ARCHIVE_URL=${OPENASE_INSTALL_RELEASES_BASE_URL%/}/download/${RELEASE_TAG}/${ASSET_NAME}
  CHECKSUM_URL=${OPENASE_INSTALL_RELEASES_BASE_URL%/}/download/${RELEASE_TAG}/checksums.txt
  ARCHIVE_PATH=${TMP_DIR}/${ASSET_NAME}
  CHECKSUM_PATH=${TMP_DIR}/checksums.txt

  info "Downloading ${ASSET_NAME}"
  curl -fsSL "$ARCHIVE_URL" -o "$ARCHIVE_PATH" || {
    die "Could not download ${ARCHIVE_URL}. Confirm that release ${RELEASE_TAG} has a ${OPENASE_OS}/${OPENASE_ARCH} archive."
  }

  info "Downloading checksums.txt"
  curl -fsSL "$CHECKSUM_URL" -o "$CHECKSUM_PATH" || {
    die "Could not download ${CHECKSUM_URL}. Confirm that release ${RELEASE_TAG} publishes checksums.txt."
  }
}

verify_release_archive() {
  expected_sha=$(awk -v asset_name="$ASSET_NAME" '$2 == asset_name { print $1; exit }' "$CHECKSUM_PATH")
  [ -n "$expected_sha" ] || die "checksums.txt does not contain an entry for ${ASSET_NAME}."

  actual_sha=$(compute_sha256 "$ARCHIVE_PATH")
  [ "$actual_sha" = "$expected_sha" ] || die "Checksum verification failed for ${ASSET_NAME}. Expected ${expected_sha}, got ${actual_sha}."
}

extract_release_binary() {
  EXTRACT_DIR=${TMP_DIR}/extract
  mkdir -p "$EXTRACT_DIR"
  tar -xzf "$ARCHIVE_PATH" -C "$EXTRACT_DIR" || die "Could not extract ${ASSET_NAME}."

  EXTRACTED_BIN=$(find "$EXTRACT_DIR" -type f -name "$OPENASE_BIN_NAME" | head -n 1)
  [ -n "$EXTRACTED_BIN" ] || die "The release archive does not contain ${OPENASE_BIN_NAME}."
  chmod +x "$EXTRACTED_BIN"
}

install_release_binary() {
  DEST_BIN=${INSTALL_DIR}/${OPENASE_BIN_NAME}
  info "Installing ${OPENASE_BIN_NAME} to ${DEST_BIN}"
  if [ "$INSTALL_WITH_SUDO" = "1" ]; then
    run_as_root mkdir -p "$INSTALL_DIR"
    run_as_root cp "$EXTRACTED_BIN" "$DEST_BIN"
    run_as_root chmod 755 "$DEST_BIN"
  else
    mkdir -p "$INSTALL_DIR"
    cp "$EXTRACTED_BIN" "$DEST_BIN"
    chmod 755 "$DEST_BIN"
  fi

  INSTALLED_BIN=$DEST_BIN
  version_output=$("$INSTALLED_BIN" version 2>&1) || {
    die "The installed binary at ${INSTALLED_BIN} did not pass 'openase version'. Output: ${version_output}"
  }
  say "$version_output"
}

start_linux_postgres_service() {
  if command -v systemctl >/dev/null 2>&1; then
    if run_as_root systemctl start postgresql >/dev/null 2>&1; then
      return 0
    fi
    if run_as_root systemctl start postgresql.service >/dev/null 2>&1; then
      return 0
    fi
  fi

  if command -v service >/dev/null 2>&1; then
    if run_as_root service postgresql start >/dev/null 2>&1; then
      return 0
    fi
  fi

  if command -v pg_ctlcluster >/dev/null 2>&1 && command -v pg_lsclusters >/dev/null 2>&1; then
    cluster_version=$(pg_lsclusters -h 2>/dev/null | awk 'NR==1 { print $1 }')
    cluster_name=$(pg_lsclusters -h 2>/dev/null | awk 'NR==1 { print $2 }')
    if [ -n "$cluster_version" ] && [ -n "$cluster_name" ]; then
      if run_as_root pg_ctlcluster "$cluster_version" "$cluster_name" start >/dev/null 2>&1; then
        return 0
      fi
    fi
  fi

  return 1
}

ensure_brew_postgres_path() {
  if [ "$SUPPORTED_PACKAGE_MANAGER" != "brew" ]; then
    return 0
  fi
  brew_prefix=$(brew --prefix postgresql@16 2>/dev/null || true)
  if [ -n "$brew_prefix" ] && [ -d "$brew_prefix/bin" ]; then
    PATH="${brew_prefix}/bin:${PATH}"
    export PATH
  fi
}

configure_postgres_role_and_database() {
  OPENASE_DB_HOST=127.0.0.1
  OPENASE_DB_PORT=5432
  OPENASE_DB_NAME=openase
  OPENASE_DB_USER=openase
  if command -v openssl >/dev/null 2>&1; then
    OPENASE_DB_PASSWORD=$(openssl rand -hex 16)
  else
    OPENASE_DB_PASSWORD=$(od -An -N16 -tx1 /dev/urandom | tr -d ' \n')
  fi
  export OPENASE_DB_PASSWORD

  role_exists=$(run_as_postgres -tAc "SELECT 1 FROM pg_roles WHERE rolname = '${OPENASE_DB_USER}'" 2>/dev/null || true)

  if [ "$role_exists" = "1" ]; then
    run_as_postgres -c "ALTER ROLE ${OPENASE_DB_USER} WITH LOGIN PASSWORD '${OPENASE_DB_PASSWORD}'" >/dev/null
  else
    run_as_postgres -c "CREATE ROLE ${OPENASE_DB_USER} WITH LOGIN PASSWORD '${OPENASE_DB_PASSWORD}'" >/dev/null
  fi

  db_exists=$(run_as_postgres -tAc "SELECT 1 FROM pg_database WHERE datname = '${OPENASE_DB_NAME}'" 2>/dev/null || true)

  if [ "$db_exists" = "1" ]; then
    run_as_postgres -c "ALTER DATABASE ${OPENASE_DB_NAME} OWNER TO ${OPENASE_DB_USER}" >/dev/null 2>&1 || true
  else
    run_as_postgres -c "CREATE DATABASE ${OPENASE_DB_NAME} OWNER ${OPENASE_DB_USER}" >/dev/null
  fi
}

bootstrap_system_postgres() {
  info "Installing PostgreSQL via ${SUPPORTED_PACKAGE_MANAGER}"
  case "$SUPPORTED_PACKAGE_MANAGER" in
    brew)
      brew list postgresql@16 >/dev/null 2>&1 || brew install postgresql@16
      ensure_brew_postgres_path
      brew services start postgresql@16 >/dev/null || die "Homebrew could not start postgresql@16. Run 'brew services start postgresql@16' manually and rerun the installer."
      ;;
    apt-get)
      run_as_root apt-get update
      run_as_root apt-get install -y postgresql postgresql-client
      start_linux_postgres_service || die "PostgreSQL was installed, but the service could not be started automatically. Start it with 'sudo service postgresql start' or 'sudo systemctl start postgresql', then rerun the installer."
      ;;
    dnf)
      run_as_root dnf install -y postgresql-server postgresql
      if command -v postgresql-setup >/dev/null 2>&1; then
        run_as_root postgresql-setup --initdb >/dev/null 2>&1 || true
      fi
      start_linux_postgres_service || die "PostgreSQL was installed, but the service could not be started automatically. Start it with 'sudo systemctl start postgresql', then rerun the installer."
      ;;
    yum)
      run_as_root yum install -y postgresql-server postgresql
      if command -v postgresql-setup >/dev/null 2>&1; then
        run_as_root postgresql-setup --initdb >/dev/null 2>&1 || true
      fi
      start_linux_postgres_service || die "PostgreSQL was installed, but the service could not be started automatically. Start it with 'sudo systemctl start postgresql', then rerun the installer."
      ;;
    *)
      die "System-package PostgreSQL bootstrap is not implemented for ${SUPPORTED_PACKAGE_MANAGER}."
      ;;
  esac

  ensure_brew_postgres_path
  command -v psql >/dev/null 2>&1 || die "PostgreSQL installed successfully, but psql is still missing from PATH."

  configure_postgres_role_and_database
}

run_setup_apply() {
  setup_input_path=${TMP_DIR}/setup-apply.json
  setup_output_path=${TMP_DIR}/setup-apply-output.json
  cat > "$setup_input_path"

  if ! "$INSTALLED_BIN" setup apply --input "$setup_input_path" >"$setup_output_path"; then
    cat "$setup_output_path" >&2 || true
    die "OpenASE setup apply failed. Resolve the database issue, then rerun the installer or run '${INSTALLED_BIN} setup' manually."
  fi

  if ! grep -q '"ready"[[:space:]]*:[[:space:]]*true' "$setup_output_path"; then
    setup_title=$(sed -n 's/.*"title"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$setup_output_path" | head -n 1)
    setup_action=$(sed -n 's/.*"action"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$setup_output_path" | head -n 1)
    [ -n "$setup_title" ] || setup_title="OpenASE setup did not finish successfully."
    [ -n "$setup_action" ] || setup_action="Rerun the installer after fixing the database issue, or run '${INSTALLED_BIN} setup' manually."
    die "${setup_title} ${setup_action}"
  fi
}

bootstrap_docker_postgres() {
  info "Preparing Docker-backed PostgreSQL"
  run_setup_apply <<EOF
{
  "database": {
    "type": "docker"
  },
  "allow_overwrite": $( [ "$OPENASE_INSTALL_ALLOW_OVERWRITE_CONFIG" = "1" ] && printf 'true' || printf 'false' )
}
EOF
}

bootstrap_manual_postgres() {
  info "Configuring OpenASE to use the local PostgreSQL service"
  run_setup_apply <<EOF
{
  "database": {
    "type": "manual",
    "manual": {
      "host": "${OPENASE_DB_HOST}",
      "port": ${OPENASE_DB_PORT},
      "name": "${OPENASE_DB_NAME}",
      "user": "${OPENASE_DB_USER}",
      "password": "${OPENASE_DB_PASSWORD}",
      "ssl_mode": "disable"
    }
  },
  "allow_overwrite": $( [ "$OPENASE_INSTALL_ALLOW_OVERWRITE_CONFIG" = "1" ] && printf 'true' || printf 'false' )
}
EOF
}

print_environment_summary() {
  say
  say "Detected environment:"
  say "  OS: ${OPENASE_OS}"
  say "  Architecture: ${OPENASE_ARCH}"
  say "  Package managers: ${DETECTED_PACKAGE_MANAGERS}"
  if [ -n "$SUPPORTED_PACKAGE_MANAGER" ]; then
    say "  Automated PostgreSQL package path: ${SUPPORTED_PACKAGE_MANAGER}"
  else
    say "  Automated PostgreSQL package path: not available"
  fi
  say "  Docker: ${DOCKER_STATUS}"
  say "  Install targets:"
  case "$SYSTEM_INSTALL_STATE" in
    writable) say "    - ${SYSTEM_INSTALL_DIR} (system-level, writable now)" ;;
    sudo) say "    - ${SYSTEM_INSTALL_DIR} (system-level, requires sudo)" ;;
    *) say "    - ${SYSTEM_INSTALL_DIR} (unavailable without sudo)" ;;
  esac
  case "$USER_INSTALL_STATE" in
    writable) say "    - ${USER_INSTALL_DIR} (user-level, writable now)" ;;
    *) say "    - ${USER_INSTALL_DIR} (not currently writable)" ;;
  esac
}

print_dry_run_plan() {
  say
  say "Dry-run plan:"
  say "  release_tag=${RELEASE_TAG}"
  say "  asset_name=${ASSET_NAME}"
  say "  install_dir=${INSTALL_DIR}"
  say "  install_with_sudo=${INSTALL_WITH_SUDO}"
  say "  pg_mode=${PG_MODE}"
}

print_completion_notes() {
  say
  say "OpenASE is installed at ${INSTALLED_BIN}."
  if [ "$PG_MODE" = "skip" ]; then
    say "PostgreSQL bootstrap was skipped."
    say "Next step: run '${INSTALLED_BIN} setup' after preparing PostgreSQL yourself, or rerun this installer and choose PostgreSQL bootstrap."
  elif [ "$AUTO_SETUP_ALLOWED" = "0" ]; then
    say "PostgreSQL bootstrap completed, but the existing OpenASE config was left unchanged at ${CONFIG_PATH}."
    say "Next step: update ${CONFIG_PATH} manually or rerun with --overwrite-config if you want the installer to rewrite it."
  else
    say "OpenASE configuration was written to ${HOME}/.openase/config.yaml."
    say "Next step: start OpenASE with '${INSTALLED_BIN} all-in-one --config ${HOME}/.openase/config.yaml'."
  fi

  if ! path_contains_dir "$INSTALL_DIR"; then
    say
    say "Your shell PATH does not currently include ${INSTALL_DIR}."
    say "Add this line to your shell profile, then open a new shell:"
    say "  export PATH=\"${INSTALL_DIR}:\$PATH\""
  fi
}

main() {
  parse_args "$@"
  open_prompt_fd

  need_command curl
  need_command tar
  need_command mktemp

  detect_platform
  detect_install_targets
  detect_package_manager
  detect_docker

  print_environment_summary
  choose_release_tag
  choose_install_dir
  choose_pg_mode

  TMP_DIR=$(mktemp -d)
  trap 'rm -rf "$TMP_DIR"' EXIT INT TERM

  ASSET_NAME="openase_${RELEASE_TAG}_${OPENASE_OS}_${OPENASE_ARCH}.tar.gz"
  print_dry_run_plan
  if [ "$OPENASE_INSTALL_DRY_RUN" = "1" ]; then
    exit 0
  fi

  download_release_assets
  verify_release_archive
  extract_release_binary
  install_release_binary

  case "$PG_MODE" in
    system)
      bootstrap_system_postgres
      if [ "$AUTO_SETUP_ALLOWED" = "1" ]; then
        bootstrap_manual_postgres
      fi
      ;;
    docker)
      if [ "$AUTO_SETUP_ALLOWED" = "1" ]; then
        bootstrap_docker_postgres
      fi
      ;;
    skip)
      ;;
  esac

  print_completion_notes
}

main "$@"
