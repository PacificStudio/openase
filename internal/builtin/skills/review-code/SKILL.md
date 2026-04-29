---
name: "review-code"
description: "Review behavior, risk, performance, and test coverage before style nits."
---

# Review Code

## Overview

Conduct a thorough code review for quality, security, performance, and maintainability. Focus on the real risks first, then provide concrete, severity-rated feedback with file and line references where possible.

## When To Use

- Before merging a pull request.
- After implementing a feature or refactor.
- When the user asks for a review of changed files or a branch.
- When the code looks correct at first glance but may still hide regressions or design debt.

## Review Workflow

1. Identify the review scope.
   - Review the current diff, named files, or the whole change set.
   - Prefer reviewing the actual changed surface before expanding further.

2. Check behavioral correctness first.
   - Regressions against existing behavior.
   - Missing edge-case handling.
   - Broken invariants or data consistency risks.

3. Review the main quality categories.
   - Security risks.
   - Code quality and complexity.
   - Performance concerns.
   - Error handling and observability.
   - Maintainability, coupling, and testability.

4. Evaluate the test story.
   - Are the important paths covered?
   - Are the tests proving behavior rather than implementation trivia?
   - Is there an obvious missing regression test for the change?

5. Rate findings by severity.
   - `CRITICAL`: must fix before merge; security or correctness issue with serious impact.
   - `HIGH`: should fix before merge; likely bug, major regression risk, or serious design flaw.
   - `MEDIUM`: worthwhile fix; non-blocking but meaningful quality issue.
   - `LOW`: suggestion, cleanup, or style improvement.

6. Report findings in priority order.
   - File and line reference.
   - What is wrong.
   - Why it matters.
   - Concrete fix guidance.

## Review Categories

- Security: secrets, unsafe input handling, authz gaps, injection, XSS, CSRF.
- Code quality: complexity, duplication, large functions, fragile branching.
- Performance: avoidable repeated work, inefficient algorithms, N+1 patterns, unnecessary re-renders.
- Maintainability: unclear naming, tight coupling, hidden assumptions, hard-to-test logic.
- Reliability: missing error handling, weak retries, silent failures, incomplete logging.

## Operating Rules

- Lead with findings, not praise.
- Prioritize behavior and risk before style nits.
- Prefer evidence and concrete examples over general taste.
- If there are no findings, say so explicitly and still mention residual risks or test gaps.
- Keep recommendations actionable enough that another engineer can implement them directly.

## Default Deliverable Shape

Return these sections:

1. `Findings` - ordered by severity, each with file and line references.
2. `Open Questions / Assumptions` - only if they affect confidence.
3. `Residual Risks` - what was not fully proven by the review.
4. `Short Summary` - brief overall assessment after the findings.
