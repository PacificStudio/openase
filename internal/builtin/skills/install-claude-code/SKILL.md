---
name: "install-claude-code"
description: "Install Claude Code on a target machine and verify the CLI is available for remote execution."
---

# Install Claude Code

Goal: make the target machine provide a working `claude` command and record the installation result.

Follow this process:

- Check the OS, package manager, and whether `claude` is already installed.
- Use an officially supported installation method. Do not download unknown binaries.
- After installation, verify `claude --version` and record the executable path.
- If login or extra authentication is still required, record the exact state and missing prerequisites.

Do not write tokens or credentials into the repository.
