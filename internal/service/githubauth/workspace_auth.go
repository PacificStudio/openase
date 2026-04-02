package githubauth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	githubauthdomain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	gittransport "github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/google/uuid"
)

func ApplyWorkspaceAuth(
	ctx context.Context,
	resolver TokenResolver,
	projectID uuid.UUID,
	request workspaceinfra.SetupRequest,
) (workspaceinfra.SetupRequest, error) {
	if resolver == nil || projectID == uuid.Nil || len(request.Repos) == 0 {
		return request, nil
	}

	resolved, err := resolver.ResolveProjectCredential(ctx, projectID)
	if err != nil {
		if errors.Is(err, ErrCredentialNotConfigured) {
			return request, nil
		}
		return workspaceinfra.SetupRequest{}, fmt.Errorf("resolve project GitHub credential for workspace preparation: %w", err)
	}

	token := strings.TrimSpace(resolved.Token)
	if token == "" {
		return request, nil
	}

	updated := request
	updated.Repos = append([]workspaceinfra.RepoRequest(nil), request.Repos...)
	for index := range updated.Repos {
		repositoryURL := strings.TrimSpace(updated.Repos[index].RepositoryURL)
		if repositoryURL == "" {
			continue
		}
		endpoint, err := gittransport.NewEndpoint(repositoryURL)
		if err != nil {
			return workspaceinfra.SetupRequest{}, fmt.Errorf("parse repository URL %q: %w", repositoryURL, err)
		}
		if endpoint.Protocol != "https" && endpoint.Protocol != "http" {
			continue
		}
		if _, ok := githubauthdomain.ParseGitHubRepositoryURL(repositoryURL); !ok {
			continue
		}
		updated.Repos[index].HTTPBasicAuth = &workspaceinfra.HTTPBasicAuthRequest{
			Username: "x-access-token",
			Password: token,
		}
	}

	return updated, nil
}
