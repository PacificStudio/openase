import type { AgentProvider } from '$lib/api/contracts'

type EphemeralChatCapabilityState = 'available' | 'unavailable' | 'unsupported'

function getEphemeralChatCapability(provider: AgentProvider) {
  return {
    state: normalizeEphemeralChatCapabilityState(provider.capabilities.ephemeral_chat?.state),
    reason: provider.capabilities.ephemeral_chat?.reason ?? null,
  }
}

function normalizeEphemeralChatCapabilityState(
  state: string | null | undefined,
): EphemeralChatCapabilityState {
  switch (state) {
    case 'available':
    case 'unavailable':
      return state
    default:
      return 'unsupported'
  }
}

export function supportsEphemeralChat(provider: AgentProvider): boolean {
  return getEphemeralChatCapability(provider).state !== 'unsupported'
}

export function hasAvailableEphemeralChat(provider: AgentProvider): boolean {
  return getEphemeralChatCapability(provider).state === 'available'
}

export function ephemeralChatCapabilityState(
  provider: AgentProvider,
): EphemeralChatCapabilityState {
  return getEphemeralChatCapability(provider).state
}

export function ephemeralChatCapabilityReason(provider: AgentProvider): string | null {
  return getEphemeralChatCapability(provider).reason ?? null
}

export function ephemeralChatCapabilityLabel(provider: AgentProvider): string {
  switch (getEphemeralChatCapability(provider).state) {
    case 'available':
      return 'Ready'
    case 'unavailable':
      return 'Unavailable'
    default:
      return 'Unsupported'
  }
}

export function listEphemeralChatProviders(providers: AgentProvider[]): AgentProvider[] {
  return providers.filter((provider) => supportsEphemeralChat(provider))
}

export function pickDefaultEphemeralChatProvider(
  providers: AgentProvider[],
  defaultProviderId: string | null | undefined,
): string {
  if (
    defaultProviderId &&
    providers.some(
      (provider) => provider.id === defaultProviderId && hasAvailableEphemeralChat(provider),
    )
  ) {
    return defaultProviderId
  }

  return providers.find((provider) => hasAvailableEphemeralChat(provider))?.id ?? ''
}

export function shouldKeepEphemeralChatProvider(
  providers: AgentProvider[],
  providerId: string,
): boolean {
  return (
    !!providerId &&
    providers.some((provider) => provider.id === providerId && hasAvailableEphemeralChat(provider))
  )
}
