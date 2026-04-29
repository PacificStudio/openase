# Installation Test Matrix

This document describes the CI coverage for OpenASE installation and bring-up
paths.

## Layers

OpenASE keeps installation validation in `scripts/ci/` with three layers:

- `test_installer.py` checks `scripts/install.sh` hermetically with fake
  release and package-manager fixtures.
- `installer_smoke.sh` validates the release-installer path end to end against a
  locally served release archive.
- `install_matrix_smoke.sh` validates Linux runtime bring-up and health checks
  across installation path and PostgreSQL backend combinations.

## Linux Runtime Matrix

The GitHub Actions `install-matrix` job runs these cases on `ubuntu-latest`:

| Install path | PostgreSQL backend | What it proves |
| --- | --- | --- |
| `installer_script` | `docker` | One-command installer can bootstrap Docker PostgreSQL, write config, and start a healthy OpenASE runtime |
| `installer_script` | `system` | One-command installer can bootstrap system PostgreSQL, write config, and start a healthy OpenASE runtime |
| `source_build` | `docker` | Current checkout can build from source, run against Docker PostgreSQL, and pass health checks |
| `source_build` | `system` | Current checkout can build from source, run against system PostgreSQL, and pass health checks |
| `release_binary` | `docker` | Downloaded release binary can run against Docker PostgreSQL and pass health checks |
| `release_binary` | `system` | Downloaded release binary can run against system PostgreSQL and pass health checks |

Every case verifies:

- `openase version`
- `openase all-in-one`
- `GET /healthz`
- `GET /api/v1/healthz`

## Local Reproduction

Run one case locally with:

```bash
INSTALL_PATH=installer_script PG_MODE=docker make install-matrix-smoke
INSTALL_PATH=source_build PG_MODE=system make install-matrix-smoke
INSTALL_PATH=release_binary PG_MODE=docker make install-matrix-smoke
```

The matrix smoke script is Linux-only because it relies on Docker and apt-based
system PostgreSQL setup for the runtime cases.
