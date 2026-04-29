// Package machinesetup holds the neutral contract between the HTTP API and
// the CLI-side SSH bootstrap implementation. The interface plus the DTO
// types let httpapi trigger a bootstrap without importing the cli package
// (which would create a cycle because cli already imports httpapi for its
// outbound API client).
package machinesetup

import (
	"context"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

// BootstrapInput captures everything the SSH bootstrap runner needs that
// isn't part of the machine record. All fields are optional — empty values
// fall back to defaults defined by the implementation.
type BootstrapInput struct {
	Topology            string
	ListenerAddress     string
	ListenerPath        string
	ListenerBearerToken string
	ControlPlaneURL     string
	TokenTTLSeconds     int
}

// BootstrapResult mirrors the CLI's machine_ssh_helper output so the
// frontend can render service manager, connection target, and retry advice
// without a second round-trip.
type BootstrapResult struct {
	MachineID        string   `json:"machine_id"`
	MachineName      string   `json:"machine_name"`
	Topology         string   `json:"topology"`
	ServiceManager   string   `json:"service_manager"`
	ServiceName      string   `json:"service_name"`
	ServiceStatus    string   `json:"service_status"`
	ConnectionTarget string   `json:"connection_target"`
	RemoteHome       string   `json:"remote_home"`
	RemoteBinaryPath string   `json:"remote_binary_path"`
	EnvironmentFile  string   `json:"environment_file"`
	ServiceFile      string   `json:"service_file"`
	TokenID          string   `json:"token_id,omitempty"`
	Commands         []string `json:"commands"`
	RetryAdvice      []string `json:"retry_advice,omitempty"`
	RollbackAdvice   []string `json:"rollback_advice,omitempty"`
	Summary          string   `json:"summary"`
}

// Bootstrapper runs `openase machine ssh-bootstrap` equivalents in-process.
// The concrete implementation lives in the cli package to keep all of the
// SSH helper internals in one spot.
type Bootstrapper interface {
	Bootstrap(ctx context.Context, machine catalogdomain.Machine, input BootstrapInput) (BootstrapResult, error)
}

// BootstrapperFactory is the signature the cli package registers via
// RegisterBootstrapperFactory. Taking opaque interface values avoids pulling
// the ssh and machine-channel packages into this contracts module (and
// therefore into everything that imports it).
type BootstrapperFactory func(pool any, channel any) Bootstrapper

var registeredFactory BootstrapperFactory

// RegisterBootstrapperFactory is called by the cli package during init so
// that the serve layer can construct a Bootstrapper without importing cli.
func RegisterBootstrapperFactory(factory BootstrapperFactory) {
	registeredFactory = factory
}

// NewBootstrapper returns a Bootstrapper built from the registered factory.
// Returns nil when the factory has not been registered yet; callers should
// treat nil as "ssh bootstrap unavailable" and surface a 503.
func NewBootstrapper(pool any, channel any) Bootstrapper {
	if registeredFactory == nil {
		return nil
	}
	return registeredFactory(pool, channel)
}
