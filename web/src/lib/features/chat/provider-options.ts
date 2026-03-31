import type { AgentProvider } from '$lib/api/contracts'

type EphemeralChatCapabilityState = AgentProvider['capabilities']['ephemeral_chat']['state']

export function supportsEphemeralChat(provider: AgentProvider): boolean {
  return provider.capabilities.ephemeral_chat.state !== 'unsupported'
}

export function hasAvailableEphemeralChat(provider: AgentProvider): boolean {
  return provider.capabilities.ephemeral_chat.state === 'available'
}

export function ephemeralChatCapabilityState(
  provider: AgentProvider,
): EphemeralChatCapabilityState {
  return provider.capabilities.ephemeral_chat.state
}

export function ephemeralChatCapabilityReason(provider: AgentProvider): string | null {
  return provider.capabilities.ephemeral_chat.reason ?? null
}

export function ephemeralChatCapabilityLabel(provider: AgentProvider): string {
  switch (provider.capabilities.ephemeral_chat.state) {
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
