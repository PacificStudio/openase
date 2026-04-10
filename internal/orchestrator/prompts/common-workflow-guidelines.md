## Shared Workflow Execution Rules

Apply these rules to every workflow execution in addition to the workflow-specific harness.

### Scope and Boundaries

- Work only on the current ticket's direct scope.
- Do not expand the task into unrelated cleanup or broad refactors unless the ticket explicitly requires it.
- Do not guess missing requirements, platform state, or undocumented interfaces.
- If critical context is missing or contradictory, record the blocker and stop expanding execution.

### Standard Execution Flow

1. Understand the ticket goal, constraints, dependencies, and current status.
2. Read the minimum necessary project and repository context before changing anything.
3. Confirm the real root cause or implementation target before editing code or configuration.
4. Make the smallest change that fully resolves the scoped task.
5. Run the most relevant validation for the work you changed.
6. Record meaningful progress, blockers, and outcomes back to OpenASE.
7. Move the ticket toward the correct workflow finish state when the work is actually complete.

### Validation Expectations

- Prefer real validation over assumption.
- Use existing tests when they cover the changed behavior.
- If no relevant automated test exists, perform the smallest credible manual or command-based verification and state what was checked.
- Do not distort the implementation just to satisfy a test harness.

### Platform Writeback

- Treat OpenASE as the control plane for ticket status, progress, and execution traceability.
- Use the runtime-provided OpenASE tools and contracts for platform reads and writes.
- Record blockers clearly with cause, impact, and the next action needed.
- When the task completes, ensure the final result, validation, and remaining risk are reflected in the platform output.

### Failure and Blocker Handling

- Stop and surface the issue when requirements are unclear, permissions are insufficient, dependencies are unavailable, or the environment is broken.
- Explain what was attempted, what remains blocked, and what additional input or change is required.
- Avoid speculative retries that do not produce new information.

### Output Contract

Final execution output should make clear:

- what changed or was delivered
- what validation ran and what the result was
- any residual risk or follow-up
- whether the ticket is ready for the workflow's finish state
