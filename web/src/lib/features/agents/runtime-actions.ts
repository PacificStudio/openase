import { ApiError } from '$lib/api/client'
import { createAgent, interruptAgent, pauseAgent, resumeAgent } from '$lib/api/openase'
import { loadAgentsPageData, type AgentsPageData } from './data'

export type RuntimeAction = 'interrupt' | 'pause' | 'resume'

export async function runRuntimeAction(input: {
  action: RuntimeAction
  agentId: string
  projectId: string
  orgId: string
  defaultProviderId: string | null
}): Promise<{ data: AgentsPageData; feedback: string }> {
  const payload =
    input.action === 'interrupt'
      ? await interruptAgent(input.agentId)
      : input.action === 'pause'
        ? await pauseAgent(input.agentId)
        : await resumeAgent(input.agentId)
  const verb =
    input.action === 'interrupt' ? 'Interrupt' : input.action === 'pause' ? 'Pause' : 'Resume'

  return {
    data: await loadAgentsPageData(input.projectId, input.orgId, input.defaultProviderId),
    feedback: `${verb} requested for ${payload.agent.name}.`,
  }
}

export function runtimeActionError(action: RuntimeAction, error: unknown) {
  return error instanceof ApiError ? error.detail : `Failed to ${action} agent runtime.`
}

export async function registerAgentAndReload(input: {
  projectId: string
  orgId: string
  defaultProviderId: string | null
  providerId: string
  name: string
}): Promise<{ data: AgentsPageData; feedback: string }> {
  await createAgent(input.projectId, {
    provider_id: input.providerId,
    name: input.name,
  })

  return {
    data: await loadAgentsPageData(input.projectId, input.orgId, input.defaultProviderId),
    feedback: `Registered ${input.name}.`,
  }
}

export function registerAgentError(error: unknown) {
  return error instanceof ApiError ? error.detail : 'Failed to register agent.'
}
