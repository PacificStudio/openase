package machinetransport

import (
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestResolverResolveRuntimeSeparatesChannelAndExecutionSurfaces(t *testing.T) {
	resolver := NewResolver(nil, nil)

	tests := []struct {
		name               string
		machine            domain.Machine
		wantMode           domain.MachineConnectionMode
		wantProbe          bool
		wantWorkspace      bool
		wantArtifactSync   bool
		wantProcess        bool
		wantCommandSession bool
		wantRuntime        bool
	}{
		{
			name: "local mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           domain.LocalMachineName,
				Host:           domain.LocalMachineHost,
				ConnectionMode: domain.MachineConnectionModeLocal,
			},
			wantMode:         domain.MachineConnectionModeLocal,
			wantProbe:        true,
			wantWorkspace:    true,
			wantArtifactSync: true,
			wantProcess:      true,
		},
		{
			name: "ssh mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           "builder-01",
				Host:           "10.0.1.8",
				ConnectionMode: domain.MachineConnectionModeSSH,
			},
			wantMode:           domain.MachineConnectionModeSSH,
			wantProbe:          true,
			wantWorkspace:      true,
			wantArtifactSync:   true,
			wantProcess:        true,
			wantCommandSession: true,
		},
		{
			name: "ws reverse mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           "daemon-01",
				Host:           "reverse.example.com",
				ConnectionMode: domain.MachineConnectionModeWSReverse,
			},
			wantMode:           domain.MachineConnectionModeWSReverse,
			wantProbe:          true,
			wantWorkspace:      true,
			wantArtifactSync:   true,
			wantProcess:        true,
			wantCommandSession: true,
			wantRuntime:        true,
		},
		{
			name: "ws listener mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           "listener-01",
				Host:           "listener.example.com",
				ConnectionMode: domain.MachineConnectionModeWSListener,
			},
			wantMode:           domain.MachineConnectionModeWSListener,
			wantProbe:          true,
			wantWorkspace:      true,
			wantArtifactSync:   true,
			wantProcess:        true,
			wantCommandSession: true,
			wantRuntime:        true,
		},
		{
			name: "legacy local machine infers local mode",
			machine: domain.Machine{
				ID:   uuid.New(),
				Name: domain.LocalMachineName,
				Host: domain.LocalMachineHost,
			},
			wantMode:         domain.MachineConnectionModeLocal,
			wantProbe:        true,
			wantWorkspace:    true,
			wantArtifactSync: true,
			wantProcess:      true,
		},
		{
			name: "legacy remote machine defaults to listener mode",
			machine: domain.Machine{
				ID:   uuid.New(),
				Name: "builder-legacy",
				Host: "10.0.9.9",
			},
			wantMode:           domain.MachineConnectionModeWSListener,
			wantProbe:          true,
			wantWorkspace:      true,
			wantArtifactSync:   true,
			wantProcess:        true,
			wantCommandSession: true,
			wantRuntime:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolver.ResolveRuntime(tt.machine)
			if err != nil {
				t.Fatalf("ResolveRuntime() error = %v", err)
			}
			if resolved.Channel == nil {
				t.Fatal("ResolveRuntime().Channel = nil")
			}
			if resolved.Channel.Mode() != tt.wantMode {
				t.Fatalf("ResolveRuntime().Channel.Mode() = %q, want %q", resolved.Channel.Mode(), tt.wantMode)
			}
			if got := resolved.Execution.Probe != nil; got != tt.wantProbe {
				t.Fatalf("ResolveRuntime().Execution.Probe != nil = %t, want %t", got, tt.wantProbe)
			}
			if got := resolved.Execution.Workspace != nil; got != tt.wantWorkspace {
				t.Fatalf("ResolveRuntime().Execution.Workspace != nil = %t, want %t", got, tt.wantWorkspace)
			}
			if got := resolved.Execution.ArtifactSync != nil; got != tt.wantArtifactSync {
				t.Fatalf("ResolveRuntime().Execution.ArtifactSync != nil = %t, want %t", got, tt.wantArtifactSync)
			}
			if got := resolved.Execution.Process != nil; got != tt.wantProcess {
				t.Fatalf("ResolveRuntime().Execution.Process != nil = %t, want %t", got, tt.wantProcess)
			}
			if got := resolved.Execution.CommandSession != nil; got != tt.wantCommandSession {
				t.Fatalf("ResolveRuntime().Execution.CommandSession != nil = %t, want %t", got, tt.wantCommandSession)
			}
			if got := resolved.Execution.Runtime != nil; got != tt.wantRuntime {
				t.Fatalf("ResolveRuntime().Execution.Runtime != nil = %t, want %t", got, tt.wantRuntime)
			}
			if tt.wantRuntime && !resolved.Execution.Runtime.SupportsAll(resolved.Execution.Runtime.Capabilities()...) {
				t.Fatal("ResolveRuntime().Execution.Runtime should report its declared capabilities")
			}
			if got := resolved.ProbeExecutor() != nil; got != (tt.wantProbe || tt.wantRuntime) {
				t.Fatalf("ResolveRuntime().ProbeExecutor() != nil = %t", got)
			}
			if got := resolved.WorkspaceExecutor() != nil; got != (tt.wantWorkspace || tt.wantRuntime) {
				t.Fatalf("ResolveRuntime().WorkspaceExecutor() != nil = %t", got)
			}
			if got := resolved.ArtifactSyncExecutor() != nil; got != (tt.wantArtifactSync || tt.wantRuntime) {
				t.Fatalf("ResolveRuntime().ArtifactSyncExecutor() != nil = %t", got)
			}
			if got := resolved.ProcessExecutor() != nil; got != (tt.wantProcess || tt.wantRuntime) {
				t.Fatalf("ResolveRuntime().ProcessExecutor() != nil = %t", got)
			}
			if got := resolved.CommandSessionExecutor() != nil; got != (tt.wantCommandSession || tt.wantRuntime) {
				t.Fatalf("ResolveRuntime().CommandSessionExecutor() != nil = %t", got)
			}
		})
	}
}
