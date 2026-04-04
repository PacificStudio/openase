package workflow

import (
	"fmt"
	"strings"
)

type PlatformAccess struct {
	Configured bool
	Allowed    []string
}

func ParsePlatformAccess(content string) (PlatformAccess, error) {
	allowed, err := parseNonEmptyStringList(strings.FieldsFunc(content, func(r rune) bool {
		return r == '\n' || r == ','
	}), "platform_access_allowed")
	if err != nil {
		return PlatformAccess{}, err
	}
	if len(allowed) == 0 {
		return PlatformAccess{}, nil
	}
	return PlatformAccess{
		Configured: true,
		Allowed:    allowed,
	}, nil
}

func parseNonEmptyStringList(raw []string, field string) ([]string, error) {
	parsed := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: %s must not contain empty values", ErrHarnessInvalid, field)
		}
		if !slicesContainsString(parsed, trimmed) {
			parsed = append(parsed, trimmed)
		}
	}
	return parsed, nil
}

func slicesContainsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
