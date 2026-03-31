---
name: prd-alignment-scan
description:
  Scan OpenASE PRD chapters against the current codebase by enumerating top-level
  chapters, optionally targeting all chapters or a selected subset, then running
  parallel Codex chapter reviews and synthesizing a checklist of PRD-vs-code
  drift. Use when asked to verify implementation alignment with OpenASE-PRD.md,
  list chapters, scan specific chapters, or summarize remaining PRD gaps.
---

# PRD Alignment Scan

Use this skill when the user wants a PRD-vs-code gap analysis based on
`OpenASE-PRD.md`.

## What This Skill Does

- Lists top-level PRD chapters with line ranges.
- Runs one `codex exec` process per selected chapter.
- Stores one output file per chapter plus a manifest.
- Optionally runs a final Codex synthesis pass that deduplicates the chapter
  outputs into one checklist.

## Files

- Skill script: `scripts/run_prd_alignment_scan.py`

## Default Workflow

1. List chapters first if the user did not specify which chapters to scan:

```bash
python3 .codex/skills/prd-alignment-scan/scripts/run_prd_alignment_scan.py --list-chapters
```

2. Run the scan:

- Full scan:

```bash
python3 .codex/skills/prd-alignment-scan/scripts/run_prd_alignment_scan.py \
  --all \
  --output-dir /tmp/prd-alignment-scan
```

- Specific chapters:

```bash
python3 .codex/skills/prd-alignment-scan/scripts/run_prd_alignment_scan.py \
  --chapters 5,6,7,8,10,11,18,25,27,29,33,34 \
  --output-dir /tmp/prd-alignment-scan
```

- Title filter:

```bash
python3 .codex/skills/prd-alignment-scan/scripts/run_prd_alignment_scan.py \
  --match "编排引擎|Hook|状态|Platform API" \
  --output-dir /tmp/prd-alignment-scan
```

3. Read:

- `manifest.json`
- `chNN.txt` per selected chapter
- `summary.md` when synthesis is enabled

4. In the final user response:

- First list the scanned chapters and skipped scope.
- Then summarize only the verified gaps.
- Separate:
  - core semantic drift
  - API/contract drift
  - UI/product drift
  - uncertain items

## Important Notes

- The script defaults to `--dangerously-bypass-approvals-and-sandbox` because
  this environment blocks local file reads under Codex read-only sandbox.
- Prefer `--chapters` or `--match` when the user only cares about a subsystem.
- Treat product-vision chapters carefully. If a chapter is not directly
  testable from code, say so instead of inventing gaps.
- The script only orchestrates Codex passes; you still need to sanity-check the
  highest-impact claims against the code before presenting them as final.

## Output Contract

Each chapter file should contain:

1. `本章关键承诺`
2. `已实现且基本符合的点`
3. `明确存在的偏差`
4. `不确定项`

The synthesized summary should deduplicate repeated findings across chapters and
produce a prioritized checklist.
