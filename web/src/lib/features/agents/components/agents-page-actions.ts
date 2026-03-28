import { parseAgentRegistrationDraft, type AgentRegistrationDraft } from '../registration'
import {
  registerAgentAndReload,
  registerAgentError,
  runRuntimeAction,
  runtimeActionError,
} from '../runtime-actions'
import type { AgentsPageData } from '../data'
import type { AgentProvider } from '$lib/api/contracts'

type RegisterAgentInput = {
  projectId: string
  orgId: string
  defaultProviderId: string | null | undefined
  draft: AgentRegistrationDraft
  providerItems: AgentProvider[]
}

export async function registerAgentPageAction(
  input: RegisterAgentInput,
): Promise<{ ok: true; data: AgentsPageData; feedback: string } | { ok: false; error: string }> {
  const parsed = parseAgentRegistrationDraft(input.draft, input.providerItems)
  if (!parsed.ok) return { ok: false, error: parsed.error }

  try {
    const result = await registerAgentAndReload({
      projectId: input.projectId,
      orgId: input.orgId,
      defaultProviderId: input.defaultProviderId ?? null,
      providerId: parsed.value.providerId,
      name: parsed.value.name,
    })
    return { ok: true, data: result.data, feedback: result.feedback || 'Agent created.' }
  } catch (caughtError) {
    return { ok: false, error: registerAgentError(caughtError) }
  }
}

export async function runAgentRuntimePageAction(input: {
  action: 'pause' | 'resume'
  agentId: string
  projectId: string
  orgId: string
  defaultProviderId: string | null | undefined
}): Promise<{ ok: true; data: AgentsPageData; feedback: string } | { ok: false; error: string }> {
  try {
    const result = await runRuntimeAction({
      action: input.action,
      agentId: input.agentId,
      projectId: input.projectId,
      orgId: input.orgId,
      defaultProviderId: input.defaultProviderId ?? null,
    })
    return { ok: true, data: result.data, feedback: result.feedback }
  } catch (caughtError) {
    return { ok: false, error: runtimeActionError(input.action, caughtError) }
  }
}
