package catalog

import (
	"strings"
	"time"
)

const (
	ProviderAvailabilityL4Interval          = 30 * time.Minute
	ProviderAvailabilityStaleAfter          = 2 * ProviderAvailabilityL4Interval
	providerReasonMachineOffline            = "machine_offline"
	providerReasonMachineDegraded           = "machine_degraded"
	providerReasonMachineMaintenance        = "machine_maintenance"
	providerReasonL4SnapshotMissing         = "l4_snapshot_missing"
	providerReasonStaleL4Snapshot           = "stale_l4_snapshot"
	providerReasonCLIMissing                = "cli_missing"
	providerReasonNotLoggedIn               = "not_logged_in"
	providerReasonNotReady                  = "not_ready"
	providerReasonConfigIncomplete          = "config_incomplete"
	providerReasonUnsupportedAdapter        = "unsupported_adapter"
	providerReasonRemoteMachineNotSupported = "remote_machine_not_supported"
	providerReasonSkillAIRequiresCodex      = "skill_ai_requires_codex"
)

func DeriveAgentProviderAvailability(item AgentProvider, now time.Time) AgentProvider {
	state, checkedAt, reason := ResolveAgentProviderAvailability(item, now)
	item.AvailabilityState = state
	item.Available = state == AgentProviderAvailabilityStateAvailable
	item.AvailabilityCheckedAt = cloneTimePointer(checkedAt)
	item.AvailabilityReason = cloneStringPointer(reason)
	return item
}

func ResolveAgentProviderAvailability(
	item AgentProvider,
	now time.Time,
) (AgentProviderAvailabilityState, *time.Time, *string) {
	switch item.MachineStatus {
	case MachineStatusOffline:
		return AgentProviderAvailabilityStateUnavailable, nil, availabilityReasonPointer(providerReasonMachineOffline)
	case MachineStatusDegraded:
		return AgentProviderAvailabilityStateUnavailable, nil, availabilityReasonPointer(providerReasonMachineDegraded)
	case MachineStatusMaintenance:
		return AgentProviderAvailabilityStateUnavailable, nil, availabilityReasonPointer(providerReasonMachineMaintenance)
	}

	l4Snapshot, checkedAt, ok := providerL4Snapshot(item.MachineResources)
	if !ok {
		return AgentProviderAvailabilityStateUnknown, nil, availabilityReasonPointer(providerReasonL4SnapshotMissing)
	}

	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if now.Sub(*checkedAt) > ProviderAvailabilityStaleAfter {
		return AgentProviderAvailabilityStateStale, checkedAt, availabilityReasonPointer(providerReasonStaleL4Snapshot)
	}

	cliSnapshot, ok := providerCLISnapshot(item.AdapterType, l4Snapshot)
	if !ok {
		return AgentProviderAvailabilityStateUnavailable, checkedAt, availabilityReasonPointer(providerReasonUnsupportedAdapter)
	}

	if installed, ok := cliSnapshot["installed"].(bool); !ok || !installed {
		return AgentProviderAvailabilityStateUnavailable, checkedAt, availabilityReasonPointer(providerReasonCLIMissing)
	}

	authStatus := MachineAgentAuthStatus(strings.ToLower(strings.TrimSpace(stringValue(cliSnapshot["auth_status"]))))
	authMode := MachineAgentAuthMode(strings.ToLower(strings.TrimSpace(stringValue(cliSnapshot["auth_mode"]))))
	if !providerAuthReady(authStatus, authMode) {
		return AgentProviderAvailabilityStateUnavailable, checkedAt, availabilityReasonPointer(providerReasonNotLoggedIn)
	}

	ready, ok := cliSnapshot["ready"].(bool)
	if !ok || !ready {
		return AgentProviderAvailabilityStateUnavailable, checkedAt, availabilityReasonPointer(providerReasonNotReady)
	}

	if !providerLaunchConfigComplete(item) {
		return AgentProviderAvailabilityStateUnavailable, checkedAt, availabilityReasonPointer(providerReasonConfigIncomplete)
	}

	return AgentProviderAvailabilityStateAvailable, checkedAt, nil
}

func providerL4Snapshot(resources map[string]any) (map[string]any, *time.Time, bool) {
	monitor, ok := providerNestedMap(resources, "monitor")
	if !ok {
		return nil, nil, false
	}
	l4Snapshot, ok := providerNestedMap(monitor, "l4")
	if !ok {
		return nil, nil, false
	}
	rawCheckedAt, ok := l4Snapshot["checked_at"].(string)
	if !ok || strings.TrimSpace(rawCheckedAt) == "" {
		return nil, nil, false
	}
	checkedAt, err := time.Parse(time.RFC3339, rawCheckedAt)
	if err != nil {
		return nil, nil, false
	}
	checkedAt = checkedAt.UTC()
	return l4Snapshot, &checkedAt, true
}

func providerCLISnapshot(
	adapterType AgentProviderAdapterType,
	l4Snapshot map[string]any,
) (map[string]any, bool) {
	entryName := ""
	switch adapterType {
	case AgentProviderAdapterTypeClaudeCodeCLI:
		entryName = "claude_code"
	case AgentProviderAdapterTypeCodexAppServer:
		entryName = "codex"
	case AgentProviderAdapterTypeGeminiCLI:
		entryName = "gemini"
	default:
		return nil, false
	}
	return providerNestedMap(l4Snapshot, entryName)
}

func providerNestedMap(raw map[string]any, key string) (map[string]any, bool) {
	value, ok := raw[key]
	if !ok {
		return nil, false
	}
	item, ok := value.(map[string]any)
	return item, ok
}

func providerAuthReady(status MachineAgentAuthStatus, mode MachineAgentAuthMode) bool {
	if mode == MachineAgentAuthModeAPIKey {
		return true
	}
	return status == MachineAgentAuthStatusLoggedIn
}

func providerLaunchConfigComplete(item AgentProvider) bool {
	command := strings.TrimSpace(item.CliCommand)
	if item.MachineAgentCLIPath != nil && strings.TrimSpace(*item.MachineAgentCLIPath) != "" {
		command = strings.TrimSpace(*item.MachineAgentCLIPath)
	}
	if command == "" {
		return false
	}

	if strings.TrimSpace(item.MachineHost) != "" && item.MachineHost != LocalMachineHost {
		return item.MachineWorkspaceRoot != nil && strings.TrimSpace(*item.MachineWorkspaceRoot) != ""
	}

	return true
}

func stringValue(raw any) string {
	value, _ := raw.(string)
	return value
}

func availabilityReasonPointer(value string) *string {
	copied := value
	return &copied
}
