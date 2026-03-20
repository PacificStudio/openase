package workflow

import (
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

type PlatformAccess struct {
	Configured bool
	Allowed    []string
}

func ParsePlatformAccess(content string) (PlatformAccess, error) {
	frontmatter, _, err := extractHarnessFrontmatter(content)
	if err != nil {
		return PlatformAccess{}, fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}

	var document struct {
		PlatformAccess *struct {
			Allowed []string `yaml:"allowed"`
		} `yaml:"platform_access"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &document); err != nil {
		return PlatformAccess{}, fmt.Errorf("%w: %s", ErrHarnessInvalid, err)
	}
	if document.PlatformAccess == nil {
		return PlatformAccess{}, nil
	}

	allowed, err := parseNonEmptyStringList(document.PlatformAccess.Allowed, "platform_access.allowed")
	if err != nil {
		return PlatformAccess{}, err
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
