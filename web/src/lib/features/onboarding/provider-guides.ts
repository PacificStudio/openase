import type { AgentProvider } from '$lib/api/contracts'
import {
  availabilityLabel,
  authDetectionLabel,
  cliDetectionLabel,
  guideList,
  guideText,
  reasonSpecificHints,
} from './provider-guide-data'
import type { DetectionStatus, GuideKey } from './provider-guide-data'

export type GuideCommand = {
  label: string
  command: string
}

export type ProviderGuide = {
  key: GuideKey
  title: string
  adapterTypes: string[]
  docsUrl: string
  docsLabel: string
  recommendedModel: string
  installCommands: GuideCommand[]
  authCommands: GuideCommand[]
  verifyCommands: GuideCommand[]
  commonFixHints: string[]
}

export { availabilityLabel, authDetectionLabel, cliDetectionLabel }
export type { DetectionStatus }

export const providerGuides: ProviderGuide[] = [
  {
    key: 'claude-code',
    title: guideText('claude-code', 'title'),
    adapterTypes: ['claude-code-cli', 'claude_code'],
    docsUrl: guideText('claude-code', 'docsUrl'),
    docsLabel: guideText('claude-code', 'docsLabel'),
    recommendedModel: guideText('claude-code', 'recommendedModel'),
    installCommands: [
      {
        label: guideText('claude-code', 'install.recommended'),
        command: 'curl -fsSL https://claude.ai/install.sh | bash',
      },
    ],
    authCommands: [
      {
        label: guideText('claude-code', 'auth.firstSignIn'),
        command: 'claude',
      },
    ],
    verifyCommands: [
      {
        label: guideText('claude-code', 'verify.cliAvailable'),
        command: 'claude --version',
      },
    ],
    commonFixHints: guideList('claude-code', 'commonFixHints'),
  },
  {
    key: 'codex',
    title: guideText('codex', 'title'),
    adapterTypes: ['codex-app-server', 'codex'],
    docsUrl: guideText('codex', 'docsUrl'),
    docsLabel: guideText('codex', 'docsLabel'),
    recommendedModel: guideText('codex', 'recommendedModel'),
    installCommands: [
      {
        label: guideText('codex', 'install.global'),
        command: 'npm i -g @openai/codex',
      },
    ],
    authCommands: [
      {
        label: guideText('codex', 'auth.chatgpt'),
        command: 'codex --login',
      },
    ],
    verifyCommands: [
      {
        label: guideText('codex', 'verify.cliAvailable'),
        command: 'codex --version',
      },
    ],
    commonFixHints: guideList('codex', 'commonFixHints'),
  },
  {
    key: 'gemini',
    title: guideText('gemini', 'title'),
    adapterTypes: ['gemini-cli', 'gemini_cli'],
    docsUrl: guideText('gemini', 'docsUrl'),
    docsLabel: guideText('gemini', 'docsLabel'),
    recommendedModel: guideText('gemini', 'recommendedModel'),
    installCommands: [
      {
        label: guideText('gemini', 'install.global'),
        command: 'npm install -g @google/gemini-cli',
      },
    ],
    authCommands: [
      {
        label: guideText('gemini', 'auth.firstSignIn'),
        command: 'gemini',
      },
    ],
    verifyCommands: [
      {
        label: guideText('gemini', 'verify.cliAvailable'),
        command: 'gemini --version',
      },
    ],
    commonFixHints: guideList('gemini', 'commonFixHints'),
  },
]

export function matchesGuide(provider: AgentProvider, guide: ProviderGuide): boolean {
  return guide.adapterTypes.includes(provider.adapter_type)
}

export function guideForProvider(provider: AgentProvider): ProviderGuide | null {
  return providerGuides.find((guide) => matchesGuide(provider, guide)) ?? null
}

export function guideProviders(providers: AgentProvider[], guide: ProviderGuide): AgentProvider[] {
  return providers.filter((provider) => matchesGuide(provider, guide))
}

export function isProviderAvailable(provider: AgentProvider): boolean {
  return provider.availability_state === 'available' || provider.availability_state === 'ready'
}

export function primaryGuideProvider(
  items: AgentProvider[],
  selectedId: string,
): AgentProvider | null {
  return (
    items.find((provider) => provider.id === selectedId) ??
    items.find((provider) => isProviderAvailable(provider)) ??
    items[0] ??
    null
  )
}

export function uniqueMachineIds(items: AgentProvider[]): string[] {
  return Array.from(
    new Set(
      items
        .map((provider) => provider.machine_id)
        .filter((value): value is string => Boolean(value && value.trim().length > 0)),
    ),
  )
}

export function cliDetectionState(items: AgentProvider[]): DetectionStatus {
  if (items.some((provider) => isProviderAvailable(provider))) {
    return 'yes'
  }
  if (items.some((provider) => provider.availability_reason === 'cli_missing')) {
    return 'no'
  }
  if (items.length === 0) {
    return 'pending'
  }
  return 'yes'
}

export function authDetectionState(items: AgentProvider[]): DetectionStatus {
  if (items.some((provider) => isProviderAvailable(provider))) {
    return 'yes'
  }
  if (items.some((provider) => provider.availability_reason === 'not_logged_in')) {
    return 'no'
  }
  return 'pending'
}

export function providerStatus(provider: AgentProvider | null) {
  if (!provider) {
    return availabilityLabel.unknown
  }
  return availabilityLabel[provider.availability_state] ?? availabilityLabel.unknown
}

export { reasonSpecificHints }
