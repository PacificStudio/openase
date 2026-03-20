---
name: report-issue
description:
  Create a detailed GitHub issue for OpenASE and add it to the OpenASE
  Automation project in Todo. Use when asked to report a bug, deployment
  regression, local repro, or design task into GitHub with reproducible
  evidence and project binding.
---

# Report Issue

## Goals

- Create a high-signal GitHub issue in the current repo.
- Add it to project `OpenASE Automation` (`#2`) with status `Todo`.
- Keep issue bodies concrete: exact branch, commit, commands, actual result,
  expected result, impact, and likely fix direction.

## Inputs To Collect

- Current branch: `git branch --show-current`
- Current commit: `git rev-parse HEAD`
- Relevant environment facts:
  - deployment mode
  - local ports
  - whether `.env` was sourced
  - whether Docker/PostgreSQL was involved
- Exact repro command or API sequence
- Exact error text or log excerpt

## Body Structure

Write issue bodies with these sections when applicable:

- `Summary`
- `Environment`
- `Reproduction`
- `Actual Result`
- `Expected Result`
- `Likely Cause`
- `Impact`
- `Suggested Fix Direction`

Prefer exact commands and exact errors over paraphrase.

## Workflow

1. Gather concrete facts from the running repo and local deployment.
2. Draft the issue body in a temp Markdown file.
3. Run:
   - `.codex/skills/report-issue/scripts/create_issue_and_add_to_project.sh --title "<title>" --body-file <file>`
4. Return the issue URL and confirm project status is `Todo`.

## Notes

- Use one issue per distinct failure mode.
- Do not merge unrelated regressions into one issue.
- If the issue is a design task, explicitly state the desired contract and
  acceptance criteria.
- If the issue references PRD drift, say that PRD must be updated to remain the
  latest semantic source of truth.
