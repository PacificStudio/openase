package machinetransport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	runtimecontract "github.com/BetterAndBetterII/openase/internal/domain/websocketruntime"
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

type ProbeExecution interface {
	Probe(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error)
}

type WorkspaceExecution interface {
	PrepareWorkspace(ctx context.Context, machine domain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error)
}

type ArtifactSyncExecution interface {
	SyncArtifacts(ctx context.Context, machine domain.Machine, request SyncArtifactsRequest) error
}

type ProcessExecution interface {
	StartProcess(ctx context.Context, machine domain.Machine, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error)
}

type CommandSessionExecution interface {
	OpenCommandSession(ctx context.Context, machine domain.Machine) (CommandSession, error)
}

type ChannelTransport interface {
	Mode() domain.MachineConnectionMode
	SessionState(ctx context.Context, machine domain.Machine) (domain.MachineTransportSessionState, error)
	Heartbeat(ctx context.Context, machine domain.Machine) (domain.MachineDaemonStatus, error)
}

type capabilitySurface struct {
	ordered []domain.MachineTransportCapability
	set     map[domain.MachineTransportCapability]struct{}
}

func newCapabilitySurface(items []domain.MachineTransportCapability) capabilitySurface {
	cloned := append([]domain.MachineTransportCapability(nil), items...)
	set := make(map[domain.MachineTransportCapability]struct{}, len(cloned))
	for _, item := range cloned {
		set[item] = struct{}{}
	}
	return capabilitySurface{
		ordered: cloned,
		set:     set,
	}
}

func (s capabilitySurface) Capabilities() []domain.MachineTransportCapability {
	return append([]domain.MachineTransportCapability(nil), s.ordered...)
}

func (s capabilitySurface) Supports(capability domain.MachineTransportCapability) bool {
	_, ok := s.set[capability]
	return ok
}

func (s capabilitySurface) SupportsAll(capabilities ...domain.MachineTransportCapability) bool {
	for _, capability := range capabilities {
		if !s.Supports(capability) {
			return false
		}
	}
	return true
}

type RemoteRuntimeSurface struct {
	capabilities   capabilitySurface
	Workspace      WorkspaceExecution
	ArtifactSync   ArtifactSyncExecution
	Process        ProcessExecution
	CommandSession CommandSessionExecution
}

func newRemoteRuntimeSurface(items []domain.MachineTransportCapability) *RemoteRuntimeSurface {
	return &RemoteRuntimeSurface{capabilities: newCapabilitySurface(items)}
}

func (s *RemoteRuntimeSurface) Capabilities() []domain.MachineTransportCapability {
	if s == nil {
		return nil
	}
	return s.capabilities.Capabilities()
}

func (s *RemoteRuntimeSurface) Supports(capability domain.MachineTransportCapability) bool {
	if s == nil {
		return false
	}
	return s.capabilities.Supports(capability)
}

func (s *RemoteRuntimeSurface) SupportsAll(capabilities ...domain.MachineTransportCapability) bool {
	if s == nil {
		return false
	}
	return s.capabilities.SupportsAll(capabilities...)
}

type ExecutionSurface struct {
	capabilities   capabilitySurface
	Probe          ProbeExecution
	Workspace      WorkspaceExecution
	ArtifactSync   ArtifactSyncExecution
	Process        ProcessExecution
	CommandSession CommandSessionExecution
	Runtime        *RemoteRuntimeSurface
}

func newExecutionSurface(items []domain.MachineTransportCapability) ExecutionSurface {
	return ExecutionSurface{capabilities: newCapabilitySurface(items)}
}

func (s ExecutionSurface) Capabilities() []domain.MachineTransportCapability {
	return s.capabilities.Capabilities()
}

func (s ExecutionSurface) Supports(capability domain.MachineTransportCapability) bool {
	return s.capabilities.Supports(capability)
}

func (s ExecutionSurface) SupportsAll(capabilities ...domain.MachineTransportCapability) bool {
	return s.capabilities.SupportsAll(capabilities...)
}

type ResolvedTransport struct {
	Channel   ChannelTransport
	Execution ExecutionSurface
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
	reverseRelay        *ReverseRuntimeRelayRegistry
}

func NewResolver(localProcessManager provider.AgentCLIProcessManager, sshPool *sshinfra.Pool) *Resolver {
	return &Resolver{
		localProcessManager: localProcessManager,
		sshPool:             sshPool,
	}
}

func (r *Resolver) WithReverseRuntimeRelay(relay *ReverseRuntimeRelayRegistry) *Resolver {
	if r == nil {
		return nil
	}
	r.reverseRelay = relay
	return r
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
		return websocketTransport{mode: mode, reverseRelay: r.reverseRelay}, nil
	default:
		return nil, fmt.Errorf("%w: unsupported connection mode %q", ErrTransportUnavailable, mode)
	}
}

func (r *Resolver) ResolveRuntime(machine domain.Machine) (ResolvedTransport, error) {
	transport, err := r.Resolve(machine)
	if err != nil {
		return ResolvedTransport{}, err
	}

	capabilities := transport.Capabilities(machine)
	resolved := ResolvedTransport{
		Channel:   transport,
		Execution: newExecutionSurface(capabilities),
	}

	switch transport.Mode() {
	case domain.MachineConnectionModeLocal:
		resolved.Execution.Probe = transport
		resolved.Execution.Workspace = transport
		resolved.Execution.ArtifactSync = transport
		resolved.Execution.Process = transport
	case domain.MachineConnectionModeSSH:
		resolved.Execution.Probe = transport
		resolved.Execution.Workspace = transport
		resolved.Execution.ArtifactSync = transport
		resolved.Execution.Process = transport
		resolved.Execution.CommandSession = transport
	case domain.MachineConnectionModeWSReverse:
		resolved.Execution.Runtime = newRemoteRuntimeSurface(capabilities)
	case domain.MachineConnectionModeWSListener:
		resolved.Execution.Probe = transport
		resolved.Execution.Workspace = transport
		resolved.Execution.ArtifactSync = transport
		resolved.Execution.Process = transport
		resolved.Execution.CommandSession = transport

		runtime := newRemoteRuntimeSurface(capabilities)
		runtime.Workspace = transport
		runtime.ArtifactSync = transport
		runtime.Process = transport
		runtime.CommandSession = transport
		resolved.Execution.Runtime = runtime
	}

	return resolved, nil
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
	channel, err := c.resolve(machine)
	if err != nil {
		checkedAt := c.currentTime()
		return domain.MachineReachability{
			CheckedAt:    checkedAt,
			Transport:    machine.ConnectionMode.String(),
			FailureCause: err.Error(),
		}, err
	}

	switch channel.Mode() {
	case domain.MachineConnectionModeWSReverse:
		checkedAt := c.currentTime()
		heartbeat, hbErr := channel.Heartbeat(ctx, machine)
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
			Transport:    channel.Mode().String(),
			Reachable:    reachable,
			FailureCause: failureCause,
		}, hbErr
	case domain.MachineConnectionModeWSListener:
		resolved, resolveErr := c.resolver.ResolveRuntime(machine)
		if resolveErr != nil {
			return domain.MachineReachability{}, resolveErr
		}
		if resolved.Execution.Probe == nil {
			err := fmt.Errorf("%w: probe unavailable for machine %s", ErrTransportUnavailable, machine.Name)
			return domain.MachineReachability{
				CheckedAt:    c.currentTime(),
				Transport:    channel.Mode().String(),
				FailureCause: err.Error(),
			}, err
		}
		probe, probeErr := resolved.Execution.Probe.Probe(ctx, machine)
		failureCause := ""
		if probeErr != nil {
			failureCause = probeErr.Error()
		}
		return domain.MachineReachability{
			CheckedAt:    probe.CheckedAt,
			Transport:    channel.Mode().String(),
			Reachable:    probeErr == nil,
			FailureCause: failureCause,
		}, probeErr
	default:
		if c.sshCollector == nil {
			checkedAt := c.currentTime()
			return domain.MachineReachability{
				CheckedAt:    checkedAt,
				Transport:    channel.Mode().String(),
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

func (c *MonitorCollector) resolve(machine domain.Machine) (ChannelTransport, error) {
	if c == nil || c.resolver == nil {
		return nil, fmt.Errorf("%w: monitor resolver unavailable", ErrTransportUnavailable)
	}
	resolved, err := c.resolver.ResolveRuntime(machine)
	if err != nil {
		return nil, err
	}
	return resolved.Channel, nil
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
		return workspaceinfra.Workspace{}, workspaceinfra.WrapPrepareTransportError(machine.Name, fmt.Errorf("ssh pool unavailable for remote machine %s", machine.Name))
	}
	return workspaceinfra.NewRemoteManager(t.pool).Prepare(ctx, machine, request)
}

func (t sshTransport) SyncArtifacts(ctx context.Context, machine domain.Machine, request SyncArtifactsRequest) error {
	if t.pool == nil {
		return fmt.Errorf("ssh pool unavailable for machine %s", machine.Name)
	}
	client, err := t.pool.Get(ctx, machine)
	if err != nil {
		return fmt.Errorf("get ssh client for machine %s: %w", machine.Name, err)
	}
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("open ssh session for machine %s: %w", machine.Name, err)
	}
	defer func() { _ = session.Close() }()
	return syncArtifactsWithSession(session, request)
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
	mode         domain.MachineConnectionMode
	reverseRelay *ReverseRuntimeRelayRegistry
}

func (t websocketTransport) Mode() domain.MachineConnectionMode { return t.mode }

func (t websocketTransport) Capabilities(machine domain.Machine) []domain.MachineTransportCapability {
	return copyCapabilities(machine.TransportCapabilities, t.mode)
}

func (t websocketTransport) Probe(ctx context.Context, machine domain.Machine) (domain.MachineProbe, error) {
	client, closeClient, err := t.openRuntimeClient(ctx, machine)
	if err != nil {
		return domain.MachineProbe{CheckedAt: time.Now().UTC(), Transport: t.mode.String()}, err
	}
	defer closeClient(nil)

	envelope, err := client.request(ctx, runtimecontract.OperationProbe, nil)
	if err != nil {
		return domain.MachineProbe{CheckedAt: time.Now().UTC(), Transport: t.mode.String()}, err
	}
	payload, err := runtimecontract.DecodePayload[runtimecontract.ProbeResponse](envelope)
	if err != nil {
		return domain.MachineProbe{CheckedAt: time.Now().UTC(), Transport: t.mode.String()}, err
	}
	return augmentRuntimeProbe(machine, payload), nil
}

func (t websocketTransport) PrepareWorkspace(ctx context.Context, machine domain.Machine, request workspaceinfra.SetupRequest) (workspaceinfra.Workspace, error) {
	client, closeClient, err := t.openRuntimeClient(ctx, machine)
	if err != nil {
		return workspaceinfra.Workspace{}, workspaceinfra.WrapPrepareTransportError(machine.Name, err)
	}
	defer closeClient(nil)

	envelope, err := client.request(ctx, runtimecontract.OperationWorkspacePrepare, runtimeWorkspacePrepareRequest(request))
	if err != nil {
		return workspaceinfra.Workspace{}, workspaceinfra.WrapPrepareTransportError(machine.Name, err)
	}
	response, err := runtimecontract.DecodePayload[runtimecontract.WorkspacePrepareResponse](envelope)
	if err != nil {
		return workspaceinfra.Workspace{}, workspaceinfra.WrapPrepareTransportError(machine.Name, err)
	}
	return runtimeWorkspaceResponse(response), nil
}

func (t websocketTransport) SyncArtifacts(ctx context.Context, machine domain.Machine, request SyncArtifactsRequest) error {
	client, closeClient, err := t.openRuntimeClient(ctx, machine)
	if err != nil {
		return err
	}
	defer closeClient(nil)

	entries, err := buildArtifactSyncEntries(request)
	if err != nil {
		return err
	}
	_, err = client.request(ctx, runtimecontract.OperationArtifactSync, runtimecontract.ArtifactSyncRequest{
		TargetRoot:  request.TargetRoot,
		RemovePaths: append([]string(nil), request.RemovePaths...),
		Entries:     entries,
	})
	return err
}

func (t websocketTransport) StartProcess(ctx context.Context, machine domain.Machine, spec provider.AgentCLIProcessSpec) (provider.AgentCLIProcess, error) {
	client, closeClient, err := t.openRuntimeClient(ctx, machine)
	if err != nil {
		return nil, err
	}
	process, err := startRuntimeRemoteProcess(ctx, client, spec, closeClient)
	if err != nil {
		closeClient(err)
		return nil, err
	}
	return process, nil
}

func (t websocketTransport) OpenCommandSession(ctx context.Context, machine domain.Machine) (CommandSession, error) {
	client, closeClient, err := t.openRuntimeClient(ctx, machine)
	if err != nil {
		return nil, err
	}
	return newRuntimeCommandSession(ctx, client, closeClient), nil
}

func (t websocketTransport) SessionState(ctx context.Context, machine domain.Machine) (domain.MachineTransportSessionState, error) {
	heartbeat, err := t.Heartbeat(ctx, machine)
	return heartbeat.SessionState, err
}

func (t websocketTransport) Heartbeat(ctx context.Context, machine domain.Machine) (domain.MachineDaemonStatus, error) {
	if t.mode == domain.MachineConnectionModeWSListener {
		_, closeClient, err := t.openRuntimeClient(ctx, machine)
		if err != nil {
			return domain.MachineDaemonStatus{
				Registered:       false,
				LastRegisteredAt: cloneTime(machine.DaemonStatus.LastRegisteredAt),
				CurrentSessionID: cloneString(machine.DaemonStatus.CurrentSessionID),
				SessionState:     domain.MachineTransportSessionStateUnavailable,
			}, err
		}
		closeClient(nil)
		checkedAt := time.Now().UTC()
		return domain.MachineDaemonStatus{
			Registered:       true,
			LastRegisteredAt: &checkedAt,
			CurrentSessionID: cloneString(machine.DaemonStatus.CurrentSessionID),
			SessionState:     domain.MachineTransportSessionStateConnected,
		}, nil
	}

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

func (t websocketTransport) openRuntimeClient(
	ctx context.Context,
	machine domain.Machine,
) (*runtimeProtocolClient, func(error), error) {
	switch t.mode {
	case domain.MachineConnectionModeWSListener:
		return dialWebsocketRuntimeClient(ctx, machine)
	case domain.MachineConnectionModeWSReverse:
		if t.reverseRelay == nil {
			return nil, nil, fmt.Errorf("%w: reverse websocket runtime relay is unavailable", ErrTransportUnavailable)
		}
		client, err := t.reverseRelay.client(machine.ID)
		if err != nil {
			return nil, nil, err
		}
		return client, func(error) {}, nil
	default:
		return nil, nil, fmt.Errorf("%w: %s runtime contract is not implemented yet", ErrTransportUnavailable, t.mode)
	}
}

func copyCapabilities(
	items []domain.MachineTransportCapability,
	mode domain.MachineConnectionMode,
) []domain.MachineTransportCapability {
	if len(items) == 0 {
		return domain.SupportedMachineTransportCapabilities(mode)
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

func pointerString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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

type commandSessionProcess struct {
	session CommandSession
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	done    chan struct{}

	waitErr  error
	waitOnce sync.Once
}

func (p *commandSessionProcess) PID() int { return 0 }

func (p *commandSessionProcess) Stdin() io.WriteCloser { return p.stdin }

func (p *commandSessionProcess) Stdout() io.ReadCloser { return io.NopCloser(p.stdout) }

func (p *commandSessionProcess) Stderr() io.ReadCloser { return io.NopCloser(p.stderr) }

func (p *commandSessionProcess) Wait() error {
	if p == nil {
		return fmt.Errorf("process must not be nil")
	}
	p.awaitExit()
	return p.waitErr
}

func (p *commandSessionProcess) Stop(ctx context.Context) error {
	if p == nil {
		return fmt.Errorf("process must not be nil")
	}
	if ctx == nil {
		return fmt.Errorf("context must not be nil")
	}

	select {
	case <-p.done:
		p.awaitExit()
		return p.waitErr
	default:
	}

	_ = p.stdin.Close()
	if err := p.session.Signal("INT"); err != nil {
		_ = p.session.Close()
	}

	select {
	case <-p.done:
		p.awaitExit()
		return p.waitErr
	case <-ctx.Done():
		closeErr := p.session.Close()
		p.awaitExit()
		if p.waitErr != nil {
			return p.waitErr
		}
		if closeErr != nil {
			return closeErr
		}
		return p.waitErr
	}
}

func (p *commandSessionProcess) waitLoop() {
	p.waitErr = p.session.Wait()
	_ = p.session.Close()
	close(p.done)
}

func (p *commandSessionProcess) awaitExit() {
	p.waitOnce.Do(func() {
		<-p.done
	})
}

func buildRemoteCommandSessionShellCommand(spec provider.AgentCLIProcessSpec) string {
	commandParts := make([]string, 0, 1+len(spec.Args))
	commandParts = append(commandParts, sshinfra.ShellQuote(spec.Command.String()))
	for _, arg := range spec.Args {
		commandParts = append(commandParts, sshinfra.ShellQuote(arg))
	}

	command := strings.Join(commandParts, " ")
	if len(spec.Environment) > 0 {
		envParts := make([]string, 0, len(spec.Environment))
		for _, entry := range spec.Environment {
			envParts = append(envParts, sshinfra.ShellQuote(entry))
		}
		command = "env " + strings.Join(envParts, " ") + " " + command
	}
	if spec.WorkingDirectory != nil {
		command = "cd " + sshinfra.ShellQuote(spec.WorkingDirectory.String()) + " && " + command
	}
	return command
}
