---
name: "setup-git"
description: "Install or repair git plus the minimum identity and credential configuration needed for agent work."
---

# Setup Git

Goal: make the target machine provide a working `git` and fill in the minimum identity configuration.

Follow this process:

- Check whether `git --version` works. If not, install git first.
- Check `git config --global user.name` and `git config --global user.email`; fill them in from ticket context when missing.
- Repair Git authentication only when there is a real credential problem, and avoid overwriting an already working setup.
- Finish with non-destructive commands that confirm the basic Git flow works, and record the effective configuration.
