# Skills

## What Is This?

Skills are **reusable automation behavior packs**. If [Workflows](./workflows.md) are "job descriptions", skills are "professional certifications". They contain scripts, code snippets, or operational standards that can be bound to multiple workflows, allowing different workflows to share the same capabilities.

## Key Concepts

| Concept | Description |
|---------|-------------|
| **Built-in Skills** | Platform-provided skills, such as Git operations, package management, etc. |
| **Custom Skills** | Skills created by users for their project |
| **Skill Bundle** | The collection of all files within a skill |
| **Skill Binding** | Attaching a skill to a workflow so agents can invoke it |

## Skill Bundle Structure

A skill consists of the following file types:

| File Type | Description |
|-----------|-------------|
| `entrypoint` | Entry file, the starting point when an agent executes |
| `metadata` | Metadata description file |
| `script` | Executable scripts |
| `reference` | Reference documentation |
| `asset` | Static resources |

## Common Operations

### Browsing Skills

Go to the Skills page to view all available skills. Supports filtering:

- **All**: Every skill
- **Built-in**: Platform-provided skills
- **Custom**: Project-created skills
- **Disabled**: Temporarily deactivated skills

### Creating Custom Skills

1. Click "New Skill"
2. Fill in name and description
3. Write skill content (Markdown template)
4. Save

### Managing Skills

- **Enable / Disable**: Temporarily turn off a skill without deleting it
- **Edit**: Modify skill content and metadata
- **View Bindings**: See which workflows are using this skill
- **Delete**: Remove custom skills no longer needed (built-in skills cannot be deleted)

## Use Cases

| Scenario | Skill Example |
|----------|---------------|
| **Git Operations** | Standard process for creating branches, committing code, opening PRs |
| **Harness Audits** | `auto-harness` diagnoses repo drift and plans stronger contract and CI guardrails |
| **Code Quality Checks** | Lint/format scripts |
| **Test Execution** | Standardized test running and report generation |
| **Deployment** | Automated deployment steps |
| **Documentation** | Auto-generate API docs or Changelog |

## Tips

- Prefer built-in skills — they are thoroughly tested
- Custom skills are ideal for encapsulating project-specific processes and standards
- A single skill can be bound to multiple workflows; changes apply to all bound workflows
- When writing skills, pay attention to entrypoint file executable permissions
