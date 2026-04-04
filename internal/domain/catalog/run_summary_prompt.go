package catalog

import "strings"

type AgentRunSummaryPromptSource string

const (
	AgentRunSummaryPromptSourceBuiltin         AgentRunSummaryPromptSource = "builtin"
	AgentRunSummaryPromptSourceProjectOverride AgentRunSummaryPromptSource = "project_override"
)

const DefaultAgentRunSummaryPrompt = `
Summarize the overall work performed by the agent.
List the major steps in execution order.
Call out operations that took unusually long.
Call out repeated trial-and-error, retries, or churn.
Call out commands or file changes with elevated security or safety risk.
Mention the main outcome and any important unresolved items.
Use concise Markdown with exactly these top-level sections:
## Overview
## Major Steps
## Long-Running Operations
## Repeated Trial-and-Error
## Security / Safety Risks
## Files Touched
## Outcome
If a section has no substantive content, write "None."
`

func (s AgentRunSummaryPromptSource) String() string {
	return string(s)
}

func EffectiveAgentRunSummaryPrompt(rawOverride string) (string, AgentRunSummaryPromptSource) {
	trimmedOverride := strings.TrimSpace(rawOverride)
	if trimmedOverride != "" {
		return trimmedOverride, AgentRunSummaryPromptSourceProjectOverride
	}
	return strings.TrimSpace(DefaultAgentRunSummaryPrompt), AgentRunSummaryPromptSourceBuiltin
}
