package chat

import (
	"context"
	"fmt"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

type RuntimeEnvironmentResolveInput struct {
	ProjectID          uuid.UUID
	ProviderAuthConfig map[string]any
	BaseEnvironment    []string
	TicketID           *uuid.UUID
	WorkflowID         *uuid.UUID
	AgentID            *uuid.UUID
}

type RuntimeEnvironmentResolver interface {
	ResolveProviderEnvironment(ctx context.Context, input RuntimeEnvironmentResolveInput) ([]string, error)
}

func resolveRuntimeEnvironment(
	ctx context.Context,
	resolver RuntimeEnvironmentResolver,
	input RuntimeTurnInput,
) ([]string, error) {
	baseEnvironment := append(providerAuthEnvironment(input.Provider), input.Environment...)
	resolveInput := RuntimeEnvironmentResolveInput{
		ProjectID:          input.ProjectID,
		ProviderAuthConfig: input.Provider.AuthConfig,
		BaseEnvironment:    baseEnvironment,
		TicketID:           input.TicketID,
		WorkflowID:         input.WorkflowID,
		AgentID:            input.AgentID,
	}
	if resolver == nil {
		explicitRefs := catalogdomain.AgentProviderExplicitSecretRefs(input.Provider.AuthConfig)
		if len(explicitRefs) == 0 {
			return baseEnvironment, nil
		}
		missing := unresolvedProviderEnvKeys(baseEnvironment, explicitRefs)
		if len(missing) > 0 {
			return nil, fmt.Errorf("provider secret resolution is unavailable for %s", strings.Join(missing, ", "))
		}
		return baseEnvironment, nil
	}
	return resolver.ResolveProviderEnvironment(ctx, resolveInput)
}

func providerAuthEnvironment(providerItem catalogdomain.AgentProvider) []string {
	return provider.AuthConfigEnvironment(providerItem.AuthConfig)
}

func unresolvedProviderEnvKeys(baseEnvironment []string, refs map[string]string) []string {
	missing := make([]string, 0)
	for envVarKey := range refs {
		if value, ok := provider.LookupEnvironmentValue(baseEnvironment, envVarKey); ok && strings.TrimSpace(value) != "" {
			continue
		}
		missing = append(missing, envVarKey)
	}
	return missing
}
