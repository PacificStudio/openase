package chat

import (
	"context"
	"slices"
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	"github.com/google/uuid"
)

type fakeChatSecretResolver struct {
	inputs   []secretsservice.ResolveRuntimeInput
	resolved []secretsdomain.ResolvedSecret
}

func (r *fakeChatSecretResolver) ResolveForRuntime(
	_ context.Context,
	input secretsservice.ResolveRuntimeInput,
) ([]secretsdomain.ResolvedSecret, []string, error) {
	r.inputs = append(r.inputs, input)
	return append([]secretsdomain.ResolvedSecret(nil), r.resolved...), nil, nil
}

func TestResolveRuntimeEnvironmentAppendsExplicitProviderSecrets(t *testing.T) {
	projectID := uuid.New()
	ticketID := uuid.New()
	resolver := &fakeChatSecretResolver{
		resolved: []secretsdomain.ResolvedSecret{
			{BindingKey: "PROJECT_OPENAI_KEY", Value: "sk-explicit"},
		},
	}

	environment, err := resolveRuntimeEnvironment(context.Background(), resolver, RuntimeTurnInput{
		ProjectID: projectID,
		TicketID:  &ticketID,
		Provider: catalogdomain.AgentProvider{
			AdapterType: catalogdomain.AgentProviderAdapterTypeCodexAppServer,
			AuthConfig: map[string]any{
				"openai_api_key": "legacy-inline",
				"secret_refs": map[string]any{
					"OPENAI_API_KEY": "PROJECT_OPENAI_KEY",
				},
			},
		},
		Environment: []string{"OPENASE_TICKET_ID=" + ticketID.String()},
	})
	if err != nil {
		t.Fatalf("resolveRuntimeEnvironment() error = %v", err)
	}

	if len(resolver.inputs) != 1 {
		t.Fatalf("resolver call count = %d, want 1", len(resolver.inputs))
	}
	if resolver.inputs[0].ProjectID != projectID {
		t.Fatalf("resolver project id = %s, want %s", resolver.inputs[0].ProjectID, projectID)
	}
	if resolver.inputs[0].TicketID == nil || *resolver.inputs[0].TicketID != ticketID {
		t.Fatalf("resolver ticket id = %#v, want %s", resolver.inputs[0].TicketID, ticketID)
	}
	if !slices.Contains(environment, "OPENAI_API_KEY=legacy-inline") {
		t.Fatalf("environment = %v, want preserved legacy auth env", environment)
	}
	if !slices.Contains(environment, "OPENAI_API_KEY=sk-explicit") {
		t.Fatalf("environment = %v, want appended explicit secret env", environment)
	}
}
