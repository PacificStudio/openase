package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/BetterAndBetterII/openase/internal/config"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type fakeCatalogService struct {
	organizations map[uuid.UUID]domain.Organization
	projects      map[uuid.UUID]domain.Project
	projectRepos  map[uuid.UUID]domain.ProjectRepo
	providers     map[uuid.UUID]domain.AgentProvider
	agents        map[uuid.UUID]domain.Agent
}

func newFakeCatalogService() *fakeCatalogService {
	return &fakeCatalogService{
		organizations: map[uuid.UUID]domain.Organization{},
		projects:      map[uuid.UUID]domain.Project{},
		projectRepos:  map[uuid.UUID]domain.ProjectRepo{},
		providers:     map[uuid.UUID]domain.AgentProvider{},
		agents:        map[uuid.UUID]domain.Agent{},
	}
}

func TestCatalogCRUDRoutes(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newFakeCatalogService(),
		nil,
	)

	orgBody := `{"name":"Acme Platform","slug":"acme-platform"}`
	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", orgBody)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var createOrgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &createOrgPayload)
	if createOrgPayload.Organization.Name != "Acme Platform" {
		t.Fatalf("expected organization name to round-trip, got %+v", createOrgPayload.Organization)
	}

	listOrgRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs", "")
	if listOrgRec.Code != http.StatusOK {
		t.Fatalf("expected organization list 200, got %d", listOrgRec.Code)
	}

	var listOrgPayload struct {
		Organizations []organizationResponse `json:"organizations"`
	}
	decodeResponse(t, listOrgRec, &listOrgPayload)
	if len(listOrgPayload.Organizations) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(listOrgPayload.Organizations))
	}

	patchOrgRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/orgs/"+createOrgPayload.Organization.ID,
		`{"name":"Acme Control Plane"}`,
	)
	if patchOrgRec.Code != http.StatusOK {
		t.Fatalf("expected organization patch 200, got %d: %s", patchOrgRec.Code, patchOrgRec.Body.String())
	}

	projectBody := `{"name":"OpenASE","slug":"openase","description":"Main control plane","status":"active","max_concurrent_agents":8}`
	projectRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+createOrgPayload.Organization.ID+"/projects",
		projectBody,
	)
	if projectRec.Code != http.StatusCreated {
		t.Fatalf("expected project create 201, got %d: %s", projectRec.Code, projectRec.Body.String())
	}

	var createProjectPayload struct {
		Project projectResponse `json:"project"`
	}
	decodeResponse(t, projectRec, &createProjectPayload)
	if createProjectPayload.Project.Status != "active" || createProjectPayload.Project.MaxConcurrentAgents != 8 {
		t.Fatalf("unexpected created project payload: %+v", createProjectPayload.Project)
	}

	listProjectRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/orgs/"+createOrgPayload.Organization.ID+"/projects",
		"",
	)
	if listProjectRec.Code != http.StatusOK {
		t.Fatalf("expected project list 200, got %d", listProjectRec.Code)
	}

	var listProjectPayload struct {
		Projects []projectResponse `json:"projects"`
	}
	decodeResponse(t, listProjectRec, &listProjectPayload)
	if len(listProjectPayload.Projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(listProjectPayload.Projects))
	}

	patchProjectRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+createProjectPayload.Project.ID,
		`{"status":"paused","max_concurrent_agents":3}`,
	)
	if patchProjectRec.Code != http.StatusOK {
		t.Fatalf("expected project patch 200, got %d: %s", patchProjectRec.Code, patchProjectRec.Body.String())
	}

	var patchProjectPayload struct {
		Project projectResponse `json:"project"`
	}
	decodeResponse(t, patchProjectRec, &patchProjectPayload)
	if patchProjectPayload.Project.Status != "paused" || patchProjectPayload.Project.MaxConcurrentAgents != 3 {
		t.Fatalf("unexpected patched project payload: %+v", patchProjectPayload.Project)
	}

	repoRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos",
		`{"name":"backend","repository_url":"https://github.com/acme/backend.git","labels":["go","api"]}`,
	)
	if repoRec.Code != http.StatusCreated {
		t.Fatalf("expected repo create 201, got %d: %s", repoRec.Code, repoRec.Body.String())
	}

	var createRepoPayload struct {
		Repo projectRepoResponse `json:"repo"`
	}
	decodeResponse(t, repoRec, &createRepoPayload)
	if !createRepoPayload.Repo.IsPrimary || createRepoPayload.Repo.DefaultBranch != "main" {
		t.Fatalf("unexpected created repo payload: %+v", createRepoPayload.Repo)
	}

	secondRepoRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos",
		`{"name":"frontend","repository_url":"https://github.com/acme/frontend.git","default_branch":"develop","is_primary":true}`,
	)
	if secondRepoRec.Code != http.StatusCreated {
		t.Fatalf("expected second repo create 201, got %d: %s", secondRepoRec.Code, secondRepoRec.Body.String())
	}

	var secondRepoPayload struct {
		Repo projectRepoResponse `json:"repo"`
	}
	decodeResponse(t, secondRepoRec, &secondRepoPayload)
	if !secondRepoPayload.Repo.IsPrimary {
		t.Fatalf("expected second repo to become primary, got %+v", secondRepoPayload.Repo)
	}

	listRepoRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos",
		"",
	)
	if listRepoRec.Code != http.StatusOK {
		t.Fatalf("expected repo list 200, got %d: %s", listRepoRec.Code, listRepoRec.Body.String())
	}

	var listRepoPayload struct {
		Repos []projectRepoResponse `json:"repos"`
	}
	decodeResponse(t, listRepoRec, &listRepoPayload)
	if len(listRepoPayload.Repos) != 2 || !listRepoPayload.Repos[0].IsPrimary {
		t.Fatalf("unexpected repo list payload: %+v", listRepoPayload.Repos)
	}

	patchRepoRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos/"+createRepoPayload.Repo.ID,
		`{"clone_path":"services/backend","is_primary":true}`,
	)
	if patchRepoRec.Code != http.StatusOK {
		t.Fatalf("expected repo patch 200, got %d: %s", patchRepoRec.Code, patchRepoRec.Body.String())
	}

	var patchRepoPayload struct {
		Repo projectRepoResponse `json:"repo"`
	}
	decodeResponse(t, patchRepoRec, &patchRepoPayload)
	if !patchRepoPayload.Repo.IsPrimary || patchRepoPayload.Repo.ClonePath == nil || *patchRepoPayload.Repo.ClonePath != "services/backend" {
		t.Fatalf("unexpected patched repo payload: %+v", patchRepoPayload.Repo)
	}

	deleteRepoRec := performJSONRequest(
		t,
		server,
		http.MethodDelete,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos/"+secondRepoPayload.Repo.ID,
		"",
	)
	if deleteRepoRec.Code != http.StatusOK {
		t.Fatalf("expected repo delete 200, got %d: %s", deleteRepoRec.Code, deleteRepoRec.Body.String())
	}

	archiveProjectRec := performJSONRequest(
		t,
		server,
		http.MethodDelete,
		"/api/v1/projects/"+createProjectPayload.Project.ID,
		"",
	)
	if archiveProjectRec.Code != http.StatusOK {
		t.Fatalf("expected project archive 200, got %d: %s", archiveProjectRec.Code, archiveProjectRec.Body.String())
	}

	var archiveProjectPayload struct {
		Project projectResponse `json:"project"`
	}
	decodeResponse(t, archiveProjectRec, &archiveProjectPayload)
	if archiveProjectPayload.Project.Status != "archived" {
		t.Fatalf("expected archived project status, got %+v", archiveProjectPayload.Project)
	}
}

func TestCatalogRoutesRejectInvalidInput(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newFakeCatalogService(),
		nil,
	)

	badSlugRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme","slug":"Acme Spaces"}`)
	if badSlugRec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad slug to return 400, got %d", badSlugRec.Code)
	}

	badUUIDRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/not-a-uuid", "")
	if badUUIDRec.Code != http.StatusBadRequest {
		t.Fatalf("expected bad uuid to return 400, got %d", badUUIDRec.Code)
	}
}

func performJSONRequest(t *testing.T, server *Server, method string, target string, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(method, target, bytes.NewBufferString(body))
	if body != "" {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	return rec
}

func decodeResponse(t *testing.T, rec *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("failed to decode response %q: %v", rec.Body.String(), err)
	}
}

func (f *fakeCatalogService) ListOrganizations(context.Context) ([]domain.Organization, error) {
	items := make([]domain.Organization, 0, len(f.organizations))
	for _, item := range f.organizations {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (f *fakeCatalogService) CreateOrganization(_ context.Context, input domain.CreateOrganization) (domain.Organization, error) {
	for _, item := range f.organizations {
		if item.Slug == input.Slug {
			return domain.Organization{}, catalogservice.ErrConflict
		}
	}

	item := domain.Organization{
		ID:                     uuid.New(),
		Name:                   input.Name,
		Slug:                   input.Slug,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
	}
	f.organizations[item.ID] = item

	return item, nil
}

func (f *fakeCatalogService) GetOrganization(_ context.Context, id uuid.UUID) (domain.Organization, error) {
	item, ok := f.organizations[id]
	if !ok {
		return domain.Organization{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateOrganization(_ context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	if _, ok := f.organizations[input.ID]; !ok {
		return domain.Organization{}, catalogservice.ErrNotFound
	}

	item := domain.Organization{
		ID:                     input.ID,
		Name:                   input.Name,
		Slug:                   input.Slug,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
	}
	f.organizations[input.ID] = item

	return item, nil
}

func (f *fakeCatalogService) ListProjects(_ context.Context, organizationID uuid.UUID) ([]domain.Project, error) {
	if _, ok := f.organizations[organizationID]; !ok {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.Project, 0)
	for _, item := range f.projects {
		if item.OrganizationID == organizationID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (f *fakeCatalogService) CreateProject(_ context.Context, input domain.CreateProject) (domain.Project, error) {
	if _, ok := f.organizations[input.OrganizationID]; !ok {
		return domain.Project{}, catalogservice.ErrNotFound
	}

	for _, item := range f.projects {
		if item.OrganizationID == input.OrganizationID && item.Slug == input.Slug {
			return domain.Project{}, catalogservice.ErrConflict
		}
	}

	item := domain.Project{
		ID:                     uuid.New(),
		OrganizationID:         input.OrganizationID,
		Name:                   input.Name,
		Slug:                   input.Slug,
		Description:            input.Description,
		Status:                 input.Status,
		DefaultWorkflowID:      input.DefaultWorkflowID,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
		MaxConcurrentAgents:    input.MaxConcurrentAgents,
	}
	f.projects[item.ID] = item

	return item, nil
}

func (f *fakeCatalogService) GetProject(_ context.Context, id uuid.UUID) (domain.Project, error) {
	item, ok := f.projects[id]
	if !ok {
		return domain.Project{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateProject(_ context.Context, input domain.UpdateProject) (domain.Project, error) {
	if _, ok := f.projects[input.ID]; !ok {
		return domain.Project{}, catalogservice.ErrNotFound
	}

	item := domain.Project{
		ID:                     input.ID,
		OrganizationID:         input.OrganizationID,
		Name:                   input.Name,
		Slug:                   input.Slug,
		Description:            strings.TrimSpace(input.Description),
		Status:                 input.Status,
		DefaultWorkflowID:      input.DefaultWorkflowID,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
		MaxConcurrentAgents:    input.MaxConcurrentAgents,
	}
	f.projects[input.ID] = item

	return item, nil
}

func (f *fakeCatalogService) ArchiveProject(_ context.Context, id uuid.UUID) (domain.Project, error) {
	item, ok := f.projects[id]
	if !ok {
		return domain.Project{}, catalogservice.ErrNotFound
	}

	item.Status = "archived"
	f.projects[id] = item

	return item, nil
}

func (f *fakeCatalogService) ListProjectRepos(_ context.Context, projectID uuid.UUID) ([]domain.ProjectRepo, error) {
	if _, ok := f.projects[projectID]; !ok {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.ProjectRepo, 0)
	for _, item := range f.projectRepos {
		if item.ProjectID == projectID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsPrimary != items[j].IsPrimary {
			return items[i].IsPrimary
		}
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (f *fakeCatalogService) CreateProjectRepo(_ context.Context, input domain.CreateProjectRepo) (domain.ProjectRepo, error) {
	if _, ok := f.projects[input.ProjectID]; !ok {
		return domain.ProjectRepo{}, catalogservice.ErrNotFound
	}

	for _, item := range f.projectRepos {
		if item.ProjectID == input.ProjectID && item.Name == input.Name {
			return domain.ProjectRepo{}, catalogservice.ErrConflict
		}
	}

	isPrimary := !f.hasProjectRepos(input.ProjectID)
	if input.RequestedPrimary != nil {
		isPrimary = *input.RequestedPrimary || !f.hasProjectRepos(input.ProjectID)
	}
	if isPrimary {
		f.clearPrimary(input.ProjectID, uuid.Nil)
	}

	item := domain.ProjectRepo{
		ID:            uuid.New(),
		ProjectID:     input.ProjectID,
		Name:          input.Name,
		RepositoryURL: input.RepositoryURL,
		DefaultBranch: input.DefaultBranch,
		ClonePath:     input.ClonePath,
		IsPrimary:     isPrimary,
		Labels:        append([]string(nil), input.Labels...),
	}
	f.projectRepos[item.ID] = item

	return item, nil
}

func (f *fakeCatalogService) GetProjectRepo(_ context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	item, ok := f.projectRepos[id]
	if !ok || item.ProjectID != projectID {
		return domain.ProjectRepo{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateProjectRepo(_ context.Context, input domain.UpdateProjectRepo) (domain.ProjectRepo, error) {
	item, ok := f.projectRepos[input.ID]
	if !ok || item.ProjectID != input.ProjectID {
		return domain.ProjectRepo{}, catalogservice.ErrNotFound
	}

	if input.IsPrimary {
		f.clearPrimary(input.ProjectID, input.ID)
	}

	item.Name = input.Name
	item.RepositoryURL = input.RepositoryURL
	item.DefaultBranch = input.DefaultBranch
	item.ClonePath = input.ClonePath
	item.IsPrimary = input.IsPrimary
	item.Labels = append([]string(nil), input.Labels...)
	f.projectRepos[item.ID] = item

	if !item.IsPrimary {
		f.ensurePrimary(input.ProjectID, item.ID)
		item = f.projectRepos[item.ID]
	}

	return item, nil
}

func (f *fakeCatalogService) DeleteProjectRepo(_ context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	item, ok := f.projectRepos[id]
	if !ok || item.ProjectID != projectID {
		return domain.ProjectRepo{}, catalogservice.ErrNotFound
	}

	delete(f.projectRepos, id)
	if item.IsPrimary {
		f.ensurePrimary(projectID, uuid.Nil)
	}

	return item, nil
}

func (f *fakeCatalogService) hasProjectRepos(projectID uuid.UUID) bool {
	for _, item := range f.projectRepos {
		if item.ProjectID == projectID {
			return true
		}
	}

	return false
}

func (f *fakeCatalogService) clearPrimary(projectID uuid.UUID, exclude uuid.UUID) {
	for id, item := range f.projectRepos {
		if item.ProjectID == projectID && id != exclude {
			item.IsPrimary = false
			f.projectRepos[id] = item
		}
	}
}

func (f *fakeCatalogService) ensurePrimary(projectID uuid.UUID, preferredExclude uuid.UUID) {
	for _, item := range f.projectRepos {
		if item.ProjectID == projectID && item.IsPrimary {
			return
		}
	}

	var fallback *domain.ProjectRepo
	for _, item := range f.projectRepos {
		if item.ProjectID == projectID && item.ID != preferredExclude {
			copied := item
			if fallback == nil || copied.Name < fallback.Name {
				fallback = &copied
			}
		}
	}
	if fallback == nil {
		for _, item := range f.projectRepos {
			if item.ProjectID == projectID {
				copied := item
				if fallback == nil || copied.Name < fallback.Name {
					fallback = &copied
				}
			}
		}
	}
	if fallback == nil {
		return
	}

	fallback.IsPrimary = true
	f.projectRepos[fallback.ID] = *fallback
}

func TestProjectRepoPrimaryLifecycleWithEntRepository(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver()),
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Acme").
		SetSlug("acme").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	var backendCreate struct {
		Repo projectRepoResponse `json:"repo"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+project.ID.String()+"/repos",
		map[string]any{
			"name":           "backend",
			"repository_url": "https://github.com/acme/backend.git",
		},
		http.StatusCreated,
		&backendCreate,
	)
	if !backendCreate.Repo.IsPrimary {
		t.Fatalf("expected first repo to be primary, got %+v", backendCreate.Repo)
	}

	var frontendCreate struct {
		Repo projectRepoResponse `json:"repo"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+project.ID.String()+"/repos",
		map[string]any{
			"name":           "frontend",
			"repository_url": "https://github.com/acme/frontend.git",
			"is_primary":     true,
		},
		http.StatusCreated,
		&frontendCreate,
	)
	if !frontendCreate.Repo.IsPrimary {
		t.Fatalf("expected frontend repo to be primary, got %+v", frontendCreate.Repo)
	}

	var backendUpdate struct {
		Repo projectRepoResponse `json:"repo"`
	}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+project.ID.String()+"/repos/"+backendCreate.Repo.ID,
		map[string]any{
			"is_primary": true,
		},
		http.StatusOK,
		&backendUpdate,
	)
	if !backendUpdate.Repo.IsPrimary {
		t.Fatalf("expected backend repo to regain primary, got %+v", backendUpdate.Repo)
	}

	executeJSON(
		t,
		server,
		http.MethodDelete,
		"/api/v1/projects/"+project.ID.String()+"/repos/"+backendUpdate.Repo.ID,
		nil,
		http.StatusOK,
		nil,
	)

	var repoList struct {
		Repos []projectRepoResponse `json:"repos"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+project.ID.String()+"/repos",
		nil,
		http.StatusOK,
		&repoList,
	)
	if len(repoList.Repos) != 1 || repoList.Repos[0].Name != "frontend" || !repoList.Repos[0].IsPrimary {
		t.Fatalf("expected surviving repo to stay primary, got %+v", repoList.Repos)
	}
}
