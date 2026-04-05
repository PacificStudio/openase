package catalog

import (
	"reflect"
	"testing"
)

func TestMachineTransportSemanticsHelpers(t *testing.T) {
	t.Run("reachability and execution enums", func(t *testing.T) {
		if got := MachineReachabilityModeDirectConnect.String(); got != "direct_connect" {
			t.Fatalf("MachineReachabilityModeDirectConnect.String() = %q", got)
		}
		if got := MachineExecutionModeSSHCompat.String(); got != "ssh_compat" {
			t.Fatalf("MachineExecutionModeSSHCompat.String() = %q", got)
		}
		if !MachineReachabilityModeReverseConnect.IsValid() {
			t.Fatal("MachineReachabilityModeReverseConnect should be valid")
		}
		if !MachineExecutionModeWebsocket.IsValid() {
			t.Fatal("MachineExecutionModeWebsocket should be valid")
		}
		if MachineReachabilityMode("bogus").IsValid() {
			t.Fatal("bogus reachability mode should be invalid")
		}
		if MachineExecutionMode("bogus").IsValid() {
			t.Fatal("bogus execution mode should be invalid")
		}
	})

	t.Run("legacy connection mode projections", func(t *testing.T) {
		if got := MachineConnectionModeLocal.ReachabilityMode(); got != MachineReachabilityModeLocal {
			t.Fatalf("local reachability = %q", got)
		}
		if got := MachineConnectionModeWSReverse.ReachabilityMode(); got != MachineReachabilityModeReverseConnect {
			t.Fatalf("ws_reverse reachability = %q", got)
		}
		if got := MachineConnectionMode("mystery").ReachabilityMode(); got != MachineReachabilityModeDirectConnect {
			t.Fatalf("unknown reachability fallback = %q", got)
		}

		if got := MachineConnectionModeLocal.ExecutionMode(); got != MachineExecutionModeLocalProcess {
			t.Fatalf("local execution = %q", got)
		}
		if got := MachineConnectionModeSSH.ExecutionMode(); got != MachineExecutionModeSSHCompat {
			t.Fatalf("ssh execution = %q", got)
		}
		if got := MachineConnectionModeWSListener.ExecutionMode(); got != MachineExecutionModeWebsocket {
			t.Fatalf("ws_listener execution = %q", got)
		}
		if got := MachineConnectionMode("mystery").ExecutionMode(); got != MachineExecutionModeWebsocket {
			t.Fatalf("unknown execution fallback = %q", got)
		}

		if !MachineConnectionModeSSH.RequiresSSHHelper() {
			t.Fatal("ssh mode should require SSH helper")
		}
		if MachineConnectionModeWSListener.RequiresSSHHelper() {
			t.Fatal("ws_listener should not require SSH helper")
		}
	})
}

func TestMachineTransportSemanticParsingAndCompatibility(t *testing.T) {
	t.Run("stored reachability mode parsing", func(t *testing.T) {
		got, err := ParseStoredMachineReachabilityMode("", LocalMachineHost)
		if err != nil || got != MachineReachabilityModeLocal {
			t.Fatalf("ParseStoredMachineReachabilityMode(local blank) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineReachabilityMode("", "builder.example.com")
		if err != nil || got != MachineReachabilityModeDirectConnect {
			t.Fatalf("ParseStoredMachineReachabilityMode(remote blank) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineReachabilityMode(" reverse_connect ", "builder.example.com")
		if err != nil || got != MachineReachabilityModeReverseConnect {
			t.Fatalf("ParseStoredMachineReachabilityMode(reverse_connect) = %q, %v", got, err)
		}
		if _, err := ParseStoredMachineReachabilityMode("bogus", "builder.example.com"); err == nil {
			t.Fatal("ParseStoredMachineReachabilityMode(bogus) expected error")
		}
	})

	t.Run("stored execution mode parsing", func(t *testing.T) {
		got, err := ParseStoredMachineExecutionMode("", LocalMachineHost)
		if err != nil || got != MachineExecutionModeLocalProcess {
			t.Fatalf("ParseStoredMachineExecutionMode(local blank) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineExecutionMode("", "builder.example.com")
		if err != nil || got != MachineExecutionModeSSHCompat {
			t.Fatalf("ParseStoredMachineExecutionMode(remote blank) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineExecutionMode(" websocket ", "builder.example.com")
		if err != nil || got != MachineExecutionModeWebsocket {
			t.Fatalf("ParseStoredMachineExecutionMode(websocket) = %q, %v", got, err)
		}
		if _, err := ParseStoredMachineExecutionMode("bogus", "builder.example.com"); err == nil {
			t.Fatal("ParseStoredMachineExecutionMode(bogus) expected error")
		}
	})

	t.Run("connection mode resolution", func(t *testing.T) {
		connectionMode, reachabilityMode, executionMode, err := ResolveMachineConnectionMode(
			"",
			"",
			"",
			LocalMachineHost,
		)
		if err != nil {
			t.Fatalf("ResolveMachineConnectionMode(legacy local fallback) error = %v", err)
		}
		if connectionMode != MachineConnectionModeLocal ||
			reachabilityMode != MachineReachabilityModeLocal ||
			executionMode != MachineExecutionModeLocalProcess {
			t.Fatalf(
				"ResolveMachineConnectionMode(legacy local fallback) = %q %q %q",
				connectionMode,
				reachabilityMode,
				executionMode,
			)
		}

		connectionMode, reachabilityMode, executionMode, err = ResolveMachineConnectionMode(
			"",
			"direct_connect",
			"websocket",
			"builder.example.com",
		)
		if err != nil {
			t.Fatalf("ResolveMachineConnectionMode(direct websocket) error = %v", err)
		}
		if connectionMode != MachineConnectionModeWSListener ||
			reachabilityMode != MachineReachabilityModeDirectConnect ||
			executionMode != MachineExecutionModeWebsocket {
			t.Fatalf(
				"ResolveMachineConnectionMode(direct websocket) = %q %q %q",
				connectionMode,
				reachabilityMode,
				executionMode,
			)
		}

		connectionMode, reachabilityMode, executionMode, err = ResolveMachineConnectionMode(
			"ssh",
			"direct_connect",
			"ssh_compat",
			"builder.example.com",
		)
		if err != nil {
			t.Fatalf("ResolveMachineConnectionMode(ssh compat) error = %v", err)
		}
		if connectionMode != MachineConnectionModeSSH ||
			reachabilityMode != MachineReachabilityModeDirectConnect ||
			executionMode != MachineExecutionModeSSHCompat {
			t.Fatalf(
				"ResolveMachineConnectionMode(ssh compat) = %q %q %q",
				connectionMode,
				reachabilityMode,
				executionMode,
			)
		}

		if _, _, _, err := ResolveMachineConnectionMode(
			"ws_reverse",
			"direct_connect",
			"websocket",
			"builder.example.com",
		); err == nil {
			t.Fatal("ResolveMachineConnectionMode(mismatched legacy mode) expected error")
		}
		if _, _, _, err := ResolveMachineConnectionMode(
			"",
			"bogus",
			"websocket",
			"builder.example.com",
		); err == nil {
			t.Fatal("ResolveMachineConnectionMode(bogus reachability) expected error")
		}
		if _, _, _, err := ResolveMachineConnectionMode(
			"bogus",
			"",
			"",
			"builder.example.com",
		); err == nil {
			t.Fatal("ResolveMachineConnectionMode(bogus legacy-only mode) expected error")
		}
		if _, _, _, err := ResolveMachineConnectionMode(
			"",
			"direct_connect",
			"bogus",
			"builder.example.com",
		); err == nil {
			t.Fatal("ResolveMachineConnectionMode(bogus execution) expected error")
		}
		if _, _, _, err := ResolveMachineConnectionMode(
			"",
			"local",
			"websocket",
			LocalMachineHost,
		); err == nil {
			t.Fatal("ResolveMachineConnectionMode(local websocket) expected semantic mismatch error")
		}
		if _, _, _, err := ResolveMachineConnectionMode(
			"bogus",
			"direct_connect",
			"websocket",
			"builder.example.com",
		); err == nil {
			t.Fatal("ResolveMachineConnectionMode(bogus legacy override) expected error")
		}
	})

	t.Run("semantic compatibility matrix", func(t *testing.T) {
		got, err := machineConnectionModeFromSemantics(
			MachineReachabilityModeLocal,
			MachineExecutionModeLocalProcess,
		)
		if err != nil || got != MachineConnectionModeLocal {
			t.Fatalf("machineConnectionModeFromSemantics(local) = %q, %v", got, err)
		}

		got, err = machineConnectionModeFromSemantics(
			MachineReachabilityModeDirectConnect,
			MachineExecutionModeSSHCompat,
		)
		if err != nil || got != MachineConnectionModeSSH {
			t.Fatalf("machineConnectionModeFromSemantics(direct ssh_compat) = %q, %v", got, err)
		}

		got, err = machineConnectionModeFromSemantics(
			MachineReachabilityModeReverseConnect,
			MachineExecutionModeWebsocket,
		)
		if err != nil || got != MachineConnectionModeWSReverse {
			t.Fatalf("machineConnectionModeFromSemantics(reverse websocket) = %q, %v", got, err)
		}

		if _, err := machineConnectionModeFromSemantics(
			MachineReachabilityModeLocal,
			MachineExecutionModeWebsocket,
		); err == nil {
			t.Fatal("machineConnectionModeFromSemantics(local websocket) expected error")
		}
		if _, err := machineConnectionModeFromSemantics(
			MachineReachabilityModeDirectConnect,
			MachineExecutionModeLocalProcess,
		); err == nil {
			t.Fatal("machineConnectionModeFromSemantics(direct local_process) expected error")
		}
		if _, err := machineConnectionModeFromSemantics(
			MachineReachabilityModeReverseConnect,
			MachineExecutionModeSSHCompat,
		); err == nil {
			t.Fatal("machineConnectionModeFromSemantics(reverse ssh_compat) expected error")
		}
		if _, err := machineConnectionModeFromSemantics(
			MachineReachabilityMode("bogus"),
			MachineExecutionModeWebsocket,
		); err == nil {
			t.Fatal("machineConnectionModeFromSemantics(bogus reachability) expected error")
		}
	})

	t.Run("transport capabilities remain truthful", func(t *testing.T) {
		got, err := ParseStoredMachineTransportCapabilities(nil, MachineConnectionModeWSReverse)
		if err != nil {
			t.Fatalf("ParseStoredMachineTransportCapabilities(ws_reverse) error = %v", err)
		}
		want := []MachineTransportCapability{
			MachineTransportCapabilityProbe,
			MachineTransportCapabilityWorkspacePrepare,
			MachineTransportCapabilityArtifactSync,
			MachineTransportCapabilityProcessStreaming,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ParseStoredMachineTransportCapabilities(ws_reverse) = %+v, want %+v", got, want)
		}
	})
}
