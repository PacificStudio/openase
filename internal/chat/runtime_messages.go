package chat

import (
	"encoding/json"
	"regexp"
	"strings"
)

const (
	chatMessageTypeText             = "text"
	chatMessageTypeActionProposal   = "action_proposal"
	chatMessageTypeTaskStarted      = "task_started"
	chatMessageTypeTaskProgress     = "task_progress"
	chatMessageTypeTaskNotification = "task_notification"
)

var codeFencePattern = regexp.MustCompile("(?s)^```(?:json)?\\s*(\\{.*\\})\\s*```$")

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
	if proposal, ok := parseActionProposalText(text); ok {
		return []StreamEvent{{Event: "message", Payload: proposal}}
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

func parseActionProposalText(text string) (map[string]any, bool) {
	trimmed := extractJSONObjectCandidate(text)
	if trimmed == "" {
		return nil, false
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return nil, false
	}
	if strings.TrimSpace(stringValue(payload["type"])) != chatMessageTypeActionProposal {
		return nil, false
	}
	if _, ok := payload["actions"]; !ok {
		return nil, false
	}
	return payload, true
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
