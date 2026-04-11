# Builtin Bundle Provenance

- Source repository: `https://github.com/PacificStudio/auto-harness-skill`
- Default branch: `main`
- Upstream skill path: `skills/auto-harness`
- Builtin skill name: `auto-harness`

## Sync Strategy

This OpenASE builtin bundle is a vendored copy of the upstream skill bundle.

- Refreshes are manual vendor updates from the upstream repository.
- Preserve the upstream structure when syncing: `SKILL.md`, `agents/`, and `references/`.
- Re-run builtin skill loading and runtime projection tests after each refresh.

## Why This File Exists

OpenASE ships `auto-harness` as a builtin capability so new sessions and new
workspaces can discover and use it without a separate GitHub installation
step, while still keeping the upstream source of truth visible to maintainers.
