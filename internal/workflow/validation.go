package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

var yamlLinePattern = regexp.MustCompile(`line (\d+)`)

type ValidationIssue struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues"`
}

func ValidateHarnessContent(content string) ValidationResult {
	if strings.TrimSpace(content) == "" {
		return ValidationResult{
			Valid: false,
			Issues: []ValidationIssue{
				{
					Level:   "error",
					Message: "Harness content must not be empty.",
					Line:    1,
					Column:  1,
				},
			},
		}
	}

	frontmatter, body, err := extractHarnessFrontmatter(content)
	if err != nil {
		return ValidationResult{
			Valid: false,
			Issues: []ValidationIssue{
				{
					Level:   "error",
					Message: err.Error(),
					Line:    1,
					Column:  1,
				},
			},
		}
	}

	issues := validateHarnessFrontmatter(frontmatter)
	if strings.TrimSpace(body) == "" {
		issues = append(issues, ValidationIssue{
			Level:   "warning",
			Message: "Harness body is empty. Add workflow instructions below the YAML frontmatter.",
			Line:    len(strings.Split(frontmatter, "\n")) + 3,
			Column:  1,
		})
	}

	return ValidationResult{
		Valid:  !hasErrorIssue(issues),
		Issues: issues,
	}
}

func validateHarnessForSave(content string) error {
	result := ValidateHarnessContent(content)
	if result.Valid {
		return nil
	}

	for _, issue := range result.Issues {
		if issue.Level != "error" {
			continue
		}
		if issue.Line > 0 && issue.Column > 0 {
			return fmt.Errorf("%w: line %d, column %d: %s", ErrHarnessInvalid, issue.Line, issue.Column, issue.Message)
		}
		if issue.Line > 0 {
			return fmt.Errorf("%w: line %d: %s", ErrHarnessInvalid, issue.Line, issue.Message)
		}
		return fmt.Errorf("%w: %s", ErrHarnessInvalid, issue.Message)
	}

	return fmt.Errorf("%w: harness validation failed", ErrHarnessInvalid)
}

func hasErrorIssue(issues []ValidationIssue) bool {
	for _, issue := range issues {
		if issue.Level == "error" {
			return true
		}
	}

	return false
}

func extractHarnessFrontmatter(content string) (string, string, error) {
	normalized := normalizeHarnessNewlines(content)
	lines := strings.Split(normalized, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return "", "", fmt.Errorf("Harness must begin with YAML frontmatter delimited by ---")
	}

	for index := 1; index < len(lines); index++ {
		if lines[index] != "---" {
			continue
		}

		frontmatter := strings.Join(lines[1:index], "\n")
		body := strings.Join(lines[index+1:], "\n")
		if strings.TrimSpace(frontmatter) == "" {
			return "", "", fmt.Errorf("Harness YAML frontmatter must not be empty")
		}

		return frontmatter, body, nil
	}

	return "", "", fmt.Errorf("Harness YAML frontmatter is missing the closing --- delimiter")
}

func validateHarnessFrontmatter(frontmatter string) []ValidationIssue {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &node); err != nil {
		issue := ValidationIssue{
			Level:   "error",
			Message: err.Error(),
		}

		if line, column, ok := extractYAMLPosition(err.Error()); ok {
			issue.Line = line + 1
			issue.Column = column
		} else {
			issue.Line = 2
			issue.Column = 1
		}

		return []ValidationIssue{issue}
	}

	return nil
}

func extractYAMLPosition(message string) (int, int, bool) {
	matches := yamlLinePattern.FindStringSubmatch(message)
	if len(matches) != 2 {
		return 0, 0, false
	}

	line := 0
	if _, err := fmt.Sscanf(matches[1], "%d", &line); err != nil {
		return 0, 0, false
	}

	if line < 1 {
		return 0, 0, false
	}

	return line, 1, true
}

func normalizeHarnessNewlines(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(normalized, "\r", "\n")
}
