package githubauth

import (
	"context"
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/google/uuid"
)

func TestApplyWorkspaceAuthInjectsHTTPSCredentials(t *testing.T) {
	t.Parallel()

	request := workspaceinfra.SetupRequest{
		Repos: []workspaceinfra.RepoRequest{
			{RepositoryURL: "https://github.com/acme/private-repo.git"},
			{RepositoryURL: "git@github.com:acme/private-repo.git"},
			{RepositoryURL: "https://gitlab.com/acme/private-repo.git"},
		},
	}

	updated, err := ApplyWorkspaceAuth(
		context.Background(),
		stubWorkspaceAuthResolver{resolved: domain.ResolvedCredential{Token: "ghu_test"}},
		uuid.New(),
		request,
	)
	if err != nil {
		t.Fatalf("ApplyWorkspaceAuth() error = %v", err)
	}
	if updated.Repos[0].HTTPBasicAuth == nil {
		t.Fatal("expected GitHub HTTPS repo auth to be injected")
	}
	if updated.Repos[0].HTTPBasicAuth.Username != "x-access-token" || updated.Repos[0].HTTPBasicAuth.Password != "ghu_test" {
		t.Fatalf("unexpected injected auth %+v", updated.Repos[0].HTTPBasicAuth)
	}
	if updated.Repos[1].HTTPBasicAuth != nil {
		t.Fatalf("expected SSH repo to skip injected HTTP auth, got %+v", updated.Repos[1].HTTPBasicAuth)
	}
	if updated.Repos[2].HTTPBasicAuth != nil {
		t.Fatalf("expected non-GitHub repo to skip injected auth, got %+v", updated.Repos[2].HTTPBasicAuth)
	}
}

func TestApplyWorkspaceAuthIgnoresMissingCredential(t *testing.T) {
	t.Parallel()

	request := workspaceinfra.SetupRequest{
		Repos: []workspaceinfra.RepoRequest{{RepositoryURL: "https://github.com/acme/private-repo.git"}},
	}
	updated, err := ApplyWorkspaceAuth(
		context.Background(),
		stubWorkspaceAuthResolver{err: ErrCredentialNotConfigured},
		uuid.New(),
		request,
	)
	if err != nil {
		t.Fatalf("ApplyWorkspaceAuth() error = %v", err)
	}
	if updated.Repos[0].HTTPBasicAuth != nil {
		t.Fatalf("expected missing credential to leave request unchanged, got %+v", updated.Repos[0].HTTPBasicAuth)
	}
}

type stubWorkspaceAuthResolver struct {
	resolved domain.ResolvedCredential
	err      error
}

func (s stubWorkspaceAuthResolver) ResolveProjectCredential(context.Context, uuid.UUID) (domain.ResolvedCredential, error) {
	if s.err != nil {
		return domain.ResolvedCredential{}, s.err
	}
	return s.resolved, nil
}
