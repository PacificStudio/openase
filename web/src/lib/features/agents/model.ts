import type { AgentPayload, AgentProvider, AgentRun, Ticket, Workflow } from '$lib/api/contracts'
import { normalizeProviderAvailabilityState } from '$lib/features/providers'
import type {
  AgentInstance,
  AgentRunInstance,
  ProviderCLIRateLimit,
  ProviderConfig,
  ProviderPermissionProfile,
} from './types'
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
    permissionProfile: normalizeProviderPermissionProfile(provider.permission_profile),
    availabilityState: normalizeProviderAvailabilityState(provider.availability_state),
    available: provider.available,
    availabilityCheckedAt: provider.availability_checked_at ?? null,
    availabilityReason: provider.availability_reason ?? null,
    cliCommand: provider.cli_command,
    cliArgs: [...provider.cli_args],
    authConfig: { ...provider.auth_config },
    cliRateLimit: normalizeProviderCLIRateLimit(provider.cli_rate_limit),
    cliRateLimitUpdatedAt: provider.cli_rate_limit_updated_at ?? null,
    modelName: provider.model_name,
    modelTemperature: provider.model_temperature,
    modelMaxTokens: provider.model_max_tokens,
    maxParallelRuns: provider.max_parallel_runs,
    costPerInputToken: provider.cost_per_input_token,
    costPerOutputToken: provider.cost_per_output_token,
    pricingConfig: provider.pricing_config ?? {},
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
      adapterType: provider?.adapter_type ?? 'custom',
      permissionProfile: normalizeProviderPermissionProfile(provider?.permission_profile),
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
      currentStepStatus: runtime?.current_step_status ?? undefined,
      currentStepSummary: runtime?.current_step_summary ?? undefined,
      currentStepChangedAt: runtime?.current_step_changed_at ?? null,
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
          permissionProfile: normalizeProviderPermissionProfile(updatedProvider.permission_profile),
          availabilityState: normalizeProviderAvailabilityState(updatedProvider.availability_state),
          available: updatedProvider.available,
          availabilityCheckedAt: updatedProvider.availability_checked_at ?? null,
          availabilityReason: updatedProvider.availability_reason ?? null,
          cliCommand: updatedProvider.cli_command,
          cliArgs: [...updatedProvider.cli_args],
          authConfig: { ...updatedProvider.auth_config },
          cliRateLimit: normalizeProviderCLIRateLimit(updatedProvider.cli_rate_limit),
          cliRateLimitUpdatedAt: updatedProvider.cli_rate_limit_updated_at ?? null,
          modelName: updatedProvider.model_name,
          modelTemperature: updatedProvider.model_temperature,
          modelMaxTokens: updatedProvider.model_max_tokens,
          maxParallelRuns: updatedProvider.max_parallel_runs,
          costPerInputToken: updatedProvider.cost_per_input_token,
          costPerOutputToken: updatedProvider.cost_per_output_token,
          pricingConfig: updatedProvider.pricing_config ?? {},
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

function normalizeProviderPermissionProfile(
  value: string | undefined | null,
): ProviderPermissionProfile {
  return value === 'standard' ? 'standard' : 'unrestricted'
}

function normalizeProviderCLIRateLimit(
  value: AgentProvider['cli_rate_limit'] | null | undefined,
): ProviderCLIRateLimit | null {
  if (!value) {
    return null
  }

  return {
    provider: value.provider ?? '',
    raw: { ...(value.raw ?? {}) },
    claudeCode: value.claude_code
      ? {
          status: value.claude_code.status ?? '',
          rateLimitType: value.claude_code.rate_limit_type ?? null,
          resetsAt: value.claude_code.resets_at ?? null,
          utilization: value.claude_code.utilization ?? null,
          surpassedThreshold: value.claude_code.surpassed_threshold ?? null,
          overageStatus: value.claude_code.overage_status ?? null,
          overageDisabledReason: value.claude_code.overage_disabled_reason ?? null,
          isUsingOverage: value.claude_code.is_using_overage ?? null,
        }
      : null,
    codex: value.codex
      ? {
          limitId: value.codex.limit_id ?? null,
          limitName: value.codex.limit_name ?? null,
          planType: value.codex.plan_type ?? null,
          primary: value.codex.primary
            ? {
                usedPercent: value.codex.primary.used_percent ?? null,
                windowMinutes: value.codex.primary.window_minutes ?? null,
                resetsAt: value.codex.primary.resets_at ?? null,
              }
            : null,
          secondary: value.codex.secondary
            ? {
                usedPercent: value.codex.secondary.used_percent ?? null,
                windowMinutes: value.codex.secondary.window_minutes ?? null,
                resetsAt: value.codex.secondary.resets_at ?? null,
              }
            : null,
        }
      : null,
    gemini: value.gemini
      ? {
          authType: value.gemini.auth_type ?? null,
          remaining: value.gemini.remaining ?? null,
          limit: value.gemini.limit ?? null,
          resetTime: value.gemini.reset_time ?? null,
          buckets: (value.gemini.buckets ?? []).map((bucket) => ({
            modelId: bucket.model_id ?? null,
            tokenType: bucket.token_type ?? null,
            remainingAmount: bucket.remaining_amount ?? null,
            remainingFraction: bucket.remaining_fraction ?? null,
            resetTime: bucket.reset_time ?? null,
          })),
        }
      : null,
  }
}
