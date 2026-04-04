---
name: "install-claude-code"
description: "Install Claude Code on a target machine and verify the CLI is available for remote execution."
---

# Install Claude Code

目标：让目标机器具备可用的 `claude` 命令，并记录安装结果。

执行时遵循：

- 先确认当前系统类型、包管理器和是否已安装 `claude`。
- 使用官方支持的安装方式完成安装，避免下载来源不明的二进制。
- 安装后至少验证 `claude --version`，并记录可执行路径。
- 如果还需要登录或额外认证，明确记录当前状态和缺失前置条件。

不要把令牌或凭据写入仓库。
