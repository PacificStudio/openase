package catalog

import (
	entagent "github.com/BetterAndBetterII/openase/ent/agent"
	entagentprovider "github.com/BetterAndBetterII/openase/ent/agentprovider"
	entmachine "github.com/BetterAndBetterII/openase/ent/machine"
	entorganization "github.com/BetterAndBetterII/openase/ent/organization"
	entproject "github.com/BetterAndBetterII/openase/ent/project"
	entticketreposcope "github.com/BetterAndBetterII/openase/ent/ticketreposcope"
	domain "github.com/BetterAndBetterII/openase/internal/domain/catalog"
)

func toEntOrganizationStatus(status domain.OrganizationStatus) entorganization.Status {
	return entorganization.Status(status)
}

func toDomainOrganizationStatus(status entorganization.Status) domain.OrganizationStatus {
	return domain.OrganizationStatus(status)
}

func toEntProjectStatus(status domain.ProjectStatus) entproject.Status {
	return entproject.Status(status)
}

func toDomainProjectStatus(status entproject.Status) domain.ProjectStatus {
	return domain.ProjectStatus(status)
}

func toEntTicketRepoScopePRStatus(status domain.TicketRepoScopePRStatus) entticketreposcope.PrStatus {
	return entticketreposcope.PrStatus(status)
}

func toDomainTicketRepoScopePRStatus(status entticketreposcope.PrStatus) domain.TicketRepoScopePRStatus {
	return domain.TicketRepoScopePRStatus(status)
}

func toEntTicketRepoScopeCIStatus(status domain.TicketRepoScopeCIStatus) entticketreposcope.CiStatus {
	return entticketreposcope.CiStatus(status)
}

func toDomainTicketRepoScopeCIStatus(status entticketreposcope.CiStatus) domain.TicketRepoScopeCIStatus {
	return domain.TicketRepoScopeCIStatus(status)
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

func toEntAgentStatus(status domain.AgentStatus) entagent.Status {
	return entagent.Status(status)
}

func toDomainAgentStatus(status entagent.Status) domain.AgentStatus {
	return domain.AgentStatus(status)
}

func toEntAgentRuntimePhase(phase domain.AgentRuntimePhase) entagent.RuntimePhase {
	return entagent.RuntimePhase(phase)
}

func toDomainAgentRuntimePhase(phase entagent.RuntimePhase) domain.AgentRuntimePhase {
	return domain.AgentRuntimePhase(phase)
}

func toEntAgentRuntimeControlState(state domain.AgentRuntimeControlState) entagent.RuntimeControlState {
	return entagent.RuntimeControlState(state)
}

func toDomainAgentRuntimeControlState(state entagent.RuntimeControlState) domain.AgentRuntimeControlState {
	return domain.AgentRuntimeControlState(state)
}
