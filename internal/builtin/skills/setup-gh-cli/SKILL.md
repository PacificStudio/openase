---
name: "setup-gh-cli"
description: "Install or repair the GitHub CLI and confirm authentication status on the target machine."
---

# Setup GitHub CLI

Goal: make the target machine provide a working `gh` command and confirm GitHub authentication status.

Follow this process:

- Check the current output of `gh --version` and `gh auth status`.
- If `gh` is missing, install it using an officially supported method.
- If `gh` is installed but unauthenticated, complete auth and verify again.
- If authentication fails, record the exact reason, such as network issues, a missing token, or an unreachable host.

Do not write plaintext tokens into shell history or repository files.
