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
		if got := MachineWebsocketTopologyRemoteListener.String(); got != "remote_listener" {
			t.Fatalf("MachineWebsocketTopologyRemoteListener.String() = %q", got)
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
		if !MachineWebsocketTopologyRemoteListener.IsValid() {
			t.Fatal("MachineWebsocketTopologyRemoteListener should be valid")
		}
		if MachineWebsocketTopology("bogus").IsValid() {
			t.Fatal("bogus websocket topology should be invalid")
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
		if got := MachineConnectionModeSSH.ExecutionMode(); got != MachineExecutionModeWebsocket {
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
	t.Run("stored connection mode parsing", func(t *testing.T) {
		got, err := ParseStoredMachineConnectionMode("", LocalMachineHost)
		if err != nil || got != MachineConnectionModeLocal {
			t.Fatalf("ParseStoredMachineConnectionMode(local blank) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineConnectionMode("", "builder.example.com")
		if err != nil || got != MachineConnectionModeWSListener {
			t.Fatalf("ParseStoredMachineConnectionMode(remote blank) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineConnectionMode(" ssh ", "builder.example.com")
		if err != nil || got != MachineConnectionModeWSListener {
			t.Fatalf("ParseStoredMachineConnectionMode(ssh legacy) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineConnectionMode(" ssh ", LocalMachineHost)
		if err != nil || got != MachineConnectionModeLocal {
			t.Fatalf("ParseStoredMachineConnectionMode(local ssh legacy) = %q, %v", got, err)
		}
		got, err = ParseStoredMachineConnectionMode(" ws_reverse ", "builder.example.com")
		if err != nil || got != MachineConnectionModeWSReverse {
			t.Fatalf("ParseStoredMachineConnectionMode(ws_reverse) = %q, %v", got, err)
		}
		if _, err := ParseStoredMachineConnectionMode("bogus", "builder.example.com"); err == nil {
			t.Fatal("ParseStoredMachineConnectionMode(bogus) expected error")
		}
	})

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
		if err != nil || got != MachineExecutionModeWebsocket {
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

	t.Run("stored transport preserves legacy ssh when semantic columns are absent", func(t *testing.T) {
		connectionMode, reachabilityMode, executionMode, err := ResolveStoredMachineTransport(
			"ssh",
			"",
			"",
			"builder.example.com",
		)
		if err != nil {
			t.Fatalf("ResolveStoredMachineTransport(legacy ssh) error = %v", err)
		}
		if connectionMode != MachineConnectionModeSSH ||
			reachabilityMode != MachineReachabilityModeDirectConnect ||
			executionMode != MachineExecutionModeWebsocket {
			t.Fatalf(
				"ResolveStoredMachineTransport(legacy ssh) = %q %q %q",
				connectionMode,
				reachabilityMode,
				executionMode,
			)
		}

		connectionMode, reachabilityMode, executionMode, err = ResolveStoredMachineTransport(
			"",
			"",
			"",
			"builder.example.com",
		)
		if err != nil {
			t.Fatalf("ResolveStoredMachineTransport(remote blank) error = %v", err)
		}
		if connectionMode != MachineConnectionModeWSListener ||
			reachabilityMode != MachineReachabilityModeDirectConnect ||
			executionMode != MachineExecutionModeWebsocket {
			t.Fatalf(
				"ResolveStoredMachineTransport(remote blank) = %q %q %q",
				connectionMode,
				reachabilityMode,
				executionMode,
			)
		}

		connectionMode, reachabilityMode, executionMode, err = ResolveStoredMachineTransport(
			"ws_listener",
			"",
			"",
			LocalMachineHost,
		)
		if err != nil {
			t.Fatalf("ResolveStoredMachineTransport(local ws_listener legacy) error = %v", err)
		}
		if connectionMode != MachineConnectionModeLocal ||
			reachabilityMode != MachineReachabilityModeLocal ||
			executionMode != MachineExecutionModeLocalProcess {
			t.Fatalf(
				"ResolveStoredMachineTransport(local ws_listener legacy) = %q %q %q",
				connectionMode,
				reachabilityMode,
				executionMode,
			)
		}

		connectionMode, reachabilityMode, executionMode, err = ResolveStoredMachineTransport(
			"",
			"reverse_connect",
			"websocket",
			"builder.example.com",
		)
		if err != nil {
			t.Fatalf("ResolveStoredMachineTransport(reverse websocket semantics) error = %v", err)
		}
		if connectionMode != MachineConnectionModeWSReverse ||
			reachabilityMode != MachineReachabilityModeReverseConnect ||
			executionMode != MachineExecutionModeWebsocket {
			t.Fatalf(
				"ResolveStoredMachineTransport(reverse websocket semantics) = %q %q %q",
				connectionMode,
				reachabilityMode,
				executionMode,
			)
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
			MachineExecutionModeWebsocket,
		)
		if err != nil || got != MachineConnectionModeWSListener {
			t.Fatalf("machineConnectionModeFromSemantics(direct websocket) = %q, %v", got, err)
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
			MachineExecutionModeLocalProcess,
		); err == nil {
			t.Fatal("machineConnectionModeFromSemantics(reverse local_process) expected error")
		}
		if _, err := machineConnectionModeFromSemantics(
			MachineReachabilityMode("bogus"),
			MachineExecutionModeWebsocket,
		); err == nil {
			t.Fatal("machineConnectionModeFromSemantics(bogus reachability) expected error")
		}
	})

	t.Run("websocket topology resolution", func(t *testing.T) {
		got, err := ResolveMachineWebsocketTopology(
			MachineReachabilityModeDirectConnect,
			MachineExecutionModeWebsocket,
		)
		if err != nil || got != MachineWebsocketTopologyRemoteListener {
			t.Fatalf("ResolveMachineWebsocketTopology(direct websocket) = %q, %v", got, err)
		}

		got, err = ResolveMachineWebsocketTopology(
			MachineReachabilityModeReverseConnect,
			MachineExecutionModeWebsocket,
		)
		if err != nil || got != MachineWebsocketTopologyReverseConnect {
			t.Fatalf("ResolveMachineWebsocketTopology(reverse websocket) = %q, %v", got, err)
		}

		got, err = ResolveMachineWebsocketTopology(
			MachineReachabilityModeLocal,
			MachineExecutionModeLocalProcess,
		)
		if err != nil || got != MachineWebsocketTopologyLocalProcess {
			t.Fatalf("ResolveMachineWebsocketTopology(local process) = %q, %v", got, err)
		}

		if _, err := ResolveMachineWebsocketTopology(
			MachineReachabilityModeDirectConnect,
			MachineExecutionModeLocalProcess,
		); err == nil {
			t.Fatal("ResolveMachineWebsocketTopology(direct local_process) expected error")
		}
		if got := MachineWebsocketTopologyReverseConnect.ConnectionMode(); got != MachineConnectionModeWSReverse {
			t.Fatalf("MachineWebsocketTopologyReverseConnect.ConnectionMode() = %q", got)
		}
		if got := MachineWebsocketTopologyLocalProcess.ConnectionMode(); got != MachineConnectionModeLocal {
			t.Fatalf("MachineWebsocketTopologyLocalProcess.ConnectionMode() = %q", got)
		}
		if got := MachineWebsocketTopologyRemoteListener.ConnectionMode(); got != MachineConnectionModeWSListener {
			t.Fatalf("MachineWebsocketTopologyRemoteListener.ConnectionMode() = %q", got)
		}
		if got := MachineWebsocketTopology("bogus").ConnectionMode(); got != MachineConnectionModeWSListener {
			t.Fatalf("MachineWebsocketTopology(bogus).ConnectionMode() = %q", got)
		}
	})

	t.Run("machine websocket topology helper follows semantic source of truth", func(t *testing.T) {
		directMachine := Machine{
			ReachabilityMode: MachineReachabilityModeDirectConnect,
			ExecutionMode:    MachineExecutionModeWebsocket,
		}
		if got, err := directMachine.WebsocketTopology(); err != nil || got != MachineWebsocketTopologyRemoteListener {
			t.Fatalf("directMachine.WebsocketTopology() = %q, %v", got, err)
		}

		reverseMachine := Machine{
			ReachabilityMode: MachineReachabilityModeReverseConnect,
			ExecutionMode:    MachineExecutionModeWebsocket,
		}
		if got, err := reverseMachine.WebsocketTopology(); err != nil || got != MachineWebsocketTopologyReverseConnect {
			t.Fatalf("reverseMachine.WebsocketTopology() = %q, %v", got, err)
		}

		invalidMachine := Machine{
			ReachabilityMode: MachineReachabilityModeDirectConnect,
			ExecutionMode:    MachineExecutionModeLocalProcess,
		}
		if _, err := invalidMachine.WebsocketTopology(); err == nil {
			t.Fatal("invalidMachine.WebsocketTopology() expected error")
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

	t.Run("transport capabilities dedupe and reject unsupported modes", func(t *testing.T) {
		got, err := ParseStoredMachineTransportCapabilities(
			[]string{"probe", "probe", "artifact_sync"},
			MachineConnectionModeSSH,
		)
		if err != nil {
			t.Fatalf("ParseStoredMachineTransportCapabilities(ssh duplicate) error = %v", err)
		}
		want := []MachineTransportCapability{
			MachineTransportCapabilityProbe,
			MachineTransportCapabilityArtifactSync,
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ParseStoredMachineTransportCapabilities(ssh duplicate) = %+v, want %+v", got, want)
		}
		if _, err := ParseStoredMachineTransportCapabilities(
			[]string{"probe"},
			MachineConnectionMode("bogus"),
		); err == nil {
			t.Fatal("ParseStoredMachineTransportCapabilities(bogus mode) expected unsupported-mode error")
		}
	})
}
