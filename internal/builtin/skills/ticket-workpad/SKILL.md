---
name: "ticket-workpad"
description: "Maintain the persistent ticket work log comment on the current ticket and use it as the execution log."
---

# Ticket Workpad

Workpad 是当前工单唯一的持久化进度板。这个 skill 应绑定到需要持续执行和续跑的 ticket workflow 上；绑定后，agent 才会明确知道应该把跨 runtime 需要保留的信息写到同一条持久化评论里，而不是散落在临时上下文中。

它的职责不是提供平台 API，而是建立一层基于平台 comment 原语之上的持久化约定：

- 平台基座由 `openase-platform` skill 提供：`ticket comment list/create/update`
- `ticket-workpad` 在这个基座上定义“哪一条评论是 workpad、如何幂等 upsert、应该记录哪些段落”
- 这样即使 runtime 重启、agent 重新调度、会话上下文丢失，后续 agent 仍然能从同一条 workpad 评论恢复执行状态

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

执行时遵循：

- 开工前先写第一版 workpad，不要先改代码再补记录。
- 每完成一个关键阶段就更新同一条评论，不要不断创建新评论。
- 至少持续维护 `Plan`、`Progress`、`Validation`、`Notes` 这些段落。
- 把 workpad 当成跨 runtime 的恢复点，记录后续 agent 续跑时真正需要的信息，而不是写成一次性聊天摘要。
- 如果当前 workflow 绑定了这个 skill，就默认必须维护 workpad；它不是可选装饰，而是执行闭环的一部分。
- 如果被阻塞，把阻塞原因和缺失前置条件写进 workpad，而不是静默退出。
