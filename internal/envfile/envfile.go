package envfile

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Upsert merges simple KEY=VALUE assignments into an env file while preserving
// unrelated lines and existing ordering where possible.
func Upsert(path string, updates map[string]string) error {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return fmt.Errorf("env file path must not be empty")
	}
	filtered := filteredUpdates(updates)
	if len(filtered) == 0 {
		return nil
	}

	existing := ""
	content, err := os.ReadFile(trimmedPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("read env file %q: %w", trimmedPath, err)
		}
	} else {
		existing = string(content)
	}

	rendered := mergeAssignments(existing, filtered)
	if err := os.MkdirAll(filepath.Dir(trimmedPath), 0o750); err != nil {
		return fmt.Errorf("create env file parent for %q: %w", trimmedPath, err)
	}
	if err := os.WriteFile(trimmedPath, []byte(rendered), 0o600); err != nil {
		return fmt.Errorf("write env file %q: %w", trimmedPath, err)
	}
	return nil
}

// NormalizePath removes empty and duplicate PATH entries while preserving order.
func NormalizePath(raw string) string {
	parts := strings.Split(raw, string(os.PathListSeparator))
	seen := make(map[string]struct{}, len(parts))
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	return strings.Join(normalized, string(os.PathListSeparator))
}

func filteredUpdates(updates map[string]string) map[string]string {
	filtered := make(map[string]string, len(updates))
	for key, value := range updates {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		if strings.TrimSpace(value) == "" {
			continue
		}
		filtered[trimmedKey] = value
	}
	return filtered
}

func mergeAssignments(existing string, updates map[string]string) string {
	pending := make(map[string]string, len(updates))
	for key, value := range updates {
		pending[key] = value
	}

	lines := strings.Split(existing, "\n")
	output := make([]string, 0, len(lines)+len(updates))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if line != "" {
				output = append(output, line)
			}
			continue
		}
		if strings.HasPrefix(trimmed, "#") {
			output = append(output, line)
			continue
		}
		key, _, ok := strings.Cut(line, "=")
		if !ok {
			output = append(output, line)
			continue
		}
		trimmedKey := strings.TrimSpace(key)
		if value, exists := pending[trimmedKey]; exists {
			output = append(output, formatAssignment(trimmedKey, value))
			delete(pending, trimmedKey)
			continue
		}
		output = append(output, line)
	}

	for _, key := range pendingKeys(pending) {
		output = append(output, formatAssignment(key, pending[key]))
	}
	return strings.Join(output, "\n") + "\n"
}

func pendingKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return assignmentPriority(keys[i]) < assignmentPriority(keys[j])
	})
	return keys
}

func assignmentPriority(key string) string {
	switch key {
	case "OPENASE_AUTH_TOKEN":
		return "00_" + key
	case "PATH":
		return "01_" + key
	default:
		return "99_" + key
	}
}

func formatAssignment(key string, value string) string {
	return key + "=" + value
}
