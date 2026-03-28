package githubauth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/githubauth"
	"github.com/google/uuid"
)

func TestResolveProjectCredentialPrefersProjectOverride(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	repo := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:      projectID,
			OrganizationID: orgID,
		},
	}
	service, err := New(repo, nil, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	orgCredential, err := service.SealToken("gho_org_token", domain.SourceGHCLIImport)
	if err != nil {
		t.Fatalf("SealToken(org) error = %v", err)
	}
	projectCredential, err := service.SealToken("ghp_project_override", domain.SourceManualPaste)
	if err != nil {
		t.Fatalf("SealToken(project) error = %v", err)
	}
	repo.context.OrganizationCredential = &orgCredential
	repo.context.OrganizationProbe = &domain.TokenProbe{State: domain.ProbeStateValid, Configured: true, Valid: true, RepoAccess: domain.RepoAccessGranted}
	repo.context.ProjectCredential = &projectCredential
	repo.context.ProjectProbe = &domain.TokenProbe{State: domain.ProbeStateConfigured, Configured: true, Valid: false, RepoAccess: domain.RepoAccessNotChecked}

	resolved, err := service.ResolveProjectCredential(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ResolveProjectCredential() error = %v", err)
	}
	if resolved.Scope != domain.ScopeProject || resolved.Source != domain.SourceManualPaste {
		t.Fatalf("ResolveProjectCredential() scope/source = %+v", resolved)
	}
	if resolved.Token != "ghp_project_override" {
		t.Fatalf("ResolveProjectCredential() token = %q", resolved.Token)
	}
	if resolved.TokenPreview == "ghp_project_override" || resolved.TokenPreview == "" {
		t.Fatalf("ResolveProjectCredential() preview = %q", resolved.TokenPreview)
	}
	if resolved.Probe.State != domain.ProbeStateConfigured {
		t.Fatalf("ResolveProjectCredential() probe = %+v", resolved.Probe)
	}
}

func TestReadProjectSecurityReturnsMissingProbeWithoutCredential(t *testing.T) {
	projectID := uuid.New()
	service, err := New(&stubRepository{
		context: domain.ProjectContext{
			ProjectID:      projectID,
			OrganizationID: uuid.New(),
		},
	}, nil, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	security, err := service.ReadProjectSecurity(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ReadProjectSecurity() error = %v", err)
	}
	if security.Scope != "" || security.Source != "" || security.TokenPreview != "" {
		t.Fatalf("ReadProjectSecurity() identity = %+v", security)
	}
	if security.Probe.State != domain.ProbeStateMissing || security.Probe.Configured {
		t.Fatalf("ReadProjectSecurity() probe = %+v", security.Probe)
	}
}

func TestProbeResolvedCredentialPersistsValidProbe(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	repository := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:      projectID,
			OrganizationID: orgID,
		},
	}
	service, err := New(repository, http.DefaultClient, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	sealed, err := service.SealToken("ghu_valid_token", domain.SourceDeviceFlow)
	if err != nil {
		t.Fatalf("SealToken() error = %v", err)
	}
	repository.context.OrganizationCredential = &sealed

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.Header().Set("X-OAuth-Scopes", "repo,read:org")
			w.WriteHeader(http.StatusOK)
		case "/repos/grandcx/openase":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	service.baseURL = server.URL
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	probe, err := service.ProbeResolvedCredential(context.Background(), projectID, "https://github.com/GrandCX/openase.git")
	if err != nil {
		t.Fatalf("ProbeResolvedCredential() error = %v", err)
	}
	if probe.State != domain.ProbeStateValid || !probe.Valid || probe.RepoAccess != domain.RepoAccessGranted {
		t.Fatalf("ProbeResolvedCredential() = %+v", probe)
	}
	if repository.savedOrganizationProbe == nil || repository.savedOrganizationProbe.State != domain.ProbeStateValid {
		t.Fatalf("expected saved organization probe, got %+v", repository.savedOrganizationProbe)
	}
}

func TestProbeResolvedCredentialMarksInsufficientPermissions(t *testing.T) {
	projectID := uuid.New()
	repository := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:      projectID,
			OrganizationID: uuid.New(),
		},
	}
	service, err := New(repository, http.DefaultClient, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	sealed, err := service.SealToken("ghu_limited_token", domain.SourceDeviceFlow)
	if err != nil {
		t.Fatalf("SealToken() error = %v", err)
	}
	repository.context.OrganizationCredential = &sealed

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.WriteHeader(http.StatusOK)
		case "/repos/grandcx/private-repo":
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	service.baseURL = server.URL

	probe, err := service.ProbeResolvedCredential(context.Background(), projectID, "https://github.com/GrandCX/private-repo.git")
	if err != nil {
		t.Fatalf("ProbeResolvedCredential() error = %v", err)
	}
	if probe.State != domain.ProbeStateInsufficientPermissions || probe.Valid || probe.RepoAccess != domain.RepoAccessDenied {
		t.Fatalf("ProbeResolvedCredential() = %+v", probe)
	}
}

type stubRepository struct {
	context                domain.ProjectContext
	savedOrganizationProbe *domain.TokenProbe
	savedProjectProbe      *domain.TokenProbe
}

func (s *stubRepository) GetProjectContext(context.Context, uuid.UUID) (domain.ProjectContext, error) {
	return s.context, nil
}

func (s *stubRepository) SaveOrganizationProbe(_ context.Context, _ uuid.UUID, probe domain.TokenProbe) error {
	cloned := probe
	s.savedOrganizationProbe = &cloned
	return nil
}

func (s *stubRepository) SaveProjectProbe(_ context.Context, _ uuid.UUID, probe domain.TokenProbe) error {
	cloned := probe
	s.savedProjectProbe = &cloned
	return nil
}
