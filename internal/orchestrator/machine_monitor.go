package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	activitysvc "github.com/BetterAndBetterII/openase/internal/activity"
	activityevent "github.com/BetterAndBetterII/openase/internal/domain/activityevent"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/BetterAndBetterII/openase/internal/provider"
	"github.com/google/uuid"
)

const (
	machineMonitorLevel1Interval  = 15 * time.Second
	machineMonitorLevel2Interval  = time.Minute
	machineMonitorLevel3Interval  = 5 * time.Minute
	machineMonitorLevel4Interval  = domain.ProviderAvailabilityL4Interval
	machineMonitorLevel5Interval  = 6 * time.Hour
	machineMonitorOfflineFailures = 3
	lowDiskThresholdGB            = 5.0
	lowMemoryThresholdPercent     = 10.0
	fullGPUMemoryThresholdGB      = 0.5
)

var (
	machineEventsTopic               = provider.MustParseTopic("machine.events")
	providerEventsTopic              = provider.MustParseTopic("provider.events")
	machineOnlineEventType           = provider.MustParseEventType("machine.online")
	machineOfflineEventType          = provider.MustParseEventType("machine.offline")
	machineDegradedEventType         = provider.MustParseEventType("machine.degraded")
	machineResourcesUpdatedEventType = provider.MustParseEventType("machine.resources_updated")
	providerAvailableEventType       = provider.MustParseEventType("provider.available")
	providerUnavailableEventType     = provider.MustParseEventType("provider.unavailable")
	providerStaleEventType           = provider.MustParseEventType("provider.stale")
)

type MachineMonitorCollector interface {
	CollectReachability(ctx context.Context, machine domain.Machine) (domain.MachineReachability, error)
	CollectSystemResources(ctx context.Context, machine domain.Machine) (domain.MachineSystemResources, error)
	CollectGPUResources(ctx context.Context, machine domain.Machine) (domain.MachineGPUResources, error)
	CollectAgentEnvironment(ctx context.Context, machine domain.Machine) (domain.MachineAgentEnvironment, error)
	CollectFullAudit(ctx context.Context, machine domain.Machine) (domain.MachineFullAudit, error)
}

type MachineMonitorReport struct {
	MachinesScanned  int `json:"machines_scanned"`
	MachinesUpdated  int `json:"machines_updated"`
	L1Checks         int `json:"l1_checks"`
	L2Checks         int `json:"l2_checks"`
	L3Checks         int `json:"l3_checks"`
	L4Checks         int `json:"l4_checks"`
	L5Checks         int `json:"l5_checks"`
	OfflineMachines  int `json:"offline_machines"`
	DegradedMachines int `json:"degraded_machines"`
}

type MachineMonitor struct {
	client    *ent.Client
	logger    *slog.Logger
	collector MachineMonitorCollector
	events    provider.EventProvider
	now       func() time.Time
}

func NewMachineMonitor(client *ent.Client, logger *slog.Logger, collector MachineMonitorCollector) *MachineMonitor {
	if logger == nil {
		logger = slog.Default()
	}

	return &MachineMonitor{
		client:    client,
		logger:    logger.With("component", "machine-monitor"),
		collector: collector,
		now:       time.Now,
	}
}

func (m *MachineMonitor) ConfigureEvents(events provider.EventProvider) {
	if m == nil {
		return
	}
	m.events = events
}

func (m *MachineMonitor) RunTick(ctx context.Context) (MachineMonitorReport, error) {
	report := MachineMonitorReport{}
	if m == nil || m.client == nil {
		return report, fmt.Errorf("machine monitor unavailable")
	}
	if m.collector == nil {
		return report, fmt.Errorf("machine monitor collector unavailable")
	}

	items, err := m.client.Machine.Query().
		Order(ent.Asc(entmachine.FieldName)).
		All(ctx)
	if err != nil {
		return report, fmt.Errorf("list machines: %w", err)
	}

	now := m.now().UTC()
	for _, item := range items {
		report.MachinesScanned++

		current := mapMachineEntity(item)
		updated, changed := m.runMachineTick(ctx, current, now, &report)
		if !changed {
			continue
		}

		builder := m.client.Machine.UpdateOneID(updated.ID).
			SetStatus(updated.Status).
			SetResources(updated.Resources)
		if updated.LastHeartbeatAt.IsZero() {
			builder.ClearLastHeartbeatAt()
		} else {
			builder.SetLastHeartbeatAt(updated.LastHeartbeatAt.UTC())
		}
		if _, err := builder.Save(ctx); err != nil {
			return report, fmt.Errorf("persist machine %s monitor state: %w", updated.ID, err)
		}

		report.MachinesUpdated++
		switch updated.Status {
		case entmachine.StatusOffline:
			report.OfflineMachines++
		case entmachine.StatusDegraded:
			report.DegradedMachines++
		}
		if err := m.publishRuntimeEvents(ctx, current, updated, now); err != nil {
			return report, fmt.Errorf("publish machine %s runtime events: %w", updated.ID, err)
		}
	}

	return report, nil
}

type monitoredMachine struct {
	OrganizationID     uuid.UUID
	ID                 uuid.UUID
	Name               string
	Host               string
	Port               int
	ConnectionMode     entmachine.ConnectionMode
	SSHUser            *string
	SSHKeyPath         *string
	AdvertisedEndpoint *string
	WorkspaceRoot      *string
	AgentCLIPath       *string
	DaemonStatus       domain.MachineDaemonStatus
	Status             entmachine.Status
	Labels             []string
	LastHeartbeatAt    time.Time
	Resources          map[string]any
}

func (m monitoredMachine) toDomain() domain.Machine {
	return domain.Machine{
		ID:                 m.ID,
		OrganizationID:     m.OrganizationID,
		Name:               m.Name,
		Host:               m.Host,
		Port:               m.Port,
		ConnectionMode:     domain.MachineConnectionMode(m.ConnectionMode),
		SSHUser:            m.SSHUser,
		SSHKeyPath:         m.SSHKeyPath,
		AdvertisedEndpoint: m.AdvertisedEndpoint,
		WorkspaceRoot:      m.WorkspaceRoot,
		AgentCLIPath:       m.AgentCLIPath,
		DaemonStatus:       m.DaemonStatus,
		Labels:             append([]string(nil), m.Labels...),
	}
}

func (m *MachineMonitor) runMachineTick(ctx context.Context, machine monitoredMachine, now time.Time, report *MachineMonitorReport) (monitoredMachine, bool) {
	level1Due := machineMonitorDue(machine.Resources, "l1", now, machineMonitorLevel1Interval)
	level2Due := machineMonitorDue(machine.Resources, "l2", now, machineMonitorLevel2Interval)
	level3Due := machineMonitorDue(machine.Resources, "l3", now, machineMonitorLevel3Interval)
	level4Due := machineMonitorDue(machine.Resources, "l4", now, machineMonitorLevel4Interval)
	level5Due := machineMonitorDue(machine.Resources, "l5", now, machineMonitorLevel5Interval)
	if !level1Due && !level2Due && !level3Due && !level4Due && !level5Due {
		return machine, false
	}

	resources := cloneResourceMap(machine.Resources)
	status := machine.Status
	if status != entmachine.StatusMaintenance {
		status = entmachine.StatusOnline
	}
	lastHeartbeatAt := machine.LastHeartbeatAt
	hardReachabilityFailure := false
	softReachabilityFailure := false
	systemProbeFailure := false
	level4ProbeFailure := false
	level5ProbeFailure := false
	domainMachine := machine.toDomain()
	isWebsocketMachine := domainMachine.ConnectionMode == domain.MachineConnectionModeWSReverse || domainMachine.ConnectionMode == domain.MachineConnectionModeWSListener

	if level1Due {
		report.L1Checks++
		reachability, err := m.collector.CollectReachability(ctx, domainMachine)
		lastHeartbeatAt = reachability.CheckedAt
		if lastHeartbeatAt.IsZero() {
			lastHeartbeatAt = now
		}
		updateL1Resources(resources, reachability)
		if err != nil || !reachability.Reachable {
			failures := machineMonitorFailures(resources) + 1
			setMachineMonitorFailures(resources, failures)
			if err != nil {
				setMachineMonitorError(resources, "l1", err.Error())
			}
			softReachabilityFailure = true
			hardReachabilityFailure = failures >= machineMonitorOfflineFailures && machine.Host != domain.LocalMachineHost
		} else {
			setMachineMonitorFailures(resources, 0)
			clearMachineMonitorError(resources, "l1")
		}
	}

	if level2Due && !softReachabilityFailure && !hardReachabilityFailure && !isWebsocketMachine {
		report.L2Checks++
		systemResources, err := m.collector.CollectSystemResources(ctx, domainMachine)
		if err != nil {
			systemProbeFailure = true
			setMachineMonitorError(resources, "l2", err.Error())
		} else {
			updateL2Resources(resources, systemResources)
			clearMachineMonitorError(resources, "l2")
		}
	}

	if level3Due && !softReachabilityFailure && !hardReachabilityFailure && !isWebsocketMachine {
		report.L3Checks++
		gpuResources, err := m.collector.CollectGPUResources(ctx, domainMachine)
		if err != nil {
			setMachineMonitorError(resources, "l3", err.Error())
		} else {
			updateL3Resources(resources, gpuResources)
			clearMachineMonitorError(resources, "l3")
		}
	}

	if level4Due && !softReachabilityFailure && !hardReachabilityFailure && !isWebsocketMachine {
		report.L4Checks++
		agentEnvironment, err := m.collector.CollectAgentEnvironment(ctx, domainMachine)
		if err != nil {
			level4ProbeFailure = true
			setMachineMonitorError(resources, "l4", err.Error())
		} else {
			updateL4Resources(resources, agentEnvironment)
			clearMachineMonitorError(resources, "l4")
		}
	}

	if level5Due && !softReachabilityFailure && !hardReachabilityFailure && !isWebsocketMachine {
		report.L5Checks++
		fullAudit, err := m.collector.CollectFullAudit(ctx, domainMachine)
		if err != nil {
			level5ProbeFailure = true
			setMachineMonitorError(resources, "l5", err.Error())
		} else {
			updateL5Resources(resources, fullAudit)
			clearMachineMonitorError(resources, "l5")
		}
	}

	if machine.Status != entmachine.StatusMaintenance {
		switch {
		case hardReachabilityFailure:
			status = entmachine.StatusOffline
		case softReachabilityFailure || systemProbeFailure || level4ProbeFailure || level5ProbeFailure || machineHasLowDisk(resources):
			status = entmachine.StatusDegraded
		default:
			status = entmachine.StatusOnline
		}
	}

	return monitoredMachine{
		OrganizationID:     machine.OrganizationID,
		ID:                 machine.ID,
		Name:               machine.Name,
		Host:               machine.Host,
		Port:               machine.Port,
		ConnectionMode:     machine.ConnectionMode,
		SSHUser:            cloneMachineString(machine.SSHUser),
		SSHKeyPath:         cloneMachineString(machine.SSHKeyPath),
		AdvertisedEndpoint: cloneMachineString(machine.AdvertisedEndpoint),
		WorkspaceRoot:      cloneMachineString(machine.WorkspaceRoot),
		AgentCLIPath:       cloneMachineString(machine.AgentCLIPath),
		DaemonStatus:       machine.DaemonStatus,
		Status:             status,
		Labels:             append([]string(nil), machine.Labels...),
		LastHeartbeatAt:    lastHeartbeatAt,
		Resources:          resources,
	}, true
}

func (m *MachineMonitor) publishRuntimeEvents(
	ctx context.Context,
	current monitoredMachine,
	updated monitoredMachine,
	publishedAt time.Time,
) error {
	if m == nil || m.events == nil {
		return nil
	}

	if eventType, ok := machineStatusEventType(updated.Status); ok && current.Status != updated.Status {
		if err := m.publishMachineEvent(ctx, eventType, updated, publishedAt); err != nil {
			return err
		}
	}
	if !reflect.DeepEqual(current.Resources, updated.Resources) {
		if err := m.publishMachineEvent(ctx, machineResourcesUpdatedEventType, updated, publishedAt); err != nil {
			return err
		}
	}

	providers, err := m.client.AgentProvider.Query().
		Where(entagentprovider.MachineIDEQ(updated.ID)).
		Order(entagentprovider.ByName()).
		All(ctx)
	if err != nil {
		return fmt.Errorf("list providers for machine %s: %w", updated.ID, err)
	}
	for _, item := range providers {
		currentProvider := mapMonitoredAgentProvider(item, current)
		updatedProvider := mapMonitoredAgentProvider(item, updated)
		currentState := domain.DeriveAgentProviderAvailability(currentProvider, publishedAt).AvailabilityState
		nextProvider := domain.DeriveAgentProviderAvailability(updatedProvider, publishedAt)
		if currentState == nextProvider.AvailabilityState {
			continue
		}
		eventType, ok := providerAvailabilityEventType(nextProvider.AvailabilityState)
		if !ok {
			continue
		}
		if err := m.publishProviderEvent(ctx, eventType, nextProvider, publishedAt); err != nil {
			return err
		}
		projectIDs, err := m.providerActivityProjectIDs(ctx, nextProvider.OrganizationID, nextProvider.ID)
		if err != nil {
			return err
		}
		for _, projectID := range projectIDs {
			if _, err := activitysvc.NewEmitter(activitysvc.EntRecorder{Client: m.client}, m.events).Emit(ctx, activitysvc.RecordInput{
				ProjectID: projectID,
				EventType: activityevent.TypeProviderAvailabilityChanged,
				Message:   fmt.Sprintf("Provider %s availability changed to %s", nextProvider.Name, nextProvider.AvailabilityState.String()),
				Metadata: map[string]any{
					"provider_id":       nextProvider.ID.String(),
					"provider_name":     nextProvider.Name,
					"from_availability": currentState.String(),
					"to_availability":   nextProvider.AvailabilityState.String(),
					"machine_id":        nextProvider.MachineID.String(),
					"changed_fields":    []string{"availability"},
				},
				CreatedAt: publishedAt,
			}); err != nil {
				return fmt.Errorf("emit provider availability activity: %w", err)
			}
		}
	}

	return nil
}

func (m *MachineMonitor) providerActivityProjectIDs(
	ctx context.Context,
	organizationID uuid.UUID,
	providerID uuid.UUID,
) ([]uuid.UUID, error) {
	projectIDs, err := providerActivityProjectIDs(ctx, m.client, organizationID, providerID)
	if err != nil {
		return nil, fmt.Errorf("list projects for provider availability activity: %w", err)
	}
	return projectIDs, nil
}

func (m *MachineMonitor) publishMachineEvent(
	ctx context.Context,
	eventType provider.EventType,
	machine monitoredMachine,
	publishedAt time.Time,
) error {
	event, err := provider.NewJSONEvent(machineEventsTopic, eventType, map[string]any{
		"organization_id": machine.OrganizationID.String(),
		"machine": map[string]any{
			"id":                machine.ID.String(),
			"organization_id":   machine.OrganizationID.String(),
			"name":              machine.Name,
			"host":              machine.Host,
			"port":              machine.Port,
			"ssh_user":          stringPointerValue(machine.SSHUser),
			"ssh_key_path":      stringPointerValue(machine.SSHKeyPath),
			"status":            string(machine.Status),
			"workspace_root":    stringPointerValue(machine.WorkspaceRoot),
			"agent_cli_path":    stringPointerValue(machine.AgentCLIPath),
			"labels":            append([]string(nil), machine.Labels...),
			"last_heartbeat_at": timePointerRFC3339(machine.LastHeartbeatAt),
			"resources":         cloneResourceMap(machine.Resources),
		},
	}, publishedAt)
	if err != nil {
		return fmt.Errorf("construct %s event: %w", eventType, err)
	}
	if err := m.events.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish %s event: %w", eventType, err)
	}
	return nil
}

func (m *MachineMonitor) publishProviderEvent(
	ctx context.Context,
	eventType provider.EventType,
	item domain.AgentProvider,
	publishedAt time.Time,
) error {
	event, err := provider.NewJSONEvent(providerEventsTopic, eventType, map[string]any{
		"organization_id": item.OrganizationID.String(),
		"provider": map[string]any{
			"id":                      item.ID.String(),
			"organization_id":         item.OrganizationID.String(),
			"machine_id":              item.MachineID.String(),
			"machine_name":            item.MachineName,
			"machine_host":            item.MachineHost,
			"machine_status":          item.MachineStatus.String(),
			"machine_ssh_user":        stringPointerValue(item.MachineSSHUser),
			"machine_workspace_root":  stringPointerValue(item.MachineWorkspaceRoot),
			"name":                    item.Name,
			"adapter_type":            item.AdapterType.String(),
			"availability_state":      item.AvailabilityState.String(),
			"available":               item.Available,
			"availability_checked_at": timePointerRFC3339Pointer(item.AvailabilityCheckedAt),
			"availability_reason":     stringPointerValue(item.AvailabilityReason),
			"cli_command":             item.CliCommand,
			"cli_args":                append([]string(nil), item.CliArgs...),
			"auth_config":             cloneResourceMap(item.AuthConfig),
			"model_name":              item.ModelName,
			"model_temperature":       item.ModelTemperature,
			"model_max_tokens":        item.ModelMaxTokens,
			"cost_per_input_token":    item.CostPerInputToken,
			"cost_per_output_token":   item.CostPerOutputToken,
		},
	}, publishedAt)
	if err != nil {
		return fmt.Errorf("construct %s event: %w", eventType, err)
	}
	if err := m.events.Publish(ctx, event); err != nil {
		return fmt.Errorf("publish %s event: %w", eventType, err)
	}
	return nil
}

func machineStatusEventType(status entmachine.Status) (provider.EventType, bool) {
	switch status {
	case entmachine.StatusOnline:
		return machineOnlineEventType, true
	case entmachine.StatusOffline:
		return machineOfflineEventType, true
	case entmachine.StatusDegraded:
		return machineDegradedEventType, true
	default:
		return "", false
	}
}

func providerAvailabilityEventType(state domain.AgentProviderAvailabilityState) (provider.EventType, bool) {
	switch state {
	case domain.AgentProviderAvailabilityStateAvailable:
		return providerAvailableEventType, true
	case domain.AgentProviderAvailabilityStateUnavailable:
		return providerUnavailableEventType, true
	case domain.AgentProviderAvailabilityStateStale:
		return providerStaleEventType, true
	default:
		return "", false
	}
}

func mapMonitoredAgentProvider(item *ent.AgentProvider, machine monitoredMachine) domain.AgentProvider {
	return domain.AgentProvider{
		ID:                   item.ID,
		OrganizationID:       item.OrganizationID,
		MachineID:            item.MachineID,
		MachineName:          machine.Name,
		MachineHost:          machine.Host,
		MachineStatus:        domain.MachineStatus(machine.Status),
		MachineSSHUser:       cloneMachineString(machine.SSHUser),
		MachineWorkspaceRoot: cloneMachineString(machine.WorkspaceRoot),
		MachineAgentCLIPath:  cloneMachineString(machine.AgentCLIPath),
		MachineResources:     cloneResourceMap(machine.Resources),
		Name:                 item.Name,
		AdapterType:          domain.AgentProviderAdapterType(item.AdapterType.String()),
		CliCommand:           item.CliCommand,
		CliArgs:              append([]string(nil), item.CliArgs...),
		AuthConfig:           cloneResourceMap(item.AuthConfig),
		ModelName:            item.ModelName,
		ModelTemperature:     item.ModelTemperature,
		ModelMaxTokens:       item.ModelMaxTokens,
		CostPerInputToken:    item.CostPerInputToken,
		CostPerOutputToken:   item.CostPerOutputToken,
	}
}

func mapMachineEntity(item *ent.Machine) monitoredMachine {
	lastHeartbeatAt := time.Time{}
	if item.LastHeartbeatAt != nil {
		lastHeartbeatAt = item.LastHeartbeatAt.UTC()
	}

	return monitoredMachine{
		OrganizationID:     item.OrganizationID,
		ID:                 item.ID,
		Name:               item.Name,
		Host:               item.Host,
		Port:               item.Port,
		ConnectionMode:     item.ConnectionMode,
		SSHUser:            optionalMachineString(item.SSHUser),
		SSHKeyPath:         optionalMachineString(item.SSHKeyPath),
		AdvertisedEndpoint: optionalMachineString(item.AdvertisedEndpoint),
		WorkspaceRoot:      optionalMachineString(item.WorkspaceRoot),
		AgentCLIPath:       optionalMachineString(item.AgentCliPath),
		DaemonStatus: domain.MachineDaemonStatus{
			Registered:       item.DaemonRegistered,
			LastRegisteredAt: cloneTimePointer(item.DaemonLastRegisteredAt),
			CurrentSessionID: optionalMachineString(item.DaemonSessionID),
			SessionState:     domain.MachineTransportSessionState(item.DaemonSessionState),
		},
		Status:          item.Status,
		Labels:          append([]string(nil), item.Labels...),
		LastHeartbeatAt: lastHeartbeatAt,
		Resources:       cloneResourceMap(item.Resources),
	}
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func optionalMachineString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	cloned := value
	return &cloned
}

func cloneMachineString(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func stringPointerValue(value *string) *string {
	return cloneMachineString(value)
}

func timePointerRFC3339(value time.Time) *string {
	if value.IsZero() {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func timePointerRFC3339Pointer(value *time.Time) *string {
	if value == nil {
		return nil
	}
	return timePointerRFC3339(*value)
}

func machineMonitorDue(resources map[string]any, level string, now time.Time, interval time.Duration) bool {
	checkedAt, ok := machineMonitorCheckedAt(resources, level)
	if !ok {
		return true
	}
	return now.Sub(checkedAt) >= interval
}

func machineMonitorCheckedAt(resources map[string]any, level string) (time.Time, bool) {
	monitor, ok := nestedMap(resources, "monitor")
	if !ok {
		return time.Time{}, false
	}
	levelMap, ok := nestedMap(monitor, level)
	if !ok {
		return time.Time{}, false
	}
	raw, ok := levelMap["checked_at"].(string)
	if !ok || strings.TrimSpace(raw) == "" {
		return time.Time{}, false
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, false
	}
	return parsed.UTC(), true
}

func machineMonitorFailures(resources map[string]any) int {
	monitor, ok := nestedMap(resources, "monitor")
	if !ok {
		return 0
	}
	levelMap, ok := nestedMap(monitor, "l1")
	if !ok {
		return 0
	}
	switch raw := levelMap["consecutive_failures"].(type) {
	case int:
		return raw
	case float64:
		return int(raw)
	default:
		return 0
	}
}

func setMachineMonitorFailures(resources map[string]any, failures int) {
	levelMap := ensureMonitorLevel(resources, "l1")
	levelMap["consecutive_failures"] = failures
}

func setMachineMonitorError(resources map[string]any, level string, message string) {
	levelMap := ensureMonitorLevel(resources, level)
	levelMap["error"] = strings.TrimSpace(message)
}

func clearMachineMonitorError(resources map[string]any, level string) {
	levelMap := ensureMonitorLevel(resources, level)
	delete(levelMap, "error")
}

func updateL1Resources(resources map[string]any, reachability domain.MachineReachability) {
	levelMap := ensureMonitorLevel(resources, "l1")
	levelMap["checked_at"] = reachability.CheckedAt.UTC().Format(time.RFC3339)
	levelMap["transport"] = reachability.Transport
	levelMap["reachable"] = reachability.Reachable
	levelMap["latency_ms"] = reachability.LatencyMS
	if strings.TrimSpace(reachability.FailureCause) != "" {
		levelMap["failure_cause"] = reachability.FailureCause
	} else {
		delete(levelMap, "failure_cause")
	}

	resources["transport"] = reachability.Transport
	resources["checked_at"] = reachability.CheckedAt.UTC().Format(time.RFC3339)
	resources["last_success"] = reachability.Reachable
}

func updateL2Resources(resources map[string]any, systemResources domain.MachineSystemResources) {
	levelMap := ensureMonitorLevel(resources, "l2")
	levelMap["checked_at"] = systemResources.CollectedAt.UTC().Format(time.RFC3339)
	levelMap["memory_low"] = systemResources.MemoryAvailablePercent < lowMemoryThresholdPercent
	levelMap["disk_low"] = systemResources.DiskAvailableGB < lowDiskThresholdGB

	resources["cpu_cores"] = systemResources.CPUCores
	resources["cpu_usage_percent"] = systemResources.CPUUsagePercent
	resources["memory_total_gb"] = systemResources.MemoryTotalGB
	resources["memory_used_gb"] = systemResources.MemoryUsedGB
	resources["memory_available_gb"] = systemResources.MemoryAvailableGB
	resources["disk_total_gb"] = systemResources.DiskTotalGB
	resources["disk_available_gb"] = systemResources.DiskAvailableGB
	resources["collected_at"] = systemResources.CollectedAt.UTC().Format(time.RFC3339)
}

func updateL3Resources(resources map[string]any, gpuResources domain.MachineGPUResources) {
	levelMap := ensureMonitorLevel(resources, "l3")
	levelMap["checked_at"] = gpuResources.CollectedAt.UTC().Format(time.RFC3339)
	levelMap["available"] = gpuResources.Available

	if !gpuResources.Available {
		resources["gpu"] = []map[string]any{}
		levelMap["gpu_dispatchable"] = false
		resources["gpu_dispatchable"] = false
		return
	}

	gpus := make([]map[string]any, 0, len(gpuResources.GPUs))
	gpuDispatchable := false
	for _, gpu := range gpuResources.GPUs {
		if gpu.MemoryTotalGB-gpu.MemoryUsedGB > fullGPUMemoryThresholdGB {
			gpuDispatchable = true
		}
		gpus = append(gpus, map[string]any{
			"index":               gpu.Index,
			"name":                gpu.Name,
			"memory_total_gb":     gpu.MemoryTotalGB,
			"memory_used_gb":      gpu.MemoryUsedGB,
			"utilization_percent": gpu.UtilizationPercent,
		})
	}
	slices.SortFunc(gpus, func(left, right map[string]any) int {
		return compareAnyInt(left["index"], right["index"])
	})

	levelMap["gpu_dispatchable"] = gpuDispatchable
	resources["gpu"] = gpus
	resources["gpu_dispatchable"] = gpuDispatchable
}

func updateL4Resources(resources map[string]any, agentEnvironment domain.MachineAgentEnvironment) {
	levelMap := ensureMonitorLevel(resources, "l4")
	levelMap["checked_at"] = agentEnvironment.CollectedAt.UTC().Format(time.RFC3339)
	levelMap["agent_dispatchable"] = agentEnvironment.Dispatchable

	environmentSummary := make(map[string]any, len(agentEnvironment.CLIs))
	for _, cli := range agentEnvironment.CLIs {
		snapshot := map[string]any{
			"installed":   cli.Installed,
			"version":     cli.Version,
			"auth_status": string(cli.AuthStatus),
			"auth_mode":   string(cli.AuthMode),
			"ready":       cli.Ready,
		}
		levelMap[cli.Name] = cloneResourceMap(snapshot)
		environmentSummary[cli.Name] = snapshot
	}

	resources["agent_dispatchable"] = agentEnvironment.Dispatchable
	resources["agent_environment_checked_at"] = agentEnvironment.CollectedAt.UTC().Format(time.RFC3339)
	resources["agent_environment"] = environmentSummary
}

func updateL5Resources(resources map[string]any, fullAudit domain.MachineFullAudit) {
	levelMap := ensureMonitorLevel(resources, "l5")
	levelMap["checked_at"] = fullAudit.CollectedAt.UTC().Format(time.RFC3339)

	gitSummary := map[string]any{
		"installed":  fullAudit.Git.Installed,
		"user_name":  fullAudit.Git.UserName,
		"user_email": fullAudit.Git.UserEmail,
	}
	ghSummary := map[string]any{
		"installed":   fullAudit.GitHubCLI.Installed,
		"auth_status": string(fullAudit.GitHubCLI.AuthStatus),
	}
	githubTokenProbe := map[string]any{
		"state":       string(fullAudit.GitHubTokenProbe.State),
		"configured":  fullAudit.GitHubTokenProbe.Configured,
		"valid":       fullAudit.GitHubTokenProbe.Valid,
		"permissions": append([]string(nil), fullAudit.GitHubTokenProbe.Permissions...),
		"repo_access": string(fullAudit.GitHubTokenProbe.RepoAccess),
		"last_error":  fullAudit.GitHubTokenProbe.LastError,
	}
	if fullAudit.GitHubTokenProbe.CheckedAt != nil {
		githubTokenProbe["checked_at"] = fullAudit.GitHubTokenProbe.CheckedAt.UTC().Format(time.RFC3339)
	}
	networkSummary := map[string]any{
		"github_reachable": fullAudit.Network.GitHubReachable,
		"pypi_reachable":   fullAudit.Network.PyPIReachable,
		"npm_reachable":    fullAudit.Network.NPMReachable,
	}

	levelMap["git"] = cloneResourceMap(gitSummary)
	levelMap["gh_cli"] = cloneResourceMap(ghSummary)
	levelMap["github_token_probe"] = cloneResourceMap(githubTokenProbe)
	levelMap["network"] = cloneResourceMap(networkSummary)
	resources["full_audit"] = map[string]any{
		"checked_at":         fullAudit.CollectedAt.UTC().Format(time.RFC3339),
		"git":                gitSummary,
		"gh_cli":             ghSummary,
		"github_token_probe": githubTokenProbe,
		"network":            networkSummary,
	}
}

func machineHasLowDisk(resources map[string]any) bool {
	value, ok := resources["disk_available_gb"]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case float64:
		return typed < lowDiskThresholdGB
	case float32:
		return float64(typed) < lowDiskThresholdGB
	case int:
		return float64(typed) < lowDiskThresholdGB
	default:
		return false
	}
}

func ensureMonitorLevel(resources map[string]any, level string) map[string]any {
	monitor, ok := nestedMap(resources, "monitor")
	if !ok {
		monitor = map[string]any{}
		resources["monitor"] = monitor
	}
	levelMap, ok := nestedMap(monitor, level)
	if !ok {
		levelMap = map[string]any{}
		monitor[level] = levelMap
	}
	return levelMap
}

func nestedMap(resources map[string]any, key string) (map[string]any, bool) {
	raw, ok := resources[key]
	if !ok {
		return nil, false
	}
	typed, ok := raw.(map[string]any)
	return typed, ok
}

func cloneResourceMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}

	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		cloned[key] = cloneAnyValue(value)
	}
	return cloned
}

func cloneAnyValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, nestedValue := range typed {
			cloned[key] = cloneAnyValue(nestedValue)
		}
		return cloned
	case []map[string]any:
		cloned := make([]map[string]any, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloneResourceMap(item))
		}
		return cloned
	case []any:
		cloned := make([]any, 0, len(typed))
		for _, item := range typed {
			cloned = append(cloned, cloneAnyValue(item))
		}
		return cloned
	default:
		return value
	}
}

func compareAnyInt(left any, right any) int {
	leftValue := anyToInt(left)
	rightValue := anyToInt(right)
	switch {
	case leftValue < rightValue:
		return -1
	case leftValue > rightValue:
		return 1
	default:
		return 0
	}
}

func anyToInt(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	default:
		return 0
	}
}

func anyToBool(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(strings.TrimSpace(typed), "true")
	default:
		return false
	}
}
