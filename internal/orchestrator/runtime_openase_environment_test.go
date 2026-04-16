package orchestrator

import (
	"testing"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func TestBuildMachineOpenASEEnvironmentRemoteFallback(t *testing.T) {
	t.Parallel()

	sshUser := "agentuser"
	environment, err := buildMachineOpenASEEnvironment(catalogdomain.Machine{
		Host:    "listener.internal",
		SSHUser: &sshUser,
	}, true, []string{"PATH=/usr/bin"})
	if err != nil {
		t.Fatalf("buildMachineOpenASEEnvironment() error = %v", err)
	}
	if !containsEnvironmentPrefix(environment, "OPENASE_REAL_BIN=/home/agentuser/.openase/bin/openase") {
		t.Fatalf("expected remote OPENASE_REAL_BIN in environment, got %+v", environment)
	}
}
