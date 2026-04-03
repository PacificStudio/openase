---
name: "pull"
description: "Sync with the latest remote branch before coding or pushing, keeping history linear."
---

# Safe Pull

开始工作前先同步远端主干：

1. 确认当前分支和工作区状态
2. 获取远端最新提交
3. 使用快进或 rebase 方式同步
4. 如果有冲突，先解决并重新验证受影响范围

不要在不理解影响的情况下覆盖本地文件。
