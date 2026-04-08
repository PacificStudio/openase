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
			listProjectSecrets: func(context.Context, uuid.UUID) ([]secretsdomain.Secret, error) {
				return []secretsdomain.Secret{
					{
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
	listProjectSecrets func(context.Context, uuid.UUID) ([]secretsdomain.Secret, error)
	createSecret       func(context.Context, secretsservice.CreateSecretInput) (secretsdomain.Secret, error)
	updateMetadata     func(context.Context, secretsservice.UpdateSecretMetadataInput) (secretsdomain.Secret, error)
	rotateSecret       func(context.Context, secretsservice.RotateSecretInput) (secretsdomain.Secret, error)
	disableSecret      func(context.Context, secretsservice.DisableSecretInput) (secretsdomain.Secret, error)
	resolveForRuntime  func(context.Context, secretsservice.ResolveRuntimeInput) ([]secretsdomain.ResolvedSecret, []string, error)
	resolveBound       func(context.Context, secretsservice.ResolveBoundRuntimeInput) ([]secretsdomain.ResolvedSecret, error)
}

func (s *stubSecretService) ListProjectSecrets(ctx context.Context, projectID uuid.UUID) ([]secretsdomain.Secret, error) {
	if s.listProjectSecrets == nil {
		return nil, nil
	}
	return s.listProjectSecrets(ctx, projectID)
}

func (s *stubSecretService) CreateSecret(ctx context.Context, input secretsservice.CreateSecretInput) (secretsdomain.Secret, error) {
	if s.createSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.createSecret(ctx, input)
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

func (s *stubSecretService) DisableSecret(ctx context.Context, input secretsservice.DisableSecretInput) (secretsdomain.Secret, error) {
	if s.disableSecret == nil {
		return secretsdomain.Secret{}, nil
	}
	return s.disableSecret(ctx, input)
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
