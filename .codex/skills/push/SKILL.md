---
name: push
description:
  Push current branch changes to origin and create or update the corresponding
  pull request for OpenASE; use when asked to push, publish updates, or create
  a pull request.
---

# Push

## Prerequisites

- `gh` CLI is installed and available in `PATH`.
- `gh auth status` succeeds for GitHub operations in this repo.
- `corepack pnpm` is available for frontend validation.

## Goals

- Push the current branch to `origin` safely.
- Create a PR if none exists for the branch, otherwise update the existing PR.
- Keep PR metadata aligned with the actual total scope of the branch.
- Run the same local validation gates that current CI expects before publishing.

## Related Skills

- `pull`: use this when push is rejected or the branch must be merged with the
  latest `origin/main`.
- `commit`: use this when there are local changes that are intended to ship but
  are not committed yet.

## Validation Gate

Use the repo-local helper script:

- `.codex/skills/push/scripts/openase_ci_gate.sh`

That script mirrors `.github/workflows/ci.yml` instead of asking the agent to
guess which checks are needed:

- Always runs `make openapi-check` because CI always runs API contract checks.
- Detects `go_changed` and `web_changed` using the same path rules as CI.
- Runs frontend CI with `make web-install` and `corepack pnpm --dir web run ci`
  when `web_changed=true`.
- Runs backend and Go lint checks with `make check`, `make build`,
  `LINT_BASE_REV=<base> make lint`, `make lint-depguard`, and
  `make lint-architecture` when `go_changed=true`.

If the branch scope is ambiguous, use the script anyway; it already biases
toward the stricter CI-compatible outcome.

## Steps

1. Identify the current branch and inspect working tree state.
2. If there are intended but uncommitted changes, use the `commit` skill first.
3. Ensure `origin/main` is available locally:
   - `git fetch --no-tags origin main`
4. Run the local CI gate for the current branch diff:
   - `.codex/skills/push/scripts/openase_ci_gate.sh`
   - Use `--plan` first if you want to inspect which jobs it will run.
5. Push the branch to `origin`, setting upstream tracking if needed.
6. If the push is rejected because the remote moved:
   - Run the `pull` skill to merge `origin/main` and/or remote branch updates.
   - Rerun `.codex/skills/push/scripts/openase_ci_gate.sh`.
   - Push again.
   - Use `--force-with-lease` only when local history was intentionally
     rewritten.
7. If the push fails due to auth, permissions, branch protection, or workflow
   restrictions, stop and surface the exact error. Do not rewrite remotes or
   switch protocols as a workaround.
8. Ensure a PR exists for the branch:
   - If no PR exists, create one.
   - If a PR exists and is open, update it.
   - If the branch is tied to a closed or merged PR, create a new branch and a
     new PR instead of reusing stale history.
9. Write a clear PR title that describes the shipped outcome, not just the last
   commit.
10. Write or refresh the PR body explicitly. OpenASE does not currently require a
   repository PR template, so include concrete sections such as:
   - Summary
   - Validation
   - Risks / follow-up
11. Reply with the PR URL from `gh pr view`.

## Commands

```sh
# Identify branch
branch=$(git branch --show-current)

# Sync base branch for deterministic validation scope
git fetch --no-tags origin main

# Preview and run the CI-compatible validation gate
.codex/skills/push/scripts/openase_ci_gate.sh --plan
.codex/skills/push/scripts/openase_ci_gate.sh

# Push branch
git push -u origin HEAD

# Only if history was intentionally rewritten
git push --force-with-lease origin HEAD

# Inspect PR state
pr_state=$(gh pr view --json state -q .state 2>/dev/null || true)
if [ "$pr_state" = "MERGED" ] || [ "$pr_state" = "CLOSED" ]; then
  echo "Current branch is tied to a closed PR; create a new branch + PR." >&2
  exit 1
fi

# Create or update PR metadata
pr_title="<clear PR title for the total branch scope>"
tmp_pr_body=$(mktemp)
cat > "$tmp_pr_body" <<'EOF'
## Summary
- <what changed>

## Validation
- <commands run>

## Risks / Follow-up
- <risks or none>
EOF

if [ -z "$pr_state" ]; then
  gh pr create --title "$pr_title" --body-file "$tmp_pr_body"
else
  gh pr edit --title "$pr_title" --body-file "$tmp_pr_body"
fi

rm -f "$tmp_pr_body"
gh pr view --json url -q .url
```

## Notes

- Do not assume `.github/pull_request_template.md` exists in this repo.
- Prefer complete, honest PR metadata over placeholder text.
- Do not replace the helper script with ad hoc command guessing. If CI changes,
  update the script and this skill together.
