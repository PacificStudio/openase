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

	if got := ResolveMachineOpenASEBinaryPath(Machine{Host: LocalMachineHost}); got != nil {
		t.Fatalf("ResolveMachineOpenASEBinaryPath(local) = %v, want nil", got)
	}

	blankUser := "   "
	if got := ResolveMachineOpenASEBinaryPath(Machine{Host: "listener.internal", SSHUser: &blankUser}); got != nil {
		t.Fatalf("ResolveMachineOpenASEBinaryPath(blank user) = %v, want nil", got)
	}

	if got := ResolveMachineOpenASEBinaryPath(Machine{
		Host: "listener.internal",
		Resources: map[string]any{
			"machine_channel": "invalid",
		},
		SSHUser: &sshUser,
	}); got == nil || *got != "/home/agentuser/.openase/bin/openase" {
		t.Fatalf("ResolveMachineOpenASEBinaryPath(invalid telemetry) = %v", got)
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

	got = UpsertMachineEnvironmentValue([]string{"PATH=/usr/bin"}, "   ", "/ignored")
	if len(got) != 1 || got[0] != "PATH=/usr/bin" {
		t.Fatalf("UpsertMachineEnvironmentValue(blank key) = %+v", got)
	}

	if value, ok := lookupMachineEnvironmentValue([]string{"PATH=/usr/bin"}, "   "); ok || value != "" {
		t.Fatalf("lookupMachineEnvironmentValue(blank key) = %q, %t", value, ok)
	}

	if value, ok := lookupMachineEnvironmentValue([]string{"PATH=/usr/bin"}, "OPENASE_REAL_BIN"); ok || value != "" {
		t.Fatalf("lookupMachineEnvironmentValue(missing) = %q, %t", value, ok)
	}

	if got := machineChannelOpenASEBinaryPath(map[string]any{}); got != "" {
		t.Fatalf("machineChannelOpenASEBinaryPath(empty) = %q", got)
	}
	if got := machineChannelOpenASEBinaryPath(map[string]any{"other": "value"}); got != "" {
		t.Fatalf("machineChannelOpenASEBinaryPath(missing channel) = %q", got)
	}
	if got := machineChannelOpenASEBinaryPath(map[string]any{"machine_channel": map[string]any{}}); got != "" {
		t.Fatalf("machineChannelOpenASEBinaryPath(missing field) = %q", got)
	}
}
