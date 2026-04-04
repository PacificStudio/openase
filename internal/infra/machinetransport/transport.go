package machinetransport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	sshinfra "github.com/BetterAndBetterII/openase/internal/infra/ssh"
	workspaceinfra "github.com/BetterAndBetterII/openase/internal/infra/workspace"
	"github.com/BetterAndBetterII/openase/internal/logging"
	"github.com/BetterAndBetterII/openase/internal/provider"
)

var _ = logging.DeclareComponent("machine-transport")

var ErrTransportUnavailable = errors.New("machine transport unavailable")

type CommandSession interface {
	CombinedOutput(cmd string) ([]byte, error)
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.Reader, error)
	StderrPipe() (io.Reader, error)
	Start(cmd string) error
	Signal(signal string) error
	Wait() error
	Close() error
}

type SyncArtifactsRequest struct {
	LocalRoot   string
	TargetRoot  string
	Paths       []string
	RemovePaths []string
}

type Transport interface {
	Mode() domain.MachineConnectionMode
	Capabilities(machine domain.Machine) []domain.MachineTransportCapability
	Probe(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error)
	PrepareWorkspace(ctx context.Context, machine domain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error)
	SyncArtifacts(ctx context.Context, machine domain.Machine, request SyncArtifactsRequest) error
	StartProcess(ctx context.Context, machine domain.Machine, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error)
	OpenCommandSession(ctx context.Context, machine domain.Machine) (CommandSession, error)
	SessionState(ctx context.Context, machine domain.Machine) (domain.MachineTransportSessionState, error)
	Heartbeat(ctx context.Context, machine domain.Machine) (domain.MachineDaemonStatus, error)
}

type Resolver struct {
	localProcessManager provider.AgentCLIProcessManager
	sshPool             *sshinfra.Pool
}

func NewResolver(localProcessManager provider.AgentCLIProcessManager, sshPool *sshinfra.Pool) *Resolver {
	return &Resolver{
		localProcessManager: localProcessManager,
		sshPool:             sshPool,
	}
}

func (r *Resolver) Resolve(machine domain.Machine) (Transport, error) {
	if r == nil {
		return nil, fmt.Errorf("%w: resolver is nil", ErrTransportUnavailable)
	}

	mode := effectiveConnectionMode(machine)
	switch mode {
	case domain.MachineConnectionModeLocal:
		return localTransport{processManager: r.localProcessManager}, nil
	case domain.MachineConnectionModeSSH:
		return sshTransport{pool: r.sshPool}, nil
	case domain.MachineConnectionModeWSReverse, domain.MachineConnectionModeWSListener:
		return websocketTransport{mode: mode}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported connection mode %q", ErrTransportUnavailable, mode)
	}
}

type resolvedProcessManager struct {
	transport Transport
	machine   domain.Machine
}

func NewProcessManager(transport Transport, machine domain.Machine) provider.AgentCLIProcessManager {
	switch typed := transport.(type) {
	case localTransport:
		if typed.processManager != nil {
			return typed.processManager
		}
	case sshTransport:
		return sshinfra.NewProcessManager(typed.pool, machine)
	}
	return &resolvedProcessManager{transport: transport, machine: machine}
}

func (m *resolvedProcessManager) Start(ctx context.Context, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	if m == nil || m.transport == nil {
		return nil, fmt.Errorf("%w: process transport unavailable", ErrTransportUnavailable)
	}
	return m.transport.StartProcess(ctx, m.machine, spec)
}

type Tester struct {
	resolver *Resolver
}

func NewTester(resolver *Resolver) *Tester {
	return &Tester{resolver: resolver}
}

func (t *Tester) TestConnection(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error) {
	if t == nil || t.resolver == nil {
		return domain.MachineProbe{}, fmt.Errorf("%w: tester resolver unavailable", ErrTransportUnavailable)
	}
	transport, err := t.resolver.Resolve(machine)
	if err != nil {
		return domain.MachineProbe{}, err
	}
	return transport.Probe(ctx, machine)
}

type MonitorCollector struct {
	resolver     *Resolver
	sshCollector *sshinfra.MonitorCollector
	now          func() time.Time
}

func NewMonitorCollector(resolver *Resolver, sshPool *sshinfra.Pool) *MonitorCollector {
	return &MonitorCollector{
		resolver:     resolver,
		sshCollector: sshinfra.NewMonitorCollector(sshPool),
		now:          time.Now,
	}
}

func (c *MonitorCollector) CollectReachability(ctx context.Context, machine domain.Machine) (domain.MachineReachability, error) {
	transport, err := c.resolve(machine)
	if err != nil {
		checkedAt := c.currentTime()
		return domain.MachineReachability{
			CheckedAt:    checkedAt,
			Transport:    machine.ConnectionMode.String(),
			FailureCause: err.Error(),
		}, err
	}

	switch transport.Mode() {
	case domain.MachineConnectionModeWSReverse, domain.MachineConnectionModeWSListener:
		checkedAt := c.currentTime()
		heartbeat, hbErr := transport.Heartbeat(ctx, machine)
		reachable := heartbeat.Registered && heartbeat.SessionState == domain.MachineTransportSessionStateConnected
		failureCause := ""
		if !reachable {
			failureCause = "machine websocket session is not connected"
		}
		if hbErr != nil {
			failureCause = hbErr.Error()
		}
		return domain.MachineReachability{
			CheckedAt:    checkedAt,
			Transport:    transport.Mode().String(),
			Reachable:    reachable,
			FailureCause: failureCause,
		}, hbErr
	default:
		if c.sshCollector == nil {
			checkedAt := c.currentTime()
			return domain.MachineReachability{
				CheckedAt:    checkedAt,
				Transport:    transport.Mode().String(),
				FailureCause: "machine monitor collector unavailable",
			}, fmt.Errorf("machine monitor collector unavailable")
		}
		return c.sshCollector.CollectReachability(ctx, machine)
	}
}

func (c *MonitorCollector) CollectSystemResources(ctx context.Context, machine domain.Machine) (domain.MachineSystemResources, error) {
	mode := effectiveConnectionMode(machine)
	if mode == domain.MachineConnectionModeWSReverse || mode == domain.MachineConnectionModeWSListener {
		return domain.MachineSystemResources{}, fmt.Errorf("%w: system resource collection is not implemented for %s", ErrTransportUnavailable, mode)
	}
	if c == nil || c.sshCollector == nil {
		return domain.MachineSystemResources{}, fmt.Errorf("machine monitor collector unavailable")
	}
	return c.sshCollector.CollectSystemResources(ctx, machine)
}

func (c *MonitorCollector) CollectGPUResources(ctx context.Context, machine domain.Machine) (domain.MachineGPUResources, error) {
	mode := effectiveConnectionMode(machine)
	if mode == domain.MachineConnectionModeWSReverse || mode == domain.MachineConnectionModeWSListener {
		return domain.MachineGPUResources{}, fmt.Errorf("%w: gpu resource collection is not implemented for %s", ErrTransportUnavailable, mode)
	}
	if c == nil || c.sshCollector == nil {
		return domain.MachineGPUResources{}, fmt.Errorf("machine monitor collector unavailable")
	}
	return c.sshCollector.CollectGPUResources(ctx, machine)
}

func (c *MonitorCollector) CollectAgentEnvironment(ctx context.Context, machine domain.Machine) (domain.MachineAgentEnvironment, error) {
	mode := effectiveConnectionMode(machine)
	if mode == domain.MachineConnectionModeWSReverse || mode == domain.MachineConnectionModeWSListener {
		return domain.MachineAgentEnvironment{}, fmt.Errorf("%w: agent environment collection is not implemented for %s", ErrTransportUnavailable, mode)
	}
	if c == nil || c.sshCollector == nil {
		return domain.MachineAgentEnvironment{}, fmt.Errorf("machine monitor collector unavailable")
	}
	return c.sshCollector.CollectAgentEnvironment(ctx, machine)
}

func (c *MonitorCollector) CollectFullAudit(ctx context.Context, machine domain.Machine) (domain.MachineFullAudit, error) {
	mode := effectiveConnectionMode(machine)
	if mode == domain.MachineConnectionModeWSReverse || mode == domain.MachineConnectionModeWSListener {
		return domain.MachineFullAudit{}, fmt.Errorf("%w: full audit collection is not implemented for %s", ErrTransportUnavailable, mode)
	}
	if c == nil || c.sshCollector == nil {
		return domain.MachineFullAudit{}, fmt.Errorf("machine monitor collector unavailable")
	}
	return c.sshCollector.CollectFullAudit(ctx, machine)
}

func (c *MonitorCollector) resolve(machine domain.Machine) (Transport, error) {
	if c == nil || c.resolver == nil {
		return nil, fmt.Errorf("%w: monitor resolver unavailable", ErrTransportUnavailable)
	}
	return c.resolver.Resolve(machine)
}

func (c *MonitorCollector) currentTime() time.Time {
	if c == nil || c.now == nil {
		return time.Now().UTC()
	}
	return c.now().UTC()
}

type localTransport struct {
	processManager provider.AgentCLIProcessManager
}

func (t localTransport) Mode() domain.MachineConnectionMode { return domain.MachineConnectionModeLocal }

func (t localTransport) Capabilities(machine domain.Machine) []domain.MachineTransportCapability {
	return copyCapabilities(machine.TransportCapabilities, t.Mode())
}

func (t localTransport) Probe(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error) {
	return sshinfra.NewTester(nil).TestConnection(ctx, machine)
}

func (t localTransport) PrepareWorkspace(ctx context.Context, machine domain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error) {
	return workspaceinfra.NewManager().Prepare(ctx, request)
}

func (t localTransport) SyncArtifacts(ctx context.Context, machine domain.Machine, request SyncArtifactsRequest) error {
	return syncLocalArtifacts(request)
}

func (t localTransport) StartProcess(ctx context.Context, machine domain.Machine, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	if t.processManager == nil {
		return nil, fmt.Errorf("local process manager unavailable")
	}
	return t.processManager.Start(ctx, spec)
}

func (t localTransport) OpenCommandSession(context.Context, domain.Machine) (CommandSession, error) {
	return nil, fmt.Errorf("%w: local command session is not implemented", ErrTransportUnavailable)
}

func (t localTransport) SessionState(context.Context, domain.Machine) (domain.MachineTransportSessionState, error) {
	return domain.MachineTransportSessionStateConnected, nil
}

func (t localTransport) Heartbeat(ctx context.Context, machine domain.Machine) (domain.MachineDaemonStatus, error) {
	return domain.MachineDaemonStatus{
		Registered:       true,
		LastRegisteredAt: cloneTime(machine.LastHeartbeatAt),
		CurrentSessionID: nil,
		SessionState:     domain.MachineTransportSessionStateConnected,
	}, nil
}

type sshTransport struct {
	pool *sshinfra.Pool
}

func (t sshTransport) Mode() domain.MachineConnectionMode { return domain.MachineConnectionModeSSH }

func (t sshTransport) Capabilities(machine domain.Machine) []domain.MachineTransportCapability {
	return copyCapabilities(machine.TransportCapabilities, t.Mode())
}

func (t sshTransport) Probe(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error) {
	return sshinfra.NewTester(t.pool).TestConnection(ctx, machine)
}

func (t sshTransport) PrepareWorkspace(ctx context.Context, machine domain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error) {
	if t.pool == nil {
		return workspaceinfra.Workspace{}, fmt.Errorf("ssh pool unavailable for remote machine %s", machine.Name)
	}
	return workspaceinfra.NewRemoteManager(t.pool).Prepare(ctx, machine, request)
}

func (t sshTransport) SyncArtifacts(ctx context.Context, machine domain.Machine, request SyncArtifactsRequest) error {
	return syncRemoteArtifacts(ctx, t.pool, machine, request)
}

func (t sshTransport) StartProcess(ctx context.Context, machine domain.Machine, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	return sshinfra.NewProcessManager(t.pool, machine).Start(ctx, spec)
}

func (t sshTransport) OpenCommandSession(ctx context.Context, machine domain.Machine) (CommandSession, error) {
	if t.pool == nil {
		return nil, fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
	}
	client, err := t.pool.Get(ctx, machine)
	if err != nil {
		return nil, fmt.Errorf("get ssh client for machine %s: %w", machine.Name, err)
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("open ssh session for machine %s: %w", machine.Name, err)
	}
	return session, nil
}

func (t sshTransport) SessionState(ctx context.Context, machine domain.Machine) (domain.MachineTransportSessionState, error) {
	if machine.DaemonStatus.SessionState != "" && machine.DaemonStatus.SessionState != domain.MachineTransportSessionStateUnknown {
		return machine.DaemonStatus.SessionState, nil
	}
	if t.pool == nil {
		return domain.MachineTransportSessionStateUnavailable, fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
	}
	if _, err := t.pool.Get(ctx, machine); err != nil {
		return domain.MachineTransportSessionStateUnavailable, err
	}
	return domain.MachineTransportSessionStateConnected, nil
}

func (t sshTransport) Heartbeat(ctx context.Context, machine domain.Machine) (domain.MachineDaemonStatus, error) {
	state, err := t.SessionState(ctx, machine)
	return domain.MachineDaemonStatus{
		Registered:       state == domain.MachineTransportSessionStateConnected,
		LastRegisteredAt: cloneTime(machine.LastHeartbeatAt),
		CurrentSessionID: cloneString(machine.DaemonStatus.CurrentSessionID),
		SessionState:     state,
	}, err
}

type websocketTransport struct {
	mode domain.MachineConnectionMode
}

func (t websocketTransport) Mode() domain.MachineConnectionMode { return t.mode }

func (t websocketTransport) Capabilities(machine domain.Machine) []domain.MachineTransportCapability {
	return copyCapabilities(machine.TransportCapabilities, t.mode)
}

func (t websocketTransport) Probe(context.Context, domain.Machine) (domain.MachineProbe, error) {
	return domain.MachineProbe{Transport: t.mode.String()}, fmt.Errorf("%w: %s transport is not implemented yet", ErrTransportUnavailable, t.mode)
}

func (t websocketTransport) PrepareWorkspace(context.Context, domain.Machine, workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error) {
	return workspaceinfra.Workspace{}, fmt.Errorf("%w: %s workspace preparation is not implemented yet", ErrTransportUnavailable, t.mode)
}

func (t websocketTransport) SyncArtifacts(context.Context, domain.Machine, SyncArtifactsRequest) error {
	return fmt.Errorf("%w: %s artifact sync is not implemented yet", ErrTransportUnavailable, t.mode)
}

func (t websocketTransport) StartProcess(context.Context, domain.Machine, provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	return nil, fmt.Errorf("%w: %s process streaming is not implemented yet", ErrTransportUnavailable, t.mode)
}

func (t websocketTransport) OpenCommandSession(context.Context, domain.Machine) (CommandSession, error) {
	return nil, fmt.Errorf("%w: %s command sessions are not implemented yet", ErrTransportUnavailable, t.mode)
}

func (t websocketTransport) SessionState(ctx context.Context, machine domain.Machine) (domain.MachineTransportSessionState, error) {
	heartbeat, err := t.Heartbeat(ctx, machine)
	return heartbeat.SessionState, err
}

func (t websocketTransport) Heartbeat(ctx context.Context, machine domain.Machine) (domain.MachineDaemonStatus, error) {
	status := domain.MachineDaemonStatus{
		Registered:       machine.DaemonStatus.Registered,
		LastRegisteredAt: cloneTime(machine.DaemonStatus.LastRegisteredAt),
		CurrentSessionID: cloneString(machine.DaemonStatus.CurrentSessionID),
		SessionState:     machine.DaemonStatus.SessionState,
	}
	if status.SessionState == "" || status.SessionState == domain.MachineTransportSessionStateUnknown {
		if status.Registered {
			status.SessionState = domain.MachineTransportSessionStateConnected
		} else {
			status.SessionState = domain.MachineTransportSessionStateUnavailable
		}
	}
	if !status.Registered {
		return status, fmt.Errorf("%w: %s machine is not registered", ErrTransportUnavailable, t.mode)
	}
	return status, nil
}

func copyCapabilities(
	items []domain.MachineTransportCapability,
	mode domain.MachineConnectionMode,
) []domain.MachineTransportCapability {
	if len(items) == 0 {
		switch mode {
		case domain.MachineConnectionModeLocal,
			domain.MachineConnectionModeSSH,
			domain.MachineConnectionModeWSReverse,
			domain.MachineConnectionModeWSListener:
			return []domain.MachineTransportCapability{
				domain.MachineTransportCapabilityProbe,
				domain.MachineTransportCapabilityWorkspacePrepare,
				domain.MachineTransportCapabilityArtifactSync,
				domain.MachineTransportCapabilityProcessStreaming,
			}
		default:
			return nil
		}
	}
	cloned := make([]domain.MachineTransportCapability, 0, len(items))
	cloned = append(cloned, items...)
	return cloned
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func cloneString(value *string) *string {
	if value == nil {
		return nil
	}
	copied := strings.TrimSpace(*value)
	return &copied
}

func effectiveConnectionMode(machine domain.Machine) domain.MachineConnectionMode {
	if machine.ConnectionMode.IsValid() {
		return machine.ConnectionMode
	}
	if machine.Host == domain.LocalMachineHost || machine.Name == domain.LocalMachineName {
		return domain.MachineConnectionModeLocal
	}
	return domain.MachineConnectionModeSSH
}

func removeLocalPath(target string) error {
	if strings.TrimSpace(target) == "" {
		return fmt.Errorf("target path must not be empty")
	}
	if err := os.RemoveAll(target); err != nil {
		return fmt.Errorf("remove local path %s: %w", target, err)
	}
	return nil
}
