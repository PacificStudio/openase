---
name: "ticket-workpad"
description: "Maintain the persistent Workpad comment on the current ticket and use it as the execution log."
---

# Ticket Workpad

Workpad 是当前工单唯一的持久化进度板。开始执行前先创建或更新它，之后在关键节点持续刷新。

Workpad upsert 不再是独立 CLI 子命令。现在应调用注入的 `openase-platform` helper script；它会自动补标准 workpad 标题，并复用或更新那条持久化评论。

推荐写法：

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

./.agent/skills/openase-platform/scripts/upsert_workpad.sh --body-file /tmp/workpad.md
```

执行时遵循：

- 开工前先写第一版 workpad，不要先改代码再补记录。
- 每完成一个关键阶段就更新同一条评论，不要不断创建新评论。
- 至少持续维护 `Plan`、`Progress`、`Validation`、`Notes` 这些段落。
- 如果被阻塞，把阻塞原因和缺失前置条件写进 workpad，而不是静默退出。
