package machinechannel

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/BetterAndBetterII/openase/ent"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entmachinechanneltoken "github.com/BetterAndBetterII/openase/ent/machinechanneltoken"
	domaincatalog "github.com/BetterAndBetterII/openase/internal/domain/catalog"
	domain "github.com/BetterAndBetterII/openase/internal/domain/machinechannel"
	service "github.com/BetterAndBetterII/openase/internal/machinechannel"
	"github.com/google/uuid"
)

type EntRepository struct {
	client *ent.Client
}

func NewEntRepository(client *ent.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) GetMachine(ctx context.Context, machineID uuid.UUID) (service.MachineRecord, error) {
	item, err := r.client.Machine.Get(ctx, machineID)
	if err != nil {
		return service.MachineRecord{}, mapReadError("get machine for machine channel", err)
	}
	return mapMachineRecord(item), nil
}

func (r *EntRepository) IssueToken(ctx context.Context, input service.CreateTokenRecord) (service.TokenRecord, error) {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return service.TokenRecord{}, fmt.Errorf("start machine channel token tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item, err := tx.MachineChannelToken.Create().
		SetMachineID(input.MachineID).
		SetTokenHash(input.TokenHash).
		SetExpiresAt(input.ExpiresAt.UTC()).
		Save(ctx)
	if err != nil {
		return service.TokenRecord{}, mapWriteError("create machine channel token", err)
	}
	if _, err := tx.Machine.UpdateOneID(input.MachineID).
		SetChannelCredentialKind(entmachine.ChannelCredentialKindToken).
		SetChannelTokenID(item.ID.String()).
		ClearChannelCertificateID().
		Save(ctx); err != nil {
		return service.TokenRecord{}, mapWriteError("set machine channel token pointer", err)
	}
	if err = tx.Commit(); err != nil {
		return service.TokenRecord{}, fmt.Errorf("commit machine channel token tx: %w", err)
	}
	return mapTokenRecord(item), nil
}

func (r *EntRepository) TokenByHash(ctx context.Context, tokenHash string) (service.TokenRecord, error) {
	item, err := r.client.MachineChannelToken.Query().
		Where(entmachinechanneltoken.TokenHashEQ(strings.TrimSpace(tokenHash))).
		Only(ctx)
	if err != nil {
		return service.TokenRecord{}, mapReadError("get machine channel token", err)
	}
	return mapTokenRecord(item), nil
}

func (r *EntRepository) TouchTokenLastUsed(ctx context.Context, tokenID uuid.UUID, usedAt time.Time) error {
	if err := r.client.MachineChannelToken.UpdateOneID(tokenID).SetLastUsedAt(usedAt.UTC()).Exec(ctx); err != nil {
		return mapWriteError("touch machine channel token", err)
	}
	return nil
}

func (r *EntRepository) RevokeToken(ctx context.Context, machineID uuid.UUID, tokenID uuid.UUID, revokedAt time.Time) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return fmt.Errorf("start revoke machine channel token tx: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	item, err := tx.MachineChannelToken.Get(ctx, tokenID)
	if err != nil {
		return mapReadError("get machine channel token for revoke", err)
	}
	if item.MachineID != machineID {
		return domain.ErrInvalidToken
	}
	if _, err := tx.MachineChannelToken.UpdateOneID(tokenID).
		SetStatus(entmachinechanneltoken.StatusRevoked).
		SetRevokedAt(revokedAt.UTC()).
		Save(ctx); err != nil {
		return mapWriteError("revoke machine channel token", err)
	}
	currentMachine, err := tx.Machine.Get(ctx, machineID)
	if err != nil {
		return mapReadError("get machine after token revoke", err)
	}
	if strings.TrimSpace(currentMachine.ChannelTokenID) == tokenID.String() {
		if _, err := tx.Machine.UpdateOneID(machineID).
			SetChannelCredentialKind(entmachine.ChannelCredentialKindNone).
			ClearChannelTokenID().
			Save(ctx); err != nil {
			return mapWriteError("clear revoked machine channel token pointer", err)
		}
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit revoke machine channel token tx: %w", err)
	}
	return nil
}

func (r *EntRepository) RecordConnectedSession(ctx context.Context, input service.ConnectedSessionRecord) (service.MachineRecord, error) {
	item, err := r.client.Machine.Get(ctx, input.MachineID)
	if err != nil {
		return service.MachineRecord{}, mapReadError("get machine for connected session", err)
	}
	resources := mergeMachineResources(item.Resources, input.ConnectedAt, input.SystemInfo, input.ToolInventory, input.ResourceSnapshot, input.WebsocketHealth)
	updated, err := r.client.Machine.UpdateOneID(input.MachineID).
		SetDaemonRegistered(true).
		SetDaemonLastRegisteredAt(input.ConnectedAt.UTC()).
		SetDaemonSessionID(strings.TrimSpace(input.SessionID)).
		SetDaemonSessionState(entmachine.DaemonSessionStateConnected).
		SetLastHeartbeatAt(input.ConnectedAt.UTC()).
		SetStatus(entmachine.StatusOnline).
		SetDetectedOs(entmachine.DetectedOs(normalizeDetectedOS(input.SystemInfo.OS))).
		SetDetectedArch(entmachine.DetectedArch(normalizeDetectedArch(input.SystemInfo.Arch))).
		SetDetectionStatus(entmachine.DetectionStatus(domaincatalog.MachineDetectionStatusOK.String())).
		SetResources(resources).
		Save(ctx)
	if err != nil {
		return service.MachineRecord{}, mapWriteError("record machine connected session", err)
	}
	return mapMachineRecord(updated), nil
}

func (r *EntRepository) RecordHeartbeat(ctx context.Context, input service.HeartbeatRecord) (service.MachineRecord, error) {
	item, err := r.client.Machine.Get(ctx, input.MachineID)
	if err != nil {
		return service.MachineRecord{}, mapReadError("get machine for heartbeat", err)
	}
	if strings.TrimSpace(item.DaemonSessionID) != strings.TrimSpace(input.SessionID) {
		return service.MachineRecord{}, domain.ErrSessionReplaced
	}
	systemInfo := input.SystemInfo
	if systemInfo == nil {
		systemInfo = &domain.SystemInfo{}
	}
	resources := mergeMachineResources(item.Resources, input.HeartbeatAt, *systemInfo, input.ToolInventory, input.ResourceSnapshot, input.WebsocketHealth)
	builder := r.client.Machine.UpdateOneID(input.MachineID).
		SetDaemonRegistered(true).
		SetDaemonLastRegisteredAt(input.HeartbeatAt.UTC()).
		SetDaemonSessionID(strings.TrimSpace(input.SessionID)).
		SetDaemonSessionState(entmachine.DaemonSessionStateConnected).
		SetLastHeartbeatAt(input.HeartbeatAt.UTC()).
		SetStatus(entmachine.StatusOnline).
		SetResources(resources)
	if strings.TrimSpace(systemInfo.OS) != "" {
		builder.SetDetectedOs(entmachine.DetectedOs(normalizeDetectedOS(systemInfo.OS)))
	}
	if strings.TrimSpace(systemInfo.Arch) != "" {
		builder.SetDetectedArch(entmachine.DetectedArch(normalizeDetectedArch(systemInfo.Arch)))
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return service.MachineRecord{}, mapWriteError("record machine heartbeat", err)
	}
	return mapMachineRecord(updated), nil
}

func (r *EntRepository) RecordDisconnectedSession(ctx context.Context, input service.DisconnectedSessionRecord) (service.MachineRecord, error) {
	item, err := r.client.Machine.Get(ctx, input.MachineID)
	if err != nil {
		return service.MachineRecord{}, mapReadError("get machine for disconnect", err)
	}
	resources := applyDisconnectedWebsocketHealth(item.Resources, input.DisconnectedAt, input.Reason, item.ConnectionMode)
	builder := r.client.Machine.UpdateOneID(input.MachineID).
		SetDaemonRegistered(false).
		SetDaemonSessionState(entmachine.DaemonSessionStateDisconnected).
		SetLastHeartbeatAt(input.DisconnectedAt.UTC()).
		SetStatus(entmachine.StatusOffline).
		SetResources(resources)
	if strings.TrimSpace(item.DaemonSessionID) == strings.TrimSpace(input.SessionID) {
		builder.ClearDaemonSessionID()
	}
	updated, err := builder.Save(ctx)
	if err != nil {
		return service.MachineRecord{}, mapWriteError("record machine disconnect", err)
	}
	return mapMachineRecord(updated), nil
}

func mapMachineRecord(item *ent.Machine) service.MachineRecord {
	return service.MachineRecord{
		ID:                    item.ID,
		OrganizationID:        item.OrganizationID,
		Name:                  item.Name,
		ConnectionMode:        string(item.ConnectionMode),
		ReachabilityMode:      string(item.ReachabilityMode),
		ExecutionMode:         string(item.ExecutionMode),
		Status:                string(item.Status),
		ChannelCredentialKind: string(item.ChannelCredentialKind),
		ChannelTokenID:        optionalString(item.ChannelTokenID),
	}
}

func mapTokenRecord(item *ent.MachineChannelToken) service.TokenRecord {
	return service.TokenRecord{
		TokenID:   item.ID,
		MachineID: item.MachineID,
		TokenHash: item.TokenHash,
		Status:    string(item.Status),
		ExpiresAt: item.ExpiresAt.UTC(),
		RevokedAt: cloneTimePointer(item.RevokedAt),
	}
}

func optionalString(raw string) *string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copied := value.UTC()
	return &copied
}

func normalizeDetectedOS(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case domaincatalog.MachineDetectedOSDarwin.String():
		return domaincatalog.MachineDetectedOSDarwin.String()
	case domaincatalog.MachineDetectedOSLinux.String():
		return domaincatalog.MachineDetectedOSLinux.String()
	default:
		return domaincatalog.MachineDetectedOSUnknown.String()
	}
}

func normalizeDetectedArch(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case domaincatalog.MachineDetectedArchAMD64.String():
		return domaincatalog.MachineDetectedArchAMD64.String()
	case domaincatalog.MachineDetectedArchARM64.String():
		return domaincatalog.MachineDetectedArchARM64.String()
	default:
		return domaincatalog.MachineDetectedArchUnknown.String()
	}
}

func mergeMachineResources(
	current map[string]any,
	collectedAt time.Time,
	systemInfo domain.SystemInfo,
	toolInventory []domain.ToolInfo,
	snapshot *domain.ResourceSnapshot,
	websocketHealth *domain.WebsocketHealth,
) map[string]any {
	resources := cloneMap(current)
	checkedAt := collectedAt.UTC().Format(time.RFC3339)
	monitor := ensureMap(resources, "monitor")
	l1 := ensureMap(monitor, "l1")
	l1["checked_at"] = checkedAt
	l1["transport"] = domaincatalog.MachineConnectionModeWSReverse.String()
	l1["reachable"] = true
	delete(l1, "failure_cause")
	resources["transport"] = domaincatalog.MachineConnectionModeWSReverse.String()
	resources["checked_at"] = checkedAt
	resources["last_success"] = true
	resources["machine_channel"] = map[string]any{
		"hostname":            strings.TrimSpace(systemInfo.Hostname),
		"os":                  strings.TrimSpace(systemInfo.OS),
		"arch":                strings.TrimSpace(systemInfo.Arch),
		"openase_binary_path": strings.TrimSpace(systemInfo.OpenASEBinaryPath),
		"agent_cli_path":      strings.TrimSpace(systemInfo.AgentCLIPath),
		"checked_at":          checkedAt,
	}
	storeWebsocketHealth(resources, collectedAt, toolInventory, websocketHealth)
	if len(toolInventory) > 0 {
		l4 := ensureMap(monitor, "l4")
		l4["checked_at"] = checkedAt
		environmentSummary := map[string]any{}
		dispatchable := false
		for _, tool := range toolInventory {
			entry := map[string]any{
				"installed":   tool.Installed,
				"version":     strings.TrimSpace(tool.Version),
				"auth_status": strings.TrimSpace(tool.AuthStatus),
				"auth_mode":   strings.TrimSpace(tool.AuthMode),
				"ready":       tool.Ready,
			}
			l4[tool.Name] = cloneMap(entry)
			environmentSummary[tool.Name] = entry
			if tool.Ready {
				dispatchable = true
			}
		}
		l4["agent_dispatchable"] = dispatchable
		resources["agent_dispatchable"] = dispatchable
		resources["agent_environment_checked_at"] = checkedAt
		resources["agent_environment"] = environmentSummary
	}
	if snapshot == nil {
		return resources
	}
	collectedValue := strings.TrimSpace(snapshot.CollectedAt)
	if collectedValue == "" {
		collectedValue = checkedAt
	}
	l2 := ensureMap(monitor, "l2")
	l2["checked_at"] = collectedValue
	l2["memory_low"] = snapshot.MemoryAvailableGB > 0 && snapshot.MemoryTotalGB > 0 && (snapshot.MemoryAvailableGB/snapshot.MemoryTotalGB)*100 < 10
	l2["disk_low"] = snapshot.DiskAvailableGB < 5
	resources["cpu_cores"] = snapshot.CPUCores
	resources["cpu_usage_percent"] = snapshot.CPUUsagePercent
	resources["memory_total_gb"] = snapshot.MemoryTotalGB
	resources["memory_used_gb"] = snapshot.MemoryUsedGB
	resources["memory_available_gb"] = snapshot.MemoryAvailableGB
	resources["disk_total_gb"] = snapshot.DiskTotalGB
	resources["disk_available_gb"] = snapshot.DiskAvailableGB
	resources["collected_at"] = collectedValue
	l3 := ensureMap(monitor, "l3")
	l3["checked_at"] = collectedValue
	l3["available"] = len(snapshot.GPUs) > 0
	gpuDispatchable := false
	gpus := make([]map[string]any, 0, len(snapshot.GPUs))
	for _, gpu := range snapshot.GPUs {
		if gpu.MemoryTotalGB-gpu.MemoryUsedGB > 0.5 {
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
	l3["gpu_dispatchable"] = gpuDispatchable
	resources["gpu"] = gpus
	resources["gpu_dispatchable"] = gpuDispatchable
	if snapshot.FullAudit != nil {
		l5 := ensureMap(monitor, "l5")
		l5["checked_at"] = collectedValue
		gitSummary := map[string]any{
			"installed":  snapshot.FullAudit.Git.Installed,
			"user_name":  strings.TrimSpace(snapshot.FullAudit.Git.UserName),
			"user_email": strings.TrimSpace(snapshot.FullAudit.Git.UserEmail),
		}
		ghSummary := map[string]any{
			"installed":   snapshot.FullAudit.GitHubCLI.Installed,
			"auth_status": strings.TrimSpace(snapshot.FullAudit.GitHubCLI.AuthStatus),
		}
		githubProbe := map[string]any{
			"state":       strings.TrimSpace(snapshot.FullAudit.GitHubTokenProbe.State),
			"configured":  snapshot.FullAudit.GitHubTokenProbe.Configured,
			"valid":       snapshot.FullAudit.GitHubTokenProbe.Valid,
			"permissions": append([]string(nil), snapshot.FullAudit.GitHubTokenProbe.Permissions...),
			"repo_access": strings.TrimSpace(snapshot.FullAudit.GitHubTokenProbe.RepoAccess),
			"last_error":  strings.TrimSpace(snapshot.FullAudit.GitHubTokenProbe.LastError),
		}
		networkSummary := map[string]any{
			"github_reachable": snapshot.FullAudit.Network.GitHubReachable,
			"pypi_reachable":   snapshot.FullAudit.Network.PyPIReachable,
			"npm_reachable":    snapshot.FullAudit.Network.NPMReachable,
		}
		l5["git"] = cloneMap(gitSummary)
		l5["gh_cli"] = cloneMap(ghSummary)
		l5["github_token_probe"] = cloneMap(githubProbe)
		l5["network"] = cloneMap(networkSummary)
		resources["full_audit"] = map[string]any{
			"checked_at":         collectedValue,
			"git":                gitSummary,
			"gh_cli":             ghSummary,
			"github_token_probe": githubProbe,
			"network":            networkSummary,
		}
	}
	return resources
}

func storeWebsocketHealth(
	resources map[string]any,
	collectedAt time.Time,
	toolInventory []domain.ToolInfo,
	report *domain.WebsocketHealth,
) {
	health := defaultWebsocketHealth(collectedAt, toolInventory)
	if parsed, err := parseReportedWebsocketHealth(report); err == nil {
		health = parsed
	} else if existing, existingErr := domaincatalog.ParseStoredWebsocketMachineHealth(resources); existingErr == nil {
		health.L2 = existing.L2
		health.L3 = existing.L3
	}
	resources["websocket_health"] = domaincatalog.StoreWebsocketMachineHealth(health)
}

func defaultWebsocketHealth(collectedAt time.Time, toolInventory []domain.ToolInfo) domaincatalog.WebsocketMachineHealth {
	observedAt := collectedAt.UTC()
	readyCount := 0
	for _, tool := range toolInventory {
		if tool.Ready {
			readyCount++
		}
	}
	l5State := domaincatalog.WebsocketHealthStateHealthy
	l5Reason := ""
	if readyCount == 0 {
		l5State = domaincatalog.WebsocketHealthStateFailed
		l5Reason = "no ready machine-agent runtimes reported"
	}
	return domaincatalog.WebsocketMachineHealth{
		TransportMode: domaincatalog.MachineConnectionModeWSReverse,
		CheckedAt:     observedAt,
		L2:            domaincatalog.WebsocketHealthUnknownLayer(observedAt, "machine-agent did not report link telemetry", map[string]any{}),
		L3:            domaincatalog.WebsocketHealthUnknownLayer(observedAt, "machine-agent did not report control-plane network telemetry", map[string]any{}),
		L4: domaincatalog.WebsocketHealthLayer{
			State:      domaincatalog.WebsocketHealthStateHealthy,
			ObservedAt: observedAt,
			Details: map[string]any{
				"session_state": "connected",
			},
		},
		L5: domaincatalog.WebsocketHealthLayer{
			State:      l5State,
			Reason:     l5Reason,
			ObservedAt: observedAt,
			Details: map[string]any{
				"ready_runtime_count": readyCount,
				"tool_count":          len(toolInventory),
			},
		},
	}
}

func parseReportedWebsocketHealth(report *domain.WebsocketHealth) (domaincatalog.WebsocketMachineHealth, error) {
	if report == nil {
		return domaincatalog.WebsocketMachineHealth{}, fmt.Errorf("missing websocket health report")
	}
	checkedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(report.CheckedAt))
	if err != nil {
		return domaincatalog.WebsocketMachineHealth{}, fmt.Errorf("parse websocket health checked_at: %w", err)
	}
	transportMode, err := domaincatalog.ParseStoredMachineConnectionMode(strings.TrimSpace(report.TransportMode), "remote")
	if err != nil {
		return domaincatalog.WebsocketMachineHealth{}, fmt.Errorf("parse websocket health transport mode: %w", err)
	}
	l2, err := parseReportedWebsocketHealthLayer(report.L2, "l2")
	if err != nil {
		return domaincatalog.WebsocketMachineHealth{}, err
	}
	l3, err := parseReportedWebsocketHealthLayer(report.L3, "l3")
	if err != nil {
		return domaincatalog.WebsocketMachineHealth{}, err
	}
	l4, err := parseReportedWebsocketHealthLayer(report.L4, "l4")
	if err != nil {
		return domaincatalog.WebsocketMachineHealth{}, err
	}
	l5, err := parseReportedWebsocketHealthLayer(report.L5, "l5")
	if err != nil {
		return domaincatalog.WebsocketMachineHealth{}, err
	}
	return domaincatalog.WebsocketMachineHealth{
		TransportMode: transportMode,
		CheckedAt:     checkedAt.UTC(),
		L2:            l2,
		L3:            l3,
		L4:            l4,
		L5:            l5,
	}, nil
}

func parseReportedWebsocketHealthLayer(report domain.WebsocketHealthLayer, layer string) (domaincatalog.WebsocketHealthLayer, error) {
	state, err := domaincatalog.ParseStoredWebsocketHealthState(report.State)
	if err != nil {
		return domaincatalog.WebsocketHealthLayer{}, fmt.Errorf("parse websocket health %s state: %w", layer, err)
	}
	observedAt, err := time.Parse(time.RFC3339, strings.TrimSpace(report.ObservedAt))
	if err != nil {
		return domaincatalog.WebsocketHealthLayer{}, fmt.Errorf("parse websocket health %s observed_at: %w", layer, err)
	}
	return domaincatalog.WebsocketHealthLayer{
		State:      state,
		Reason:     strings.TrimSpace(report.Reason),
		ObservedAt: observedAt.UTC(),
		Details:    cloneMap(report.Details),
	}, nil
}

func applyDisconnectedWebsocketHealth(
	current map[string]any,
	disconnectedAt time.Time,
	reason string,
	connectionMode entmachine.ConnectionMode,
) map[string]any {
	resources := cloneMap(current)
	if connectionMode != entmachine.ConnectionModeWsReverse && connectionMode != entmachine.ConnectionModeWsListener {
		return resources
	}
	health, err := domaincatalog.ParseStoredWebsocketMachineHealth(resources)
	if err != nil {
		health = domaincatalog.WebsocketMachineHealth{
			TransportMode: domaincatalog.MachineConnectionMode(connectionMode),
			CheckedAt:     disconnectedAt.UTC(),
			L2:            domaincatalog.WebsocketHealthUnknownLayer(disconnectedAt, "link state unavailable after disconnect", map[string]any{}),
			L3:            domaincatalog.WebsocketHealthUnknownLayer(disconnectedAt, "network state unavailable after disconnect", map[string]any{}),
			L5:            domaincatalog.WebsocketHealthUnknownLayer(disconnectedAt, "application state unavailable after transport disconnect", map[string]any{}),
		}
	}
	health.CheckedAt = disconnectedAt.UTC()
	health.L4 = domaincatalog.WebsocketHealthLayer{
		State:      domaincatalog.WebsocketHealthStateFailed,
		Reason:     strings.TrimSpace(reason),
		ObservedAt: disconnectedAt.UTC(),
		Details: map[string]any{
			"session_state": "disconnected",
		},
	}
	health.L5 = domaincatalog.WebsocketHealthUnknownLayer(
		disconnectedAt,
		"application state unavailable after transport disconnect",
		map[string]any{"transport_reason": strings.TrimSpace(reason)},
	)
	resources["websocket_health"] = domaincatalog.StoreWebsocketMachineHealth(health)
	return resources
}

func cloneMap(raw map[string]any) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(raw))
	for key, value := range raw {
		nested, ok := value.(map[string]any)
		if ok {
			cloned[key] = cloneMap(nested)
			continue
		}
		cloned[key] = value
	}
	return cloned
}

func ensureMap(target map[string]any, key string) map[string]any {
	if value, ok := target[key].(map[string]any); ok {
		return value
	}
	value := map[string]any{}
	target[key] = value
	return value
}

func mapReadError(action string, err error) error {
	if ent.IsNotFound(err) {
		return domain.ErrNotFound
	}
	return fmt.Errorf("%s: %w", action, err)
}

func mapWriteError(action string, err error) error {
	if ent.IsNotFound(err) {
		return domain.ErrNotFound
	}
	if ent.IsConstraintError(err) {
		return fmt.Errorf("%s: conflict: %w", action, err)
	}
	return fmt.Errorf("%s: %w", action, err)
}
