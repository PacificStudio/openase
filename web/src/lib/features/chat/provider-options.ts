import type { AgentProvider } from '$lib/api/contracts'

const chatCapableAdapterTypes = new Set(['claude-code-cli', 'codex-app-server', 'gemini-cli'])

export function listEphemeralChatProviders(providers: AgentProvider[]): AgentProvider[] {
  return providers.filter((provider) => chatCapableAdapterTypes.has(provider.adapter_type))
}

export function pickDefaultEphemeralChatProvider(
  providers: AgentProvider[],
  defaultProviderId: string | null | undefined,
): string {
  if (
    defaultProviderId &&
    providers.some((provider) => provider.id === defaultProviderId && provider.available)
  ) {
    return defaultProviderId
  }

  return providers.find((provider) => provider.available)?.id ?? ''
}
