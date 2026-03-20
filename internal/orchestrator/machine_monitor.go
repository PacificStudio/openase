package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	"github.com/google/uuid"
)

const (
	machineMonitorLevel1Interval  = 15 * time.Second
	machineMonitorLevel2Interval  = time.Minute
	machineMonitorLevel3Interval  = 5 * time.Minute
	machineMonitorOfflineFailures = 3
	lowDiskThresholdGB            = 5.0
	lowMemoryThresholdPercent     = 10.0
	fullGPUMemoryThresholdGB      = 0.5
)

type MachineMonitorCollector interface {
	CollectReachability(ctx context.Context, machine domain.Machine) (domain.MachineReachability, error)
	CollectSystemResources(ctx context.Context, machine domain.Machine) (domain.MachineSystemResources, error)
	CollectGPUResources(ctx context.Context, machine domain.Machine) (domain.MachineGPUResources, error)
}

type MachineMonitorReport struct {
	MachinesScanned  int `json:"machines_scanned"`
	MachinesUpdated  int `json:"machines_updated"`
	L1Checks         int `json:"l1_checks"`
	L2Checks         int `json:"l2_checks"`
	L3Checks         int `json:"l3_checks"`
	OfflineMachines  int `json:"offline_machines"`
	DegradedMachines int `json:"degraded_machines"`
}

type MachineMonitor struct {
	client    *ent.Client
	logger    *slog.Logger
	collector MachineMonitorCollector
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
	if shouldSkipMachineMonitor(items) {
		return report, nil
	}

	now := m.now().UTC()
	for _, item := range items {
		report.MachinesScanned++

		updated, changed := m.runMachineTick(ctx, mapMachineEntity(item), now, &report)
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
	}

	return report, nil
}

type monitoredMachine struct {
	ID              uuid.UUID
	Name            string
	Host            string
	Port            int
	SSHUser         *string
	SSHKeyPath      *string
	Status          entmachine.Status
	Labels          []string
	LastHeartbeatAt time.Time
	Resources       map[string]any
}

func (m *MachineMonitor) runMachineTick(ctx context.Context, machine monitoredMachine, now time.Time, report *MachineMonitorReport) (monitoredMachine, bool) {
	level1Due := machineMonitorDue(machine.Resources, "l1", now, machineMonitorLevel1Interval)
	level2Due := machineMonitorDue(machine.Resources, "l2", now, machineMonitorLevel2Interval)
	level3Due := hasMachineLabel(machine.Labels, "gpu") && machineMonitorDue(machine.Resources, "l3", now, machineMonitorLevel3Interval)
	if !level1Due && !level2Due && !level3Due {
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

	if level1Due {
		report.L1Checks++
		reachability, err := m.collector.CollectReachability(ctx, domain.Machine{
			ID:         machine.ID,
			Name:       machine.Name,
			Host:       machine.Host,
			Port:       machine.Port,
			SSHUser:    machine.SSHUser,
			SSHKeyPath: machine.SSHKeyPath,
			Labels:     append([]string(nil), machine.Labels...),
		})
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

	if level2Due && !softReachabilityFailure && !hardReachabilityFailure {
		report.L2Checks++
		systemResources, err := m.collector.CollectSystemResources(ctx, domain.Machine{
			ID:         machine.ID,
			Name:       machine.Name,
			Host:       machine.Host,
			Port:       machine.Port,
			SSHUser:    machine.SSHUser,
			SSHKeyPath: machine.SSHKeyPath,
			Labels:     append([]string(nil), machine.Labels...),
		})
		if err != nil {
			systemProbeFailure = true
			setMachineMonitorError(resources, "l2", err.Error())
		} else {
			updateL2Resources(resources, systemResources)
			clearMachineMonitorError(resources, "l2")
		}
	}

	if level3Due && !softReachabilityFailure && !hardReachabilityFailure {
		report.L3Checks++
		gpuResources, err := m.collector.CollectGPUResources(ctx, domain.Machine{
			ID:         machine.ID,
			Name:       machine.Name,
			Host:       machine.Host,
			Port:       machine.Port,
			SSHUser:    machine.SSHUser,
			SSHKeyPath: machine.SSHKeyPath,
			Labels:     append([]string(nil), machine.Labels...),
		})
		if err != nil {
			setMachineMonitorError(resources, "l3", err.Error())
		} else {
			updateL3Resources(resources, gpuResources)
			clearMachineMonitorError(resources, "l3")
		}
	}

	if machine.Status != entmachine.StatusMaintenance {
		switch {
		case hardReachabilityFailure:
			status = entmachine.StatusOffline
		case softReachabilityFailure || systemProbeFailure || machineHasLowDisk(resources):
			status = entmachine.StatusDegraded
		default:
			status = entmachine.StatusOnline
		}
	}

	return monitoredMachine{
		ID:              machine.ID,
		Name:            machine.Name,
		Host:            machine.Host,
		Status:          status,
		Labels:          append([]string(nil), machine.Labels...),
		LastHeartbeatAt: lastHeartbeatAt,
		Resources:       resources,
	}, true
}

func shouldSkipMachineMonitor(items []*ent.Machine) bool {
	return len(items) == 1 && items[0].Name == domain.LocalMachineName && items[0].Host == domain.LocalMachineHost
}

func mapMachineEntity(item *ent.Machine) monitoredMachine {
	lastHeartbeatAt := time.Time{}
	if item.LastHeartbeatAt != nil {
		lastHeartbeatAt = item.LastHeartbeatAt.UTC()
	}

	return monitoredMachine{
		ID:              item.ID,
		Name:            item.Name,
		Host:            item.Host,
		Port:            item.Port,
		SSHUser:         optionalMachineString(item.SSHUser),
		SSHKeyPath:      optionalMachineString(item.SSHKeyPath),
		Status:          item.Status,
		Labels:          append([]string(nil), item.Labels...),
		LastHeartbeatAt: lastHeartbeatAt,
		Resources:       cloneResourceMap(item.Resources),
	}
}

func optionalMachineString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	cloned := value
	return &cloned
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
		levelMap["gpu_dispatchable"] = true
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

func hasMachineLabel(labels []string, label string) bool {
	for _, item := range labels {
		if strings.EqualFold(strings.TrimSpace(item), label) {
			return true
		}
	}
	return false
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
