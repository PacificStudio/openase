# OpenASE Desktop v1

The `desktop/` package hosts the Electron shell for OpenASE desktop v1.

It keeps the current architecture intact:

- Go `openase all-in-one` remains the local service kernel.
- The existing SvelteKit app remains the UI.
- PostgreSQL remains the database dependency for v1.
- Electron only owns process lifecycle, window hosting, logs, and packaging.

Use the repo-root documentation for the full workflow and packaging guide:

- `docs/en/desktop-v1.md`
- `docs/zh/desktop-v1.md`
