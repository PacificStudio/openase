import type { AgentProvider } from '$lib/api/contracts'

export type ProviderCapabilityName = 'ephemeral_chat' | 'harness_ai' | 'skill_ai'
type ProviderCapabilityState = 'available' | 'unavailable' | 'unsupported'

function readProviderCapability(provider: AgentProvider, capability: ProviderCapabilityName) {
  switch (capability) {
    case 'harness_ai':
      return provider.capabilities.harness_ai
    case 'skill_ai':
      return provider.capabilities.skill_ai
    default:
      return provider.capabilities.ephemeral_chat
  }
}

function getProviderCapability(provider: AgentProvider, capability: ProviderCapabilityName) {
  const value = readProviderCapability(provider, capability)
  return {
    state: normalizeProviderCapabilityState(value?.state),
    reason: value?.reason ?? null,
  }
}

function normalizeProviderCapabilityState(
  state: string | null | undefined,
): ProviderCapabilityState {
  switch (state) {
    case 'available':
    case 'unavailable':
      return state
    default:
      return 'unsupported'
  }
}

export function supportsProviderCapability(
  provider: AgentProvider,
  capability: ProviderCapabilityName,
): boolean {
  return getProviderCapability(provider, capability).state !== 'unsupported'
}

export function hasAvailableProviderCapability(
  provider: AgentProvider,
  capability: ProviderCapabilityName,
): boolean {
  return getProviderCapability(provider, capability).state === 'available'
}

export function providerCapabilityState(
  provider: AgentProvider,
  capability: ProviderCapabilityName,
): ProviderCapabilityState {
  return getProviderCapability(provider, capability).state
}

export function providerCapabilityReason(
  provider: AgentProvider,
  capability: ProviderCapabilityName,
): string | null {
  return getProviderCapability(provider, capability).reason ?? null
}

export function providerCapabilityLabel(
  provider: AgentProvider,
  capability: ProviderCapabilityName,
): string {
  switch (getProviderCapability(provider, capability).state) {
    case 'available':
      return 'Ready'
    case 'unavailable':
      return 'Unavailable'
    default:
      return 'Unsupported'
  }
}

export function listProviderCapabilityProviders(
  providers: AgentProvider[],
  capability: ProviderCapabilityName,
): AgentProvider[] {
  return providers.filter((provider) => supportsProviderCapability(provider, capability))
}

export function pickDefaultProviderCapability(
  providers: AgentProvider[],
  defaultProviderId: string | null | undefined,
  capability: ProviderCapabilityName,
): string {
  if (
    defaultProviderId &&
    providers.some(
      (provider) =>
        provider.id === defaultProviderId && hasAvailableProviderCapability(provider, capability),
    )
  ) {
    return defaultProviderId
  }

  return (
    providers.find((provider) => hasAvailableProviderCapability(provider, capability))?.id ?? ''
  )
}

export function shouldKeepProviderCapability(
  providers: AgentProvider[],
  providerId: string,
  capability: ProviderCapabilityName,
): boolean {
  return (
    !!providerId &&
    providers.some(
      (provider) =>
        provider.id === providerId && hasAvailableProviderCapability(provider, capability),
    )
  )
}

export function supportsEphemeralChat(provider: AgentProvider): boolean {
  return supportsProviderCapability(provider, 'ephemeral_chat')
}

export function hasAvailableEphemeralChat(provider: AgentProvider): boolean {
  return hasAvailableProviderCapability(provider, 'ephemeral_chat')
}

export function ephemeralChatCapabilityState(provider: AgentProvider): ProviderCapabilityState {
  return providerCapabilityState(provider, 'ephemeral_chat')
}

export function ephemeralChatCapabilityReason(provider: AgentProvider): string | null {
  return providerCapabilityReason(provider, 'ephemeral_chat')
}

export function ephemeralChatCapabilityLabel(provider: AgentProvider): string {
  return providerCapabilityLabel(provider, 'ephemeral_chat')
}

export function listEphemeralChatProviders(providers: AgentProvider[]): AgentProvider[] {
  return listProviderCapabilityProviders(providers, 'ephemeral_chat')
}

export function pickDefaultEphemeralChatProvider(
  providers: AgentProvider[],
  defaultProviderId: string | null | undefined,
): string {
  return pickDefaultProviderCapability(providers, defaultProviderId, 'ephemeral_chat')
}

export function shouldKeepEphemeralChatProvider(
  providers: AgentProvider[],
  providerId: string,
): boolean {
  return shouldKeepProviderCapability(providers, providerId, 'ephemeral_chat')
}
