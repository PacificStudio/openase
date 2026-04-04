package workflow

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nikolalohinski/gonja/v2"
)

var gonjaPositionPattern = regexp.MustCompile(`Line:\s*(\d+)\s+Col:\s*(\d+)`)

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
	normalized := normalizeHarnessNewlines(content)
	if strings.TrimSpace(normalized) == "" {
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

	if startsWithHarnessFrontmatter(normalized) {
		return ValidationResult{
			Valid: false,
			Issues: []ValidationIssue{
				{
					Level:   "error",
					Message: "Harness content must be pure Markdown/Gonja body text. YAML frontmatter is no longer supported.",
					Line:    1,
					Column:  1,
				},
			},
		}
	}

	issues := validateHarnessTemplateBody(normalized, 0)

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

func startsWithHarnessFrontmatter(content string) bool {
	normalized := normalizeHarnessNewlines(content)
	lines := strings.Split(normalized, "\n")
	return len(lines) > 0 && strings.TrimSpace(lines[0]) == "---"
}

func validateHarnessTemplateBody(body string, bodyOffset int) []ValidationIssue {
	if strings.TrimSpace(body) == "" {
		return nil
	}

	if _, err := gonja.FromString(body); err != nil {
		issue := ValidationIssue{
			Level:   "error",
			Message: normalizeGonjaError(err.Error()),
		}
		if line, column, ok := extractGonjaPosition(err.Error()); ok {
			issue.Line = line + bodyOffset
			issue.Column = column
		}
		return []ValidationIssue{issue}
	}

	return nil
}

func extractGonjaPosition(message string) (int, int, bool) {
	matches := gonjaPositionPattern.FindStringSubmatch(message)
	if len(matches) != 3 {
		return 0, 0, false
	}

	line := 0
	column := 0
	if _, err := fmt.Sscanf(matches[1], "%d", &line); err != nil {
		return 0, 0, false
	}
	if _, err := fmt.Sscanf(matches[2], "%d", &column); err != nil {
		return 0, 0, false
	}
	if line < 1 || column < 1 {
		return 0, 0, false
	}

	return line, column, true
}

func normalizeGonjaError(message string) string {
	normalized := strings.TrimPrefix(message, "failed to parse template '")
	if normalized != message {
		if index := strings.Index(normalized, "': "); index >= 0 {
			normalized = normalized[index+3:]
		}
	}
	return normalized
}

func normalizeHarnessNewlines(content string) string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(normalized, "\r", "\n")
}
