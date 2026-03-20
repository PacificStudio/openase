package provider

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

func AuthConfigEnvironment(authConfig map[string]any) []string {
	if len(authConfig) == 0 {
		return nil
	}

	keys := make([]string, 0, len(authConfig))
	for key := range authConfig {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	env := make([]string, 0, len(keys))
	for _, key := range keys {
		value, ok := stringifyEnvValue(authConfig[key])
		if !ok {
			continue
		}
		normalized := normalizeEnvKey(key)
		if normalized == "" {
			continue
		}
		env = append(env, normalized+"="+value)
	}
	return env
}

func LookupEnvironmentValue(environment []string, key string) (string, bool) {
	normalizedKey := normalizeEnvKey(key)
	if normalizedKey == "" {
		return "", false
	}

	for index := len(environment) - 1; index >= 0; index-- {
		name, value, found := strings.Cut(environment[index], "=")
		if !found {
			continue
		}
		if normalizeEnvKey(name) == normalizedKey {
			return value, true
		}
	}
	return "", false
}

func stringifyEnvValue(value any) (string, bool) {
	switch typed := value.(type) {
	case string:
		return typed, true
	case bool:
		return strconv.FormatBool(typed), true
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64), true
	case int:
		return strconv.Itoa(typed), true
	case json.Number:
		return typed.String(), true
	default:
		return "", false
	}
}

func normalizeEnvKey(raw string) string {
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
