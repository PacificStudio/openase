package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	githubrepodomain "github.com/BetterAndBetterII/openase/internal/domain/githubrepo"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	githubreposervice "github.com/BetterAndBetterII/openase/internal/service/githubrepo"
	"github.com/google/uuid"
)

type stubGitHubRepoService struct {
	namespaces []githubrepodomain.Namespace
	page       githubrepodomain.RepositoryPage
	created    githubrepodomain.Repository
	listInput  githubrepodomain.ListRepositoriesInput
	createIn   githubrepodomain.CreateRepositoryInput
	err        error
}

func (s *stubGitHubRepoService) ListNamespaces(context.Context, uuid.UUID) ([]githubrepodomain.Namespace, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]githubrepodomain.Namespace(nil), s.namespaces...), nil
}

func (s *stubGitHubRepoService) ListRepositories(_ context.Context, input githubrepodomain.ListRepositoriesInput) (githubrepodomain.RepositoryPage, error) {
	if s.err != nil {
		return githubrepodomain.RepositoryPage{}, s.err
	}
	s.listInput = input
	return s.page, nil
}

func (s *stubGitHubRepoService) CreateRepository(_ context.Context, input githubrepodomain.CreateRepositoryInput) (githubrepodomain.Repository, error) {
	if s.err != nil {
		return githubrepodomain.Repository{}, s.err
	}
	s.createIn = input
	return s.created, nil
}

func TestGitHubRepoRoutes(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{
		ID:             projectID,
		OrganizationID: uuid.New(),
		Name:           "OpenASE",
		Slug:           "openase",
		Description:    "control plane",
		Status:         domain.ProjectStatusPlanned,
	}

	githubSvc := &stubGitHubRepoService{
		namespaces: []githubrepodomain.Namespace{
			{Login: "octocat", Kind: githubrepodomain.NamespaceKindUser},
			{Login: "acme", Kind: githubrepodomain.NamespaceKindOrganization},
		},
		page: githubrepodomain.RepositoryPage{
			Repositories: []githubrepodomain.Repository{{
				ID:            42,
				Name:          "backend",
				FullName:      "acme/backend",
				Owner:         "acme",
				DefaultBranch: "main",
				Visibility:    githubrepodomain.VisibilityPrivate,
				Private:       true,
				HTMLURL:       "https://github.com/acme/backend",
				CloneURL:      "https://github.com/acme/backend.git",
			}},
			NextCursor: "3",
		},
		created: githubrepodomain.Repository{
			ID:            88,
			Name:          "frontend",
			FullName:      "octocat/frontend",
			Owner:         "octocat",
			DefaultBranch: "main",
			Visibility:    githubrepodomain.VisibilityPublic,
			HTMLURL:       "https://github.com/octocat/frontend",
			CloneURL:      "https://github.com/octocat/frontend.git",
		},
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubRepoService(githubSvc),
	)

	var namespacesPayload githubRepositoryNamespacesResponse
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/github/namespaces",
		nil,
		http.StatusOK,
		&namespacesPayload,
	)
	if len(namespacesPayload.Namespaces) != 2 || namespacesPayload.Namespaces[0].Login != "octocat" {
		t.Fatalf("namespaces payload = %+v", namespacesPayload)
	}

	var reposPayload githubRepositoryListResponse
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+projectID.String()+"/github/repos?query=back&cursor=2",
		nil,
		http.StatusOK,
		&reposPayload,
	)
	if githubSvc.listInput.Query != "back" || githubSvc.listInput.Page != 2 {
		t.Fatalf("ListRepositories input = %+v", githubSvc.listInput)
	}
	if len(reposPayload.Repositories) != 1 || reposPayload.NextCursor != "3" {
		t.Fatalf("repos payload = %+v", reposPayload)
	}

	var createdPayload githubRepositoryCreateResponse
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/github/repos",
		map[string]any{
			"owner":      "octocat",
			"name":       "frontend",
			"visibility": "public",
		},
		http.StatusCreated,
		&createdPayload,
	)
	if githubSvc.createIn.Owner != "octocat" || githubSvc.createIn.Name != "frontend" {
		t.Fatalf("CreateRepository input = %+v", githubSvc.createIn)
	}
	if createdPayload.Repository.FullName != "octocat/frontend" {
		t.Fatalf("created payload = %+v", createdPayload)
	}
}

func TestGitHubRepoRoutesMapCredentialRequirement(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = domain.Project{
		ID:             projectID,
		OrganizationID: uuid.New(),
		Name:           "OpenASE",
		Slug:           "openase",
		Description:    "control plane",
		Status:         domain.ProjectStatusPlanned,
	}

	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalog,
		nil,
		WithGitHubRepoService(&stubGitHubRepoService{err: githubreposervice.ErrCredentialMissing}),
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+projectID.String()+"/github/repos", "")
	if rec.Code != http.StatusPreconditionFailed {
		t.Fatalf("expected 412, got %d: %s", rec.Code, rec.Body.String())
	}
}
