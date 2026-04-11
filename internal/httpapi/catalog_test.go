package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	"github.com/BetterAndBetterII/openase/internal/infra/executable"
	"github.com/BetterAndBetterII/openase/internal/provider"
	catalogrepo "github.com/BetterAndBetterII/openase/internal/repo/catalog"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	"github.com/BetterAndBetterII/openase/internal/ticketstatus"
	git "github.com/go-git/go-git/v5"
	gitconfig "github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type fakeCatalogService struct {
	organizations        map[uuid.UUID]domain.Organization
	machines             map[uuid.UUID]domain.Machine
	projects             map[uuid.UUID]domain.Project
	tickets              map[uuid.UUID]fakeCatalogTicket
	projectRepos         map[uuid.UUID]domain.ProjectRepo
	ticketScopes         map[uuid.UUID]domain.TicketRepoScope
	providers            map[uuid.UUID]domain.AgentProvider
	agents               map[uuid.UUID]domain.Agent
	agentRuns            map[uuid.UUID]domain.AgentRun
	agentDeleteConflicts map[uuid.UUID]*domain.AgentDeleteConflict
	activityEvents       []domain.ActivityEvent
	traceEvents          []domain.AgentTraceEntry
	stepEvents           []domain.AgentStepEntry
}

type fakeCatalogTicket struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
}

func newFakeCatalogService() *fakeCatalogService {
	return &fakeCatalogService{
		organizations:        map[uuid.UUID]domain.Organization{},
		machines:             map[uuid.UUID]domain.Machine{},
		projects:             map[uuid.UUID]domain.Project{},
		tickets:              map[uuid.UUID]fakeCatalogTicket{},
		projectRepos:         map[uuid.UUID]domain.ProjectRepo{},
		ticketScopes:         map[uuid.UUID]domain.TicketRepoScope{},
		providers:            map[uuid.UUID]domain.AgentProvider{},
		agents:               map[uuid.UUID]domain.Agent{},
		agentRuns:            map[uuid.UUID]domain.AgentRun{},
		agentDeleteConflicts: map[uuid.UUID]*domain.AgentDeleteConflict{},
		activityEvents:       []domain.ActivityEvent{},
		traceEvents:          []domain.AgentTraceEntry{},
		stepEvents:           []domain.AgentStepEntry{},
	}
}

func TestCatalogCRUDRoutes(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
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
	if createOrgPayload.Organization.Status != "active" {
		t.Fatalf("expected created organization to be active, got %+v", createOrgPayload.Organization)
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

	accessibleMachineID := uuid.NewString()
	projectBody := fmt.Sprintf(`{"name":"OpenASE","slug":"openase","description":"Main control plane","status":"In Progress","accessible_machine_ids":["%s"],"max_concurrent_agents":8,"agent_run_summary_prompt":"Summarize ticket runs."}`, accessibleMachineID)
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
	if createProjectPayload.Project.Status != "In Progress" || createProjectPayload.Project.MaxConcurrentAgents != 8 {
		t.Fatalf("unexpected created project payload: %+v", createProjectPayload.Project)
	}
	if createProjectPayload.Project.AgentRunSummaryPrompt == nil || *createProjectPayload.Project.AgentRunSummaryPrompt != "Summarize ticket runs." {
		t.Fatalf("expected project summary prompt to round-trip on create, got %+v", createProjectPayload.Project)
	}
	if createProjectPayload.Project.EffectiveAgentRunSummaryPrompt != "Summarize ticket runs." {
		t.Fatalf("expected effective project summary prompt to match override, got %+v", createProjectPayload.Project)
	}
	if createProjectPayload.Project.AgentRunSummaryPromptSource != domain.AgentRunSummaryPromptSourceProjectOverride.String() {
		t.Fatalf("expected project summary prompt source to be project override, got %+v", createProjectPayload.Project)
	}
	if len(createProjectPayload.Project.AccessibleMachineIDs) != 1 || createProjectPayload.Project.AccessibleMachineIDs[0] != accessibleMachineID {
		t.Fatalf("expected project accessible machines to round-trip, got %+v", createProjectPayload.Project)
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

	getProjectRec := performJSONRequest(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+createProjectPayload.Project.ID,
		"",
	)
	if getProjectRec.Code != http.StatusOK {
		t.Fatalf("expected project get 200, got %d: %s", getProjectRec.Code, getProjectRec.Body.String())
	}

	var getProjectPayload struct {
		Project projectResponse `json:"project"`
	}
	decodeResponse(t, getProjectRec, &getProjectPayload)
	if getProjectPayload.Project.ID != createProjectPayload.Project.ID || getProjectPayload.Project.Name != "OpenASE" {
		t.Fatalf("unexpected project get payload: %+v", getProjectPayload.Project)
	}

	updatedAccessibleMachineID := uuid.NewString()
	patchProjectRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+createProjectPayload.Project.ID,
		fmt.Sprintf(`{"status":"Canceled","accessible_machine_ids":["%s"],"max_concurrent_agents":3,"agent_run_summary_prompt":""}`, updatedAccessibleMachineID),
	)
	if patchProjectRec.Code != http.StatusOK {
		t.Fatalf("expected project patch 200, got %d: %s", patchProjectRec.Code, patchProjectRec.Body.String())
	}

	var patchProjectPayload struct {
		Project projectResponse `json:"project"`
	}
	decodeResponse(t, patchProjectRec, &patchProjectPayload)
	if patchProjectPayload.Project.Status != "Canceled" || patchProjectPayload.Project.MaxConcurrentAgents != 3 {
		t.Fatalf("unexpected patched project payload: %+v", patchProjectPayload.Project)
	}
	if patchProjectPayload.Project.AgentRunSummaryPrompt != nil {
		t.Fatalf("expected blank prompt patch to fall back to unset, got %+v", patchProjectPayload.Project)
	}
	if patchProjectPayload.Project.EffectiveAgentRunSummaryPrompt != strings.TrimSpace(domain.DefaultAgentRunSummaryPrompt) {
		t.Fatalf("expected blank prompt patch to surface built-in effective prompt, got %+v", patchProjectPayload.Project)
	}
	if patchProjectPayload.Project.AgentRunSummaryPromptSource != domain.AgentRunSummaryPromptSourceBuiltin.String() {
		t.Fatalf("expected blank prompt patch to use built-in prompt source, got %+v", patchProjectPayload.Project)
	}
	if len(patchProjectPayload.Project.AccessibleMachineIDs) != 1 || patchProjectPayload.Project.AccessibleMachineIDs[0] != updatedAccessibleMachineID {
		t.Fatalf("expected patched project accessible machines to round-trip, got %+v", patchProjectPayload.Project)
	}

	repoRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos",
		`{"name":"backend","repository_url":"file:///srv/git/backend.git","labels":["go","api"]}`,
	)
	if repoRec.Code != http.StatusCreated {
		t.Fatalf("expected repo create 201, got %d: %s", repoRec.Code, repoRec.Body.String())
	}

	var createRepoPayload struct {
		Repo projectRepoResponse `json:"repo"`
	}
	decodeResponse(t, repoRec, &createRepoPayload)
	if createRepoPayload.Repo.DefaultBranch != "main" || createRepoPayload.Repo.RepositoryURL != "file:///srv/git/backend.git" {
		t.Fatalf("unexpected created repo payload: %+v", createRepoPayload.Repo)
	}

	secondRepoRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos",
		`{"name":"frontend","repository_url":"https://github.com/acme/frontend.git","default_branch":"develop"}`,
	)
	if secondRepoRec.Code != http.StatusCreated {
		t.Fatalf("expected second repo create 201, got %d: %s", secondRepoRec.Code, secondRepoRec.Body.String())
	}

	var secondRepoPayload struct {
		Repo projectRepoResponse `json:"repo"`
	}
	decodeResponse(t, secondRepoRec, &secondRepoPayload)
	if !strings.Contains(secondRepoRec.Body.String(), `"labels":[]`) {
		t.Fatalf("expected second repo response to include empty labels array, got %s", secondRepoRec.Body.String())
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
	if len(listRepoPayload.Repos) != 2 {
		t.Fatalf("unexpected repo list payload: %+v", listRepoPayload.Repos)
	}
	if !strings.Contains(listRepoRec.Body.String(), `"labels":[]`) {
		t.Fatalf("expected repo list response to include empty labels array for unlabeled repos, got %s", listRepoRec.Body.String())
	}

	patchRepoRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+createProjectPayload.Project.ID+"/repos/"+createRepoPayload.Repo.ID,
		`{"repository_url":"file:///srv/git/backend-mirror.git","workspace_dirname":"services/backend"}`,
	)
	if patchRepoRec.Code != http.StatusOK {
		t.Fatalf("expected repo patch 200, got %d: %s", patchRepoRec.Code, patchRepoRec.Body.String())
	}

	var patchRepoPayload struct {
		Repo projectRepoResponse `json:"repo"`
	}
	decodeResponse(t, patchRepoRec, &patchRepoPayload)
	if patchRepoPayload.Repo.WorkspaceDirname != "services/backend" || patchRepoPayload.Repo.RepositoryURL != "file:///srv/git/backend-mirror.git" {
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
	if archiveProjectPayload.Project.Status != "Archived" {
		t.Fatalf("expected archived project status, got %+v", archiveProjectPayload.Project)
	}
}

func TestCatalogRoutesErrorMappingsAndInvalidPayloads(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	orgID := uuid.New()
	localMachineID := uuid.New()
	machineID := uuid.New()
	projectID := uuid.New()
	repoID := uuid.New()
	ticketID := uuid.New()
	scopeID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme", Status: "active"}
	service.machines[localMachineID] = domain.Machine{
		ID:             localMachineID,
		OrganizationID: orgID,
		Name:           domain.LocalMachineName,
		Host:           domain.LocalMachineHost,
		Port:           22,
		Status:         "online",
	}
	service.machines[machineID] = domain.Machine{ID: machineID, OrganizationID: orgID, Name: "builder", Host: "builder.local", Port: 22, Status: "online"}
	service.projects[projectID] = domain.Project{ID: projectID, OrganizationID: orgID, Name: "OpenASE", Slug: "openase", Status: "In Progress"}
	service.projectRepos[repoID] = domain.ProjectRepo{
		ID:            repoID,
		ProjectID:     projectID,
		Name:          "backend",
		RepositoryURL: "https://github.com/acme/backend.git",
		DefaultBranch: "main",
	}
	service.tickets[ticketID] = fakeCatalogTicket{ID: ticketID, ProjectID: projectID}
	service.ticketScopes[scopeID] = domain.TicketRepoScope{
		ID:         scopeID,
		TicketID:   ticketID,
		RepoID:     repoID,
		BranchName: "main",
	}

	for _, testCase := range []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
		wantBody   string
	}{
		{name: "get organization invalid id", method: http.MethodGet, path: "/api/v1/orgs/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "orgId must be a valid UUID"},
		{name: "get organization missing", method: http.MethodGet, path: "/api/v1/orgs/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "patch organization unknown field", method: http.MethodPatch, path: "/api/v1/orgs/" + orgID.String(), body: `{"unknown":true}`, wantStatus: http.StatusBadRequest, wantBody: "invalid JSON body"},
		{name: "list projects invalid org id", method: http.MethodGet, path: "/api/v1/orgs/not-a-uuid/projects", wantStatus: http.StatusBadRequest, wantBody: "orgId must be a valid UUID"},
		{name: "list machines invalid org id", method: http.MethodGet, path: "/api/v1/orgs/not-a-uuid/machines", wantStatus: http.StatusBadRequest, wantBody: "orgId must be a valid UUID"},
		{name: "create project invalid payload", method: http.MethodPost, path: "/api/v1/orgs/" + orgID.String() + "/projects", body: `{"name":" ","slug":"openase"}`, wantStatus: http.StatusBadRequest, wantBody: "name must not be empty"},
		{name: "create project legacy status", method: http.MethodPost, path: "/api/v1/orgs/" + orgID.String() + "/projects", body: `{"name":"OpenASE","slug":"openase","status":"active"}`, wantStatus: http.StatusBadRequest, wantBody: "status must be one of Backlog, Planned, In Progress, Completed, Canceled, Archived"},
		{name: "create project whitespace status", method: http.MethodPost, path: "/api/v1/orgs/" + orgID.String() + "/projects", body: `{"name":"OpenASE","slug":"openase","status":" Planned "}`, wantStatus: http.StatusBadRequest, wantBody: "status must be one of Backlog, Planned, In Progress, Completed, Canceled, Archived"},
		{name: "create machine invalid payload", method: http.MethodPost, path: "/api/v1/orgs/" + orgID.String() + "/machines", body: `{"name":"builder","host":" ","port":22}`, wantStatus: http.StatusBadRequest, wantBody: "host must not be empty"},
		{name: "get machine invalid id", method: http.MethodGet, path: "/api/v1/machines/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "machineId must be a valid UUID"},
		{name: "get machine missing", method: http.MethodGet, path: "/api/v1/machines/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "patch machine invalid id", method: http.MethodPatch, path: "/api/v1/machines/not-a-uuid", body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "machineId must be a valid UUID"},
		{name: "patch machine multiple values", method: http.MethodPatch, path: "/api/v1/machines/" + machineID.String(), body: `{"name":"builder"}{"name":"again"}`, wantStatus: http.StatusBadRequest, wantBody: "multiple JSON values are not allowed"},
		{name: "patch machine invalid status", method: http.MethodPatch, path: "/api/v1/machines/" + localMachineID.String(), body: `{"status":"broken"}`, wantStatus: http.StatusBadRequest, wantBody: "status must be one of"},
		{name: "delete machine invalid id", method: http.MethodDelete, path: "/api/v1/machines/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "machineId must be a valid UUID"},
		{name: "delete machine local invalid input", method: http.MethodDelete, path: "/api/v1/machines/" + localMachineID.String(), wantStatus: http.StatusBadRequest, wantBody: "invalid input"},
		{name: "test machine invalid id", method: http.MethodPost, path: "/api/v1/machines/not-a-uuid/test", wantStatus: http.StatusBadRequest, wantBody: "machineId must be a valid UUID"},
		{name: "test machine missing", method: http.MethodPost, path: "/api/v1/machines/" + uuid.NewString() + "/test", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "refresh machine health invalid id", method: http.MethodPost, path: "/api/v1/machines/not-a-uuid/refresh-health", wantStatus: http.StatusBadRequest, wantBody: "machineId must be a valid UUID"},
		{name: "refresh machine health missing", method: http.MethodPost, path: "/api/v1/machines/" + uuid.NewString() + "/refresh-health", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "get machine resources missing", method: http.MethodGet, path: "/api/v1/machines/" + uuid.NewString() + "/resources", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "get project invalid id", method: http.MethodGet, path: "/api/v1/projects/not-a-uuid", wantStatus: http.StatusBadRequest, wantBody: "projectId must be a valid UUID"},
		{name: "get project missing", method: http.MethodGet, path: "/api/v1/projects/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "patch project legacy status", method: http.MethodPatch, path: "/api/v1/projects/" + projectID.String(), body: `{"status":"paused"}`, wantStatus: http.StatusBadRequest, wantBody: "status must be one of Backlog, Planned, In Progress, Completed, Canceled, Archived"},
		{name: "patch project lowercase status", method: http.MethodPatch, path: "/api/v1/projects/" + projectID.String(), body: `{"status":"planned"}`, wantStatus: http.StatusBadRequest, wantBody: "status must be one of Backlog, Planned, In Progress, Completed, Canceled, Archived"},
		{name: "list repos invalid project id", method: http.MethodGet, path: "/api/v1/projects/not-a-uuid/repos", wantStatus: http.StatusBadRequest, wantBody: "projectId must be a valid UUID"},
		{name: "create repo invalid project id", method: http.MethodPost, path: "/api/v1/projects/not-a-uuid/repos", body: `{"name":"backend","repository_url":"https://github.com/acme/backend.git"}`, wantStatus: http.StatusBadRequest, wantBody: "projectId must be a valid UUID"},
		{name: "create repo invalid payload", method: http.MethodPost, path: "/api/v1/projects/" + projectID.String() + "/repos", body: `{"name":" ","repository_url":"https://github.com/acme/backend.git"}`, wantStatus: http.StatusBadRequest, wantBody: "name must not be empty"},
		{name: "create repo conflict", method: http.MethodPost, path: "/api/v1/projects/" + projectID.String() + "/repos", body: `{"name":"backend","repository_url":"https://github.com/acme/other.git"}`, wantStatus: http.StatusConflict, wantBody: "REPOSITORY_NAME_CONFLICT"},
		{name: "patch repo invalid repo id", method: http.MethodPatch, path: "/api/v1/projects/" + projectID.String() + "/repos/not-a-uuid", body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "repoId must be a valid UUID"},
		{name: "patch repo invalid payload", method: http.MethodPatch, path: "/api/v1/projects/" + projectID.String() + "/repos/" + repoID.String(), body: `{"name":" "}`, wantStatus: http.StatusBadRequest, wantBody: "name must not be empty"},
		{name: "delete repo invalid project id", method: http.MethodDelete, path: "/api/v1/projects/not-a-uuid/repos/" + repoID.String(), wantStatus: http.StatusBadRequest, wantBody: "projectId must be a valid UUID"},
		{name: "delete repo missing", method: http.MethodDelete, path: "/api/v1/projects/" + projectID.String() + "/repos/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "list repo scopes invalid project id", method: http.MethodGet, path: "/api/v1/projects/not-a-uuid/tickets/" + ticketID.String() + "/repo-scopes", wantStatus: http.StatusBadRequest, wantBody: "projectId must be a valid UUID"},
		{name: "list repo scopes invalid ticket id", method: http.MethodGet, path: "/api/v1/projects/" + projectID.String() + "/tickets/not-a-uuid/repo-scopes", wantStatus: http.StatusBadRequest, wantBody: "ticketId must be a valid UUID"},
		{name: "list repo scopes missing ticket", method: http.MethodGet, path: "/api/v1/projects/" + projectID.String() + "/tickets/" + uuid.NewString() + "/repo-scopes", wantStatus: http.StatusNotFound, wantBody: "resource not found"},
		{name: "create repo scope invalid payload", method: http.MethodPost, path: "/api/v1/projects/" + projectID.String() + "/tickets/" + ticketID.String() + "/repo-scopes", body: `{"repo_id":"bad"}`, wantStatus: http.StatusBadRequest, wantBody: "repo_id must be a valid UUID"},
		{name: "create repo scope conflict", method: http.MethodPost, path: "/api/v1/projects/" + projectID.String() + "/tickets/" + ticketID.String() + "/repo-scopes", body: `{"repo_id":"` + repoID.String() + `"}`, wantStatus: http.StatusConflict, wantBody: "TICKET_REPO_SCOPE_CONFLICT"},
		{name: "patch repo scope invalid scope id", method: http.MethodPatch, path: "/api/v1/projects/" + projectID.String() + "/tickets/" + ticketID.String() + "/repo-scopes/not-a-uuid", body: `{}`, wantStatus: http.StatusBadRequest, wantBody: "scopeId must be a valid UUID"},
		{name: "patch repo scope invalid payload", method: http.MethodPatch, path: "/api/v1/projects/" + projectID.String() + "/tickets/" + ticketID.String() + "/repo-scopes/" + scopeID.String(), body: `{"repo_id":"bad"}`, wantStatus: http.StatusBadRequest, wantBody: "invalid JSON body"},
		{name: "delete repo scope invalid ticket id", method: http.MethodDelete, path: "/api/v1/projects/" + projectID.String() + "/tickets/not-a-uuid/repo-scopes/" + scopeID.String(), wantStatus: http.StatusBadRequest, wantBody: "ticketId must be a valid UUID"},
		{name: "delete repo scope missing", method: http.MethodDelete, path: "/api/v1/projects/" + projectID.String() + "/tickets/" + ticketID.String() + "/repo-scopes/" + uuid.NewString(), wantStatus: http.StatusNotFound, wantBody: "resource not found"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rec := performJSONRequest(t, server, testCase.method, testCase.path, testCase.body)
			if rec.Code != testCase.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", rec.Code, testCase.wantStatus, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), testCase.wantBody) {
				t.Fatalf("body %q does not contain %q", rec.Body.String(), testCase.wantBody)
			}
		})
	}
}

func TestArchiveOrganizationRouteArchivesOrganizationAndProjects(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	orgID := uuid.New()
	service.organizations[orgID] = domain.Organization{
		ID:     orgID,
		Name:   "Acme",
		Slug:   "acme",
		Status: "active",
	}
	projectID := uuid.New()
	service.projects[projectID] = domain.Project{
		ID:             projectID,
		OrganizationID: orgID,
		Name:           "OpenASE",
		Slug:           "openase",
		Status:         "In Progress",
	}

	rec := performJSONRequest(t, server, http.MethodDelete, "/api/v1/orgs/"+orgID.String(), "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected organization archive 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, rec, &payload)
	if payload.Organization.Status != "archived" {
		t.Fatalf("expected archived organization response, got %+v", payload.Organization)
	}
	if project := service.projects[projectID]; project.Status != "Archived" {
		t.Fatalf("expected project to be archived with org, got %+v", project)
	}

	listRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs", "")
	if listRec.Code != http.StatusOK {
		t.Fatalf("expected organization list 200, got %d: %s", listRec.Code, listRec.Body.String())
	}

	var listPayload struct {
		Organizations []organizationResponse `json:"organizations"`
	}
	decodeResponse(t, listRec, &listPayload)
	if len(listPayload.Organizations) != 0 {
		t.Fatalf("expected archived organization to be hidden from list, got %+v", listPayload.Organizations)
	}
}

func TestMachineRoutes(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		newFakeCatalogService(),
		nil,
	)

	orgRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/orgs", `{"name":"Acme Platform","slug":"acme-platform"}`)
	if orgRec.Code != http.StatusCreated {
		t.Fatalf("expected organization create 201, got %d: %s", orgRec.Code, orgRec.Body.String())
	}

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	decodeResponse(t, orgRec, &orgPayload)

	listMachinesRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/"+orgPayload.Organization.ID+"/machines", "")
	if listMachinesRec.Code != http.StatusOK {
		t.Fatalf("expected machine list 200, got %d: %s", listMachinesRec.Code, listMachinesRec.Body.String())
	}

	var listMachinesPayload struct {
		Machines []machineResponse `json:"machines"`
	}
	decodeResponse(t, listMachinesRec, &listMachinesPayload)
	if len(listMachinesPayload.Machines) != 1 || listMachinesPayload.Machines[0].Name != "local" {
		t.Fatalf("expected auto-seeded local machine, got %+v", listMachinesPayload.Machines)
	}

	createMachineRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+orgPayload.Organization.ID+"/machines",
		`{"name":"gpu-01","host":"10.0.1.10","advertised_endpoint":"wss://gpu-01.example.com/openase","ssh_user":"openase","ssh_key_path":"keys/gpu-01.pem","labels":["gpu","a100"],"workspace_root":"/srv/openase/workspaces","env_vars":["CUDA_VISIBLE_DEVICES=0"]}`,
	)
	if createMachineRec.Code != http.StatusCreated {
		t.Fatalf("expected machine create 201, got %d: %s", createMachineRec.Code, createMachineRec.Body.String())
	}

	var createMachinePayload struct {
		Machine machineResponse `json:"machine"`
	}
	decodeResponse(t, createMachineRec, &createMachinePayload)
	if createMachinePayload.Machine.Status != "maintenance" {
		t.Fatalf("expected created remote machine to default to maintenance, got %+v", createMachinePayload.Machine)
	}
	getMachineRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/machines/"+createMachinePayload.Machine.ID, "")
	if getMachineRec.Code != http.StatusOK {
		t.Fatalf("expected machine get 200, got %d: %s", getMachineRec.Code, getMachineRec.Body.String())
	}

	var getMachinePayload struct {
		Machine machineResponse `json:"machine"`
	}
	decodeResponse(t, getMachineRec, &getMachinePayload)
	if getMachinePayload.Machine.ID != createMachinePayload.Machine.ID || getMachinePayload.Machine.Name != "gpu-01" {
		t.Fatalf("unexpected machine get payload: %+v", getMachinePayload.Machine)
	}

	patchMachineRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/machines/"+createMachinePayload.Machine.ID,
		`{"status":"online","description":"A100 worker","advertised_endpoint":"wss://gpu-01.example.com/openase"}`,
	)
	if patchMachineRec.Code != http.StatusOK {
		t.Fatalf("expected machine patch 200, got %d: %s", patchMachineRec.Code, patchMachineRec.Body.String())
	}

	var patchMachinePayload struct {
		Machine machineResponse `json:"machine"`
	}
	decodeResponse(t, patchMachineRec, &patchMachinePayload)
	if patchMachinePayload.Machine.Status != "online" || patchMachinePayload.Machine.Description != "A100 worker" {
		t.Fatalf("unexpected patched machine payload: %+v", patchMachinePayload.Machine)
	}

	testMachineRec := performJSONRequest(t, server, http.MethodPost, "/api/v1/machines/"+createMachinePayload.Machine.ID+"/test", "")
	if testMachineRec.Code != http.StatusOK {
		t.Fatalf("expected machine test 200, got %d: %s", testMachineRec.Code, testMachineRec.Body.String())
	}

	var testMachinePayload struct {
		Machine machineResponse      `json:"machine"`
		Probe   machineProbeResponse `json:"probe"`
	}
	decodeResponse(t, testMachineRec, &testMachinePayload)
	if testMachinePayload.Probe.Transport != "ssh" {
		t.Fatalf("expected ssh probe transport, got %+v", testMachinePayload.Probe)
	}
	if testMachinePayload.Probe.DetectedOS != "linux" ||
		testMachinePayload.Probe.DetectedArch != "amd64" ||
		testMachinePayload.Probe.DetectionStatus != "ok" {
		t.Fatalf("expected probe detection metadata, got %+v", testMachinePayload.Probe)
	}
	if testMachinePayload.Probe.DetectionMessage == "" || testMachinePayload.Machine.DetectionMessage == "" {
		t.Fatalf("expected detection messages in machine test payload, got %+v", testMachinePayload)
	}
	if testMachinePayload.Machine.LastHeartbeatAt == nil {
		t.Fatalf("expected machine test to stamp heartbeat, got %+v", testMachinePayload.Machine)
	}

	refreshMachineHealthRec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/machines/"+createMachinePayload.Machine.ID+"/refresh-health",
		"",
	)
	if refreshMachineHealthRec.Code != http.StatusOK {
		t.Fatalf("expected machine health refresh 200, got %d: %s", refreshMachineHealthRec.Code, refreshMachineHealthRec.Body.String())
	}

	var refreshMachineHealthPayload struct {
		Machine machineResponse `json:"machine"`
	}
	decodeResponse(t, refreshMachineHealthRec, &refreshMachineHealthPayload)
	monitor, ok := refreshMachineHealthPayload.Machine.Resources["monitor"].(map[string]any)
	if !ok {
		t.Fatalf("expected monitor payload after health refresh, got %+v", refreshMachineHealthPayload.Machine.Resources)
	}
	if _, ok := monitor["l4"].(map[string]any); !ok {
		t.Fatalf("expected l4 payload after health refresh, got %+v", monitor)
	}

	resourcesRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/machines/"+createMachinePayload.Machine.ID+"/resources", "")
	if resourcesRec.Code != http.StatusOK {
		t.Fatalf("expected machine resources 200, got %d: %s", resourcesRec.Code, resourcesRec.Body.String())
	}

	var resourcesPayload struct {
		MachineID               string                                    `json:"machine_id"`
		Status                  string                                    `json:"status"`
		LastHeartbeatAt         *string                                   `json:"last_heartbeat_at"`
		Resources               map[string]any                            `json:"resources"`
		EnvironmentProvisioning domain.MachineEnvironmentProvisioningPlan `json:"environment_provisioning"`
	}
	decodeResponse(t, resourcesRec, &resourcesPayload)
	if resourcesPayload.Status != "online" || resourcesPayload.Resources["transport"] != "ssh" {
		t.Fatalf("unexpected machine resources payload: %+v", resourcesPayload)
	}
	if resourcesPayload.EnvironmentProvisioning.Available {
		t.Fatalf("expected empty machine resources to have no provisioning plan, got %+v", resourcesPayload.EnvironmentProvisioning)
	}

	deleteMachineRec := performJSONRequest(t, server, http.MethodDelete, "/api/v1/machines/"+createMachinePayload.Machine.ID, "")
	if deleteMachineRec.Code != http.StatusOK {
		t.Fatalf("expected machine delete 200, got %d: %s", deleteMachineRec.Code, deleteMachineRec.Body.String())
	}
}

func TestCatalogRoutesRejectInvalidInput(t *testing.T) {
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
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

func TestMachineResourcesExposeEnvironmentProvisioningPlan(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	orgID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	machineID := uuid.New()
	service.machines[machineID] = domain.Machine{
		ID:             machineID,
		OrganizationID: orgID,
		Name:           "builder-01",
		Host:           "10.0.1.13",
		Status:         "degraded",
		Resources: map[string]any{
			"last_success": true,
			"monitor": map[string]any{
				"l1": map[string]any{
					"reachable": true,
				},
			},
			"agent_environment": map[string]any{
				"claude_code": map[string]any{
					"installed":   false,
					"auth_status": "unknown",
				},
				"codex": map[string]any{
					"installed":   true,
					"auth_status": "not_logged_in",
				},
			},
			"full_audit": map[string]any{
				"git": map[string]any{
					"installed":  true,
					"user_name":  "OpenASE",
					"user_email": "",
				},
				"gh_cli": map[string]any{
					"installed":   true,
					"auth_status": "not_logged_in",
				},
				"network": map[string]any{
					"github_reachable": true,
					"pypi_reachable":   false,
					"npm_reachable":    false,
				},
			},
		},
	}

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/machines/"+machineID.String()+"/resources", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected machine resources 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		EnvironmentProvisioning domain.MachineEnvironmentProvisioningPlan `json:"environment_provisioning"`
	}
	decodeResponse(t, rec, &payload)
	if !payload.EnvironmentProvisioning.Available || !payload.EnvironmentProvisioning.Runnable {
		t.Fatalf("expected runnable environment provisioning plan, got %+v", payload.EnvironmentProvisioning)
	}
	if len(payload.EnvironmentProvisioning.RequiredSkills) != 4 {
		t.Fatalf("expected 4 required skills, got %+v", payload.EnvironmentProvisioning)
	}
}

func TestMachineRoutesMaskSecretLikeEnvVarsAndPreserveMaskedPatchValues(t *testing.T) {
	service := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		service,
		nil,
	)

	orgID := uuid.New()
	service.organizations[orgID] = domain.Organization{ID: orgID, Name: "Acme", Slug: "acme"}
	machineID := uuid.New()
	advertisedEndpoint := "wss://builder-01.example.com/openase"
	service.machines[machineID] = domain.Machine{
		ID:                 machineID,
		OrganizationID:     orgID,
		Name:               "builder-01",
		Host:               "10.0.1.13",
		Port:               22,
		AdvertisedEndpoint: &advertisedEndpoint,
		Status:             "online",
		EnvVars:            []string{"OPENAI_API_KEY=sk-live-1234", "CUDA_VISIBLE_DEVICES=0"},
	}

	getRec := performJSONRequest(t, server, http.MethodGet, "/api/v1/machines/"+machineID.String(), "")
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected machine get 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), "OPENAI_API_KEY=[redacted]") ||
		strings.Contains(getRec.Body.String(), "sk-live-1234") {
		t.Fatalf("expected masked machine env vars, got %s", getRec.Body.String())
	}

	patchRec := performJSONRequest(
		t,
		server,
		http.MethodPatch,
		"/api/v1/machines/"+machineID.String(),
		`{"env_vars":["OPENAI_API_KEY=[redacted]","CUDA_VISIBLE_DEVICES=1"],"status":"online"}`,
	)
	if patchRec.Code != http.StatusOK {
		t.Fatalf("expected machine patch 200, got %d: %s", patchRec.Code, patchRec.Body.String())
	}
	updated := service.machines[machineID]
	if got := updated.EnvVars[0]; got != "OPENAI_API_KEY=sk-live-1234" {
		t.Fatalf("expected masked patch to preserve secret env var, got %+v", updated.EnvVars)
	}
	if got := updated.EnvVars[1]; got != "CUDA_VISIBLE_DEVICES=1" {
		t.Fatalf("expected non-secret env var update to apply, got %+v", updated.EnvVars)
	}
}

func TestCreateProjectSeedsDefaultTicketStatuses(t *testing.T) {
	client := openTestEntClient(t)
	statusService := newTicketStatusService(client)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		statusService,
		nil,
		catalogservice.New(
			catalogrepo.NewEntRepository(client),
			executable.NewPathResolver(),
			nil,
			catalogservice.WithProjectStatusBootstrapper(catalogservice.ProjectStatusBootstrapperFunc(func(ctx context.Context, projectID uuid.UUID) error {
				_, err := statusService.ResetToDefaultTemplate(ctx, projectID)
				return err
			})),
		),
		nil,
	)

	var orgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs",
		map[string]any{
			"name": "Acme Platform",
			"slug": "acme-platform",
		},
		http.StatusCreated,
		&orgPayload,
	)

	var projectPayload struct {
		Project projectResponse `json:"project"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		fmt.Sprintf("/api/v1/orgs/%s/projects", orgPayload.Organization.ID),
		map[string]any{
			"name":                  "OpenASE",
			"slug":                  "openase",
			"description":           "Main control plane",
			"status":                "In Progress",
			"max_concurrent_agents": 4,
		},
		http.StatusCreated,
		&projectPayload,
	)

	var statusesPayload ticketstatus.ListResult
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf("/api/v1/projects/%s/statuses", projectPayload.Project.ID),
		nil,
		http.StatusOK,
		&statusesPayload,
	)

	names := make([]string, 0, len(statusesPayload.Statuses))
	for _, status := range statusesPayload.Statuses {
		names = append(names, status.Name)
	}
	if strings.Join(names, ",") != "Backlog,Todo,In Progress,In Review,Done,Cancelled" {
		t.Fatalf("unexpected default status order for new project: %v", names)
	}
	if len(statusesPayload.Statuses) == 0 || statusesPayload.Statuses[0].Name != "Backlog" || !statusesPayload.Statuses[0].IsDefault {
		t.Fatalf("expected Backlog to be the default seeded status, got %+v", statusesPayload.Statuses)
	}
}

func TestAppContextAggregatesOrganizationsProjectsProvidersAndAgentCount(t *testing.T) {
	svc := newFakeCatalogService()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		svc,
		nil,
	)

	orgID := uuid.New()
	projectID := uuid.New()
	machineID := uuid.New()
	providerID := uuid.New()
	agentID := uuid.New()

	svc.organizations[orgID] = domain.Organization{
		ID:     orgID,
		Name:   "Acme Platform",
		Slug:   "acme-platform",
		Status: domain.OrganizationStatusActive,
	}
	svc.projects[projectID] = domain.Project{
		ID:             projectID,
		OrganizationID: orgID,
		Name:           "OpenASE",
		Slug:           "openase",
		Status:         domain.ProjectStatusInProgress,
	}
	svc.machines[machineID] = domain.Machine{
		ID:             machineID,
		OrganizationID: orgID,
		Name:           "runner-1",
		Host:           "runner-1.internal",
		Port:           22,
		Status:         domain.MachineStatusOnline,
	}
	svc.providers[providerID] = domain.AgentProvider{
		ID:              providerID,
		OrganizationID:  orgID,
		MachineID:       machineID,
		MachineName:     "runner-1",
		MachineHost:     "runner-1.internal",
		MachineStatus:   domain.MachineStatusOnline,
		Name:            "OpenAI Codex",
		AdapterType:     domain.AgentProviderAdapterTypeCodexAppServer,
		CliCommand:      "/usr/bin/codex",
		ModelName:       "gpt-5.4",
		MaxParallelRuns: domain.DefaultAgentProviderMaxParallelRuns,
	}
	svc.agents[agentID] = domain.Agent{
		ID:         agentID,
		ProviderID: providerID,
		ProjectID:  projectID,
		Name:       "Primary Agent",
	}

	contextPayload := struct {
		Organizations []organizationResponse  `json:"organizations"`
		Projects      []projectResponse       `json:"projects"`
		Providers     []agentProviderResponse `json:"providers"`
		AgentCount    int                     `json:"agent_count"`
	}{}
	executeJSON(
		t,
		server,
		http.MethodGet,
		fmt.Sprintf(
			"/api/v1/app-context?org_id=%s&project_id=%s",
			orgID,
			projectID,
		),
		nil,
		http.StatusOK,
		&contextPayload,
	)
	if len(contextPayload.Organizations) != 1 {
		t.Fatalf("expected 1 organization in app context, got %d", len(contextPayload.Organizations))
	}
	if len(contextPayload.Projects) != 1 || contextPayload.Projects[0].ID != projectID.String() {
		t.Fatalf("expected project in app context, got %+v", contextPayload.Projects)
	}
	if len(contextPayload.Providers) != 1 || contextPayload.Providers[0].ID != providerID.String() {
		t.Fatalf("expected provider in app context, got %+v", contextPayload.Providers)
	}
	if contextPayload.AgentCount != 1 {
		t.Fatalf("expected aggregated agent count to be 1, got %d", contextPayload.AgentCount)
	}
}

func TestTicketRepoScopeRoutesWithEntRepository(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
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
	status, err := client.TicketStatus.Create().
		SetProjectID(project.ID).
		SetName("Todo").
		SetColor("#111111").
		SetPosition(1).
		SetIsDefault(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create status: %v", err)
	}
	ticket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-8").
		SetTitle("bind repos").
		SetStatusID(status.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	backendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create backend repo: %v", err)
	}
	frontendRepo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("frontend").
		SetRepositoryURL("https://github.com/acme/frontend.git").
		SetDefaultBranch("develop").
		Save(ctx)
	if err != nil {
		t.Fatalf("create frontend repo: %v", err)
	}

	var backendCreate struct {
		RepoScope ticketRepoScopeResponse `json:"repo_scope"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes",
		map[string]any{
			"repo_id": backendRepo.ID.String(),
		},
		http.StatusCreated,
		&backendCreate,
	)
	if backendCreate.RepoScope.BranchName != "" {
		t.Fatalf("unexpected backend scope payload: %+v", backendCreate.RepoScope)
	}

	var frontendCreate struct {
		RepoScope ticketRepoScopeResponse `json:"repo_scope"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes",
		map[string]any{
			"repo_id":          frontendRepo.ID.String(),
			"branch_name":      "agent/codex/ASE-8",
			"pull_request_url": "https://github.com/acme/frontend/pull/8",
		},
		http.StatusCreated,
		&frontendCreate,
	)
	if frontendCreate.RepoScope.BranchName != "agent/codex/ASE-8" {
		t.Fatalf("unexpected frontend scope payload: %+v", frontendCreate.RepoScope)
	}

	var scopeList struct {
		RepoScopes []ticketRepoScopeResponse `json:"repo_scopes"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes",
		nil,
		http.StatusOK,
		&scopeList,
	)
	if len(scopeList.RepoScopes) != 2 {
		t.Fatalf("unexpected scope ordering: %+v", scopeList.RepoScopes)
	}

	var backendUpdate struct {
		RepoScope ticketRepoScopeResponse `json:"repo_scope"`
	}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes/"+backendCreate.RepoScope.ID,
		map[string]any{
			"branch_name":      "",
			"pull_request_url": "https://github.com/acme/backend/pull/7",
		},
		http.StatusOK,
		&backendUpdate,
	)
	if backendUpdate.RepoScope.BranchName != "" || backendUpdate.RepoScope.PullRequestURL == nil || *backendUpdate.RepoScope.PullRequestURL != "https://github.com/acme/backend/pull/7" {
		t.Fatalf("unexpected backend scope after update: %+v", backendUpdate.RepoScope)
	}

	executeJSON(
		t,
		server,
		http.MethodDelete,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes/"+backendUpdate.RepoScope.ID,
		nil,
		http.StatusOK,
		nil,
	)

	var finalList struct {
		RepoScopes []ticketRepoScopeResponse `json:"repo_scopes"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes",
		nil,
		http.StatusOK,
		&finalList,
	)
	if len(finalList.RepoScopes) != 1 || finalList.RepoScopes[0].ID != frontendCreate.RepoScope.ID {
		t.Fatalf("unexpected surviving scopes: %+v", finalList.RepoScopes)
	}

	otherProject, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("Other").
		SetSlug("other").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other project: %v", err)
	}
	otherRepo, err := client.ProjectRepo.Create().
		SetProjectID(otherProject.ID).
		SetName("shared").
		SetRepositoryURL("https://github.com/acme/shared.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create other repo: %v", err)
	}

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes",
		`{"repo_id":"`+otherRepo.ID.String()+`"}`,
	)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected cross-project repo scope create to return 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestTicketRepoScopeRoutesPublishTicketRefreshEvents(t *testing.T) {
	client := openTestEntClient(t)
	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		newTicketService(client),
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Acme Repo Scope Events").
		SetSlug("acme-repo-scope-events").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Repo Scope Events").
		SetSlug("openase-repo-scope-events").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	status, err := client.TicketStatus.Create().
		SetProjectID(project.ID).
		SetName("Todo").
		SetColor("#111111").
		SetPosition(1).
		SetIsDefault(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create status: %v", err)
	}
	ticket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-31").
		SetTitle("repo scope refresh").
		SetStatusID(status.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	repo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	stream := subscribeTopicEvents(t, bus, ticketEventsTopic)

	var createResp struct {
		RepoScope ticketRepoScopeResponse `json:"repo_scope"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes",
		map[string]any{"repo_id": repo.ID.String()},
		http.StatusCreated,
		&createResp,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), ticket.ID.String())

	executeJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes/"+createResp.RepoScope.ID,
		map[string]any{"branch_name": "feature/repo-scope-events"},
		http.StatusOK,
		nil,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), ticket.ID.String())

	executeJSON(
		t,
		server,
		http.MethodDelete,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes/"+createResp.RepoScope.ID,
		nil,
		http.StatusOK,
		nil,
	)
	assertStringSet(t, readTicketEventTicketIDs(t, stream, 1), ticket.ID.String())
}

func TestTicketRepoScopeRoutesPublishPullRequestActivityEvents(t *testing.T) {
	client := openTestEntClient(t)
	bus := eventinfra.NewChannelBus()
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		bus,
		newTicketService(client),
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	ctx := context.Background()
	org, err := client.Organization.Create().
		SetName("Acme Repo Scope PR Events").
		SetSlug("acme-repo-scope-pr-events").
		Save(ctx)
	if err != nil {
		t.Fatalf("create org: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE Repo Scope PR Events").
		SetSlug("openase-repo-scope-pr-events").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	status, err := client.TicketStatus.Create().
		SetProjectID(project.ID).
		SetName("Todo").
		SetColor("#111111").
		SetPosition(1).
		SetIsDefault(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("create status: %v", err)
	}
	ticket, err := client.Ticket.Create().
		SetProjectID(project.ID).
		SetIdentifier("ASE-44").
		SetTitle("repo scope pr events").
		SetStatusID(status.ID).
		SetCreatedBy("codex").
		Save(ctx)
	if err != nil {
		t.Fatalf("create ticket: %v", err)
	}
	repo, err := client.ProjectRepo.Create().
		SetProjectID(project.ID).
		SetName("backend").
		SetRepositoryURL("https://github.com/acme/backend.git").
		SetDefaultBranch("main").
		Save(ctx)
	if err != nil {
		t.Fatalf("create repo: %v", err)
	}

	activityStream := subscribeTopicEvents(t, bus, activityStreamTopic)

	var createResp struct {
		RepoScope ticketRepoScopeResponse `json:"repo_scope"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes",
		map[string]any{"repo_id": repo.ID.String()},
		http.StatusCreated,
		&createResp,
	)

	executeJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes/"+createResp.RepoScope.ID,
		map[string]any{"pull_request_url": "https://github.com/acme/backend/pull/44"},
		http.StatusOK,
		nil,
	)

	opened := readEventType(t, activityStream, provider.MustParseEventType(activityevent.TypePROpened.String()), 3)

	var openedPayload struct {
		Event struct {
			Message   string         `json:"message"`
			EventType string         `json:"event_type"`
			Metadata  map[string]any `json:"metadata"`
		} `json:"event"`
	}
	if err := json.Unmarshal(opened.Payload, &openedPayload); err != nil {
		t.Fatalf("decode pr.opened payload: %v", err)
	}
	if openedPayload.Event.EventType != activityevent.TypePROpened.String() || openedPayload.Event.Metadata["pull_request_url"] != "https://github.com/acme/backend/pull/44" {
		t.Fatalf("unexpected pr.opened payload: %+v", openedPayload)
	}

	executeJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes/"+createResp.RepoScope.ID,
		map[string]any{"pull_request_url": "https://github.com/acme/backend/pull/45"},
		http.StatusOK,
		nil,
	)

	select {
	case event := <-activityStream:
		if event.Type != provider.MustParseEventType(activityevent.TypeTicketUpdated.String()) {
			t.Fatalf("expected PR URL update to fall back to ticket.updated only, got %+v", event)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected PR URL update to keep emitting the generic ticket.updated activity")
	}
	select {
	case event := <-activityStream:
		t.Fatalf("expected PR URL update to avoid synthetic PR activity after ticket.updated, got %+v", event)
	case <-time.After(200 * time.Millisecond):
	}

	executeJSON(
		t,
		server,
		http.MethodDelete,
		"/api/v1/projects/"+project.ID.String()+"/tickets/"+ticket.ID.String()+"/repo-scopes/"+createResp.RepoScope.ID,
		nil,
		http.StatusOK,
		nil,
	)

	closed := readEventType(t, activityStream, provider.MustParseEventType(activityevent.TypePRClosed.String()), 2)

	var closedPayload struct {
		Event struct {
			Message   string         `json:"message"`
			EventType string         `json:"event_type"`
			Metadata  map[string]any `json:"metadata"`
		} `json:"event"`
	}
	if err := json.Unmarshal(closed.Payload, &closedPayload); err != nil {
		t.Fatalf("decode pr.closed payload: %v", err)
	}
	if closedPayload.Event.EventType != activityevent.TypePRClosed.String() || closedPayload.Event.Metadata["previous_pr_url"] != "https://github.com/acme/backend/pull/45" {
		t.Fatalf("unexpected pr.closed payload: %+v", closedPayload)
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
		if item.Status == "archived" {
			continue
		}
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
			return domain.Organization{}, domain.ErrOrganizationSlugConflict
		}
	}

	item := domain.Organization{
		ID:                     uuid.New(),
		Name:                   input.Name,
		Slug:                   input.Slug,
		Status:                 "active",
		DefaultAgentProviderID: input.DefaultAgentProviderID,
	}
	f.organizations[item.ID] = item
	localID := uuid.New()
	f.machines[localID] = domain.Machine{
		ID:             localID,
		OrganizationID: item.ID,
		Name:           domain.LocalMachineName,
		Host:           domain.LocalMachineHost,
		Port:           22,
		Status:         "online",
		Description:    "Control-plane local execution host.",
		Resources: map[string]any{
			"transport":    "local",
			"last_success": true,
		},
	}
	for _, template := range domain.BuiltinAgentProviderTemplates() {
		providerID := uuid.New()
		f.providers[providerID] = domain.AgentProvider{
			ID:              providerID,
			OrganizationID:  item.ID,
			MachineID:       localID,
			MachineName:     domain.LocalMachineName,
			MachineHost:     domain.LocalMachineHost,
			MachineStatus:   domain.MachineStatusOnline,
			Name:            template.Name,
			AdapterType:     template.AdapterType,
			CliCommand:      template.Command,
			CliArgs:         append([]string(nil), template.CliArgs...),
			AuthConfig:      map[string]any{},
			ModelName:       template.ModelName,
			MaxParallelRuns: domain.DefaultAgentProviderMaxParallelRuns,
		}
	}

	return item, nil
}

func (f *fakeCatalogService) GetOrganization(_ context.Context, id uuid.UUID) (domain.Organization, error) {
	item, ok := f.organizations[id]
	if !ok || item.Status == "archived" {
		return domain.Organization{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateOrganization(_ context.Context, input domain.UpdateOrganization) (domain.Organization, error) {
	current, ok := f.organizations[input.ID]
	if !ok || current.Status == "archived" {
		return domain.Organization{}, catalogservice.ErrNotFound
	}

	item := domain.Organization{
		ID:                     input.ID,
		Name:                   input.Name,
		Slug:                   input.Slug,
		Status:                 current.Status,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
	}
	f.organizations[input.ID] = item

	return item, nil
}

func (f *fakeCatalogService) ArchiveOrganization(_ context.Context, id uuid.UUID) (domain.Organization, error) {
	item, ok := f.organizations[id]
	if !ok || item.Status == "archived" {
		return domain.Organization{}, catalogservice.ErrNotFound
	}
	item.Status = "archived"
	f.organizations[id] = item
	for projectID, project := range f.projects {
		if project.OrganizationID != id {
			continue
		}
		project.Status = "Archived"
		f.projects[projectID] = project
	}
	return item, nil
}

func (f *fakeCatalogService) ListMachines(_ context.Context, organizationID uuid.UUID) ([]domain.Machine, error) {
	item, ok := f.organizations[organizationID]
	if !ok || item.Status == "Archived" {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.Machine, 0)
	for _, item := range f.machines {
		if item.OrganizationID == organizationID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	return items, nil
}

func (f *fakeCatalogService) CreateMachine(_ context.Context, input domain.CreateMachine) (domain.Machine, error) {
	org, ok := f.organizations[input.OrganizationID]
	if !ok || org.Status == "archived" {
		return domain.Machine{}, catalogservice.ErrNotFound
	}
	for _, item := range f.machines {
		if item.OrganizationID == input.OrganizationID && item.Name == input.Name {
			return domain.Machine{}, domain.ErrMachineNameConflict
		}
	}

	machine := domain.Machine{
		ID:             uuid.New(),
		OrganizationID: input.OrganizationID,
		Name:           input.Name,
		Host:           input.Host,
		Port:           input.Port,
		SSHUser:        input.SSHUser,
		SSHKeyPath:     input.SSHKeyPath,
		Description:    input.Description,
		Labels:         append([]string(nil), input.Labels...),
		Status:         input.Status,
		WorkspaceRoot:  input.WorkspaceRoot,
		AgentCLIPath:   input.AgentCLIPath,
		EnvVars:        append([]string(nil), input.EnvVars...),
		Resources:      map[string]any{},
	}
	f.machines[machine.ID] = machine

	return machine, nil
}

func (f *fakeCatalogService) GetMachine(_ context.Context, id uuid.UUID) (domain.Machine, error) {
	item, ok := f.machines[id]
	if !ok {
		return domain.Machine{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateMachine(_ context.Context, input domain.UpdateMachine) (domain.Machine, error) {
	current, ok := f.machines[input.ID]
	if !ok {
		return domain.Machine{}, catalogservice.ErrNotFound
	}
	for _, item := range f.machines {
		if item.ID != input.ID && item.OrganizationID == input.OrganizationID && item.Name == input.Name {
			return domain.Machine{}, domain.ErrMachineNameConflict
		}
	}

	current.Name = input.Name
	current.Host = input.Host
	current.Port = input.Port
	current.SSHUser = input.SSHUser
	current.SSHKeyPath = input.SSHKeyPath
	current.Description = input.Description
	current.Labels = append([]string(nil), input.Labels...)
	current.Status = input.Status
	current.WorkspaceRoot = input.WorkspaceRoot
	current.AgentCLIPath = input.AgentCLIPath
	current.EnvVars = append([]string(nil), input.EnvVars...)
	f.machines[input.ID] = current

	return current, nil
}

func (f *fakeCatalogService) DeleteMachine(_ context.Context, id uuid.UUID) (domain.Machine, error) {
	item, ok := f.machines[id]
	if !ok {
		return domain.Machine{}, catalogservice.ErrNotFound
	}
	if item.Name == domain.LocalMachineName {
		return domain.Machine{}, catalogservice.ErrInvalidInput
	}

	delete(f.machines, id)
	return item, nil
}

func (f *fakeCatalogService) TestMachineConnection(_ context.Context, id uuid.UUID) (domain.Machine, domain.MachineProbe, error) {
	item, ok := f.machines[id]
	if !ok {
		return domain.Machine{}, domain.MachineProbe{}, catalogservice.ErrNotFound
	}

	checkedAt := time.Now().UTC()
	transport := map[bool]string{true: "local", false: "ssh"}[item.Host == domain.LocalMachineHost]
	probe := domain.MachineProbe{
		CheckedAt:       checkedAt,
		Transport:       transport,
		Output:          "probe-ok",
		DetectedOS:      domain.MachineDetectedOSLinux,
		DetectedArch:    domain.MachineDetectedArchAMD64,
		DetectionStatus: domain.MachineDetectionStatusOK,
		Resources: map[string]any{
			"transport":    transport,
			"checked_at":   checkedAt.Format(time.RFC3339),
			"last_success": true,
		},
	}
	item.Status = "online"
	item.DetectedOS = probe.DetectedOS
	item.DetectedArch = probe.DetectedArch
	item.DetectionStatus = probe.DetectionStatus
	item.LastHeartbeatAt = &checkedAt
	item.Resources = cloneMap(probe.Resources)
	f.machines[id] = item

	return item, probe, nil
}

func (f *fakeCatalogService) RefreshMachineHealth(_ context.Context, id uuid.UUID) (domain.Machine, error) {
	item, ok := f.machines[id]
	if !ok {
		return domain.Machine{}, catalogservice.ErrNotFound
	}

	checkedAt := time.Now().UTC()
	item.Status = domain.MachineStatusOnline
	item.LastHeartbeatAt = &checkedAt
	item.Resources = map[string]any{
		"transport":    "ssh",
		"checked_at":   checkedAt.Format(time.RFC3339),
		"last_success": true,
		"monitor": map[string]any{
			"l1": map[string]any{
				"checked_at": checkedAt.Format(time.RFC3339),
				"reachable":  true,
				"transport":  "ssh",
			},
			"l4": map[string]any{
				"checked_at":         checkedAt.Format(time.RFC3339),
				"agent_dispatchable": true,
				"codex": map[string]any{
					"installed":   true,
					"version":     "0.117.0",
					"auth_status": "logged_in",
					"auth_mode":   "login",
					"ready":       true,
				},
			},
			"l5": map[string]any{
				"checked_at": checkedAt.Format(time.RFC3339),
				"git": map[string]any{
					"installed":  true,
					"user_name":  "Codex",
					"user_email": "codex@openai.com",
				},
			},
		},
		"agent_dispatchable": true,
		"agent_environment": map[string]any{
			"codex": map[string]any{
				"installed":   true,
				"version":     "0.117.0",
				"auth_status": "logged_in",
				"auth_mode":   "login",
				"ready":       true,
			},
		},
	}
	f.machines[id] = item

	return item, nil
}

func (f *fakeCatalogService) ListProjects(_ context.Context, organizationID uuid.UUID) ([]domain.Project, error) {
	item, ok := f.organizations[organizationID]
	if !ok || item.Status == "Archived" {
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
	org, ok := f.organizations[input.OrganizationID]
	if !ok || org.Status == "archived" {
		return domain.Project{}, catalogservice.ErrNotFound
	}

	for _, item := range f.projects {
		if item.OrganizationID == input.OrganizationID && item.Slug == input.Slug {
			return domain.Project{}, domain.ErrProjectSlugConflict
		}
	}

	project := domain.Project{
		ID:                     uuid.New(),
		OrganizationID:         input.OrganizationID,
		Name:                   input.Name,
		Slug:                   input.Slug,
		Description:            input.Description,
		Status:                 input.Status,
		DefaultAgentProviderID: input.DefaultAgentProviderID,
		AccessibleMachineIDs:   append([]uuid.UUID(nil), input.AccessibleMachineIDs...),
		MaxConcurrentAgents:    input.MaxConcurrentAgents,
		AgentRunSummaryPrompt:  input.AgentRunSummaryPrompt,
	}
	f.projects[project.ID] = project

	return project, nil
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
		DefaultAgentProviderID: input.DefaultAgentProviderID,
		AccessibleMachineIDs:   append([]uuid.UUID(nil), input.AccessibleMachineIDs...),
		MaxConcurrentAgents:    input.MaxConcurrentAgents,
		AgentRunSummaryPrompt:  input.AgentRunSummaryPrompt,
	}
	f.projects[input.ID] = item

	return item, nil
}

func (f *fakeCatalogService) ArchiveProject(_ context.Context, id uuid.UUID) (domain.Project, error) {
	item, ok := f.projects[id]
	if !ok {
		return domain.Project{}, catalogservice.ErrNotFound
	}

	item.Status = "Archived"
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
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	return items, nil
}

func (f *fakeCatalogService) CreateProjectRepo(_ context.Context, input domain.CreateProjectRepo) (domain.ProjectRepo, error) {
	if _, ok := f.projects[input.ProjectID]; !ok {
		return domain.ProjectRepo{}, catalogservice.ErrNotFound
	}

	for _, item := range f.projectRepos {
		if item.ProjectID == input.ProjectID && item.Name == input.Name {
			return domain.ProjectRepo{}, domain.ErrProjectRepoNameConflict
		}
	}

	item := domain.ProjectRepo{
		ID:               uuid.New(),
		ProjectID:        input.ProjectID,
		Name:             input.Name,
		RepositoryURL:    input.RepositoryURL,
		DefaultBranch:    input.DefaultBranch,
		WorkspaceDirname: input.WorkspaceDirname,
		Labels:           append([]string(nil), input.Labels...),
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

	item.Name = input.Name
	item.RepositoryURL = input.RepositoryURL
	item.DefaultBranch = input.DefaultBranch
	item.WorkspaceDirname = input.WorkspaceDirname
	item.Labels = append([]string(nil), input.Labels...)
	f.projectRepos[item.ID] = item

	return item, nil
}

func (f *fakeCatalogService) DeleteProjectRepo(_ context.Context, projectID uuid.UUID, id uuid.UUID) (domain.ProjectRepo, error) {
	item, ok := f.projectRepos[id]
	if !ok || item.ProjectID != projectID {
		return domain.ProjectRepo{}, catalogservice.ErrNotFound
	}

	delete(f.projectRepos, id)
	return item, nil
}

func (f *fakeCatalogService) ListTicketRepoScopes(_ context.Context, projectID uuid.UUID, ticketID uuid.UUID) ([]domain.TicketRepoScope, error) {
	ticket, ok := f.tickets[ticketID]
	if !ok || ticket.ProjectID != projectID {
		return nil, catalogservice.ErrNotFound
	}

	items := make([]domain.TicketRepoScope, 0)
	for _, item := range f.ticketScopes {
		if item.TicketID == ticketID {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID.String() < items[j].ID.String() })

	return items, nil
}

func (f *fakeCatalogService) CreateTicketRepoScope(_ context.Context, input domain.CreateTicketRepoScope) (domain.TicketRepoScope, error) {
	ticket, ok := f.tickets[input.TicketID]
	if !ok || ticket.ProjectID != input.ProjectID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}
	repo, ok := f.projectRepos[input.RepoID]
	if !ok || repo.ProjectID != input.ProjectID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}
	for _, item := range f.ticketScopes {
		if item.TicketID == input.TicketID && item.RepoID == input.RepoID {
			return domain.TicketRepoScope{}, domain.ErrTicketRepoScopeConflict
		}
	}

	branchName := repo.DefaultBranch
	if input.BranchName != nil {
		branchName = *input.BranchName
	}
	item := domain.TicketRepoScope{
		ID:             uuid.New(),
		TicketID:       input.TicketID,
		RepoID:         input.RepoID,
		BranchName:     branchName,
		PullRequestURL: input.PullRequestURL,
	}
	f.ticketScopes[item.ID] = item

	return item, nil
}

func (f *fakeCatalogService) GetTicketRepoScope(_ context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error) {
	ticket, ok := f.tickets[ticketID]
	if !ok || ticket.ProjectID != projectID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}
	item, ok := f.ticketScopes[id]
	if !ok || item.TicketID != ticketID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}

	return item, nil
}

func (f *fakeCatalogService) UpdateTicketRepoScope(_ context.Context, input domain.UpdateTicketRepoScope) (domain.TicketRepoScope, error) {
	ticket, ok := f.tickets[input.TicketID]
	if !ok || ticket.ProjectID != input.ProjectID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}
	item, ok := f.ticketScopes[input.ID]
	if !ok || item.TicketID != input.TicketID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}

	if input.BranchName != nil {
		item.BranchName = *input.BranchName
	}
	item.PullRequestURL = input.PullRequestURL
	f.ticketScopes[item.ID] = item

	return item, nil
}

func (f *fakeCatalogService) DeleteTicketRepoScope(_ context.Context, projectID uuid.UUID, ticketID uuid.UUID, id uuid.UUID) (domain.TicketRepoScope, error) {
	ticket, ok := f.tickets[ticketID]
	if !ok || ticket.ProjectID != projectID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}
	item, ok := f.ticketScopes[id]
	if !ok || item.TicketID != ticketID {
		return domain.TicketRepoScope{}, catalogservice.ErrNotFound
	}

	delete(f.ticketScopes, id)
	return item, nil
}

func (f *fakeCatalogService) GetWorkspaceDashboardSummary(context.Context) (domain.WorkspaceDashboardSummary, error) {
	return domain.WorkspaceDashboardSummary{}, nil
}

func (f *fakeCatalogService) GetOrganizationDashboardSummary(context.Context, uuid.UUID) (domain.OrganizationDashboardSummary, error) {
	return domain.OrganizationDashboardSummary{}, nil
}

func (f *fakeCatalogService) GetOrganizationTokenUsage(context.Context, domain.GetOrganizationTokenUsage) (domain.OrganizationTokenUsageReport, error) {
	return domain.OrganizationTokenUsageReport{}, nil
}

func (f *fakeCatalogService) GetProjectTokenUsage(context.Context, domain.GetProjectTokenUsage) (domain.ProjectTokenUsageReport, error) {
	return domain.ProjectTokenUsageReport{}, nil
}

func (f *fakeCatalogService) hasProjectRepos(projectID uuid.UUID) bool {
	for _, item := range f.projectRepos {
		if item.ProjectID == projectID {
			return true
		}
	}

	return false
}

func (f *fakeCatalogService) hasTicketRepoScopes(ticketID uuid.UUID) bool {
	for _, item := range f.ticketScopes {
		if item.TicketID == ticketID {
			return true
		}
	}

	return false
}

func TestProjectRepoLifecycleWithEntRepository(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
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
		},
		http.StatusCreated,
		&frontendCreate,
	)

	var backendUpdate struct {
		Repo projectRepoResponse `json:"repo"`
	}
	executeJSON(
		t,
		server,
		http.MethodPatch,
		"/api/v1/projects/"+project.ID.String()+"/repos/"+backendCreate.Repo.ID,
		map[string]any{},
		http.StatusOK,
		&backendUpdate,
	)
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
	if len(repoList.Repos) != 1 || repoList.Repos[0].Name != "frontend" {
		t.Fatalf("unexpected surviving repos: %+v", repoList.Repos)
	}
}

func TestOrganizationCreateSeedsLocalMachineWithEntRepository(t *testing.T) {
	client := openTestEntClient(t)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		catalogservice.New(catalogrepo.NewEntRepository(client), executable.NewPathResolver(), nil),
		nil,
	)

	var createOrgPayload struct {
		Organization organizationResponse `json:"organization"`
	}
	executeJSON(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs",
		map[string]any{"name": "Acme", "slug": "acme"},
		http.StatusCreated,
		&createOrgPayload,
	)

	var machinesPayload struct {
		Machines []machineResponse `json:"machines"`
	}
	executeJSON(
		t,
		server,
		http.MethodGet,
		"/api/v1/orgs/"+createOrgPayload.Organization.ID+"/machines",
		nil,
		http.StatusOK,
		&machinesPayload,
	)
	if len(machinesPayload.Machines) != 1 {
		t.Fatalf("expected one seeded machine, got %+v", machinesPayload.Machines)
	}
	if machinesPayload.Machines[0].Name != "local" || machinesPayload.Machines[0].Status != "online" {
		t.Fatalf("unexpected seeded local machine: %+v", machinesPayload.Machines[0])
	}
}

func createGitRepository(t *testing.T) (string, string) {
	t.Helper()

	repoPath := filepath.Join(t.TempDir(), "remote")
	if err := os.MkdirAll(repoPath, 0o750); err != nil {
		t.Fatalf("create git repo dir: %v", err)
	}

	repository, err := git.PlainInit(repoPath, false)
	if err != nil {
		t.Fatalf("git init: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte("mirror test\n"), 0o600); err != nil {
		t.Fatalf("write git file: %v", err)
	}

	worktree, err := repository.Worktree()
	if err != nil {
		t.Fatalf("git worktree: %v", err)
	}
	if _, err := worktree.Add("README.md"); err != nil {
		t.Fatalf("git add: %v", err)
	}
	hash, err := worktree.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Codex",
			Email: "codex@openai.com",
			When:  time.Date(2026, 3, 29, 14, 0, 0, 0, time.UTC),
		},
	})
	if err != nil {
		t.Fatalf("git commit: %v", err)
	}
	if _, err := repository.CreateRemote(&gitconfig.RemoteConfig{
		Name: "origin",
		URLs: []string{repoPath},
	}); err != nil {
		t.Fatalf("git create remote: %v", err)
	}

	return repoPath, hash.String()
}
