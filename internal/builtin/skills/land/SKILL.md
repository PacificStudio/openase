---
name: "land"
description: "Land a reviewed change only after CI and branch-state checks pass."
---

# Land Safely

Before merging, confirm:

1. The branch is synced with the latest mainline.
2. Critical tests and CI are green.
3. Review comments have been addressed.
4. No temporary debug code remains after merge.

Prefer a linear, traceable history.
