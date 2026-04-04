---
name: "land"
description: "Land a reviewed change only after CI and branch-state checks pass."
---

# Land Safely

合并前确认：

1. 当前分支已同步最新主干
2. 关键测试和 CI 通过
3. 已处理 review comment
4. 合并后不留下临时调试代码

优先保持线性、可追踪的历史。
