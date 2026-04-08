package secretenv

import (
	"context"
	"fmt"
	"sort"
	"strings"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
	"github.com/BetterAndBetterII/openase/internal/provider"
	secretsservice "github.com/BetterAndBetterII/openase/internal/service/secrets"
	"github.com/google/uuid"
)

type Resolver interface {
	ResolveForRuntime(ctx context.Context, input secretsservice.ResolveRuntimeInput) ([]secretsdomain.ResolvedSecret, []string, error)
}

type ResolveInput struct {
	ProjectID          uuid.UUID
	ProviderAuthConfig map[string]any
	BaseEnvironment    []string
	TicketID           *uuid.UUID
	WorkflowID         *uuid.UUID
	AgentID            *uuid.UUID
}

func AppendResolvedProviderSecrets(
	ctx context.Context,
	resolver Resolver,
	input ResolveInput,
) ([]string, error) {
	environment := append([]string(nil), input.BaseEnvironment...)
	explicitRefs := catalogdomain.AgentProviderExplicitSecretRefs(input.ProviderAuthConfig)
	if len(explicitRefs) == 0 {
		return environment, nil
	}

	envVarKeys := make([]string, 0, len(explicitRefs))
	bindingKeySet := make(map[string]struct{}, len(explicitRefs))
	bindingKeys := make([]string, 0, len(explicitRefs))
	for envVarKey, bindingKey := range explicitRefs {
		envVarKeys = append(envVarKeys, envVarKey)
		if _, ok := bindingKeySet[bindingKey]; ok {
			continue
		}
		bindingKeySet[bindingKey] = struct{}{}
		bindingKeys = append(bindingKeys, bindingKey)
	}
	sort.Strings(envVarKeys)
	sort.Strings(bindingKeys)

	if resolver == nil {
		missing := unresolvedEnvVarKeys(environment, explicitRefs, nil)
		if len(missing) > 0 {
			return nil, fmt.Errorf("provider secret resolution is unavailable for %s", strings.Join(missing, ", "))
		}
		return environment, nil
	}

	resolved, missingBindingKeys, err := resolver.ResolveForRuntime(ctx, secretsservice.ResolveRuntimeInput{
		ProjectID:   input.ProjectID,
		BindingKeys: bindingKeys,
		TicketID:    input.TicketID,
		WorkflowID:  input.WorkflowID,
		AgentID:     input.AgentID,
	})
	if err != nil {
		return nil, err
	}

	resolvedByBindingKey := make(map[string]string, len(resolved))
	for _, item := range resolved {
		resolvedByBindingKey[item.BindingKey] = item.Value
	}
	missingByBindingKey := make(map[string]struct{}, len(missingBindingKeys))
	for _, item := range missingBindingKeys {
		missingByBindingKey[item] = struct{}{}
	}

	missing := unresolvedEnvVarKeys(environment, explicitRefs, resolvedByBindingKey)
	if len(missing) > 0 {
		for _, envVarKey := range envVarKeys {
			bindingKey := explicitRefs[envVarKey]
			if _, ok := missingByBindingKey[bindingKey]; ok && !strings.Contains(strings.Join(missing, ","), envVarKey) {
				missing = append(missing, envVarKey)
			}
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("missing provider secret bindings for %s", strings.Join(missing, ", "))
	}

	for _, envVarKey := range envVarKeys {
		bindingKey := explicitRefs[envVarKey]
		if value, ok := resolvedByBindingKey[bindingKey]; ok {
			environment = append(environment, envVarKey+"="+value)
		}
	}

	return environment, nil
}

func unresolvedEnvVarKeys(baseEnvironment []string, refs map[string]string, resolved map[string]string) []string {
	missing := make([]string, 0)
	for envVarKey, bindingKey := range refs {
		if resolved != nil {
			if value, ok := resolved[bindingKey]; ok && strings.TrimSpace(value) != "" {
				continue
			}
		}
		if value, ok := provider.LookupEnvironmentValue(baseEnvironment, envVarKey); ok && strings.TrimSpace(value) != "" {
			continue
		}
		missing = append(missing, envVarKey)
	}
	sort.Strings(missing)
	return missing
}
