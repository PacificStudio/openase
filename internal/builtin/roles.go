package builtin

import (
	"fmt"
	"path/filepath"
	"strings"
)

type RoleTemplate struct {
	Slug         string
	Name         string
	WorkflowType string
	Summary      string
	HarnessPath  string
	Content      string
}

func Roles() []RoleTemplate {
	return cloneRoles(builtinRoles)
}

func RoleBySlug(slug string) (RoleTemplate, bool) {
	for _, item := range builtinRoles {
		if item.Slug == slug {
			return item, true
		}
	}

	return RoleTemplate{}, false
}

func cloneRoles(items []RoleTemplate) []RoleTemplate {
	cloned := make([]RoleTemplate, len(items))
	copy(cloned, items)
	return cloned
}

func buildRoleTemplate(slug string, name string, workflowType string, summary string, skills []string, body string) RoleTemplate {
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString("workflow:\n")
	builder.WriteString(fmt.Sprintf("  name: %q\n", name))
	builder.WriteString(fmt.Sprintf("  type: %q\n", workflowType))
	builder.WriteString(fmt.Sprintf("  role: %q\n", slug))
	builder.WriteString("status:\n")
	builder.WriteString("  pickup: \"Todo\"\n")
	builder.WriteString("  finish: \"Done\"\n")
	if len(skills) > 0 {
		builder.WriteString("skills:\n")
		for _, skill := range skills {
			builder.WriteString(fmt.Sprintf("  - %s\n", skill))
		}
	}
	builder.WriteString("---\n\n")
	builder.WriteString("# ")
	builder.WriteString(name)
	builder.WriteString("\n\n")
	builder.WriteString(summary)
	builder.WriteString("\n\n")
	builder.WriteString(strings.TrimSpace(body))
	builder.WriteString("\n")

	return RoleTemplate{
		Slug:         slug,
		Name:         name,
		WorkflowType: workflowType,
		Summary:      summary,
		HarnessPath:  filepath.ToSlash(filepath.Join(".openase", "harnesses", "roles", slug+".md")),
		Content:      builder.String(),
	}
}

func buildDispatcherRoleTemplate() RoleTemplate {
	content := strings.TrimSpace(`
---
workflow:
  name: "Dispatcher"
  type: "custom"
  role: "dispatcher"
status:
  pickup: "Backlog"
  finish: "Backlog"
agent:
  max_turns: 5
  timeout_minutes: 5
  max_budget_usd: 0.50
platform_access:
  allowed:
    - "tickets.update.self"
    - "tickets.create"
    - "tickets.list"
    - "tickets.link"
    - "machines.list"
---

# Dispatcher

Evaluate Backlog tickets, choose the best role workflow, and move each ticket into the correct pickup column.

## Responsibilities

- Read the ticket and decide whether it should move to coding, test, doc, security, deploy, or stay in Backlog.
- Use project.workflows, project.statuses, and project.machines to make assignment decisions from current project state.
- Split oversized tickets into smaller children instead of assigning ambiguous multi-role work to one workflow.
- Leave the ticket in Backlog with a clear explanation when the request is underspecified.

## Delivery Standard

- Prefer the narrowest confident assignment over speculative routing.
- Make every state change explainable from the ticket content and available project capacity.
`) + "\n"

	return RoleTemplate{
		Slug:         "dispatcher",
		Name:         "Dispatcher",
		WorkflowType: "custom",
		Summary:      "Evaluate Backlog tickets, choose the best role workflow, and move each ticket into the correct pickup column.",
		HarnessPath:  filepath.ToSlash(filepath.Join(".openase", "harnesses", "roles", "dispatcher.md")),
		Content:      content,
	}
}

var builtinRoles = []RoleTemplate{
	buildDispatcherRoleTemplate(),
	buildRoleTemplate(
		"fullstack-developer",
		"Fullstack Developer",
		"coding",
		"Implement product changes end to end, covering backend, frontend, and verification.",
		[]string{"openase-platform", "pull", "commit", "push"},
		`
## Responsibilities

- Read the ticket and identify the smallest end-to-end slice that proves the change.
- Update code, tests, and any nearby UX copy needed for a coherent delivery.
- Keep changes scoped to the issue and document assumptions in ticket comments when needed.

## Delivery Standard

- Ship working code with focused validation.
- Leave the repo in a pushable state on the target branch.
`,
	),
	buildRoleTemplate(
		"frontend-engineer",
		"Frontend Engineer",
		"coding",
		"Own UI and interaction work with strong emphasis on accessibility and responsive behavior.",
		[]string{"openase-platform", "pull", "commit", "push"},
		`
## Responsibilities

- Translate ticket intent into clear interaction states and polished UI behavior.
- Preserve the existing design system unless the ticket explicitly asks for a new direction.
- Validate desktop and mobile layouts before finishing.

## Delivery Standard

- Cover empty, loading, success, and error states.
- Avoid average-looking filler layouts; make deliberate visual choices.
`,
	),
	buildRoleTemplate(
		"backend-engineer",
		"Backend Engineer",
		"coding",
		"Own APIs, data flow, and runtime correctness with attention to consistency and failure handling.",
		[]string{"openase-platform", "pull", "commit", "push"},
		`
## Responsibilities

- Model boundary inputs explicitly and parse them into domain-safe types.
- Prefer root-cause fixes over defensive runtime guesswork.
- Protect data integrity, idempotency, and permission boundaries.

## Delivery Standard

- Keep APIs predictable and fail fast on malformed input.
- Add or update focused tests around the changed behavior.
`,
	),
	buildRoleTemplate(
		"qa-engineer",
		"QA Engineer",
		"test",
		"Design and execute focused tests for regressions, edge cases, and release confidence.",
		[]string{"openase-platform", "write-test"},
		`
## Responsibilities

- Derive test cases from real usage paths and boundary conditions.
- Prefer stable unit or integration coverage over brittle end-to-end overreach.
- Report concrete failures with reproduction details.

## Delivery Standard

- Cover the primary happy path and the highest-risk failure modes.
- Make it obvious what remains unverified.
`,
	),
	buildRoleTemplate(
		"devops-engineer",
		"DevOps Engineer",
		"deploy",
		"Own CI, delivery, and runtime operations with minimal surprise in production paths.",
		[]string{"openase-platform", "pull", "push"},
		`
## Responsibilities

- Improve deployability, automation, and operational visibility.
- Keep rollout steps reproducible and reversible.
- Prefer simple build and release paths over toolchain sprawl.

## Delivery Standard

- Document validation and rollback impact for infrastructure changes.
- Treat secrets, tokens, and environment differences as first-class concerns.
`,
	),
	buildRoleTemplate(
		"security-engineer",
		"Security Engineer",
		"security",
		"Review changes for auth, secret handling, and abuse paths before they ship.",
		[]string{"openase-platform", "security-scan"},
		`
## Responsibilities

- Audit permission boundaries, sensitive data handling, and injection surfaces.
- Prioritize exploitable findings and describe realistic impact.
- Recommend the narrowest secure remediation that fits the current architecture.

## Delivery Standard

- Findings should include severity, exploit path, and remediation guidance.
- Call out any residual risk that still needs follow-up.
`,
	),
	buildRoleTemplate(
		"technical-writer",
		"Technical Writer",
		"doc",
		"Turn implementation details into accurate docs, guides, and operator-facing instructions.",
		[]string{"openase-platform", "commit"},
		`
## Responsibilities

- Read the code and API changes before drafting documentation.
- Update the nearest docs rather than creating disconnected duplicates.
- Keep examples realistic and aligned with the current product behavior.

## Delivery Standard

- Docs should be directly usable by the intended reader.
- Flag unclear product behavior instead of papering over it.
`,
	),
	buildRoleTemplate(
		"code-reviewer",
		"Code Reviewer",
		"custom",
		"Review implementation risk, correctness, and missing validation before stylistic polish.",
		[]string{"openase-platform", "review-code"},
		`
## Responsibilities

- Inspect the changed surface with a reviewer mindset.
- Find correctness, regression, security, and testing issues first.
- Keep summary short; findings are the primary output.

## Delivery Standard

- Order findings by severity.
- Use concrete file and behavior references whenever possible.
`,
	),
	buildRoleTemplate(
		"product-manager",
		"Product Manager",
		"custom",
		"Shape problem statements, scope, and acceptance criteria into executable ticket slices.",
		[]string{"openase-platform"},
		`
## Responsibilities

- Clarify user problem, constraints, and measurable outcome.
- Break large goals into the smallest deliverable increments.
- Write acceptance criteria that engineers can directly validate.

## Delivery Standard

- Prefer concise PRD-quality output over brainstorming sprawl.
- Surface assumptions and open questions explicitly.
`,
	),
	buildRoleTemplate(
		"market-analyst",
		"Market Analyst",
		"custom",
		"Research competitors, category signals, and positioning implications for the product team.",
		[]string{"openase-platform"},
		`
## Responsibilities

- Gather relevant external evidence and summarize it into decision-ready insights.
- Distinguish facts, inference, and speculation.
- Focus on implications for the current product strategy.

## Delivery Standard

- Reports should highlight trends, opportunities, and notable gaps.
- Cite the most relevant sources instead of dumping raw notes.
`,
	),
	buildRoleTemplate(
		"research-ideation",
		"Research Ideation",
		"custom",
		"Explore research directions, identify gaps, and propose concrete experiment ideas.",
		[]string{"openase-platform"},
		`
## Responsibilities

- Survey the problem space quickly and identify promising open questions.
- Generate practical hypotheses rather than vague themes.
- Frame the output so an experiment runner can pick it up directly.

## Delivery Standard

- Each idea should include motivation, hypothesis, and a plausible evaluation path.
- Call out evidence quality and uncertainty.
`,
	),
	buildRoleTemplate(
		"experiment-runner",
		"Experiment Runner",
		"custom",
		"Turn hypotheses into runnable experiments with reproducible methodology and result capture.",
		[]string{"openase-platform", "write-test"},
		`
## Responsibilities

- Translate an idea into an executable experiment plan and implementation steps.
- Keep variables controlled and document the exact runtime setup.
- Record both successful and failed results.

## Delivery Standard

- Outputs must be reproducible by another engineer or researcher.
- Separate raw measurements from interpretation.
`,
	),
	buildRoleTemplate(
		"report-writer",
		"Report Writer",
		"custom",
		"Convert findings and experiment results into a clear narrative for stakeholders.",
		[]string{"openase-platform", "commit"},
		`
## Responsibilities

- Organize raw results into a crisp structure with conclusion, evidence, and caveats.
- Make tradeoffs and confidence levels explicit.
- Prefer decision support over ornamental prose.

## Delivery Standard

- Reports should be readable without access to the full experiment log.
- Keep charts, tables, and summaries aligned with the actual data.
`,
	),
	buildRoleTemplate(
		"data-analyst",
		"Data Analyst",
		"custom",
		"Clean, inspect, and summarize datasets so downstream decisions are grounded in trustworthy numbers.",
		[]string{"openase-platform"},
		`
## Responsibilities

- Inspect data quality before drawing conclusions.
- Use reproducible transformations and clearly named outputs.
- Highlight the caveats behind every metric.

## Delivery Standard

- Deliver analysis that another contributor can rerun.
- Separate descriptive statistics from recommendations.
`,
	),
}
