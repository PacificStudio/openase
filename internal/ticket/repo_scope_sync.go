package ticket

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	"github.com/google/uuid"
)

type SyncRepoScopePRStatusInput struct {
	RepositoryURL      string
	RepositoryFullName string
	BranchName         string
	PullRequestURL     string
	PRStatus           ticketreposcope.PrStatus
}

func (s *Service) SyncRepoScopePRStatus(ctx context.Context, input SyncRepoScopePRStatusInput) (bool, error) {
	if s.client == nil {
		return false, ErrUnavailable
	}

	repositoryKey, err := normalizeGitHubRepositoryKey(input.RepositoryURL, input.RepositoryFullName)
	if err != nil {
		return false, err
	}

	branchName := strings.TrimSpace(input.BranchName)
	if branchName == "" {
		return false, fmt.Errorf("branch name must not be empty")
	}

	scopes, err := s.client.TicketRepoScope.Query().
		Where(ticketreposcope.BranchNameEQ(branchName)).
		WithRepo().
		All(ctx)
	if err != nil {
		return false, fmt.Errorf("query ticket repo scopes: %w", err)
	}

	var matchedScopeID uuid.UUID
	matched := false
	for _, scope := range scopes {
		scopeRepositoryKey, repoErr := normalizeGitHubRepositoryKey(scope.Edges.Repo.RepositoryURL, "")
		if repoErr != nil {
			return false, fmt.Errorf("normalize project repo %q: %w", scope.Edges.Repo.RepositoryURL, repoErr)
		}
		if scopeRepositoryKey != repositoryKey {
			continue
		}
		if matched {
			return false, fmt.Errorf("multiple ticket repo scopes matched repository %q and branch %q", repositoryKey, branchName)
		}
		matchedScopeID = scope.ID
		matched = true
	}

	if !matched {
		return false, nil
	}

	update := s.client.TicketRepoScope.UpdateOneID(matchedScopeID).
		SetPrStatus(input.PRStatus)
	if pullRequestURL := strings.TrimSpace(input.PullRequestURL); pullRequestURL != "" {
		update.SetPullRequestURL(pullRequestURL)
	}

	if _, err := update.Save(ctx); err != nil {
		return false, fmt.Errorf("update ticket repo scope: %w", err)
	}

	return true, nil
}

func normalizeGitHubRepositoryKey(rawURL string, rawFullName string) (string, error) {
	if fullName := strings.TrimSpace(rawFullName); fullName != "" {
		return normalizeOwnerRepoPath(fullName)
	}

	repositoryURL := strings.TrimSpace(rawURL)
	if repositoryURL == "" {
		return "", fmt.Errorf("repository URL must not be empty")
	}

	switch {
	case strings.HasPrefix(repositoryURL, "git@github.com:"):
		return normalizeOwnerRepoPath(strings.TrimPrefix(repositoryURL, "git@github.com:"))
	case strings.HasPrefix(repositoryURL, "ssh://git@github.com/"):
		return normalizeOwnerRepoPath(strings.TrimPrefix(repositoryURL, "ssh://git@github.com/"))
	case strings.HasPrefix(repositoryURL, "https://github.com/"), strings.HasPrefix(repositoryURL, "http://github.com/"):
		parsedURL, err := url.Parse(repositoryURL)
		if err != nil {
			return "", fmt.Errorf("parse repository URL %q: %w", repositoryURL, err)
		}
		return normalizeOwnerRepoPath(parsedURL.Path)
	default:
		return normalizeOwnerRepoPath(repositoryURL)
	}
}

func normalizeOwnerRepoPath(raw string) (string, error) {
	trimmed := strings.Trim(strings.TrimSpace(raw), "/")
	trimmed = strings.TrimSuffix(trimmed, ".git")

	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid GitHub repository reference %q", raw)
	}
	if strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", fmt.Errorf("invalid GitHub repository reference %q", raw)
	}

	return strings.ToLower(parts[0] + "/" + parts[1]), nil
}
