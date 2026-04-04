---
name: "install-codex"
description: "Install the Codex CLI on a target machine and verify it can start successfully."
---

# Install Codex CLI

Goal: make the target machine provide a working `codex` command and verify that the CLI can start successfully.

Follow this process:

- Check whether `codex` already exists and what version is installed.
- Install or upgrade the Codex CLI using an officially supported method.
- Verify `codex --version` afterward, and add a minimal auth check if needed.
- If network, Python, Node, or system dependencies block installation, record the exact blocker and avoid leaving a half-installed state.
