package catalog

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const (
	AgentOutputStreamStdout = "stdout"
	AgentOutputStreamStderr = "stderr"
	AgentOutputStreamSystem = "system"
)

type AgentOutputEntry struct {
	ID        uuid.UUID
	TicketID  *uuid.UUID
	EventType string
	Stream    string
	Message   string
	Metadata  map[string]any
	CreatedAt time.Time
}

type AgentOutput struct {
	Agent   Agent
	Entries []AgentOutputEntry
}

type AgentOutputListInput struct {
	Limit string
}

type GetAgentOutput struct {
	AgentID uuid.UUID
	Limit   int
}

func ParseGetAgentOutput(agentID uuid.UUID, raw AgentOutputListInput) (GetAgentOutput, error) {
	limit, err := parseActivityEventLimit(raw.Limit)
	if err != nil {
		return GetAgentOutput{}, err
	}

	return GetAgentOutput{
		AgentID: agentID,
		Limit:   limit,
	}, nil
}

func ParseAgentOutputStream(raw string, fallback string) (string, error) {
	stream := strings.TrimSpace(strings.ToLower(raw))
	if stream == "" {
		stream = strings.TrimSpace(strings.ToLower(fallback))
	}

	switch stream {
	case AgentOutputStreamStdout, AgentOutputStreamStderr, AgentOutputStreamSystem:
		return stream, nil
	default:
		return "", fmt.Errorf("agent output stream must be one of stdout, stderr, system")
	}
}
