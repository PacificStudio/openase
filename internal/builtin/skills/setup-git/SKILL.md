---
name: "setup-git"
description: "Install or repair git plus the minimum identity and credential configuration needed for agent work."
---

# Setup Git

目标：让目标机器具备可用的 `git`，并补齐最小身份配置。

执行时遵循：

- 检查 `git --version` 是否可用；如果不可用，先安装 git。
- 检查 `git config --global user.name` 和 `git config --global user.email`；缺失时按工单上下文补齐。
- 仅在确有凭据问题时修复 git 认证，避免覆盖已有可用配置。
- 最后用非破坏性命令确认 git 基础能力可用，并记录生效配置。
