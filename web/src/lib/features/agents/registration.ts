import type { AgentProvider } from '$lib/api/contracts'
import { providerIsDispatchReady } from '$lib/features/providers'

export type AgentRegistrationDraft = {
  providerId: string
  name: string
}

export type AgentRegistrationDraftField = keyof AgentRegistrationDraft

export type AgentRegistrationInput = {
  providerId: string
  name: string
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

  return {
    ok: true,
    value: {
      providerId,
      name,
    },
  }
}

export function deriveWorkspaceConvention(
  provider: AgentProvider | undefined,
  orgSlug?: string | null,
  projectSlug?: string | null,
) {
  const root =
    provider && provider.machine_host && provider.machine_host !== 'local'
      ? (provider.machine_workspace_root ?? '{machine.workspace_root}')
      : '~/.openase/workspace'
  const orgSegment = orgSlug?.trim() || '{org}'
  const projectSegment = projectSlug?.trim() || '{project}'
  return `${root}/${orgSegment}/${projectSegment}/{ticket}`
}

function resolveProviderId(providers: AgentProvider[], defaultProviderId?: string | null) {
  if (defaultProviderId && providers.some((provider) => provider.id === defaultProviderId)) {
    return defaultProviderId
  }

  return (
    providers.find((provider) => providerIsDispatchReady(provider.availability_state))?.id ??
    providers[0]?.id ??
    ''
  )
}
