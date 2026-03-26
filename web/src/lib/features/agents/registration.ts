import type { AgentProvider } from '$lib/api/contracts'

export type AgentRegistrationDraft = {
  providerId: string
  name: string
  workspacePath: string
  capabilitiesText: string
}

export type AgentRegistrationDraftField = keyof AgentRegistrationDraft

export type AgentRegistrationInput = {
  providerId: string
  name: string
  workspacePath: string
  capabilities: string[]
}

type AgentRegistrationParseResult =
  | { ok: true; value: AgentRegistrationInput }
  | { ok: false; error: string }

export function createAgentRegistrationDraft(
  providers: AgentProvider[],
  defaultProviderId?: string | null,
): AgentRegistrationDraft {
  return {
    providerId: resolveProviderId(providers, defaultProviderId),
    name: '',
    workspacePath: '',
    capabilitiesText: '',
  }
}

export function parseAgentRegistrationDraft(
  draft: AgentRegistrationDraft,
  providers: AgentProvider[],
): AgentRegistrationParseResult {
  const providerId = draft.providerId.trim()
  if (providerId === '') {
    return { ok: false, error: 'Select a provider before registering an agent.' }
  }

  if (!providers.some((provider) => provider.id === providerId)) {
    return { ok: false, error: 'Selected provider is no longer available.' }
  }

  const name = draft.name.trim()
  if (name === '') {
    return { ok: false, error: 'Agent name must not be empty.' }
  }

  const workspacePath = draft.workspacePath.trim()
  if (workspacePath === '') {
    return { ok: false, error: 'Workspace path must not be empty.' }
  }

  return {
    ok: true,
    value: {
      providerId,
      name,
      workspacePath,
      capabilities: parseCapabilities(draft.capabilitiesText),
    },
  }
}

function resolveProviderId(providers: AgentProvider[], defaultProviderId?: string | null) {
  if (defaultProviderId && providers.some((provider) => provider.id === defaultProviderId)) {
    return defaultProviderId
  }

  return providers.find((provider) => provider.available)?.id ?? providers[0]?.id ?? ''
}

function parseCapabilities(raw: string) {
  const capabilities: string[] = []
  const seen = new Set<string>()

  for (const value of raw.split(/[\n,]/)) {
    const capability = value.trim()
    if (capability === '' || seen.has(capability)) {
      continue
    }

    seen.add(capability)
    capabilities.push(capability)
  }

  return capabilities
}
