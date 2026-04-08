package chat

import (
	"context"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	runtimesecretenv "github.com/BetterAndBetterII/openase/internal/runtime/secretenv"
)

type runtimeSecretResolver interface {
	ResolveForRuntime(ctx context.Context, input runtimesecretenv.ResolveInput) ([]string, error)
}

type providerSecretResolver struct {
	resolver runtimesecretenv.Resolver
}

func (r providerSecretResolver) ResolveForRuntime(
	ctx context.Context,
	input runtimesecretenv.ResolveInput,
) ([]string, error) {
	return runtimesecretenv.AppendResolvedProviderSecrets(ctx, r.resolver, input)
}

func resolveRuntimeEnvironment(
	ctx context.Context,
	resolver runtimesecretenv.Resolver,
	input RuntimeTurnInput,
) ([]string, error) {
	baseEnvironment := append(providerAuthEnvironment(input.Provider), input.Environment...)
	return providerSecretResolver{resolver: resolver}.ResolveForRuntime(ctx, runtimesecretenv.ResolveInput{
		ProjectID:          input.ProjectID,
		ProviderAuthConfig: input.Provider.AuthConfig,
		BaseEnvironment:    baseEnvironment,
		TicketID:           input.TicketID,
		WorkflowID:         input.WorkflowID,
		AgentID:            input.AgentID,
	})
}

func providerAuthEnvironment(providerItem catalogdomain.AgentProvider) []string {
	return provider.AuthConfigEnvironment(providerItem.AuthConfig)
}
