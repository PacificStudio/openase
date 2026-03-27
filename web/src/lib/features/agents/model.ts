import type { AgentPayload, AgentProvider, AgentRun, Ticket, Workflow } from '$lib/api/contracts'
import type { AgentInstance, AgentRunInstance, ProviderConfig } from './types'
import { normalizeAgentStatus, normalizeRuntimeControlState, normalizeRuntimePhase } from './state'

export function buildProviderCards(
  providerItems: AgentProvider[],
  agentItems: AgentPayload['agents'],
  defaultProviderId: string | null,
): ProviderConfig[] {
  return providerItems.map((provider) => ({
    id: provider.id,
    machineId: provider.machine_id,
    machineName: provider.machine_name,
    machineHost: provider.machine_host,
    machineStatus: provider.machine_status,
    machineWorkspaceRoot: provider.machine_workspace_root ?? null,
    name: provider.name,
    adapterType: provider.adapter_type,
    available: provider.available,
    cliCommand: provider.cli_command,
    cliArgs: [...provider.cli_args],
    authConfig: { ...provider.auth_config },
    modelName: provider.model_name,
    modelTemperature: provider.model_temperature,
    modelMaxTokens: provider.model_max_tokens,
    costPerInputToken: provider.cost_per_input_token,
    costPerOutputToken: provider.cost_per_output_token,
    agentCount: agentItems.filter((agent) => agent.provider_id === provider.id).length,
    isDefault: defaultProviderId === provider.id,
  }))
}

export function buildAgentRows(
  providerItems: AgentProvider[],
  ticketItems: Ticket[],
  agentItems: AgentPayload['agents'],
): AgentInstance[] {
  const ticketMap = new Map(ticketItems.map((ticket) => [ticket.id, ticket]))
  const providerMap = new Map(providerItems.map((provider) => [provider.id, provider]))

  return agentItems.map((agent) => {
    const provider = providerMap.get(agent.provider_id)
    const runtime = agent.runtime ?? null
    const activeRunCount =
      typeof runtime?.active_run_count === 'number'
        ? runtime.active_run_count
        : runtime?.current_run_id
          ? 1
          : 0
    const currentTicket =
      activeRunCount === 1 && runtime?.current_ticket_id
        ? ticketMap.get(runtime.current_ticket_id)
        : null

    return {
      id: agent.id,
      name: agent.name,
      providerId: agent.provider_id,
      providerName: provider?.name ?? 'Unknown provider',
      modelName: provider?.model_name ?? 'Unknown model',
      status: normalizeAgentStatus(runtime?.status ?? 'idle'),
      runtimePhase: normalizeRuntimePhase(runtime?.runtime_phase ?? 'none'),
      runtimeControlState: normalizeRuntimeControlState(agent.runtime_control_state),
      activeRunCount,
      currentTicket: currentTicket
        ? {
            id: currentTicket.id,
            identifier: currentTicket.identifier,
            title: currentTicket.title,
          }
        : undefined,
      lastHeartbeat: runtime?.last_heartbeat_at ?? null,
      runtimeStartedAt: runtime?.runtime_started_at ?? null,
      sessionId: runtime?.session_id ?? '',
      lastError: runtime?.last_error ?? '',
      todayCompleted: agent.total_tickets_completed,
      todayCost: 0,
    }
  })
}

export function buildAgentRunRows(
  providerItems: AgentProvider[],
  ticketItems: Ticket[],
  workflowItems: Workflow[],
  agentItems: AgentPayload['agents'],
  agentRunItems: AgentRun[],
): AgentRunInstance[] {
  const ticketMap = new Map(ticketItems.map((ticket) => [ticket.id, ticket]))
  const workflowMap = new Map(workflowItems.map((workflow) => [workflow.id, workflow]))
  const providerMap = new Map(providerItems.map((provider) => [provider.id, provider]))
  const agentMap = new Map(agentItems.map((agent) => [agent.id, agent]))

  return agentRunItems
    .map((agentRun) => {
      const ticket = ticketMap.get(agentRun.ticket_id)
      if (!ticket) {
        return null
      }

      const agent = agentMap.get(agentRun.agent_id)
      const provider = providerMap.get(agentRun.provider_id)
      const workflow = workflowMap.get(agentRun.workflow_id)

      return {
        id: agentRun.id,
        agentId: agentRun.agent_id,
        agentName: agent?.name ?? 'Unknown agent',
        providerId: agentRun.provider_id,
        providerName: provider?.name ?? 'Unknown provider',
        modelName: provider?.model_name ?? 'Unknown model',
        workflowId: agentRun.workflow_id,
        workflowName: workflow?.name ?? 'Unknown workflow',
        status: normalizeAgentRunStatus(agentRun.status),
        ticket: {
          id: ticket.id,
          identifier: ticket.identifier,
          title: ticket.title,
        },
        lastHeartbeat: agentRun.last_heartbeat_at ?? null,
        runtimeStartedAt: agentRun.runtime_started_at ?? null,
        sessionId: agentRun.session_id ?? '',
        lastError: agentRun.last_error ?? '',
        createdAt: agentRun.created_at,
      } satisfies AgentRunInstance
    })
    .filter((item): item is AgentRunInstance => item !== null)
}

function normalizeAgentRunStatus(status: string): AgentRunInstance['status'] {
  if (
    status === 'launching' ||
    status === 'ready' ||
    status === 'executing' ||
    status === 'completed' ||
    status === 'errored' ||
    status === 'terminated'
  ) {
    return status
  }

  return 'launching'
}

export function applyUpdatedProviderState(
  providers: ProviderConfig[],
  agents: AgentInstance[],
  updatedProvider: AgentProvider,
) {
  const nextProviders = providers.map((provider) =>
    provider.id === updatedProvider.id
      ? {
          ...provider,
          machineId: updatedProvider.machine_id,
          machineName: updatedProvider.machine_name,
          machineHost: updatedProvider.machine_host,
          machineStatus: updatedProvider.machine_status,
          machineWorkspaceRoot: updatedProvider.machine_workspace_root ?? null,
          name: updatedProvider.name,
          adapterType: updatedProvider.adapter_type,
          available: updatedProvider.available,
          cliCommand: updatedProvider.cli_command,
          cliArgs: [...updatedProvider.cli_args],
          authConfig: { ...updatedProvider.auth_config },
          modelName: updatedProvider.model_name,
          modelTemperature: updatedProvider.model_temperature,
          modelMaxTokens: updatedProvider.model_max_tokens,
          costPerInputToken: updatedProvider.cost_per_input_token,
          costPerOutputToken: updatedProvider.cost_per_output_token,
        }
      : provider,
  )
  const nextAgents = agents.map((agent) =>
    agent.providerId === updatedProvider.id
      ? {
          ...agent,
          providerName: updatedProvider.name,
          modelName: updatedProvider.model_name,
        }
      : agent,
  )

  return {
    providers: nextProviders,
    agents: nextAgents,
    provider: nextProviders.find((provider) => provider.id === updatedProvider.id) ?? null,
  }
}
