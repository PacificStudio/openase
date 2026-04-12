#!/usr/bin/env python3

from __future__ import annotations

import argparse
import contextlib
import http.server
import io
import os
import pathlib
import shutil
import socketserver
import subprocess
import sys
import tarfile
import tempfile
import textwrap
import threading
from dataclasses import dataclass
from hashlib import sha256


REPO_ROOT = pathlib.Path(__file__).resolve().parents[2]
INSTALLER_PATH = REPO_ROOT / "scripts" / "install.sh"


def fail(message: str) -> None:
    raise AssertionError(message)


def write_executable(path: pathlib.Path, content: str) -> None:
    path.write_text(content)
    path.chmod(0o755)


def build_fake_openase_script(tag: str) -> str:
    return textwrap.dedent(
        f"""\
        #!/bin/sh
        set -eu

        case "${{1:-}}" in
          version)
            printf 'openase {tag}\\n'
            ;;
          setup)
            [ "${{2:-}}" = "apply" ] || {{
              printf 'unexpected setup subcommand\\n' >&2
              exit 1
            }}
            [ "${{3:-}}" = "--input" ] || {{
              printf 'missing --input\\n' >&2
              exit 1
            }}
            python3 - "$4" <<'PY'
        import json
        import os
        import pathlib
        import sys

        request_path = pathlib.Path(sys.argv[1])
        payload = json.loads(request_path.read_text())
        home = pathlib.Path(os.environ["HOME"])
        root = home / ".openase"
        root.mkdir(parents=True, exist_ok=True)
        database = payload["database"]["type"]
        (root / "config.yaml").write_text(f"database_source: {{database}}\\n")
        (root / ".env").write_text("OPENASE_AUTH_TOKEN=test-token\\n")
        result = {{
            "ready": True,
            "config_path": str(root / "config.yaml"),
            "env_path": str(root / ".env"),
            "database_source": database,
        }}
        print(json.dumps(result, indent=2))
        PY
            ;;
          *)
            printf 'unexpected openase invocation: %s\\n' "$*" >&2
            exit 1
            ;;
        esac
        """
    )


def build_release_tree(root: pathlib.Path, tags: list[str]) -> None:
    platforms = [
        ("linux", "amd64"),
        ("linux", "arm64"),
        ("darwin", "amd64"),
        ("darwin", "arm64"),
    ]

    for tag in tags:
        download_dir = root / "releases" / "download" / tag
        download_dir.mkdir(parents=True, exist_ok=True)
        checksum_lines: list[str] = []

        for goos, goarch in platforms:
            asset_name = f"openase_{tag}_{goos}_{goarch}.tar.gz"
            archive_path = download_dir / asset_name
            package_dir = f"openase_{tag}_{goos}_{goarch}"

            with tempfile.TemporaryDirectory() as temp_dir:
                temp_root = pathlib.Path(temp_dir)
                package_root = temp_root / package_dir
                package_root.mkdir(parents=True, exist_ok=True)
                write_executable(package_root / "openase", build_fake_openase_script(tag))
                (package_root / "README.md").write_text("fixture\n")
                (package_root / "LICENSE").write_text("fixture\n")

                with tarfile.open(archive_path, "w:gz") as tar:
                    tar.add(package_root, arcname=package_root.name)

            digest = sha256(archive_path.read_bytes()).hexdigest()
            checksum_lines.append(f"{digest}  {asset_name}\n")

        (download_dir / "checksums.txt").write_text("".join(checksum_lines))


class ReleaseHandler(http.server.SimpleHTTPRequestHandler):
    def __init__(self, *args, directory: str, latest_tag: str, **kwargs):
        self.latest_tag = latest_tag
        super().__init__(*args, directory=directory, **kwargs)

    def do_GET(self) -> None:  # noqa: N802
        if self.path == "/releases/latest":
            self.send_response(302)
            self.send_header("Location", f"/releases/tag/{self.latest_tag}")
            self.end_headers()
            return
        if self.path == f"/releases/tag/{self.latest_tag}":
            body = f"<html><body>{self.latest_tag}</body></html>".encode()
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        super().do_GET()

    def log_message(self, fmt: str, *args) -> None:  # noqa: A003
        return


@contextlib.contextmanager
def release_server(root: pathlib.Path, latest_tag: str):
    server = socketserver.TCPServer(
        ("127.0.0.1", 0),
        lambda *args, **kwargs: ReleaseHandler(*args, directory=str(root), latest_tag=latest_tag, **kwargs),
    )
    thread = threading.Thread(target=server.serve_forever, daemon=True)
    thread.start()
    try:
        host, port = server.server_address
        yield f"http://{host}:{port}"
    finally:
        server.shutdown()
        server.server_close()
        thread.join(timeout=5)


def fake_command_script() -> str:
    return textwrap.dedent(
        """\
        #!/bin/sh
        set -eu

        cmd_name=$(basename "$0")
        log_file=${FAKE_LOG:?}

        printf '%s %s\\n' "$cmd_name" "$*" >>"$log_file"

        case "$cmd_name" in
          docker)
            if [ "${1:-}" = "info" ] && [ "${FAKE_DOCKER_INFO_OK:-0}" = "1" ]; then
              exit 0
            fi
            exit 1
            ;;
          brew)
            if [ "${1:-}" = "list" ]; then
              exit 1
            fi
            if [ "${1:-}" = "install" ]; then
              exit 0
            fi
            if [ "${1:-}" = "--prefix" ]; then
              printf '%s\\n' "${FAKE_BREW_PREFIX:?}"
              exit 0
            fi
            if [ "${1:-}" = "services" ] && [ "${2:-}" = "start" ]; then
              exit 0
            fi
            exit 0
            ;;
          apt-get|dnf|yum|systemctl|service|pg_ctlcluster)
            exit 0
            ;;
          pg_lsclusters)
            printf '16 main 5432 online postgres /var/lib/postgresql/16/main /var/log/postgresql/postgresql-16-main.log\\n'
            exit 0
            ;;
          sudo)
            if [ "${1:-}" = "-u" ]; then
              shift 2
            fi
            exec "$@"
            ;;
          psql)
            case "$*" in
              *"SELECT 1 FROM pg_roles"*)
                exit 0
                ;;
              *"SELECT 1 FROM pg_database"*)
                exit 0
                ;;
            esac
            exit 0
            ;;
          *)
            printf 'unexpected fake command: %s\\n' "$cmd_name" >&2
            exit 1
            ;;
        esac
        """
    )


def build_fake_bin(root: pathlib.Path) -> tuple[pathlib.Path, pathlib.Path]:
    fake_bin = root / "fake-bin"
    fake_bin.mkdir(parents=True, exist_ok=True)
    script = fake_command_script()
    for name in ["docker", "brew", "apt-get", "dnf", "yum", "systemctl", "service", "pg_ctlcluster", "pg_lsclusters", "sudo", "psql"]:
        write_executable(fake_bin / name, script)

    brew_prefix = root / "fake-brew-prefix"
    brew_bin = brew_prefix / "bin"
    brew_bin.mkdir(parents=True, exist_ok=True)
    write_executable(brew_bin / "psql", script)

    return fake_bin, brew_prefix


@dataclass
class RunResult:
    process: subprocess.CompletedProcess[str]
    home_dir: pathlib.Path
    fake_log: pathlib.Path


def run_installer(
    *,
    release_base_url: str,
    fake_bin: pathlib.Path,
    brew_prefix: pathlib.Path,
    stdin: str = "",
    args: list[str] | None = None,
    extra_env: dict[str, str] | None = None,
) -> RunResult:
    args = args or []
    extra_env = extra_env or {}

    root = pathlib.Path(tempfile.mkdtemp(prefix="openase-installer-test-"))
    home_dir = root / "home"
    home_dir.mkdir(parents=True, exist_ok=True)
    fake_log = root / "fake.log"
    fake_log.write_text("")

    env = os.environ.copy()
    env.update(
        {
            "HOME": str(home_dir),
            "PATH": f"{fake_bin}:{env['PATH']}",
            "FAKE_LOG": str(fake_log),
            "FAKE_BREW_PREFIX": str(brew_prefix),
            "OPENASE_INSTALL_RELEASES_BASE_URL": f"{release_base_url}/releases",
        }
    )
    env.update(extra_env)

    process = subprocess.run(
        ["sh", str(INSTALLER_PATH), *args],
        input=stdin,
        text=True,
        capture_output=True,
        env=env,
        cwd=str(REPO_ROOT),
        check=False,
    )
    return RunResult(process=process, home_dir=home_dir, fake_log=fake_log)


def assert_success(result: RunResult, context: str) -> None:
    if result.process.returncode != 0:
      fail(
          f"{context} failed with exit code {result.process.returncode}\n"
          f"stdout:\n{result.process.stdout}\n"
          f"stderr:\n{result.process.stderr}"
      )


def assert_failure(result: RunResult, expected_substring: str, context: str) -> None:
    if result.process.returncode == 0:
        fail(f"{context} unexpectedly succeeded\nstdout:\n{result.process.stdout}")
    combined = result.process.stdout + result.process.stderr
    if expected_substring not in combined:
        fail(f"{context} did not include expected text {expected_substring!r}\ncombined output:\n{combined}")


def test_latest_default_linux_docker(release_base_url: str, fake_bin: pathlib.Path, brew_prefix: pathlib.Path) -> None:
    result = run_installer(
        release_base_url=release_base_url,
        fake_bin=fake_bin,
        brew_prefix=brew_prefix,
        stdin="\n\n\n",
        extra_env={
            "OPENASE_INSTALL_UNAME_S": "Linux",
            "OPENASE_INSTALL_UNAME_M": "x86_64",
            "FAKE_DOCKER_INFO_OK": "1",
        },
    )
    assert_success(result, "latest default linux installer")

    config_path = result.home_dir / ".openase" / "config.yaml"
    if "database_source: docker" not in config_path.read_text():
        fail(f"default linux docker setup did not write docker config:\n{config_path.read_text()}")

    installed_candidates = [
        pathlib.Path("/usr/local/bin/openase"),
        result.home_dir / ".local" / "bin" / "openase",
    ]
    if not any(path.exists() for path in installed_candidates):
        fail(
            "default linux install did not place the binary in either supported default location:\n"
            + "\n".join(str(path) for path in installed_candidates)
        )

    combined = result.process.stdout + result.process.stderr
    if "Automated PostgreSQL package path: apt-get" not in combined:
        fail(f"default linux install did not print package-manager detection:\n{combined}")
    if "Docker: installed and usable" not in combined:
        fail(f"default linux install did not print docker detection:\n{combined}")


def test_pinned_version_system_package_linux(release_base_url: str, fake_bin: pathlib.Path, brew_prefix: pathlib.Path) -> None:
    custom_dir = pathlib.Path(tempfile.mkdtemp(prefix="openase-installer-custom-")) / "bin"
    result = run_installer(
        release_base_url=release_base_url,
        fake_bin=fake_bin,
        brew_prefix=brew_prefix,
        args=["--version", "v1.2.3", "--pg-mode", "system", "--install-dir", str(custom_dir), "--yes"],
        extra_env={
            "OPENASE_INSTALL_UNAME_S": "Linux",
            "OPENASE_INSTALL_UNAME_M": "x86_64",
        },
    )
    assert_success(result, "pinned linux system-package installer")

    log_text = result.fake_log.read_text()
    if "apt-get update" not in log_text or "apt-get install -y postgresql postgresql-client" not in log_text:
        fail(f"system-package linux install did not invoke apt-get as expected:\n{log_text}")
    if "psql postgres -c CREATE ROLE openase WITH LOGIN PASSWORD" not in log_text:
        fail(f"system-package linux install did not configure PostgreSQL credentials:\n{log_text}")

    config_path = result.home_dir / ".openase" / "config.yaml"
    if "database_source: manual" not in config_path.read_text():
        fail(f"system-package linux install did not write manual config:\n{config_path.read_text()}")

    installed_bin = custom_dir / "openase"
    if not installed_bin.exists():
        fail(f"pinned linux install did not place binary at {installed_bin}")


def test_macos_brew_path(release_base_url: str, fake_bin: pathlib.Path, brew_prefix: pathlib.Path) -> None:
    result = run_installer(
        release_base_url=release_base_url,
        fake_bin=fake_bin,
        brew_prefix=brew_prefix,
        args=["--version", "v9.9.9", "--pg-mode", "system", "--yes"],
        extra_env={
            "OPENASE_INSTALL_UNAME_S": "Darwin",
            "OPENASE_INSTALL_UNAME_M": "arm64",
        },
    )
    assert_success(result, "macOS brew installer")

    log_text = result.fake_log.read_text()
    if "brew install postgresql@16" not in log_text or "brew services start postgresql@16" not in log_text:
        fail(f"macOS installer did not invoke brew PostgreSQL commands:\n{log_text}")

    config_path = result.home_dir / ".openase" / "config.yaml"
    if "database_source: manual" not in config_path.read_text():
        fail(f"macOS installer did not write manual config:\n{config_path.read_text()}")


def test_checksum_failure(release_base_url: str, fake_bin: pathlib.Path, brew_prefix: pathlib.Path, fixtures_root: pathlib.Path) -> None:
    checksums_path = fixtures_root / "releases" / "download" / "v9.9.9" / "checksums.txt"
    original = checksums_path.read_text()
    checksums_path.write_text(original.replace(original.splitlines()[0][:64], "0" * 64, 1))
    try:
        result = run_installer(
            release_base_url=release_base_url,
            fake_bin=fake_bin,
            brew_prefix=brew_prefix,
            args=["--version", "v9.9.9", "--pg-mode", "skip", "--yes"],
            extra_env={
                "OPENASE_INSTALL_UNAME_S": "Linux",
                "OPENASE_INSTALL_UNAME_M": "x86_64",
            },
        )
        assert_failure(result, "Checksum verification failed", "checksum mismatch installer")
    finally:
        checksums_path.write_text(original)


def test_docker_unavailable_failure(release_base_url: str, fake_bin: pathlib.Path, brew_prefix: pathlib.Path) -> None:
    result = run_installer(
        release_base_url=release_base_url,
        fake_bin=fake_bin,
        brew_prefix=brew_prefix,
        args=["--version", "v9.9.9", "--pg-mode", "docker", "--yes"],
        extra_env={
            "OPENASE_INSTALL_UNAME_S": "Linux",
            "OPENASE_INSTALL_UNAME_M": "x86_64",
        },
    )
    assert_failure(result, "Docker-backed PostgreSQL is not available here", "docker unavailable installer")


def test_arch_mapping_dry_run(release_base_url: str, fake_bin: pathlib.Path, brew_prefix: pathlib.Path) -> None:
    cases = [
        ("Linux", "x86_64", "openase_v9.9.9_linux_amd64.tar.gz"),
        ("Linux", "arm64", "openase_v9.9.9_linux_arm64.tar.gz"),
        ("Darwin", "x86_64", "openase_v9.9.9_darwin_amd64.tar.gz"),
        ("Darwin", "arm64", "openase_v9.9.9_darwin_arm64.tar.gz"),
    ]

    for uname_s, uname_m, expected_asset in cases:
        result = run_installer(
            release_base_url=release_base_url,
            fake_bin=fake_bin,
            brew_prefix=brew_prefix,
            args=["--version", "v9.9.9", "--pg-mode", "skip", "--dry-run", "--yes"],
            extra_env={
                "OPENASE_INSTALL_UNAME_S": uname_s,
                "OPENASE_INSTALL_UNAME_M": uname_m,
            },
        )
        assert_success(result, f"dry-run mapping for {uname_s}/{uname_m}")
        if expected_asset not in result.process.stdout:
            fail(
                f"dry-run mapping for {uname_s}/{uname_m} did not include {expected_asset}\n"
                f"stdout:\n{result.process.stdout}"
            )


def run_suite() -> None:
    with tempfile.TemporaryDirectory(prefix="openase-installer-fixtures-") as temp_dir:
        fixtures_root = pathlib.Path(temp_dir)
        build_release_tree(fixtures_root, ["v9.9.9", "v1.2.3"])
        fake_bin, brew_prefix = build_fake_bin(fixtures_root)

        with release_server(fixtures_root, latest_tag="v9.9.9") as base_url:
            test_latest_default_linux_docker(base_url, fake_bin, brew_prefix)
            test_pinned_version_system_package_linux(base_url, fake_bin, brew_prefix)
            test_macos_brew_path(base_url, fake_bin, brew_prefix)
            test_checksum_failure(base_url, fake_bin, brew_prefix, fixtures_root)
            test_docker_unavailable_failure(base_url, fake_bin, brew_prefix)
            test_arch_mapping_dry_run(base_url, fake_bin, brew_prefix)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Run hermetic tests for scripts/install.sh")
    return parser.parse_args()


def main() -> int:
    parse_args()
    try:
        run_suite()
    except Exception as exc:  # noqa: BLE001
        print(str(exc), file=sys.stderr)
        return 1
    print("installer tests passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
