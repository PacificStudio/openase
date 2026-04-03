---
name: "setup-gh-cli"
description: "Install or repair the GitHub CLI and confirm authentication status on the target machine."
---

# Setup GitHub CLI

目标：让目标机器具备可用的 `gh`，并确认 GitHub 认证状态。

执行时遵循：

- 检查 `gh --version` 和 `gh auth status` 的当前输出。
- 如果 `gh` 缺失，使用官方支持的安装方式安装。
- 如果 `gh` 已安装但未认证，补齐认证并再次验证状态。
- 认证失败时记录准确原因，例如网络、令牌缺失或主机不可达。

不要把明文令牌写入 shell 历史或仓库文件。
