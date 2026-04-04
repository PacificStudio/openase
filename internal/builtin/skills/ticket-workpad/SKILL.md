---
name: "ticket-workpad"
description: "Maintain the persistent ticket work log comment on the current ticket and use it as the execution log."
---

# Ticket Workpad

The Workpad is the single persistent progress board for the current ticket. Bind this skill to ticket workflows that need long-running execution and resume support so agents consistently write durable state into one persistent comment instead of scattering it across ephemeral context.

Its job is not to expose platform APIs directly. It defines a persistence convention on top of primitive platform comments:

- The platform base comes from the `openase-platform` skill: `ticket comment list/create/update`.
- `ticket-workpad` defines which comment counts as the workpad, how to upsert it idempotently, and which sections should be maintained.
- This lets later agents recover execution state from the same workpad comment even after runtime restarts, rescheduling, or lost conversation context.

Workpad upsert is no longer a standalone CLI subcommand. Call the injected `openase-platform` helper script instead; it will add the standard heading automatically and reuse or update the persistent comment.

Recommended usage:

```bash
cat <<'EOF' >/tmp/workpad.md
Environment
- <host>:<abs-workdir>@<short-sha>

Plan
- step 1
- step 2

Progress
- inspecting current implementation

Validation
- not run yet

Notes
- assumptions or blockers
EOF

OPENASE_PLATFORM_HELPER=""
for candidate in \
  ./.codex/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.claude/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.gemini/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.agents/skills/openase-platform/scripts/upsert_workpad.sh \
  ./.agent/skills/openase-platform/scripts/upsert_workpad.sh
do
  if [ -x "$candidate" ]; then
    OPENASE_PLATFORM_HELPER="$candidate"
    break
  fi
done

if [ -z "$OPENASE_PLATFORM_HELPER" ]; then
  echo "openase-platform helper script not found" >&2
  exit 1
fi

"$OPENASE_PLATFORM_HELPER" --body-file /tmp/workpad.md
```

Execution rules:

- Write the first workpad before you start changing code. Do not backfill it afterward.
- Update the same comment after each major phase instead of creating new comments repeatedly.
- Continuously maintain at least `Plan`, `Progress`, `Validation`, and `Notes`.
- Treat the workpad as the resume point across runtimes and record the information the next agent will actually need, not a one-off chat summary.
- If the workflow binds this skill, maintaining the workpad is mandatory. It is part of the execution loop, not optional decoration.
- If you are blocked, record the blocker and missing prerequisites in the workpad instead of exiting silently.
