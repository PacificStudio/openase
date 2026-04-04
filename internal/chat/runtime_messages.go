package chat

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	chatMessageTypeText             = "text"
	chatMessageTypeDiff             = "diff"
	chatMessageTypeBundleDiff       = "bundle_diff"
	chatMessageTypeTaskStarted      = "task_started"
	chatMessageTypeTaskProgress     = "task_progress"
	chatMessageTypeTaskNotification = "task_notification"
)

var codeFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(\\{.*\\})\\s*```$")
var trailingCodeFencePattern = regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*?\\})\\s*```\\s*$")

type diffPayload struct {
	Type  string     `json:"type"`
	File  string     `json:"file"`
	Hunks []diffHunk `json:"hunks"`
}

type bundleDiffPayload struct {
	Type  string           `json:"type"`
	Files []bundleDiffFile `json:"files"`
}

type bundleDiffFile struct {
	File  string     `json:"file"`
	Hunks []diffHunk `json:"hunks"`
}

type diffHunk struct {
	OldStart int        `json:"old_start"`
	OldLines int        `json:"old_lines"`
	NewStart int        `json:"new_start"`
	NewLines int        `json:"new_lines"`
	Lines    []diffLine `json:"lines"`
}

type diffLine struct {
	Op   string `json:"op"`
	Text string `json:"text"`
}

func newTextMessageEvent(content string) StreamEvent {
	return StreamEvent{
		Event:   "message",
		Payload: textPayload{Type: chatMessageTypeText, Content: content},
	}
}

func newTaskMessageEvent(kind string, raw any) StreamEvent {
	payload := map[string]any{"type": kind}
	if raw != nil {
		payload["raw"] = raw
	}

	return StreamEvent{Event: "message", Payload: payload}
}

func normalizeAssistantText(text string) []StreamEvent {
	if leadingText, payload, ok := extractStructuredAssistantPayload(text); ok {
		events := make([]StreamEvent, 0, 2)
		if trimmed := strings.TrimSpace(leadingText); trimmed != "" {
			events = append(events, newTextMessageEvent(trimmed))
		}
		events = append(events, StreamEvent{Event: "message", Payload: payload})
		return events
	}

	return []StreamEvent{newTextMessageEvent(text)}
}

func extractAssistantTextBlocks(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}

	var message struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil
	}

	items := make([]string, 0, len(message.Content))
	for _, block := range message.Content {
		if block.Type != chatMessageTypeText {
			continue
		}
		text := strings.TrimSpace(block.Text)
		if text == "" {
			continue
		}
		items = append(items, text)
	}
	return items
}

func parseDiffPayloadText(text string) (diffPayload, bool) {
	trimmed := extractJSONObjectCandidate(text)
	if trimmed == "" {
		return diffPayload{}, false
	}

	var payload diffPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return diffPayload{}, false
	}
	if strings.TrimSpace(payload.Type) != chatMessageTypeDiff {
		return diffPayload{}, false
	}
	if strings.TrimSpace(payload.File) == "" || len(payload.Hunks) == 0 {
		return diffPayload{}, false
	}
	for _, hunk := range payload.Hunks {
		if !isValidDiffHunk(hunk) {
			return diffPayload{}, false
		}
	}
	return payload, true
}

func parseBundleDiffPayloadText(text string) (bundleDiffPayload, bool) {
	trimmed := extractJSONObjectCandidate(text)
	if trimmed == "" {
		return bundleDiffPayload{}, false
	}

	var payload bundleDiffPayload
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return bundleDiffPayload{}, false
	}
	if strings.TrimSpace(payload.Type) != chatMessageTypeBundleDiff {
		return bundleDiffPayload{}, false
	}
	if len(payload.Files) == 0 {
		return bundleDiffPayload{}, false
	}
	seenFiles := make(map[string]struct{}, len(payload.Files))
	for _, item := range payload.Files {
		file := strings.TrimSpace(item.File)
		if file == "" {
			return bundleDiffPayload{}, false
		}
		if _, exists := seenFiles[file]; exists {
			return bundleDiffPayload{}, false
		}
		seenFiles[file] = struct{}{}
		if len(item.Hunks) == 0 {
			return bundleDiffPayload{}, false
		}
		for _, hunk := range item.Hunks {
			if !isValidDiffHunk(hunk) {
				return bundleDiffPayload{}, false
			}
		}
	}
	return payload, true
}

func extractStructuredAssistantPayload(text string) (string, any, bool) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", nil, false
	}

	if payload, ok := parseStructuredPayloadCandidate(trimmed); ok {
		return "", payload, true
	}

	if matches := trailingCodeFencePattern.FindAllStringSubmatchIndex(trimmed, -1); len(matches) > 0 {
		last := matches[len(matches)-1]
		if len(last) >= 4 {
			candidate := strings.TrimSpace(trimmed[last[2]:last[3]])
			if payload, ok := parseStructuredPayloadCandidate(candidate); ok {
				return strings.TrimSpace(trimmed[:last[0]]), payload, true
			}
		}
	}

	if leadingText, payload, ok := extractTrailingStructuredPayloadSequence(trimmed); ok {
		return leadingText, payload, true
	}

	return "", nil, false
}

func parseStructuredPayloadCandidate(text string) (any, bool) {
	if bundleDiff, ok := parseBundleDiffPayloadText(text); ok {
		return bundleDiff, true
	}
	if diff, ok := parseDiffPayloadText(text); ok {
		return diff, true
	}
	return nil, false
}

func extractTrailingStructuredPayloadSequence(text string) (string, any, bool) {
	for start := 0; start < len(text); start++ {
		if text[start] != '{' {
			continue
		}

		payload, next, ok := decodeStructuredPayloadAt(text, start)
		if !ok {
			continue
		}

		fingerprint := structuredPayloadFingerprint(payload)
		position := skipStructuredWhitespace(text, next)
		for position < len(text) {
			nextPayload, nextEnd, nextOK := decodeStructuredPayloadAt(text, position)
			if !nextOK || structuredPayloadFingerprint(nextPayload) != fingerprint {
				ok = false
				break
			}
			payload = nextPayload
			position = skipStructuredWhitespace(text, nextEnd)
		}

		if !ok || position != len(text) {
			continue
		}

		return strings.TrimSpace(text[:start]), payload, true
	}

	return "", nil, false
}

func decodeStructuredPayloadAt(text string, start int) (any, int, bool) {
	decoder := json.NewDecoder(strings.NewReader(text[start:]))
	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return nil, 0, false
	}

	payload, ok := parseStructuredPayloadCandidate(string(raw))
	if !ok {
		return nil, 0, false
	}
	return payload, start + int(decoder.InputOffset()), true
}

func skipStructuredWhitespace(text string, start int) int {
	for start < len(text) {
		switch text[start] {
		case ' ', '\t', '\r', '\n':
			start++
		default:
			return start
		}
	}
	return start
}

func structuredPayloadFingerprint(payload any) string {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%T", payload)
	}
	return string(encoded)
}

func isValidDiffHunk(hunk diffHunk) bool {
	if hunk.OldStart < 1 || hunk.NewStart < 1 || hunk.OldLines < 0 || hunk.NewLines < 0 || len(hunk.Lines) == 0 {
		return false
	}

	oldLineCount := 0
	newLineCount := 0
	for _, line := range hunk.Lines {
		switch strings.TrimSpace(line.Op) {
		case "context":
			oldLineCount++
			newLineCount++
		case "remove":
			oldLineCount++
		case "add":
			newLineCount++
		default:
			return false
		}
	}

	return oldLineCount == hunk.OldLines && newLineCount == hunk.NewLines
}

func extractJSONObjectCandidate(text string) string {
	trimmed := strings.TrimSpace(text)
	if matches := codeFencePattern.FindStringSubmatch(trimmed); len(matches) == 2 {
		return strings.TrimSpace(matches[1])
	}

	return trimmed
}

func decodeRawJSON(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}

	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return string(raw)
	}
	return decoded
}

func stringValue(value any) string {
	typed, _ := value.(string)
	return typed
}
