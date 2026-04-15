package orchestrator

import (
	"embed"
	"fmt"
	"strings"

	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
)

//go:embed prompts/common-workflow-guidelines.md
var workflowPromptAssets embed.FS

var sharedWorkflowExecutionRules = mustReadWorkflowPromptAsset("prompts/common-workflow-guidelines.md")

func composeWorkflowDeveloperInstructions(
	renderedHarness string,
	ticketContext string,
	platformContract string,
) string {
	sections := []string{
		strings.TrimSpace(renderedHarness),
		strings.TrimSpace(ticketContext),
		sharedWorkflowExecutionRules,
		strings.TrimSpace(platformContract),
	}
	nonEmpty := make([]string, 0, len(sections))
	for _, section := range sections {
		if section == "" {
			continue
		}
		nonEmpty = append(nonEmpty, section)
	}
	return strings.Join(nonEmpty, "\n\n")
}

func composeWorkflowTicketContext(data workflowservice.HarnessTemplateData) string {
	var sb strings.Builder
	sb.WriteString("## Ticket Execution Context\n")
	_, _ = fmt.Fprintf(
		&sb,
		"Ticket: %s - %s\n",
		strings.TrimSpace(data.Ticket.Identifier),
		strings.TrimSpace(data.Ticket.Title),
	)
	_, _ = fmt.Fprintf(
		&sb,
		"Status: %s | Priority: %s | Type: %s | Attempts: %d/%d\n",
		strings.TrimSpace(data.Ticket.Status),
		strings.TrimSpace(data.Ticket.Priority),
		strings.TrimSpace(data.Ticket.Type),
		data.Ticket.AttemptCount,
		data.Ticket.MaxAttempts,
	)
	if strings.TrimSpace(data.Workspace) != "" {
		_, _ = fmt.Fprintf(&sb, "Workspace: %s\n", strings.TrimSpace(data.Workspace))
	}
	if strings.TrimSpace(data.Ticket.ParentIdentifier) != "" {
		_, _ = fmt.Fprintf(&sb, "Parent Ticket: %s\n", strings.TrimSpace(data.Ticket.ParentIdentifier))
	}
	if strings.TrimSpace(data.Ticket.Description) != "" {
		sb.WriteString("\n### Ticket Description\n")
		sb.WriteString(strings.TrimSpace(data.Ticket.Description))
		sb.WriteString("\n")
	}
	if len(data.Ticket.Dependencies) > 0 {
		sb.WriteString("\n### Dependencies\n")
		for _, dependency := range data.Ticket.Dependencies {
			_, _ = fmt.Fprintf(
				&sb,
				"- [%s] %s (%s, status=%s)\n",
				strings.TrimSpace(dependency.Identifier),
				strings.TrimSpace(dependency.Title),
				strings.TrimSpace(dependency.Type),
				strings.TrimSpace(dependency.Status),
			)
		}
	}
	if len(data.Ticket.Links) > 0 {
		sb.WriteString("\n### External Links\n")
		for _, link := range data.Ticket.Links {
			_, _ = fmt.Fprintf(
				&sb,
				"- %s | %s | status=%s | url=%s\n",
				strings.TrimSpace(link.Type),
				strings.TrimSpace(link.Title),
				strings.TrimSpace(link.Status),
				strings.TrimSpace(link.URL),
			)
		}
	}
	if len(data.Repos) > 0 {
		sb.WriteString("\n### Scoped Repositories\n")
		for _, repo := range data.Repos {
			_, _ = fmt.Fprintf(
				&sb,
				"- %s path=%s branch=%s labels=%s\n",
				strings.TrimSpace(repo.Name),
				strings.TrimSpace(repo.Path),
				strings.TrimSpace(repo.Branch),
				strings.Join(repo.Labels, ", "),
			)
		}
	} else if len(data.AllRepos) > 0 {
		sb.WriteString("\n### Project Repositories\n")
		for _, repo := range data.AllRepos {
			_, _ = fmt.Fprintf(
				&sb,
				"- %s path=%s branch=%s labels=%s\n",
				strings.TrimSpace(repo.Name),
				strings.TrimSpace(repo.Path),
				strings.TrimSpace(repo.Branch),
				strings.Join(repo.Labels, ", "),
			)
		}
	}
	return strings.TrimSpace(sb.String())
}

func mustReadWorkflowPromptAsset(path string) string {
	data, err := workflowPromptAssets.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("read workflow prompt asset %q: %v", path, err))
	}
	return strings.TrimSpace(string(data))
}
