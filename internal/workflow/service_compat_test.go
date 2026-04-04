package workflow

import (
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entticketdependency "github.com/BetterAndBetterII/openase/ent/ticketdependency"
	ticketingdomain "github.com/BetterAndBetterII/openase/internal/domain/ticketing"
	"github.com/google/uuid"
)

func mapHarnessScopedRepos(ticketIdentifier string, scopes []*ent.TicketRepoScope, workspace string) ([]HarnessRepoData, map[uuid.UUID]string) {
	repos := make([]HarnessRepoData, 0, len(scopes))
	branches := make(map[uuid.UUID]string, len(scopes))
	for _, scope := range scopes {
		repo := scope.Edges.Repo
		if repo == nil {
			continue
		}
		effectiveBranchName := ticketingdomain.ResolveRepoWorkBranch(ticketIdentifier, scope.BranchName)
		branches[repo.ID] = effectiveBranchName
		repos = append(repos, HarnessRepoData{
			Name:          repo.Name,
			URL:           repo.RepositoryURL,
			Path:          resolveRepoPath(repo.WorkspaceDirname, workspace, repo.Name),
			Branch:        effectiveBranchName,
			DefaultBranch: repo.DefaultBranch,
			Labels:        append([]string(nil), repo.Labels...),
		})
	}
	return repos, branches
}

func mapHarnessAllRepos(ticketIdentifier string, repos []*ent.ProjectRepo, repoBranchByID map[uuid.UUID]string, workspace string) []HarnessRepoData {
	items := make([]HarnessRepoData, 0, len(repos))
	for _, repo := range repos {
		branch := repoBranchByID[repo.ID]
		if branch == "" {
			branch = ticketingdomain.DefaultRepoWorkBranch(ticketIdentifier)
		}
		items = append(items, HarnessRepoData{
			Name:          repo.Name,
			URL:           repo.RepositoryURL,
			Path:          resolveRepoPath(repo.WorkspaceDirname, workspace, repo.Name),
			Branch:        branch,
			DefaultBranch: repo.DefaultBranch,
			Labels:        append([]string(nil), repo.Labels...),
		})
	}
	return items
}

func joinStatusNames(statuses []*ent.TicketStatus) string {
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, status.Name)
	}
	return strings.Join(names, ", ")
}

func mapHarnessAgent(item *ent.Agent) HarnessAgentData {
	if item == nil {
		return HarnessAgentData{}
	}
	providerName := ""
	adapterType := ""
	modelName := ""
	if item.Edges.Provider != nil {
		providerName = item.Edges.Provider.Name
		adapterType = item.Edges.Provider.AdapterType.String()
		modelName = item.Edges.Provider.ModelName
	}
	return HarnessAgentData{
		ID:                    item.ID.String(),
		Name:                  item.Name,
		Provider:              providerName,
		AdapterType:           adapterType,
		Model:                 modelName,
		TotalTicketsCompleted: item.TotalTicketsCompleted,
	}
}

func mapHarnessTicketLinks(links []*ent.TicketExternalLink) []HarnessTicketLinkData {
	items := make([]HarnessTicketLinkData, 0, len(links))
	for _, link := range links {
		items = append(items, HarnessTicketLinkData{
			Type:     link.LinkType.String(),
			URL:      link.URL,
			Title:    link.Title,
			Status:   link.Status,
			Relation: link.Relation.String(),
		})
	}
	return items
}

func mapHarnessDependencies(dependencies []*ent.TicketDependency) []HarnessTicketDependencyData {
	items := make([]HarnessTicketDependencyData, 0, len(dependencies))
	for _, dependency := range dependencies {
		target := dependency.Edges.TargetTicket
		if target == nil {
			continue
		}
		items = append(items, HarnessTicketDependencyData{
			Identifier: target.Identifier,
			Title:      target.Title,
			Type:       normalizeDependencyType(dependency.Type),
			Status:     edgeTicketStatusName(target.Edges.Status),
		})
	}
	return items
}

func edgeTicketStatusName(status *ent.TicketStatus) string {
	if status == nil {
		return ""
	}
	return status.Name
}

func parentIdentifier(ticketItem *ent.Ticket) string {
	if ticketItem == nil || ticketItem.Edges.Parent == nil {
		return ""
	}
	return ticketItem.Edges.Parent.Identifier
}

func normalizeDependencyType(value entticketdependency.Type) string {
	return strings.ReplaceAll(value.String(), "-", "_")
}

func mapHarnessProjectWorkflowTicket(item *ent.Ticket) HarnessProjectWorkflowTicketData {
	return HarnessProjectWorkflowTicketData{
		Identifier:        item.Identifier,
		Title:             item.Title,
		Status:            edgeTicketStatusName(item.Edges.Status),
		Priority:          item.Priority.String(),
		Type:              item.Type.String(),
		AttemptCount:      normalizeAttemptCount(item.AttemptCount),
		ConsecutiveErrors: item.ConsecutiveErrors,
		RetryPaused:       item.RetryPaused,
		PauseReason:       item.PauseReason,
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
		StartedAt:         formatOptionalTime(item.StartedAt),
		CompletedAt:       formatOptionalTime(item.CompletedAt),
	}
}

func statusNamesFromEdges(statuses []*ent.TicketStatus) []string {
	names := make([]string, 0, len(statuses))
	for _, status := range statuses {
		names = append(names, status.Name)
	}
	return names
}

func rollback(tx *ent.Tx) {
	if tx == nil {
		return
	}
	_ = tx.Rollback()
}
