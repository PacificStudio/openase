---
name: "ticket-workpad"
description: "Maintain the persistent Workpad comment on the current ticket and use it as the execution log."
---

# Ticket Workpad

Workpad 是当前工单唯一的持久化进度板。开始执行前先创建或更新它，之后在关键节点持续刷新。

调用 `ticket comment workpad` 时，不需要自己手动维护标题；只需要提供正文内容，让平台命令去复用或更新那条持久化 workpad 评论。

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

./.openase/bin/openase ticket comment workpad --body-file /tmp/workpad.md
```

执行时遵循：

- 开工前先写第一版 workpad，不要先改代码再补记录。
- 每完成一个关键阶段就更新同一条评论，不要不断创建新评论。
- 至少持续维护 `Plan`、`Progress`、`Validation`、`Notes` 这些段落。
- 如果被阻塞，把阻塞原因和缺失前置条件写进 workpad，而不是静默退出。
