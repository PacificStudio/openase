---
name: "security-scan"
description: "Review trust boundaries, auth, input handling, secrets, and dependency risk before release."
---

# Security Scan

## Overview

Conduct a security-focused review of the code or change set. Map trust boundaries first, then inspect how untrusted input, credentials, permissions, dependencies, and deployment defaults are handled.

## When To Use

- After adding or changing authentication or authorization logic.
- After adding new API endpoints, uploads, background jobs, or external integrations.
- Before production deployment.
- When the change handles untrusted input, secrets, tokens, files, or network calls.
- When the user explicitly requests a security review or audit.

## Review Workflow

1. Map the attack surface.
   - Entry points, user-controlled input, file access, network access, storage boundaries, and privileged operations.

2. Review authentication and authorization.
   - Authentication bypasses.
   - Missing authorization checks.
   - Confused trust boundaries between user, service, and admin actions.

3. Review input handling and injection surfaces.
   - SQL, NoSQL, shell, template, and path injection.
   - XSS and unsafe HTML rendering.
   - SSRF, open redirects, and unsafe outbound requests.

4. Review secrets and cryptography.
   - Hardcoded keys, tokens, passwords, or connection strings.
   - Weak password hashing or unsafe token generation.
   - Sensitive data exposure in logs, errors, or client payloads.

5. Review dependency and configuration risk.
   - Known vulnerable dependencies.
   - Overly permissive defaults.
   - Missing secure headers, unsafe CORS, or weak production settings.

6. Review detection and recovery.
   - Are important security-relevant failures logged?
   - Would an operator be able to detect abuse or investigate an incident?

7. Report severity-rated findings with remediation guidance.
   - `CRITICAL`: exploitable issue or active secret exposure; block deployment.
   - `HIGH`: serious vulnerability or major trust-boundary weakness; fix before release.
   - `MEDIUM`: meaningful hardening gap or partial control failure.
   - `LOW`: defense-in-depth improvement or cleanup.

## Security Checklist

- Authentication and authorization boundaries are explicit and enforced.
- Untrusted input is validated, normalized, and escaped at the correct boundary.
- Queries and commands are parameterized and never built from raw user input.
- Secrets are not stored in source, test fixtures, logs, or client-visible responses.
- File and network access respect allowlists and path or host validation where appropriate.
- Dependency versions and deployment defaults do not introduce known high-risk vulnerabilities.

## Operating Rules

- Think in trust boundaries and attacker-controlled inputs, not only code style.
- Prefer reproducible exploit paths over vague warnings.
- Name the impacted asset or permission boundary for every serious finding.
- Include a practical remediation path, not just the vulnerability label.
- If no critical issues are found, still call out hardening gaps and any unreviewed surfaces.

## Default Deliverable Shape

Return these sections:

1. `Attack Surface`
2. `Findings` - ordered by severity with file and line references where possible.
3. `Exploit Path / Impact` - why each important issue matters.
4. `Remediation` - concrete fixes and safer patterns.
5. `Residual Risks` - remaining uncertainty or unreviewed areas.
