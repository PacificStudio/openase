package secretenv

import (
	"context"
	"errors"
	"slices"
	"testing"

	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	"github.com/google/uuid"
)

type fakeResolver struct {
	resolved []secretsdomain.ResolvedSecret
	missing  []string
	err      error
	inputs   []secretsservice.ResolveRuntimeInput
}

func (r *fakeResolver) ResolveForRuntime(
	_ context.Context,
	input secretsservice.ResolveRuntimeInput,
) ([]secretsdomain.ResolvedSecret, []string, error) {
	r.inputs = append(r.inputs, input)
	if r.err != nil {
		return nil, nil, r.err
	}
	return append([]secretsdomain.ResolvedSecret(nil), r.resolved...),
		append([]string(nil), r.missing...),
		nil
}

func TestAppendResolvedProviderSecretsAppendsResolvedBindings(t *testing.T) {
	projectID := uuid.New()
	resolver := &fakeResolver{
		resolved: []secretsdomain.ResolvedSecret{
			{BindingKey: "PROJECT_OPENAI_KEY", Value: "sk-explicit"},
		},
	}

	environment, err := AppendResolvedProviderSecrets(context.Background(), resolver, ResolveInput{
		ProjectID: projectID,
		ProviderAuthConfig: map[string]any{
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
			},
		},
		BaseEnvironment: []string{"PATH=/usr/bin"},
	})
	if err != nil {
		t.Fatalf("AppendResolvedProviderSecrets() error = %v", err)
	}

	if len(resolver.inputs) != 1 {
		t.Fatalf("resolver call count = %d, want 1", len(resolver.inputs))
	}
	if resolver.inputs[0].ProjectID != projectID {
		t.Fatalf("resolver project id = %s, want %s", resolver.inputs[0].ProjectID, projectID)
	}
	if !slices.Contains(environment, "OPENAI_API_KEY=sk-explicit") {
		t.Fatalf("environment = %v, want resolved OPENAI_API_KEY", environment)
	}
}

func TestAppendResolvedProviderSecretsAllowsBaseEnvironmentFallback(t *testing.T) {
	environment, err := AppendResolvedProviderSecrets(context.Background(), nil, ResolveInput{
		ProjectID: uuid.New(),
		ProviderAuthConfig: map[string]any{
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
			},
		},
		BaseEnvironment: []string{"OPENAI_API_KEY=already-present"},
	})
	if err != nil {
		t.Fatalf("AppendResolvedProviderSecrets() error = %v", err)
	}
	if len(environment) != 1 || environment[0] != "OPENAI_API_KEY=already-present" {
		t.Fatalf("environment = %v, want preserved base environment", environment)
	}
}

func TestAppendResolvedProviderSecretsFailsWhenBindingIsMissingAndEnvUnset(t *testing.T) {
	_, err := AppendResolvedProviderSecrets(context.Background(), &fakeResolver{missing: []string{"PROJECT_OPENAI_KEY"}}, ResolveInput{
		ProjectID: uuid.New(),
		ProviderAuthConfig: map[string]any{
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
			},
		},
		BaseEnvironment: []string{"PATH=/usr/bin"},
	})
	if err == nil || err.Error() != "missing provider secret bindings for OPENAI_API_KEY" {
		t.Fatalf("AppendResolvedProviderSecrets() error = %v, want missing OPENAI_API_KEY", err)
	}
}

func TestAppendResolvedProviderSecretsPropagatesResolverError(t *testing.T) {
	wantErr := errors.New("boom")
	_, err := AppendResolvedProviderSecrets(context.Background(), &fakeResolver{err: wantErr}, ResolveInput{
		ProjectID: uuid.New(),
		ProviderAuthConfig: map[string]any{
			"secret_refs": map[string]any{
				"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
			},
		},
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("AppendResolvedProviderSecrets() error = %v, want %v", err, wantErr)
	}
}
