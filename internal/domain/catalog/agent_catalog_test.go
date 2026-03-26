package catalog

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseCreateAgentProviderRequiresMachineID(t *testing.T) {
	_, err := ParseCreateAgentProvider(uuid.New(), AgentProviderInput{
		Name:        "Codex",
		AdapterType: "codex-app-server",
		ModelName:   "gpt-5.4",
	})
	if err == nil || err.Error() != "machine_id must not be empty" {
		t.Fatalf("expected machine_id validation error, got %v", err)
	}
}

func TestParseCreateAgentProviderParsesMachineID(t *testing.T) {
	machineID := uuid.New()
	input, err := ParseCreateAgentProvider(uuid.New(), AgentProviderInput{
		MachineID:   machineID.String(),
		Name:        "Codex",
		AdapterType: "codex-app-server",
		CliCommand:  "codex",
		ModelName:   "gpt-5.4",
	})
	if err != nil {
		t.Fatalf("ParseCreateAgentProvider returned error: %v", err)
	}
	if input.MachineID != machineID {
		t.Fatalf("expected machine_id %s, got %s", machineID, input.MachineID)
	}
}
