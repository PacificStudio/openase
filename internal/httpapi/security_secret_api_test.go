package httpapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/BetterAndBetterII/openase/internal/config"
	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	eventinfra "github.com/BetterAndBetterII/openase/internal/infra/event"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	"github.com/google/uuid"
)

func TestScopedSecretRoutesListSecrets(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = catalogdomain.Project{ID: projectID, OrganizationID: orgID}

	rotatedAt := time.Date(2026, time.April, 8, 12, 0, 0, 0, time.UTC)
	createdAt := rotatedAt.Add(-2 * time.Hour)
	updatedAt := rotatedAt.Add(-1 * time.Hour)
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
		WithSecretService(&stubSecretService{
			listProjectSecretInventory: func(context.Context, uuid.UUID) ([]secretsdomain.InventorySecret, error) {
				return []secretsdomain.InventorySecret{
					{
						Secret: secretsdomain.Secret{
							ID:             uuid.New(),
							OrganizationID: orgID,
							ProjectID:      projectID,
							Scope:          secretsdomain.ScopeKindProject,
							Name:           "OPENAI_API_KEY",
							Kind:           secretsdomain.KindOpaque,
							Description:    "Primary runtime key",
							CreatedAt:      createdAt,
							UpdatedAt:      updatedAt,
							StoredValue: secretsdomain.StoredValue{
								Algorithm: secretsdomain.CipherAlgorithmAES256GCM,
								KeySource: secretsdomain.KeySourceDatabaseDSNSHA256,
								KeyID:     secretsdomain.DefaultKeyID,
								Preview:   "sk-live...cdef",
								RotatedAt: rotatedAt,
							},
						},
						UsageCount:  2,
						UsageScopes: []secretsdomain.BindingScopeKind{secretsdomain.BindingScopeKindProject, secretsdomain.BindingScopeKindWorkflow},
					},
				}, nil
			},
		}),
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/projects/"+projectID.String()+"/security-settings/secrets", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var payload struct {
		Secrets []securityScopedSecretResponse `json:"secrets"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if len(payload.Secrets) != 1 {
		t.Fatalf("expected 1 secret, got %+v", payload.Secrets)
	}
	if payload.Secrets[0].Name != "OPENAI_API_KEY" {
		t.Fatalf("unexpected secret payload: %+v", payload.Secrets[0])
	}
	if payload.Secrets[0].Encryption.ValuePreview != "sk-live...cdef" {
		t.Fatalf("expected preview to be redacted in response, got %+v", payload.Secrets[0].Encryption)
	}
	if payload.Secrets[0].UsageCount != 2 {
		t.Fatalf("expected usage count in response, got %+v", payload.Secrets[0])
	}
}

func TestScopedSecretRoutesCreateSecret(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = catalogdomain.Project{ID: projectID, OrganizationID: orgID}

	var gotInput secretsservice.CreateSecretInput
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
		WithSecretService(&stubSecretService{
			createSecret: func(_ context.Context, input secretsservice.CreateSecretInput) (secretsdomain.Secret, error) {
				gotInput = input
				return secretsdomain.Secret{
					ID:             uuid.New(),
					OrganizationID: orgID,
					ProjectID:      projectID,
					Scope:          secretsdomain.ScopeKindProject,
					Name:           "OPENAI_API_KEY",
					Kind:           secretsdomain.KindOpaque,
					Description:    "runtime key",
					CreatedAt:      time.Now().UTC(),
					UpdatedAt:      time.Now().UTC(),
					StoredValue: secretsdomain.StoredValue{
						Algorithm: secretsdomain.CipherAlgorithmAES256GCM,
						KeySource: secretsdomain.KeySourceDatabaseDSNSHA256,
						KeyID:     secretsdomain.DefaultKeyID,
						Preview:   "sk-live...cdef",
						RotatedAt: time.Now().UTC(),
					},
				}, nil
			},
		}),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/security-settings/secrets",
		`{"scope":"project","name":"OPENAI_API_KEY","kind":"opaque","description":"runtime key","value":"sk-live-1234cdef"}`,
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotInput.ProjectID != projectID || gotInput.Scope != "project" || gotInput.Name != "OPENAI_API_KEY" || gotInput.Value != "sk-live-1234cdef" {
		t.Fatalf("CreateSecret() input = %+v", gotInput)
	}
}

func TestScopedSecretRoutesDeleteSecret(t *testing.T) {
	projectID := uuid.New()
	orgID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = catalogdomain.Project{ID: projectID, OrganizationID: orgID}

	var gotInput secretsservice.DeleteSecretInput
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
		WithSecretService(&stubSecretService{
			deleteSecret: func(_ context.Context, input secretsservice.DeleteSecretInput) error {
				gotInput = input
				return nil
			},
		}),
	)

	secretID := uuid.New()
	rec := performJSONRequest(
		t,
		server,
		http.MethodDelete,
		"/api/v1/projects/"+projectID.String()+"/security-settings/secrets/"+secretID.String(),
		"",
	)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotInput.ProjectID != projectID || gotInput.SecretID != secretID {
		t.Fatalf("DeleteSecret() input = %+v", gotInput)
	}
}

func TestOrganizationScopedSecretRoutesListSecrets(t *testing.T) {
	organizationID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.organizations[organizationID] = catalogdomain.Organization{ID: organizationID, Name: "Acme", Slug: "acme"}

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
		WithSecretService(&stubSecretService{
			listOrganizationSecretInventory: func(context.Context, uuid.UUID) ([]secretsdomain.InventorySecret, error) {
				return []secretsdomain.InventorySecret{
					{
						Secret: secretsdomain.Secret{
							ID:             uuid.New(),
							OrganizationID: organizationID,
							ProjectID:      uuid.Nil,
							Scope:          secretsdomain.ScopeKindOrganization,
							Name:           "GH_TOKEN",
							Kind:           secretsdomain.KindOpaque,
							Description:    "Shared automation token",
							CreatedAt:      time.Now().UTC(),
							UpdatedAt:      time.Now().UTC(),
							StoredValue: secretsdomain.StoredValue{
								Algorithm: secretsdomain.CipherAlgorithmAES256GCM,
								KeySource: secretsdomain.KeySourceDatabaseDSNSHA256,
								KeyID:     secretsdomain.DefaultKeyID,
								Preview:   "ghp_xxx...1234",
								RotatedAt: time.Now().UTC(),
							},
						},
						UsageCount: 1,
						UsageScopes: []secretsdomain.BindingScopeKind{
							secretsdomain.BindingScopeKindOrganization,
						},
					},
				}, nil
			},
		}),
	)

	rec := performJSONRequest(t, server, http.MethodGet, "/api/v1/orgs/"+organizationID.String()+"/security-settings/secrets", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "GH_TOKEN") || !strings.Contains(rec.Body.String(), "organization") {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestOrganizationScopedSecretRoutesCreateSecret(t *testing.T) {
	organizationID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.organizations[organizationID] = catalogdomain.Organization{ID: organizationID, Name: "Acme", Slug: "acme"}

	var gotInput secretsservice.CreateOrganizationSecretInput
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
		WithSecretService(&stubSecretService{
			createOrganizationSecret: func(_ context.Context, input secretsservice.CreateOrganizationSecretInput) (secretsdomain.Secret, error) {
				gotInput = input
				return secretsdomain.Secret{
					ID:             uuid.New(),
					OrganizationID: organizationID,
					ProjectID:      uuid.Nil,
					Scope:          secretsdomain.ScopeKindOrganization,
					Name:           "GH_TOKEN",
					Kind:           secretsdomain.KindOpaque,
					Description:    "Shared automation token",
					CreatedAt:      time.Now().UTC(),
					UpdatedAt:      time.Now().UTC(),
					StoredValue: secretsdomain.StoredValue{
						Algorithm: secretsdomain.CipherAlgorithmAES256GCM,
						KeySource: secretsdomain.KeySourceDatabaseDSNSHA256,
						KeyID:     secretsdomain.DefaultKeyID,
						Preview:   "ghp_xxx...1234",
						RotatedAt: time.Now().UTC(),
					},
				}, nil
			},
			listOrganizationSecretInventory: func(context.Context, uuid.UUID) ([]secretsdomain.InventorySecret, error) {
				return nil, nil
			},
		}),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/orgs/"+organizationID.String()+"/security-settings/secrets",
		`{"name":"GH_TOKEN","kind":"opaque","description":"Shared automation token","value":"ghp_1234"}`,
	)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if gotInput.OrganizationID != organizationID || gotInput.Name != "GH_TOKEN" || gotInput.Value != "ghp_1234" {
		t.Fatalf("CreateOrganizationSecret() input = %+v", gotInput)
	}
}

func TestScopedSecretRoutesResolveRejectsInvalidUUID(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = catalogdomain.Project{ID: projectID, OrganizationID: uuid.New()}

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
		WithSecretService(&stubSecretService{}),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/security-settings/secrets/resolve-for-runtime",
		`{"binding_keys":["OPENAI_API_KEY"],"workflow_id":"not-a-uuid"}`,
	)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "workflow_id") {
		t.Fatalf("expected invalid workflow_id error, got %s", rec.Body.String())
	}
}

func TestScopedSecretRoutesResolveMapsConflict(t *testing.T) {
	projectID := uuid.New()
	catalog := newFakeCatalogService()
	catalog.projects[projectID] = catalogdomain.Project{ID: projectID, OrganizationID: uuid.New()}

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
		WithSecretService(&stubSecretService{
			resolveForRuntime: func(context.Context, secretsservice.ResolveRuntimeInput) ([]secretsdomain.ResolvedSecret, []string, error) {
				return nil, nil, secretsdomain.ErrResolutionScopeConflict
			},
		}),
	)

	rec := performJSONRequest(
		t,
		server,
		http.MethodPost,
		"/api/v1/projects/"+projectID.String()+"/security-settings/secrets/resolve-for-runtime",
		`{"binding_keys":["OPENAI_API_KEY"]}`,
	)
	if rec.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "SECRET_BINDING_CONFLICT") {
		t.Fatalf("expected conflict code, got %s", rec.Body.String())
	}
}

type stubSecretService struct {
	listProjectSecretInventory      func(context.Context, uuid.UUID) ([]secretsdomain.InventorySecret, error)
	listOrganizationSecretInventory func(context.Context, uuid.UUID) ([]secretsdomain.InventorySecret, error)
	createSecret                    func(context.Context, secretsservice.CreateSecretInput) (secretsdomain.Secret, error)
	createOrganizationSecret        func(context.Context, secretsservice.CreateOrganizationSecretInput) (secretsdomain.Secret, error)
	updateMetadata                  func(context.Context, secretsservice.UpdateSecretMetadataInput) (secretsdomain.Secret, error)
	rotateSecret                    func(context.Context, secretsservice.RotateSecretInput) (secretsdomain.Secret, error)
	rotateOrganizationSecret        func(context.Context, secretsservice.RotateOrganizationSecretInput) (secretsdomain.Secret, error)
	disableSecret                   func(context.Context, secretsservice.DisableSecretInput) (secretsdomain.Secret, error)
	disableOrganizationSecret       func(context.Context, secretsservice.DisableOrganizationSecretInput) (secretsdomain.Secret, error)
	deleteSecret                    func(context.Context, secretsservice.DeleteSecretInput) error
	deleteOrganizationSecret        func(context.Context, secretsservice.DeleteOrganizationSecretInput) error
	resolveForRuntime               func(context.Context, secretsservice.ResolveRuntimeInput) ([]secretsdomain.ResolvedSecret, []string, error)
	resolveBound                    func(context.Context, secretsservice.ResolveBoundRuntimeInput) ([]secretsdomain.ResolvedSecret, error)
}

func (s *stubSecretService) ListProjectSecretInventory(ctx context.Context, projectID uuid.UUID) ([]secretsdomain.InventorySecret, error) {
	if s.listProjectSecretInventory == nil {
		return nil, nil
	}
	return s.listProjectSecretInventory(ctx, projectID)
}

func (s *stubSecretService) ListOrganizationSecretInventory(ctx context.Context, organizationID uuid.UUID) ([]secretsdomain.InventorySecret, error) {
	if s.listOrganizationSecretInventory == nil {
		return nil, nil
	}
	return s.listOrganizationSecretInventory(ctx, organizationID)
}

func (s *stubSecretService) CreateSecret(ctx context.Context, input secretsservice.CreateSecretInput) (secretsdomain.Secret, error) {
	if s.createSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.createSecret(ctx, input)
}

func (s *stubSecretService) CreateOrganizationSecret(ctx context.Context, input secretsservice.CreateOrganizationSecretInput) (secretsdomain.Secret, error) {
	if s.createOrganizationSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.createOrganizationSecret(ctx, input)
}

func (s *stubSecretService) UpdateSecretMetadata(ctx context.Context, input secretsservice.UpdateSecretMetadataInput) (secretsdomain.Secret, error) {
	if s.updateMetadata == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.updateMetadata(ctx, input)
}

func (s *stubSecretService) RotateSecret(ctx context.Context, input secretsservice.RotateSecretInput) (secretsdomain.Secret, error) {
	if s.rotateSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.rotateSecret(ctx, input)
}

func (s *stubSecretService) RotateOrganizationSecret(ctx context.Context, input secretsservice.RotateOrganizationSecretInput) (secretsdomain.Secret, error) {
	if s.rotateOrganizationSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.rotateOrganizationSecret(ctx, input)
}

func (s *stubSecretService) DisableSecret(ctx context.Context, input secretsservice.DisableSecretInput) (secretsdomain.Secret, error) {
	if s.disableSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.disableSecret(ctx, input)
}

func (s *stubSecretService) DisableOrganizationSecret(ctx context.Context, input secretsservice.DisableOrganizationSecretInput) (secretsdomain.Secret, error) {
	if s.disableOrganizationSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.disableOrganizationSecret(ctx, input)
}

func (s *stubSecretService) DeleteSecret(ctx context.Context, input secretsservice.DeleteSecretInput) error {
	if s.deleteSecret == nil {
		return nil
	}
	return s.deleteSecret(ctx, input)
}

func (s *stubSecretService) DeleteOrganizationSecret(ctx context.Context, input secretsservice.DeleteOrganizationSecretInput) error {
	if s.deleteOrganizationSecret == nil {
		return nil
	}
	return s.deleteOrganizationSecret(ctx, input)
}

func (s *stubSecretService) ResolveForRuntime(ctx context.Context, input secretsservice.ResolveRuntimeInput) ([]secretsdomain.ResolvedSecret, []string, error) {
	if s.resolveForRuntime == nil {
		return nil, nil, nil
	}
	return s.resolveForRuntime(ctx, input)
}

func (s *stubSecretService) ResolveBoundForRuntime(ctx context.Context, input secretsservice.ResolveBoundRuntimeInput) ([]secretsdomain.ResolvedSecret, error) {
	if s.resolveBound == nil {
		return nil, nil
	}
	return s.resolveBound(ctx, input)
}
