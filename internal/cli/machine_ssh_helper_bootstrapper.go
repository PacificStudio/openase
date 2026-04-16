package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	catalogdomain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	machinechanneldomain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	machinechannelservice "github.com/BetterAndBetterII/openase/internal/machinechannel"
	"github.com/BetterAndBetterII/openase/internal/machinesetup"
	"github.com/google/uuid"
)

// NewSSHBootstrapper wires the CLI's in-process SSH bootstrap runner behind
// the machinesetup.Bootstrapper contract so httpapi can trigger bootstrap
// without importing the cli package.
func NewSSHBootstrapper(pool *sshinfra.Pool, channel *machinechannelservice.Service) machinesetup.Bootstrapper {
	return &sshBootstrapper{pool: pool, channel: channel}
}

func init() {
	machinesetup.RegisterBootstrapperFactory(func(pool any, channel any) machinesetup.Bootstrapper {
		typedPool, _ := pool.(*sshinfra.Pool)
		typedChannel, _ := channel.(*machinechannelservice.Service)
		if typedPool == nil || typedChannel == nil {
			return nil
		}
		return NewSSHBootstrapper(typedPool, typedChannel)
	})
}

type sshBootstrapper struct {
	pool    *sshinfra.Pool
	channel *machinechannelservice.Service
}

func (b *sshBootstrapper) Bootstrap(
	ctx context.Context,
	machine catalogdomain.Machine,
	input machinesetup.BootstrapInput,
) (machinesetup.BootstrapResult, error) {
	if b == nil || b.pool == nil {
		return machinesetup.BootstrapResult{}, fmt.Errorf("ssh bootstrap pool unavailable")
	}
	if b.channel == nil {
		return machinesetup.BootstrapResult{}, fmt.Errorf("machine channel service unavailable")
	}

	topology, err := resolveMachineSSHBootstrapTopology(machine, input.Topology)
	if err != nil {
		return machinesetup.BootstrapResult{}, err
	}

	ttl := 24 * time.Hour
	if input.TokenTTLSeconds > 0 {
		ttl = time.Duration(input.TokenTTLSeconds) * time.Second
	}

	controlPlaneURL := strings.TrimSpace(input.ControlPlaneURL)

	deps := machineSSHBootstrapDeps{
		getClient: func(ctx context.Context, m catalogdomain.Machine) (sshinfra.Client, error) {
			return b.pool.Get(ctx, m)
		},
		issueToken: func(ctx context.Context, machineID uuid.UUID, tokenTTL time.Duration, explicitURL, rawToken string) (machineChannelTokenResponse, error) {
			return b.issueChannelToken(ctx, machineID, tokenTTL, firstNonEmpty(strings.TrimSpace(explicitURL), controlPlaneURL), rawToken)
		},
		readLocalFile:     os.ReadFile,
		resolveExecutable: os.Executable,
	}

	result, err := runMachineSSHBootstrap(ctx, deps, machineSSHBootstrapInput{
		Machine:             machine,
		Topology:            topology,
		TokenTTL:            ttl,
		ControlPlaneURL:     controlPlaneURL,
		ListenerAddress:     strings.TrimSpace(input.ListenerAddress),
		ListenerPath:        strings.TrimSpace(input.ListenerPath),
		ListenerBearerToken: strings.TrimSpace(input.ListenerBearerToken),
	})
	if err != nil {
		return machinesetup.BootstrapResult{}, err
	}

	return machinesetup.BootstrapResult{
		MachineID:        result.MachineID,
		MachineName:      result.MachineName,
		Topology:         result.Topology,
		ServiceManager:   result.ServiceManager,
		ServiceName:      result.ServiceName,
		ServiceStatus:    result.ServiceStatus,
		ConnectionTarget: result.ConnectionTarget,
		RemoteHome:       result.RemoteHome,
		RemoteBinaryPath: result.RemoteBinaryPath,
		EnvironmentFile:  result.EnvironmentFile,
		ServiceFile:      result.ServiceFile,
		TokenID:          result.TokenID,
		Commands:         append([]string(nil), result.Commands...),
		RetryAdvice:      append([]string(nil), result.RetryAdvice...),
		RollbackAdvice:   append([]string(nil), result.RollbackAdvice...),
		Summary:          result.Summary,
	}, nil
}

// issueChannelToken issues a fresh machine channel token via the local
// machinechannel service. Callers may still pass a rawToken to reuse an
// existing credential, mirroring the CLI code path — in that case we only
// normalize the control-plane URL into the response envelope.
func (b *sshBootstrapper) issueChannelToken(
	ctx context.Context,
	machineID uuid.UUID,
	ttl time.Duration,
	controlPlaneURL string,
	rawToken string,
) (machineChannelTokenResponse, error) {
	controlPlaneURL = strings.TrimSpace(controlPlaneURL)

	trimmed := strings.TrimSpace(rawToken)
	if trimmed != "" {
		if controlPlaneURL == "" {
			return machineChannelTokenResponse{}, fmt.Errorf("control_plane_url is required when reusing an existing channel token")
		}
		return machineChannelTokenResponse{
			Token:           trimmed,
			MachineID:       machineID.String(),
			ControlPlaneURL: controlPlaneURL,
			Environment: map[string]string{
				machinechanneldomain.EnvMachineID:              machineID.String(),
				machinechanneldomain.EnvMachineChannelToken:    trimmed,
				machinechanneldomain.EnvMachineControlPlaneURL: controlPlaneURL,
			},
		}, nil
	}

	issued, err := b.channel.IssueToken(ctx, machinechanneldomain.IssueInput{MachineID: machineID, TTL: ttl})
	if err != nil {
		return machineChannelTokenResponse{}, err
	}

	return machineChannelTokenResponse{
		Token:           issued.Token,
		TokenID:         issued.TokenID.String(),
		MachineID:       machineID.String(),
		ExpiresAt:       issued.ExpiresAt.UTC().Format(time.RFC3339),
		ControlPlaneURL: controlPlaneURL,
		Environment: map[string]string{
			machinechanneldomain.EnvMachineID:              machineID.String(),
			machinechanneldomain.EnvMachineChannelToken:    issued.Token,
			machinechanneldomain.EnvMachineControlPlaneURL: controlPlaneURL,
		},
	}, nil
}
