---
name: land
description:
  Land an OpenASE PR by syncing with main, resolving conflicts, waiting for CI,
  addressing review feedback, and squash-merging when green.
---

# Land

## Goals

- Ensure the PR is conflict-free with `origin/main`.
- Keep the current OpenASE validation and CI green.
- Address human review feedback before merge.
- Squash-merge the PR once checks pass.
- Do not yield while the branch is mergeable and there is still concrete work to
  do.

## Preconditions

- `gh` CLI is authenticated.
- You are on the PR branch with a clean working tree, or you are prepared to
  commit/push outstanding intended changes first.

## Validation Gate

Use the repo's actual Go + Svelte checks instead of Symphony's Elixir flow.

- Go/backend changes or unclear scope:
  - `PATH="$PWD/.tooling/go/bin:/home/yuzhong/.local/go1.26.1/bin:$PATH" go test ./...`
  - `./scripts/ci/lint.sh`
- Frontend changes under `web/` or unclear scope:
  - `npm --prefix web install`
  - `npm --prefix web run check`
  - `npm --prefix web run build`
- Cross-cutting changes or uncertainty:
  - Run all of the above before the final merge attempt.

## Steps

1. Locate the PR for the current branch.
2. If the working tree has uncommitted intended changes, use the `commit` skill
   and then the `push` skill before proceeding.
3. Check mergeability and conflicts against `origin/main`.
4. If conflicts exist, use the `pull` skill to fetch and merge `origin/main`,
   resolve conflicts, rerun the applicable validation gate, then use the `push`
   skill to publish the updated branch.
5. Review outstanding feedback:
   - Top-level PR comments
   - Inline review comments
   - Review summaries / `CHANGES_REQUESTED`
6. Before implementing review feedback, confirm it does not conflict with the
   task's intent. If you disagree, respond with concise rationale and only ask
   the user if product intent is genuinely ambiguous.
7. Reply to review threads before or alongside the corresponding code change so
   the record is clear.
8. Watch CI and review state until the PR is ready:
   - Preferred: `python3 .codex/skills/land/land_watch.py`
   - Fallback: `gh pr checks --watch` plus explicit review comment inspection
     with `gh api`.
9. If checks fail, inspect the failing run, fix the issue, rerun the applicable
   validation gate locally, commit, push, and restart the watch loop.
10. When all checks are green and outstanding feedback is resolved or answered,
    squash-merge using the PR title/body.

## Commands

```sh
# Ensure branch and PR context
branch=$(git branch --show-current)
pr_number=$(gh pr view --json number -q .number)
pr_title=$(gh pr view --json title -q .title)
pr_body=$(gh pr view --json body -q .body)

# Check mergeability and conflicts
mergeable=$(gh pr view --json mergeable -q .mergeable)
if [ "$mergeable" = "CONFLICTING" ]; then
  echo "Run the pull skill, resolve conflicts, rerun checks, then push." >&2
  exit 1
fi

# Go validation
PATH="$PWD/.tooling/go/bin:/home/yuzhong/.local/go1.26.1/bin:$PATH" go test ./...
./scripts/ci/lint.sh

# Frontend validation
npm --prefix web install
npm --prefix web run check
npm --prefix web run build

# Preferred watcher
python3 .codex/skills/land/land_watch.py

# Failing CI investigation examples
gh pr checks
gh run list --branch "$branch"
gh run view <run-id> --log

# Squash merge
gh pr merge --squash --subject "$pr_title" --body "$pr_body"
```

## Review Handling

- Treat human review comments and blocking review states as merge blockers until
  they are fixed or explicitly answered with rationale.
- Prefer replying inline for inline feedback and on the PR conversation for
  top-level feedback.
- Keep the PR title and body aligned with the final merged scope if review
  expands or narrows the work.
- Do not merge while there are unaddressed correctness concerns.

## Failure Handling

- If CI fails, inspect logs with `gh pr checks` and `gh run view --log`, then
  fix locally and rerun the watch.
- If the PR head changes remotely, sync the branch, rerun the applicable
  validation gate, and continue.
- If mergeability is `UNKNOWN`, wait and re-check.
- Do not enable auto-merge just to bypass the manual validation loop.
