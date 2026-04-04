package httpapi

import (
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

type projectRepoPatchRequest struct {
	Name             *string   `json:"name"`
	RepositoryURL    *string   `json:"repository_url"`
	DefaultBranch    *string   `json:"default_branch"`
	WorkspaceDirname *string   `json:"workspace_dirname"`
	Labels           *[]string `json:"labels"`
}

type ticketRepoScopePatchRequest struct {
	BranchName     *string `json:"branch_name"`
	PullRequestURL *string `json:"pull_request_url"`
}

func parseProjectRepoPatchRequest(
	projectID uuid.UUID,
	repoID uuid.UUID,
	current domain.ProjectRepo,
	patch projectRepoPatchRequest,
) (domain.UpdateProjectRepo, error) {
	request := domain.ProjectRepoInput{
		Name:             current.Name,
		RepositoryURL:    current.RepositoryURL,
		DefaultBranch:    current.DefaultBranch,
		WorkspaceDirname: stringPointer(current.WorkspaceDirname),
		Labels:           append([]string(nil), current.Labels...),
	}
	if patch.Name != nil {
		request.Name = *patch.Name
	}
	if patch.RepositoryURL != nil {
		request.RepositoryURL = *patch.RepositoryURL
	}
	if patch.DefaultBranch != nil {
		request.DefaultBranch = *patch.DefaultBranch
	}
	if patch.WorkspaceDirname != nil {
		request.WorkspaceDirname = patch.WorkspaceDirname
	}
	if patch.Labels != nil {
		request.Labels = *patch.Labels
	}

	return domain.ParseUpdateProjectRepo(repoID, projectID, request)
}

func parseTicketRepoScopePatchRequest(
	scopeID uuid.UUID,
	projectID uuid.UUID,
	ticketID uuid.UUID,
	current domain.TicketRepoScope,
	patch ticketRepoScopePatchRequest,
) (domain.UpdateTicketRepoScope, error) {
	request := domain.TicketRepoScopeInput{
		RepoID:         current.RepoID.String(),
		BranchName:     patch.BranchName,
		PullRequestURL: patch.PullRequestURL,
	}

	return domain.ParseUpdateTicketRepoScope(scopeID, projectID, ticketID, request)
}
