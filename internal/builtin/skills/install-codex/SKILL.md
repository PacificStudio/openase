---
name: "install-codex"
description: "Install the Codex CLI on a target machine and verify it can start successfully."
---

# Install Codex CLI

目标：让目标机器具备可用的 `codex` 命令，并验证 CLI 能正常启动。

执行时遵循：

- 先检查 `codex` 是否已存在以及当前版本。
- 使用官方支持的安装方式安装或升级 Codex CLI。
- 安装后验证 `codex --version`，必要时补充最小认证检查。
- 如果网络、Python、Node 或系统依赖阻塞安装，记录准确阻塞点，不要留下半安装状态。
