package catalog

import (
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entagentrun "github.com/BetterAndBetterII/openase/ent/agentrun"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func toEntOrganizationStatus(status domain.OrganizationStatus) entorganization.Status {
	return entorganization.Status(status)
}

func toDomainOrganizationStatus(status entorganization.Status) domain.OrganizationStatus {
	return domain.OrganizationStatus(status)
}

func toEntProjectStatus(status domain.ProjectStatus) string {
	return string(status)
}

func toDomainProjectStatus(status string) domain.ProjectStatus {
	return domain.ProjectStatus(status)
}

func toEntMachineStatus(status domain.MachineStatus) entmachine.Status {
	return entmachine.Status(status)
}

func toDomainMachineStatus(status entmachine.Status) domain.MachineStatus {
	return domain.MachineStatus(status)
}

func toEntAgentProviderAdapterType(adapterType domain.AgentProviderAdapterType) entagentprovider.AdapterType {
	return entagentprovider.AdapterType(adapterType)
}

func toDomainAgentProviderAdapterType(adapterType entagentprovider.AdapterType) domain.AgentProviderAdapterType {
	return domain.AgentProviderAdapterType(adapterType)
}

func toEntAgentRuntimeControlState(state domain.AgentRuntimeControlState) entagent.RuntimeControlState {
	return entagent.RuntimeControlState(state)
}

func toDomainAgentRuntimeControlState(state entagent.RuntimeControlState) domain.AgentRuntimeControlState {
	return domain.AgentRuntimeControlState(state)
}

func toDomainAgentRunStatus(status entagentrun.Status) domain.AgentRunStatus {
	return domain.AgentRunStatus(status)
}
