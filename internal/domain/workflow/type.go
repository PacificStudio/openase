package workflow

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxTypeLabelRunes = 64

type TypeLabel string

const (
	TypeCoding        TypeLabel = "coding"
	TypeTest          TypeLabel = "test"
	TypeDoc           TypeLabel = "doc"
	TypeSecurity      TypeLabel = "security"
	TypeDeploy        TypeLabel = "deploy"
	TypeRefineHarness TypeLabel = "refine-harness"
	TypeCustom        TypeLabel = "custom"
)

type Type = TypeLabel

func ParseTypeLabel(raw string) (TypeLabel, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("type must not be empty")
	}
	if utf8.RuneCountInString(trimmed) > maxTypeLabelRunes {
		return "", fmt.Errorf("type must be %d characters or fewer", maxTypeLabelRunes)
	}
	for _, r := range trimmed {
		if unicode.IsControl(r) {
			return "", fmt.Errorf("type must not contain control characters")
		}
	}
	return TypeLabel(trimmed), nil
}

func MustParseTypeLabel(raw string) TypeLabel {
	parsed, err := ParseTypeLabel(raw)
	if err != nil {
		panic(err)
	}
	return parsed
}

func ParseType(raw string) (Type, error) {
	return ParseTypeLabel(raw)
}

func (t TypeLabel) String() string {
	return string(t)
}

func (t TypeLabel) NormalizedKey() string {
	return normalizeSemanticKey(string(t))
}

func normalizeSemanticKey(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range strings.TrimSpace(value) {
		switch {
		case unicode.IsLetter(r), unicode.IsDigit(r):
			builder.WriteRune(unicode.ToLower(r))
		case unicode.IsSpace(r), unicode.IsPunct(r), unicode.IsSymbol(r):
			continue
		default:
			builder.WriteRune(unicode.ToLower(r))
		}
	}
	return builder.String()
}
