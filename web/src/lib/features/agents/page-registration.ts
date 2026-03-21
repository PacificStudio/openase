import { ApiError } from '$lib/api/client'
import { createAgent } from '$lib/api/openase'
import type { AgentProvider } from '$lib/api/contracts'
import { parseAgentRegistrationDraft, type AgentRegistrationDraft } from './registration'

type RegisterAgentInput = {
  projectId: string
  draft: AgentRegistrationDraft
  providerItems: AgentProvider[]
}

export async function registerAgent(input: RegisterAgentInput) {
  const parsed = parseAgentRegistrationDraft(input.draft, input.providerItems)
  if (!parsed.ok) {
    throw new Error(parsed.error)
  }

  await createAgent(input.projectId, {
    provider_id: parsed.value.providerId,
    name: parsed.value.name,
    workspace_path: parsed.value.workspacePath,
    capabilities: parsed.value.capabilities,
  })

  return parsed.value.name
}

export function describeRegisterAgentError(caughtError: unknown) {
  return caughtError instanceof ApiError ? caughtError.detail : 'Failed to register agent.'
}
