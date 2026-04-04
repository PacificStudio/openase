package httpapi

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	chatservice "github.com/BetterAndBetterII/openase/internal/chat"
	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	humanauthdomain "github.com/BetterAndBetterII/openase/internal/domain/humanauth"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	humanauthrepo "github.com/BetterAndBetterII/openase/internal/repo/humanauth"
	catalogservice "github.com/BetterAndBetterII/openase/internal/service/catalog"
	humanauthservice "github.com/BetterAndBetterII/openase/internal/service/humanauth"
	workflowservice "github.com/BetterAndBetterII/openase/internal/workflow"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

func TestProtectedRoutesRequireDefinedAuthorizationRules(t *testing.T) {
	t.Parallel()

	echoServer := echo.New()
	server := &Server{
		echo: echoServer,
		catalog: catalogservice.Services{
			OrganizationService: authorizationRouteCoverageOrganizationService{},
		},
	}

	api := echoServer.Group("/api/v1")
	protected := api.Group("")
	routeRegistrar{server: server, api: api}.registerProtectedAPIRoutes(protected)

	missing := make([]string, 0)
	for _, route := range echoServer.Routes() {
		if !strings.HasPrefix(route.Path, "/api/v1/") {
			continue
		}
		if route.Method == http.MethodOptions || route.Method == http.MethodHead {
			continue
		}
		if _, ok := humanRouteAuthorizationRuleFor(route.Path, route.Method); ok {
			continue
		}
		missing = append(missing, route.Method+" "+route.Path)
	}

	slices.Sort(missing)
	if len(missing) > 0 {
		t.Fatalf("protected routes missing authorization rules:\n%s", strings.Join(missing, "\n"))
	}
}

func TestRequiredScopeAndPermissionResolvesSkillRefinementSessionDeleteScope(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	client := openTestEntClient(t)
	ctx := context.Background()

	org, err := client.Organization.Create().
		SetName("Better And Better").
		SetSlug("better-and-better").
		Save(ctx)
	if err != nil {
		t.Fatalf("create organization: %v", err)
	}
	project, err := client.Project.Create().
		SetOrganizationID(org.ID).
		SetName("OpenASE").
		SetSlug("openase").
		Save(ctx)
	if err != nil {
		t.Fatalf("create project: %v", err)
	}
	skill, err := client.Skill.Create().
		SetProjectID(project.ID).
		SetName("deploy-openase").
		SetDescription("Refine deploy skill").
		SetIsBuiltin(false).
		SetIsEnabled(true).
		SetCreatedBy("user:test").
		Save(ctx)
	if err != nil {
		t.Fatalf("create skill: %v", err)
	}

	providerID := uuid.New()
	refinementSvc := chatservice.NewSkillRefinementService(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		authorizationTestSkillRefinementRuntime{},
		authorizationTestCatalog{
			project: catalogdomain.Project{
				ID:                     project.ID,
				OrganizationID:         org.ID,
				Name:                   project.Name,
				DefaultAgentProviderID: &providerID,
			},
			providers: []catalogdomain.AgentProvider{
				{
					ID:             providerID,
					OrganizationID: org.ID,
					Name:           "Codex",
					AdapterType:    catalogdomain.AgentProviderAdapterTypeCodexAppServer,
					Available:      true,
				},
			},
		},
		authorizationTestWorkflow{
			skill: workflowservice.SkillDetail{
				Skill: workflowservice.Skill{
					ID:             skill.ID,
					Name:           skill.Name,
					Description:    skill.Description,
					Path:           ".openase/skills/deploy-openase",
					CurrentVersion: 1,
					IsEnabled:      skill.IsEnabled,
				},
			},
		},
	)

	userUUID := uuid.New()
	userID := chatservice.UserID("user:" + userUUID.String())
	stream, err := refinementSvc.Start(ctx, userID, chatservice.SkillRefinementInput{
		ProjectID:  project.ID,
		SkillID:    skill.ID,
		ProviderID: &providerID,
		Message:    "Refine the skill.",
		DraftFiles: []workflowservice.SkillBundleFileInput{
			{
				Path:         "SKILL.md",
				Content:      []byte("---\nname: deploy-openase\ndescription: Refine deploy skill\n---\n\n# Deploy\n\nRefine this skill.\n"),
				MediaType:    "text/markdown; charset=utf-8",
				IsExecutable: false,
			},
		},
	})
	if err != nil {
		t.Fatalf("start skill refinement: %v", err)
	}
	firstEvent := <-stream.Events
	if firstEvent.Event != "session" {
		t.Fatalf("first event = %+v, want session", firstEvent)
	}
	payload, ok := firstEvent.Payload.(chatservice.SkillRefinementSessionPayload)
	if !ok {
		t.Fatalf("session payload type = %T", firstEvent.Payload)
	}

	repository := humanauthrepo.NewEntRepository(client)
	authorizer := humanauthservice.NewAuthorizer(repository)
	server := NewServer(
		config.ServerConfig{Port: 40023},
		config.GitHubConfig{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		eventinfra.NewChannelBus(),
		nil,
		nil,
		nil,
		nil,
		nil,
		WithHumanAuthConfig(config.AuthConfig{Mode: config.AuthModeOIDC}),
		WithHumanAuthService(nil, authorizer),
		WithSkillRefinementService(refinementSvc),
	)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/skills/refinement-runs/"+payload.SessionID, nil)
	rec := httptest.NewRecorder()
	echoCtx := server.echo.NewContext(req, rec)
	echoCtx.SetPath("/api/v1/skills/refinement-runs/:sessionId")
	echoCtx.SetParamNames("sessionId")
	echoCtx.SetParamValues(payload.SessionID)

	principal := humanauthdomain.AuthenticatedPrincipal{
		User: humanauthdomain.User{ID: userUUID},
	}
	scope, permission, checkRequired, err := server.requiredScopeAndPermission(echoCtx, principal)
	if err != nil {
		t.Fatalf("requiredScopeAndPermission() error = %v", err)
	}
	if !checkRequired {
		t.Fatal("requiredScopeAndPermission() checkRequired = false, want true")
	}
	if scope.Kind != humanauthdomain.ScopeKindProject || scope.ID != project.ID.String() {
		t.Fatalf("requiredScopeAndPermission() scope = %+v", scope)
	}
	if permission != humanauthdomain.PermissionSkillManage {
		t.Fatalf("requiredScopeAndPermission() permission = %q, want %q", permission, humanauthdomain.PermissionSkillManage)
	}
}

type authorizationTestSkillRefinementRuntime struct{}

func (authorizationTestSkillRefinementRuntime) Supports(catalogdomain.AgentProvider) bool {
	return true
}

func (authorizationTestSkillRefinementRuntime) StartTurn(context.Context, chatservice.RuntimeTurnInput) (chatservice.TurnStream, error) {
	events := make(chan chatservice.StreamEvent)
	close(events)
	return chatservice.TurnStream{Events: events}, nil
}

func (authorizationTestSkillRefinementRuntime) CloseSession(chatservice.SessionID) bool {
	return true
}

func (authorizationTestSkillRefinementRuntime) SessionAnchor(chatservice.SessionID) chatservice.RuntimeSessionAnchor {
	return chatservice.RuntimeSessionAnchor{}
}

type authorizationTestCatalog struct {
	project   catalogdomain.Project
	providers []catalogdomain.AgentProvider
}

func (c authorizationTestCatalog) GetProject(context.Context, uuid.UUID) (catalogdomain.Project, error) {
	return c.project, nil
}

func (authorizationTestCatalog) ListActivityEvents(context.Context, catalogdomain.ListActivityEvents) ([]catalogdomain.ActivityEvent, error) {
	return nil, nil
}

func (authorizationTestCatalog) ListProjectRepos(context.Context, uuid.UUID) ([]catalogdomain.ProjectRepo, error) {
	return nil, nil
}

func (authorizationTestCatalog) ListTicketRepoScopes(context.Context, uuid.UUID, uuid.UUID) ([]catalogdomain.TicketRepoScope, error) {
	return nil, nil
}

func (c authorizationTestCatalog) ListAgentProviders(context.Context, uuid.UUID) ([]catalogdomain.AgentProvider, error) {
	return c.providers, nil
}

func (c authorizationTestCatalog) GetAgentProvider(context.Context, uuid.UUID) (catalogdomain.AgentProvider, error) {
	return c.providers[0], nil
}

type authorizationTestWorkflow struct {
	skill workflowservice.SkillDetail
}

func (authorizationTestWorkflow) Get(context.Context, uuid.UUID) (workflowservice.WorkflowDetail, error) {
	return workflowservice.WorkflowDetail{}, nil
}

func (authorizationTestWorkflow) List(context.Context, uuid.UUID) ([]workflowservice.Workflow, error) {
	return nil, nil
}

type authorizationRouteCoverageOrganizationService struct{}

func (authorizationRouteCoverageOrganizationService) ListOrganizations(context.Context) ([]catalogdomain.Organization, error) {
	return nil, nil
}

func (authorizationRouteCoverageOrganizationService) CreateOrganization(context.Context, catalogdomain.CreateOrganization) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}

func (authorizationRouteCoverageOrganizationService) GetOrganization(context.Context, uuid.UUID) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}

func (authorizationRouteCoverageOrganizationService) UpdateOrganization(context.Context, catalogdomain.UpdateOrganization) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}

func (authorizationRouteCoverageOrganizationService) ArchiveOrganization(context.Context, uuid.UUID) (catalogdomain.Organization, error) {
	return catalogdomain.Organization{}, nil
}

func (w authorizationTestWorkflow) GetSkill(context.Context, uuid.UUID) (workflowservice.SkillDetail, error) {
	return w.skill, nil
}
