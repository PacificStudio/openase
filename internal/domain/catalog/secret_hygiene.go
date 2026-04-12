package catalog

import "strings"

const MaskedMachineEnvVarValue = "[redacted]"

func CountLegacyProviderInlineSecretBindings(items []AgentProvider) (int, int) {
	providersWithLegacy := 0
	legacyBindingCount := 0
	for _, item := range items {
		providerLegacyCount := len(LegacyAgentProviderSecretBindings(item.AdapterType, item.AuthConfig))
		if providerLegacyCount == 0 {
			continue
		}
		providersWithLegacy++
		legacyBindingCount += providerLegacyCount
	}
	return providersWithLegacy, legacyBindingCount
}

func CountSensitiveMachineEnvVars(items []Machine) (int, int) {
	machinesWithSensitive := 0
	sensitiveEnvVarCount := 0
	for _, item := range items {
		machineSensitiveCount := SensitiveMachineEnvVarCount(item.EnvVars)
		if machineSensitiveCount == 0 {
			continue
		}
		machinesWithSensitive++
		sensitiveEnvVarCount += machineSensitiveCount
	}
	return machinesWithSensitive, sensitiveEnvVarCount
}

func SensitiveMachineEnvVarCount(raw []string) int {
	count := 0
	for _, item := range raw {
		key, _, ok := splitMachineEnvVar(item)
		if ok && IsSensitiveMachineEnvVarKey(key) {
			count++
		}
	}
	return count
}

func MaskMachineEnvVars(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	masked := make([]string, 0, len(raw))
	for _, item := range raw {
		key, value, ok := splitMachineEnvVar(item)
		if !ok || !IsSensitiveMachineEnvVarKey(key) {
			masked = append(masked, strings.TrimSpace(item))
			continue
		}
		if strings.TrimSpace(value) == "" {
			masked = append(masked, key+"=")
			continue
		}
		masked = append(masked, key+"="+MaskedMachineEnvVarValue)
	}
	return masked
}

func MergeMaskedMachineEnvVars(current []string, requested []string) []string {
	if len(requested) == 0 {
		return nil
	}
	currentByKey := make(map[string]string, len(current))
	for _, item := range current {
		key, _, ok := splitMachineEnvVar(item)
		if !ok {
			continue
		}
		currentByKey[key] = strings.TrimSpace(item)
	}

	merged := make([]string, 0, len(requested))
	for _, item := range requested {
		key, value, ok := splitMachineEnvVar(item)
		if !ok {
			merged = append(merged, strings.TrimSpace(item))
			continue
		}
		if IsSensitiveMachineEnvVarKey(key) &&
			strings.TrimSpace(value) == MaskedMachineEnvVarValue {
			if preserved, exists := currentByKey[key]; exists {
				merged = append(merged, preserved)
				continue
			}
		}
		merged = append(merged, strings.TrimSpace(item))
	}
	return merged
}

func IsSensitiveMachineEnvVarKey(raw string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(raw))
	if normalized == "" {
		return false
	}
	for _, token := range []string{
		"SECRET",
		"TOKEN",
		"PASSWORD",
		"PASSWD",
		"PASS",
		"API_KEY",
		"ACCESS_KEY",
		"ACCESS_TOKEN",
		"AUTH",
		"CREDENTIAL",
		"CERT",
		"PRIVATE_KEY",
		"SESSION_KEY",
		"WEBHOOK",
		"DSN",
	} {
		if strings.Contains(normalized, token) {
			return true
		}
	}
	return false
}

func splitMachineEnvVar(raw string) (string, string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" || !strings.Contains(trimmed, "=") {
		return "", "", false
	}
	key, value, _ := strings.Cut(trimmed, "=")
	key = strings.TrimSpace(key)
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
