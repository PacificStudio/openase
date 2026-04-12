package catalog

import (
	"fmt"
	"sort"
	"strings"

	secretsdomain "github.com/BetterAndBetterII/openase/internal/domain/secrets"
)

const agentProviderSecretRefsField = "secret_refs"

type AgentProviderSecretBindingSource string

const (
	AgentProviderSecretBindingSourceBinding AgentProviderSecretBindingSource = "binding"
	// #nosec G101 -- descriptive source label, not a credential literal.
	AgentProviderSecretBindingSourceLegacyAuthConfig AgentProviderSecretBindingSource = "legacy_auth_config"
	AgentProviderSecretBindingSourceDefault          AgentProviderSecretBindingSource = "default"
)

type AgentProviderSecretBinding struct {
	EnvVarKey  string                           `json:"env_var_key"`
	BindingKey string                           `json:"binding_key"`
	Configured bool                             `json:"configured"`
	Source     AgentProviderSecretBindingSource `json:"source"`
}

type AgentProviderSecretBindingInput struct {
	EnvVarKey  string `json:"env_var_key"`
	BindingKey string `json:"binding_key"`
}

type agentProviderAuthConfigParts struct {
	plainConfig  map[string]any
	explicitRefs map[string]string
	legacyInline map[string]any
}

func BuildAgentProviderAuthConfig(
	aAdapterType AgentProviderAdapterType,
	plainConfig map[string]any,
	secretBindings []AgentProviderSecretBindingInput,
) (map[string]any, error) {
	parts := splitAgentProviderAuthConfig(aAdapterType, plainConfig)
	explicitRefs := parts.explicitRefs
	legacyInline := parts.legacyInline
	if len(secretBindings) > 0 {
		parsed, err := parseAgentProviderSecretBindingInputs(secretBindings)
		if err != nil {
			return nil, err
		}
		explicitRefs = parsed
		legacyInline = removeProviderLegacyInlineOverlaps(legacyInline, explicitRefs)
	}
	return composeAgentProviderAuthConfig(parts.plainConfig, legacyInline, explicitRefs), nil
}

func MergeAgentProviderAuthConfig(
	aAdapterType AgentProviderAdapterType,
	current map[string]any,
	plainConfig *map[string]any,
	secretBindings *[]AgentProviderSecretBindingInput,
) (map[string]any, error) {
	currentParts := splitAgentProviderAuthConfig(aAdapterType, current)
	plain := currentParts.plainConfig
	legacy := currentParts.legacyInline
	explicit := currentParts.explicitRefs

	if plainConfig != nil {
		parts := splitAgentProviderAuthConfig(aAdapterType, *plainConfig)
		plain = parts.plainConfig
		for rawKey, value := range parts.legacyInline {
			legacy[rawKey] = value
		}
	}
	if secretBindings != nil {
		parsed, err := parseAgentProviderSecretBindingInputs(*secretBindings)
		if err != nil {
			return nil, err
		}
		explicit = parsed
		legacy = removeProviderLegacyInlineOverlaps(legacy, explicit)
	}

	return composeAgentProviderAuthConfig(plain, legacy, explicit), nil
}

func VisibleAgentProviderAuthConfig(
	aAdapterType AgentProviderAdapterType,
	raw map[string]any,
) map[string]any {
	return splitAgentProviderAuthConfig(aAdapterType, raw).plainConfig
}

func AgentProviderSecretBindings(
	aAdapterType AgentProviderAdapterType,
	raw map[string]any,
) []AgentProviderSecretBinding {
	parts := splitAgentProviderAuthConfig(aAdapterType, raw)
	keys := make([]string, 0, len(parts.explicitRefs)+len(requiredProviderSecretEnvVars(aAdapterType)))
	seen := make(map[string]struct{}, len(parts.explicitRefs))
	for _, envVarKey := range requiredProviderSecretEnvVars(aAdapterType) {
		seen[envVarKey] = struct{}{}
		keys = append(keys, envVarKey)
	}
	for envVarKey := range parts.explicitRefs {
		if _, ok := seen[envVarKey]; ok {
			continue
		}
		seen[envVarKey] = struct{}{}
		keys = append(keys, envVarKey)
	}
	sort.Strings(keys)

	bindings := make([]AgentProviderSecretBinding, 0, len(keys))
	for _, envVarKey := range keys {
		binding := AgentProviderSecretBinding{
			EnvVarKey:  envVarKey,
			BindingKey: envVarKey,
			Configured: false,
			Source:     AgentProviderSecretBindingSourceDefault,
		}
		if explicitBinding, ok := parts.explicitRefs[envVarKey]; ok {
			binding.BindingKey = explicitBinding
			binding.Configured = true
			binding.Source = AgentProviderSecretBindingSourceBinding
		}
		bindings = append(bindings, binding)
	}
	return bindings
}

func LegacyAgentProviderSecretBindings(
	aAdapterType AgentProviderAdapterType,
	raw map[string]any,
) []AgentProviderSecretBinding {
	parts := splitAgentProviderAuthConfig(aAdapterType, raw)
	keys := make([]string, 0, len(parts.legacyInline))
	seen := make(map[string]struct{}, len(parts.legacyInline))
	for rawKey := range parts.legacyInline {
		envVarKey := normalizeProviderEnvKey(rawKey)
		if envVarKey == "" {
			continue
		}
		if _, ok := seen[envVarKey]; ok {
			continue
		}
		seen[envVarKey] = struct{}{}
		keys = append(keys, envVarKey)
	}
	sort.Strings(keys)

	bindings := make([]AgentProviderSecretBinding, 0, len(keys))
	for _, envVarKey := range keys {
		bindings = append(bindings, AgentProviderSecretBinding{
			EnvVarKey:  envVarKey,
			BindingKey: envVarKey,
			Configured: true,
			Source:     AgentProviderSecretBindingSourceLegacyAuthConfig,
		})
	}
	return bindings
}

func AgentProviderExplicitSecretRefs(raw map[string]any) map[string]string {
	return splitAgentProviderAuthConfig(AgentProviderAdapterTypeCustom, raw).explicitRefs
}

func splitAgentProviderAuthConfig(
	aAdapterType AgentProviderAdapterType,
	raw map[string]any,
) agentProviderAuthConfigParts {
	plain := make(map[string]any, len(raw))
	legacy := make(map[string]any)
	for key, value := range raw {
		if strings.EqualFold(strings.TrimSpace(key), agentProviderSecretRefsField) {
			continue
		}
		if shouldTreatProviderAuthConfigKeyAsSecret(aAdapterType, key) {
			legacy[key] = value
			continue
		}
		plain[key] = value
	}
	return agentProviderAuthConfigParts{
		plainConfig:  plain,
		explicitRefs: parseStoredAgentProviderSecretRefs(raw),
		legacyInline: legacy,
	}
}

func composeAgentProviderAuthConfig(
	plainConfig map[string]any,
	legacyInline map[string]any,
	explicitRefs map[string]string,
) map[string]any {
	result := make(map[string]any, len(plainConfig)+len(legacyInline)+1)
	for key, value := range legacyInline {
		if strings.EqualFold(strings.TrimSpace(key), agentProviderSecretRefsField) {
			continue
		}
		result[key] = value
	}
	for key, value := range plainConfig {
		if strings.EqualFold(strings.TrimSpace(key), agentProviderSecretRefsField) {
			continue
		}
		result[key] = value
	}
	if len(explicitRefs) > 0 {
		secretRefs := make(map[string]any, len(explicitRefs))
		for envVarKey, bindingKey := range explicitRefs {
			secretRefs[envVarKey] = bindingKey
		}
		result[agentProviderSecretRefsField] = secretRefs
	}
	return result
}

func removeProviderLegacyInlineOverlaps(legacyInline map[string]any, explicitRefs map[string]string) map[string]any {
	if len(legacyInline) == 0 || len(explicitRefs) == 0 {
		return legacyInline
	}
	filtered := make(map[string]any, len(legacyInline))
	for rawKey, value := range legacyInline {
		if _, ok := explicitRefs[normalizeProviderEnvKey(rawKey)]; ok {
			continue
		}
		filtered[rawKey] = value
	}
	return filtered
}

func parseAgentProviderSecretBindingInputs(raw []AgentProviderSecretBindingInput) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	parsed := make(map[string]string, len(raw))
	for index, item := range raw {
		envVarKey, err := parseProviderBindingName(item.EnvVarKey)
		if err != nil {
			return nil, fmt.Errorf("secret_bindings[%d].env_var_key: %w", index, err)
		}
		bindingKey, err := parseProviderBindingName(item.BindingKey)
		if err != nil {
			return nil, fmt.Errorf("secret_bindings[%d].binding_key: %w", index, err)
		}
		parsed[envVarKey] = bindingKey
	}
	return parsed, nil
}

func parseStoredAgentProviderSecretRefs(raw map[string]any) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	stored, ok := raw[agentProviderSecretRefsField]
	if !ok || stored == nil {
		return map[string]string{}
	}

	parsed := make(map[string]string)
	switch typed := stored.(type) {
	case map[string]string:
		for envVarKey, bindingKey := range typed {
			normalizedEnvVarKey, envErr := parseProviderBindingName(envVarKey)
			normalizedBindingKey, bindingErr := parseProviderBindingName(bindingKey)
			if envErr != nil || bindingErr != nil {
				continue
			}
			parsed[normalizedEnvVarKey] = normalizedBindingKey
		}
	case map[string]any:
		for envVarKey, bindingValue := range typed {
			normalizedEnvVarKey, envErr := parseProviderBindingName(envVarKey)
			if envErr != nil {
				continue
			}
			normalizedBindingKey, bindingErr := parseProviderBindingName(fmt.Sprint(bindingValue))
			if bindingErr != nil {
				continue
			}
			parsed[normalizedEnvVarKey] = normalizedBindingKey
		}
	}
	return parsed
}

func shouldTreatProviderAuthConfigKeyAsSecret(
	aAdapterType AgentProviderAdapterType,
	rawKey string,
) bool {
	envVarKey := normalizeProviderEnvKey(rawKey)
	if envVarKey == "" {
		return false
	}
	for _, requiredKey := range requiredProviderSecretEnvVars(aAdapterType) {
		if envVarKey == requiredKey {
			return true
		}
	}
	parts := strings.Split(envVarKey, "_")
	for _, part := range parts {
		switch part {
		case "TOKEN", "SECRET", "PASSWORD", "PASSPHRASE":
			return true
		}
	}
	for index := 0; index < len(parts)-1; index++ {
		if parts[index] == "API" && parts[index+1] == "KEY" {
			return true
		}
	}
	return false
}

func requiredProviderSecretEnvVars(adapterType AgentProviderAdapterType) []string {
	switch adapterType {
	case AgentProviderAdapterTypeClaudeCodeCLI:
		return []string{"ANTHROPIC_API_KEY"}
	case AgentProviderAdapterTypeCodexAppServer:
		return []string{"OPENAI_API_KEY"}
	case AgentProviderAdapterTypeGeminiCLI:
		return []string{"GEMINI_API_KEY", "GOOGLE_API_KEY"}
	default:
		return nil
	}
}

func parseProviderBindingName(raw string) (string, error) {
	normalized := normalizeProviderEnvKey(raw)
	if normalized == "" {
		return "", fmt.Errorf("must not be empty")
	}
	return secretsdomain.NormalizeName(normalized)
}

func normalizeProviderEnvKey(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	normalized := strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - ('a' - 'A')
		case r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		default:
			return '_'
		}
	}, trimmed)
	return strings.Trim(normalized, "_")
}
