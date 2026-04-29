package catalog

import (
	"testing"

	"github.com/google/uuid"
)

func TestParseCreateMachineRejectsInvalidAgentCLIPaths(t *testing.T) {
	t.Parallel()

	_, err := ParseCreateMachine(uuid.New(), MachineInput{
		Name:          "gpu-01",
		Host:          "10.0.1.10",
		AgentCLIPaths: map[string]string{"bad-adapter": "/bin/sh"},
	})
	if err == nil {
		t.Fatal("ParseCreateMachine() expected agent_cli_paths validation error")
	}
}
