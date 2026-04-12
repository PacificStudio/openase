#!/usr/bin/env bash

set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
release_tag="v9.9.9"
tmp_root="$(mktemp -d)"
server_pid=""

cleanup() {
  exit_code=$?
  if [[ -n "${server_pid}" ]]; then
    kill "${server_pid}" 2>/dev/null || true
    wait "${server_pid}" 2>/dev/null || true
  fi
  rm -rf "${tmp_root}"
  exit "${exit_code}"
}
trap cleanup EXIT

case "$(uname -s)" in
  Linux) goos="linux" ;;
  Darwin) goos="darwin" ;;
  *)
    printf 'Unsupported smoke-test OS: %s\n' "$(uname -s)" >&2
    exit 1
    ;;
esac

case "$(uname -m)" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *)
    printf 'Unsupported smoke-test architecture: %s\n' "$(uname -m)" >&2
    exit 1
    ;;
esac

fixture_root="${tmp_root}/fixture"
download_dir="${fixture_root}/releases/download/${release_tag}"
package_dir="${tmp_root}/openase_${release_tag}_${goos}_${goarch}"
install_dir="${tmp_root}/bin"
mkdir -p "${download_dir}" "${fixture_root}/scripts" "${install_dir}"

cp "${repo_root}/scripts/install.sh" "${fixture_root}/scripts/install.sh"
chmod +x "${fixture_root}/scripts/install.sh"

make -C "${repo_root}" build VERSION="${release_tag}" OPENASE_BIN="${package_dir}/openase"
cp "${repo_root}/LICENSE" "${repo_root}/README.md" "${package_dir}/"

archive_name="openase_${release_tag}_${goos}_${goarch}.tar.gz"
archive_path="${download_dir}/${archive_name}"
tar -C "${tmp_root}" -czf "${archive_path}" "$(basename "${package_dir}")"

(
  cd "${download_dir}"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "${archive_name}" > checksums.txt
  else
    shasum -a 256 "${archive_name}" > checksums.txt
  fi
)

port="$(
python3 - <<'PY'
import socket

with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
    sock.bind(("127.0.0.1", 0))
    print(sock.getsockname()[1])
PY
)"

python3 -m http.server "${port}" --bind 127.0.0.1 --directory "${fixture_root}" >/dev/null 2>&1 &
server_pid=$!
sleep 1

curl -fsSL "http://127.0.0.1:${port}/scripts/install.sh" \
  | env \
      OPENASE_INSTALL_RELEASES_BASE_URL="http://127.0.0.1:${port}/releases" \
      sh -s -- --version "${release_tag}" --install-dir "${install_dir}" --pg-mode system --yes

"${install_dir}/openase" version | grep -F "${release_tag}" >/dev/null
[[ -f "${HOME}/.openase/config.yaml" ]]
[[ -f "${HOME}/.openase/.env" ]]
