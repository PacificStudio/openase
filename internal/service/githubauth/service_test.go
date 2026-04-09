package githubauth

import (
	"context"
	"errors"
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
	repository := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:            projectID,
			OrganizationID:       orgID,
			ProjectRepositoryURL: "https://github.com/PacificStudio/openase.git",
		},
	}
	service, err := New(repository, nil, "postgres://openase:test@localhost/openase")
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
	repository.context.OrganizationCredential = &orgCredential
	repository.context.OrganizationProbe = &domain.TokenProbe{
		State:      domain.ProbeStateValid,
		Configured: true,
		Valid:      true,
		RepoAccess: domain.RepoAccessGranted,
	}
	repository.context.ProjectCredential = &projectCredential
	repository.context.ProjectProbe = &domain.TokenProbe{
		State:      domain.ProbeStateConfigured,
		Configured: true,
		Valid:      false,
		RepoAccess: domain.RepoAccessNotChecked,
	}

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

func TestReadProjectSecurityReturnsMissingSlotsWithoutCredential(t *testing.T) {
	projectID := uuid.New()
	service, err := New(&stubRepository{
		context: domain.ProjectContext{
			ProjectID:            projectID,
			OrganizationID:       uuid.New(),
			ProjectRepositoryURL: "https://github.com/PacificStudio/openase.git",
		},
	}, nil, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	security, err := service.ReadProjectSecurity(context.Background(), projectID)
	if err != nil {
		t.Fatalf("ReadProjectSecurity() error = %v", err)
	}
	if security.Effective.Configured || security.Effective.Scope != "" {
		t.Fatalf("ReadProjectSecurity().Effective = %+v", security.Effective)
	}
	if security.Organization.Scope != domain.ScopeOrganization || security.Organization.Configured {
		t.Fatalf("ReadProjectSecurity().Organization = %+v", security.Organization)
	}
	if security.ProjectOverride.Scope != domain.ScopeProject || security.ProjectOverride.Configured {
		t.Fatalf("ReadProjectSecurity().ProjectOverride = %+v", security.ProjectOverride)
	}
	if security.Effective.Probe.State != domain.ProbeStateMissing {
		t.Fatalf("ReadProjectSecurity().Effective.Probe = %+v", security.Effective.Probe)
	}
}

func TestSaveManualCredentialPersistsOrganizationProbeLifecycle(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	repository := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:            projectID,
			OrganizationID:       orgID,
			ProjectRepositoryURL: "https://github.com/PacificStudio/openase.git",
		},
	}
	service, err := New(repository, http.DefaultClient, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.Header().Set("X-OAuth-Scopes", "repo,read:org")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"login":"octocat"}`))
		case "/repos/pacificstudio/openase":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	service.baseURL = server.URL
	now := time.Date(2026, 3, 31, 9, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	security, err := service.SaveManualCredential(context.Background(), SaveCredentialInput{
		ProjectID: projectID,
		Scope:     domain.ScopeOrganization,
		Token:     "ghu_manual_token",
	})
	if err != nil {
		t.Fatalf("SaveManualCredential() error = %v", err)
	}

	if !security.Organization.Configured || security.Organization.Source != domain.SourceManualPaste {
		t.Fatalf("SaveManualCredential().Organization = %+v", security.Organization)
	}
	if security.Organization.Probe.State != domain.ProbeStateValid || !security.Organization.Probe.Valid {
		t.Fatalf("SaveManualCredential().Probe = %+v", security.Organization.Probe)
	}
	if security.Organization.Probe.Login != "octocat" {
		t.Fatalf("SaveManualCredential().Probe.Login = %q", security.Organization.Probe.Login)
	}
	if repository.context.OrganizationCredential == nil {
		t.Fatal("expected organization credential to be persisted")
	}
	if repository.context.OrganizationCredential.Source != domain.SourceManualPaste {
		t.Fatalf("organization credential source = %q", repository.context.OrganizationCredential.Source)
	}
	if got := probeStates(repository.organizationProbeHistory); len(got) != 3 ||
		got[0] != domain.ProbeStateConfigured ||
		got[1] != domain.ProbeStateProbing ||
		got[2] != domain.ProbeStateValid {
		t.Fatalf("organization probe history = %v", got)
	}
}

func TestImportGHCLICredentialPersistsProjectOverride(t *testing.T) {
	projectID := uuid.New()
	repository := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:            projectID,
			OrganizationID:       uuid.New(),
			ProjectRepositoryURL: "https://github.com/PacificStudio/openase.git",
		},
	}
	service, err := New(repository, http.DefaultClient, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	service.tokenImporter = stubTokenImporter{token: "ghu_imported_once"}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/user":
			w.Header().Set("X-OAuth-Scopes", "repo")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"login":"octocat"}`))
		case "/repos/pacificstudio/openase":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	service.baseURL = server.URL

	security, err := service.ImportGHCLICredential(context.Background(), ScopeInput{
		ProjectID: projectID,
		Scope:     domain.ScopeProject,
	})
	if err != nil {
		t.Fatalf("ImportGHCLICredential() error = %v", err)
	}

	if !security.ProjectOverride.Configured || security.ProjectOverride.Source != domain.SourceGHCLIImport {
		t.Fatalf("ImportGHCLICredential().ProjectOverride = %+v", security.ProjectOverride)
	}
	if security.ProjectOverride.Probe.Login != "octocat" {
		t.Fatalf("ImportGHCLICredential().ProjectOverride.Probe.Login = %q", security.ProjectOverride.Probe.Login)
	}
	if repository.context.ProjectCredential == nil || repository.context.ProjectCredential.Source != domain.SourceGHCLIImport {
		t.Fatalf("persisted project credential = %+v", repository.context.ProjectCredential)
	}
}

func TestRetestCredentialRejectsMissingScope(t *testing.T) {
	projectID := uuid.New()
	service, err := New(&stubRepository{
		context: domain.ProjectContext{
			ProjectID:            projectID,
			OrganizationID:       uuid.New(),
			ProjectRepositoryURL: "https://github.com/PacificStudio/openase.git",
		},
	}, http.DefaultClient, "postgres://openase:test@localhost/openase")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = service.RetestCredential(context.Background(), ScopeInput{
		ProjectID: projectID,
		Scope:     domain.ScopeProject,
	})
	if !errors.Is(err, ErrCredentialNotConfigured) {
		t.Fatalf("RetestCredential() error = %v, want %v", err, ErrCredentialNotConfigured)
	}
}

func TestDeleteCredentialFallsBackToOrganizationDefault(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	repository := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:            projectID,
			OrganizationID:       orgID,
			ProjectRepositoryURL: "https://github.com/PacificStudio/openase.git",
		},
	}
	service, err := New(repository, nil, "postgres://openase:test@localhost/openase")
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
	repository.context.OrganizationCredential = &orgCredential
	repository.context.OrganizationProbe = &domain.TokenProbe{
		State:      domain.ProbeStateValid,
		Configured: true,
		Valid:      true,
		RepoAccess: domain.RepoAccessGranted,
	}
	repository.context.ProjectCredential = &projectCredential
	repository.context.ProjectProbe = &domain.TokenProbe{
		State:      domain.ProbeStateValid,
		Configured: true,
		Valid:      true,
		RepoAccess: domain.RepoAccessGranted,
	}

	security, err := service.DeleteCredential(context.Background(), ScopeInput{
		ProjectID: projectID,
		Scope:     domain.ScopeProject,
	})
	if err != nil {
		t.Fatalf("DeleteCredential() error = %v", err)
	}

	if security.ProjectOverride.Configured {
		t.Fatalf("expected project override to be cleared, got %+v", security.ProjectOverride)
	}
	if !security.Effective.Configured || security.Effective.Scope != domain.ScopeOrganization {
		t.Fatalf("expected effective credential to fall back to organization, got %+v", security.Effective)
	}
}

func TestExplicitCipherSeedAllowsCrossDSNDecrypt(t *testing.T) {
	t.Parallel()

	repository := &stubRepository{}
	sourceService, err := New(repository, nil, "shared-cluster-seed")
	if err != nil {
		t.Fatalf("New(source) error = %v", err)
	}
	targetService, err := New(repository, nil, "shared-cluster-seed")
	if err != nil {
		t.Fatalf("New(target) error = %v", err)
	}

	sealed, err := sourceService.SealToken("ghu_cross_env_token", domain.SourceManualPaste)
	if err != nil {
		t.Fatalf("SealToken() error = %v", err)
	}

	opened, err := targetService.decryptStoredCredential(sealed)
	if err != nil {
		t.Fatalf("decryptStoredCredential() error = %v", err)
	}
	if opened != "ghu_cross_env_token" {
		t.Fatalf("decryptStoredCredential() = %q", opened)
	}
}

func TestProbeResolvedCredentialPersistsValidProbe(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	repository := &stubRepository{
		context: domain.ProjectContext{
			ProjectID:            projectID,
			OrganizationID:       orgID,
			ProjectRepositoryURL: "https://github.com/PacificStudio/openase.git",
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
			_, _ = w.Write([]byte(`{"login":"octocat"}`))
		case "/repos/pacificstudio/openase":
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	service.baseURL = server.URL
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC)
	service.now = func() time.Time { return now }

	probe, err := service.ProbeResolvedCredential(context.Background(), projectID, "")
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
			ProjectID:            projectID,
			OrganizationID:       uuid.New(),
			ProjectRepositoryURL: "https://github.com/GrandCX/private-repo.git",
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
			_, _ = w.Write([]byte(`{"login":"octocat"}`))
		case "/repos/grandcx/private-repo":
			w.WriteHeader(http.StatusForbidden)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	service.baseURL = server.URL

	probe, err := service.ProbeResolvedCredential(context.Background(), projectID, "")
	if err != nil {
		t.Fatalf("ProbeResolvedCredential() error = %v", err)
	}
	if probe.State != domain.ProbeStateInsufficientPermissions || probe.Valid || probe.RepoAccess != domain.RepoAccessDenied {
		t.Fatalf("ProbeResolvedCredential() = %+v", probe)
	}
}

type stubRepository struct {
	context                  domain.ProjectContext
	orgContext               domain.OrgContext
	savedOrganizationProbe   *domain.TokenProbe
	savedProjectProbe        *domain.TokenProbe
	organizationProbeHistory []domain.TokenProbe
	projectProbeHistory      []domain.TokenProbe
}

func (s *stubRepository) GetOrganizationContext(context.Context, uuid.UUID) (domain.OrgContext, error) {
	if s.orgContext.OrganizationID != uuid.Nil {
		return s.orgContext, nil
	}
	return domain.OrgContext{
		OrganizationID: s.context.OrganizationID,
		Credential:     s.context.OrganizationCredential,
		Probe:          s.context.OrganizationProbe,
	}, nil
}

func (s *stubRepository) GetProjectContext(context.Context, uuid.UUID) (domain.ProjectContext, error) {
	return s.context, nil
}

func (s *stubRepository) SaveOrganizationCredential(
	_ context.Context,
	_ uuid.UUID,
	credential domain.StoredCredential,
	probe domain.TokenProbe,
) error {
	clonedCredential := credential
	clonedProbe := probe
	s.context.OrganizationCredential = &clonedCredential
	s.context.OrganizationProbe = &clonedProbe
	s.organizationProbeHistory = append(s.organizationProbeHistory, clonedProbe)
	return nil
}

func (s *stubRepository) SaveProjectCredential(
	_ context.Context,
	_ uuid.UUID,
	credential domain.StoredCredential,
	probe domain.TokenProbe,
) error {
	clonedCredential := credential
	clonedProbe := probe
	s.context.ProjectCredential = &clonedCredential
	s.context.ProjectProbe = &clonedProbe
	s.projectProbeHistory = append(s.projectProbeHistory, clonedProbe)
	return nil
}

func (s *stubRepository) ClearOrganizationCredential(context.Context, uuid.UUID) error {
	s.context.OrganizationCredential = nil
	s.context.OrganizationProbe = nil
	return nil
}

func (s *stubRepository) ClearProjectCredential(context.Context, uuid.UUID) error {
	s.context.ProjectCredential = nil
	s.context.ProjectProbe = nil
	return nil
}

func (s *stubRepository) SaveOrganizationProbe(_ context.Context, _ uuid.UUID, probe domain.TokenProbe) error {
	cloned := probe
	s.context.OrganizationProbe = &cloned
	s.savedOrganizationProbe = &cloned
	s.organizationProbeHistory = append(s.organizationProbeHistory, cloned)
	return nil
}

func (s *stubRepository) SaveProjectProbe(_ context.Context, _ uuid.UUID, probe domain.TokenProbe) error {
	cloned := probe
	s.context.ProjectProbe = &cloned
	s.savedProjectProbe = &cloned
	s.projectProbeHistory = append(s.projectProbeHistory, cloned)
	return nil
}

type stubTokenImporter struct {
	token string
	err   error
}

func (s stubTokenImporter) ReadToken(context.Context) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.token, nil
}

func probeStates(history []domain.TokenProbe) []domain.ProbeState {
	states := make([]domain.ProbeState, 0, len(history))
	for _, item := range history {
		states = append(states, item.State)
	}
	return states
}
