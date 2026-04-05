import type { AgentProvider } from '$lib/api/contracts'

export type GuideKey = 'claude-code' | 'codex' | 'gemini'
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

export type DetectionStatus = 'yes' | 'no' | 'pending'

export const providerGuides: ProviderGuide[] = [
  {
    key: 'claude-code',
    title: 'Claude Code',
    adapterTypes: ['claude-code-cli', 'claude_code'],
    docsUrl: 'https://code.claude.com/docs/en/getting-started',
    docsLabel: 'Claude Code official docs',
    recommendedModel: 'claude-opus-4-6',
    installCommands: [
      {
        label: 'Recommended install',
        command: 'curl -fsSL https://claude.ai/install.sh | bash',
      },
    ],
    authCommands: [
      {
        label: 'First sign-in',
        command: 'claude',
      },
    ],
    verifyCommands: [
      {
        label: 'Verify CLI is available',
        command: 'claude --version',
      },
    ],
    commonFixHints: [
      'On Windows, install Git for Windows first because Claude Code depends on Git Bash.',
      'If the account does not have Claude Code access, it will remain unavailable after sign-in.',
    ],
  },
  {
    key: 'codex',
    title: 'OpenAI Codex',
    adapterTypes: ['codex-app-server', 'codex'],
    docsUrl: 'https://developers.openai.com/codex/cli',
    docsLabel: 'Codex CLI official docs',
    recommendedModel: 'gpt-5.4',
    installCommands: [
      {
        label: 'Global install',
        command: 'npm i -g @openai/codex',
      },
    ],
    authCommands: [
      {
        label: 'Sign in to ChatGPT / API',
        command: 'codex --login',
      },
    ],
    verifyCommands: [
      {
        label: 'Verify CLI is available',
        command: 'codex --version',
      },
    ],
    commonFixHints: [
      'If the command exists but cannot run, first confirm the global npm bin directory is on PATH.',
      'If the login state is lost, rerun `codex --login` or check API key / ChatGPT plan access.',
    ],
  },
  {
    key: 'gemini',
    title: 'Gemini CLI',
    adapterTypes: ['gemini-cli', 'gemini_cli'],
    docsUrl: 'https://github.com/google-gemini/gemini-cli',
    docsLabel: 'Gemini CLI official repository',
    recommendedModel: 'gemini-2.5-pro',
    installCommands: [
      {
        label: 'Global install',
        command: 'npm install -g @google/gemini-cli',
      },
    ],
    authCommands: [
      {
        label: 'First sign-in / choose auth method',
        command: 'gemini',
      },
    ],
    verifyCommands: [
      {
        label: 'Verify CLI is available',
        command: 'gemini --version',
      },
    ],
    commonFixHints: [
      'On first launch, you can sign in with a Google account or switch to `GEMINI_API_KEY` / Vertex AI.',
      'If the command is missing, first confirm Node 18+ and the global npm bin directory are on PATH.',
    ],
  },
]

export const availabilityLabel: Record<string, { text: string; className: string }> = {
  available: { text: 'Available', className: 'text-emerald-600 dark:text-emerald-400' },
  ready: { text: 'Available', className: 'text-emerald-600 dark:text-emerald-400' },
  unavailable: { text: 'Unavailable', className: 'text-destructive' },
  stale: { text: 'Status stale', className: 'text-amber-600 dark:text-amber-400' },
  unknown: { text: 'Pending check', className: 'text-muted-foreground' },
}

export const cliDetectionLabel: Record<DetectionStatus, string> = {
  yes: 'Detected',
  no: 'Not detected',
  pending: 'Pending setup',
}

export const authDetectionLabel: Record<DetectionStatus, string> = {
  yes: 'Signed in',
  no: 'Signed out',
  pending: 'Pending confirmation',
}

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

export function reasonSpecificHints(provider: AgentProvider | null): string[] {
  switch (provider?.availability_reason) {
    case 'machine_offline':
      return [
        'The bound machine is currently offline. Bring it back online, then rerun detection here.',
      ]
    case 'machine_degraded':
      return ['The machine is connected but degraded. Fix host health first, then rerun detection.']
    case 'machine_maintenance':
      return ['The machine is in maintenance mode. Exit maintenance and rerun detection.']
    case 'cli_missing':
      return [
        'OpenASE found this provider, but could not detect the matching CLI in the target machine PATH.',
      ]
    case 'not_logged_in':
      return [
        'The CLI is installed, but authentication is missing or expired. Rerun the sign-in command, then detect again.',
      ]
    case 'not_ready':
      return [
        'The CLI exists, but the readiness probe still fails. Run a verification command first to inspect the exact error.',
      ]
    case 'config_incomplete':
      return [
        'This provider registration is incomplete. Check that the CLI command, model, and machine binding are all filled in.',
      ]
    case 'l4_snapshot_missing':
    case 'stale_l4_snapshot':
      return [
        'OpenASE does not yet have a usable fresh machine snapshot. Rerunning detection will fetch one again.',
      ]
    default:
      return []
  }
}
