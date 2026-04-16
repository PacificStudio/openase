package catalog

import "testing"

func TestResolveMachineOpenASEBinaryPath(t *testing.T) {
	t.Parallel()

	sshUser := "agentuser"
	explicit := ResolveMachineOpenASEBinaryPath(Machine{
		Host:    "listener.internal",
		SSHUser: &sshUser,
		EnvVars: []string{"OPENASE_REAL_BIN=/opt/openase/bin/openase"},
	})
	if explicit == nil || *explicit != "/opt/openase/bin/openase" {
		t.Fatalf("ResolveMachineOpenASEBinaryPath(explicit) = %v", explicit)
	}

	telemetry := ResolveMachineOpenASEBinaryPath(Machine{
		Host:    "listener.internal",
		SSHUser: &sshUser,
		Resources: map[string]any{
			"machine_channel": map[string]any{
				"openase_binary_path": "/srv/openase/bin/openase",
			},
		},
	})
	if telemetry == nil || *telemetry != "/srv/openase/bin/openase" {
		t.Fatalf("ResolveMachineOpenASEBinaryPath(telemetry) = %v", telemetry)
	}

	fallback := ResolveMachineOpenASEBinaryPath(Machine{
		Host:    "listener.internal",
		SSHUser: &sshUser,
	})
	if fallback == nil || *fallback != "/home/agentuser/.openase/bin/openase" {
		t.Fatalf("ResolveMachineOpenASEBinaryPath(fallback) = %v", fallback)
	}
}

func TestUpsertMachineEnvironmentValue(t *testing.T) {
	t.Parallel()

	got := UpsertMachineEnvironmentValue([]string{"PATH=/usr/bin", "OPENASE_REAL_BIN="}, "OPENASE_REAL_BIN", "/home/agentuser/.openase/bin/openase")
	if len(got) != 2 || got[1] != "OPENASE_REAL_BIN=/home/agentuser/.openase/bin/openase" {
		t.Fatalf("UpsertMachineEnvironmentValue(replace) = %+v", got)
	}

	got = UpsertMachineEnvironmentValue([]string{"PATH=/usr/bin"}, "OPENASE_REAL_BIN", "/home/agentuser/.openase/bin/openase")
	if len(got) != 2 || got[1] != "OPENASE_REAL_BIN=/home/agentuser/.openase/bin/openase" {
		t.Fatalf("UpsertMachineEnvironmentValue(append) = %+v", got)
	}
}
