package machinetransport

import (
	"testing"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

func TestResolverResolveSelectsTransportByMachineMode(t *testing.T) {
	resolver := NewResolver(nil, nil)

	tests := []struct {
		name     string
		machine  domain.Machine
		wantMode domain.MachineConnectionMode
		wantType any
	}{
		{
			name: "local mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           domain.LocalMachineName,
				Host:           domain.LocalMachineHost,
				ConnectionMode: domain.MachineConnectionModeLocal,
			},
			wantMode: domain.MachineConnectionModeLocal,
			wantType: localTransport{},
		},
		{
			name: "ssh mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           "builder-01",
				Host:           "10.0.1.8",
				ConnectionMode: domain.MachineConnectionModeSSH,
			},
			wantMode: domain.MachineConnectionModeSSH,
			wantType: sshTransport{},
		},
		{
			name: "ws reverse mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           "daemon-01",
				Host:           "reverse.example.com",
				ConnectionMode: domain.MachineConnectionModeWSReverse,
			},
			wantMode: domain.MachineConnectionModeWSReverse,
			wantType: websocketTransport{},
		},
		{
			name: "ws listener mode",
			machine: domain.Machine{
				ID:             uuid.New(),
				Name:           "listener-01",
				Host:           "listener.example.com",
				ConnectionMode: domain.MachineConnectionModeWSListener,
			},
			wantMode: domain.MachineConnectionModeWSListener,
			wantType: websocketTransport{},
		},
		{
			name: "legacy local machine infers local mode",
			machine: domain.Machine{
				ID:   uuid.New(),
				Name: domain.LocalMachineName,
				Host: domain.LocalMachineHost,
			},
			wantMode: domain.MachineConnectionModeLocal,
			wantType: localTransport{},
		},
		{
			name: "legacy remote machine defaults to ssh mode",
			machine: domain.Machine{
				ID:   uuid.New(),
				Name: "builder-legacy",
				Host: "10.0.9.9",
			},
			wantMode: domain.MachineConnectionModeSSH,
			wantType: sshTransport{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport, err := resolver.Resolve(tt.machine)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if transport.Mode() != tt.wantMode {
				t.Fatalf("Resolve().Mode() = %q, want %q", transport.Mode(), tt.wantMode)
			}
			switch tt.wantType.(type) {
			case localTransport:
				if _, ok := transport.(localTransport); !ok {
					t.Fatalf("Resolve() type = %T, want localTransport", transport)
				}
			case sshTransport:
				if _, ok := transport.(sshTransport); !ok {
					t.Fatalf("Resolve() type = %T, want sshTransport", transport)
				}
			case websocketTransport:
				if _, ok := transport.(websocketTransport); !ok {
					t.Fatalf("Resolve() type = %T, want websocketTransport", transport)
				}
			default:
				t.Fatalf("unexpected test transport type %T", tt.wantType)
			}
		})
	}
}
