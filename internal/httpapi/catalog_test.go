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
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type fakeCatalogService struct {
	organizations map[uuid.UUID]domain.Organization
	projects      map[uuid.UUID]domain.Project
}

func newFakeCatalogService() *fakeCatalogService {
	return &fakeCatalogService{
		organizations: map[uuid.UUID]domain.Organization{},
		projects:      map[uuid.UUID]domain.Project{},
	}
}

func TestCatalogCRUDRoutes(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		newFakeCatalogService(),
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
