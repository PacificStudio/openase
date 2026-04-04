---
name: "push"
description: "Push verified changes carefully and avoid destructive history rewrites."
---

# Safe Push

推送前确认：

- 相关验证已完成
- 本地分支已和远端同步
- 不使用破坏性强推覆盖他人历史

推送失败时先同步远端，再解决冲突后重新验证。
