package chatconversation

import (
	"fmt"
	"regexp"
	"strings"
)

const MaxConversationTitleRunes = 120

var conversationTitleWhitespacePattern = regexp.MustCompile(`\s+`)

type ConversationTitle string

func ParseConversationTitleFromFirstUserMessage(raw string) (ConversationTitle, error) {
	normalized := normalizeConversationTitleWhitespace(raw)
	if normalized == "" {
		return "", fmt.Errorf("%w: first user message must not be empty", ErrInvalidInput)
	}

	title := firstConversationSentence(normalized)
	if title == "" {
		title = firstNonEmptyConversationTitleLine(raw)
	}
	title = normalizeConversationTitleWhitespace(title)

	runes := []rune(title)
	if len(runes) > MaxConversationTitleRunes {
		title = strings.TrimSpace(string(runes[:MaxConversationTitleRunes]))
	}

	return ConversationTitle(title), nil
}

func (t ConversationTitle) String() string {
	return strings.TrimSpace(string(t))
}

func firstNonEmptyConversationTitleLine(raw string) string {
	for _, line := range strings.Split(raw, "\n") {
		normalized := normalizeConversationTitleWhitespace(line)
		if normalized != "" {
			return normalized
		}
	}
	return ""
}

func normalizeConversationTitleWhitespace(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return conversationTitleWhitespacePattern.ReplaceAllString(trimmed, " ")
}

func firstConversationSentence(value string) string {
	if value == "" {
		return ""
	}
	for index, r := range value {
		switch r {
		case '。', '！', '？', '.', '!', '?':
			return value[:index+len(string(r))]
		}
	}
	return ""
}
