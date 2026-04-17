import type { AgentProvider } from '$lib/api/contracts'

export type GuideKey = 'claude-code' | 'codex' | 'gemini'
export type DetectionStatus = 'yes' | 'no' | 'pending'

type GuideCopyEntry = {
  fallback: string | string[]
  translationKey?: string
}

const guideCopy: Record<string, GuideCopyEntry> = {
  'claude-code.title': {
    translationKey: 'onboarding.providerGuide.claudeCode.title',
    fallback: 'Claude Code',
  },
  'claude-code.docsUrl': {
    fallback: 'https://code.claude.com/docs/en/getting-started',
  },
  'claude-code.docsLabel': {
    translationKey: 'onboarding.providerGuide.claudeCode.docsLabel',
    fallback: 'Claude Code official docs',
  },
  'claude-code.recommendedModel': {
    fallback: 'claude-opus-4-6',
  },
  'claude-code.install.recommended': {
    translationKey: 'onboarding.providerGuide.claudeCode.install.recommended',
    fallback: 'Recommended install',
  },
  'claude-code.auth.firstSignIn': {
    translationKey: 'onboarding.providerGuide.claudeCode.auth.firstSignIn',
    fallback: 'First sign-in',
  },
  'claude-code.verify.cliAvailable': {
    translationKey: 'onboarding.providerGuide.claudeCode.verify.cliAvailable',
    fallback: 'Verify CLI is available',
  },
  'claude-code.commonFixHints': {
    translationKey: 'onboarding.providerGuide.claudeCode.commonFixHints',
    fallback: [
      'On Windows, install Git for Windows first because Claude Code depends on Git Bash.',
      'If the account does not have Claude Code access, it will remain unavailable after sign-in.',
    ],
  },
  'codex.title': {
    translationKey: 'onboarding.providerGuide.codex.title',
    fallback: 'OpenAI Codex',
  },
  'codex.docsUrl': {
    fallback: 'https://developers.openai.com/codex/cli',
  },
  'codex.docsLabel': {
    translationKey: 'onboarding.providerGuide.codex.docsLabel',
    fallback: 'Codex CLI official docs',
  },
  'codex.recommendedModel': {
    fallback: 'gpt-5.4',
  },
  'codex.install.global': {
    translationKey: 'onboarding.providerGuide.codex.install.global',
    fallback: 'Global install',
  },
  'codex.auth.chatgpt': {
    translationKey: 'onboarding.providerGuide.codex.auth.chatgpt',
    fallback: 'Sign in to ChatGPT / API',
  },
  'codex.verify.cliAvailable': {
    translationKey: 'onboarding.providerGuide.codex.verify.cliAvailable',
    fallback: 'Verify CLI is available',
  },
  'codex.commonFixHints': {
    translationKey: 'onboarding.providerGuide.codex.commonFixHints',
    fallback: [
      'If the command exists but cannot run, first confirm the global npm bin directory is on PATH.',
      'If the login state is lost, rerun `codex --login` or check API key / ChatGPT plan access.',
    ],
  },
  'gemini.title': {
    translationKey: 'onboarding.providerGuide.gemini.title',
    fallback: 'Gemini CLI',
  },
  'gemini.docsUrl': {
    fallback: 'https://github.com/google-gemini/gemini-cli',
  },
  'gemini.docsLabel': {
    translationKey: 'onboarding.providerGuide.gemini.docsLabel',
    fallback: 'Gemini CLI official repository',
  },
  'gemini.recommendedModel': {
    fallback: 'gemini-2.5-pro',
  },
  'gemini.install.global': {
    translationKey: 'onboarding.providerGuide.gemini.install.global',
    fallback: 'Global install',
  },
  'gemini.auth.firstSignIn': {
    translationKey: 'onboarding.providerGuide.gemini.auth.firstSignIn',
    fallback: 'First sign-in / choose auth method',
  },
  'gemini.verify.cliAvailable': {
    translationKey: 'onboarding.providerGuide.gemini.verify.cliAvailable',
    fallback: 'Verify CLI is available',
  },
  'gemini.commonFixHints': {
    translationKey: 'onboarding.providerGuide.gemini.commonFixHints',
    fallback: [
      'On first launch, you can sign in with a Google account or switch to `GEMINI_API_KEY` / Vertex AI.',
      'If the command is missing, first confirm Node 18+ and the global npm bin directory are on PATH.',
    ],
  },
}

export function guideText(key: GuideKey, field: string) {
  return (guideCopy[`${key}.${field}`]?.fallback as string) ?? ''
}

export function guideList(key: GuideKey, field: string) {
  return (guideCopy[`${key}.${field}`]?.fallback as string[]) ?? []
}

const availabilityText: Record<string, string> = {
  available: 'Available',
  unavailable: 'Unavailable',
  stale: 'Status stale',
  unknown: 'Pending check',
}

export const availabilityLabel: Record<string, { text: string; className: string }> = {
  available: {
    text: availabilityText.available,
    className: 'text-emerald-600 dark:text-emerald-400',
  },
  ready: {
    text: availabilityText.available,
    className: 'text-emerald-600 dark:text-emerald-400',
  },
  unavailable: {
    text: availabilityText.unavailable,
    className: 'text-destructive',
  },
  stale: {
    text: availabilityText.stale,
    className: 'text-amber-600 dark:text-amber-400',
  },
  unknown: {
    text: availabilityText.unknown,
    className: 'text-muted-foreground',
  },
}

const detectionCopy: Record<string, string> = {
  detected: 'Detected',
  notDetected: 'Not detected',
  pendingSetup: 'Pending setup',
  signedIn: 'Signed in',
  signedOut: 'Signed out',
  pendingConfirmation: 'Pending confirmation',
}

export const cliDetectionLabel: Record<DetectionStatus, string> = {
  yes: detectionCopy.detected,
  no: detectionCopy.notDetected,
  pending: detectionCopy.pendingSetup,
}

export const authDetectionLabel: Record<DetectionStatus, string> = {
  yes: detectionCopy.signedIn,
  no: detectionCopy.signedOut,
  pending: detectionCopy.pendingConfirmation,
}

const reasonHints: Record<string, string[]> = {
  machine_offline: [
    'The bound machine is currently offline. Bring it back online, then rerun detection here.',
  ],
  machine_degraded: [
    'The machine is connected but degraded. Fix host health first, then rerun detection.',
  ],
  machine_maintenance: [
    'The machine is manually in maintenance. Exit maintenance before rerunning detection or dispatch.',
  ],
  cli_missing: [
    'OpenASE found this provider, but could not detect the matching CLI in the target machine PATH.',
  ],
  not_logged_in: [
    'The CLI is installed, but authentication is missing or expired. Rerun the sign-in command, then detect again.',
  ],
  not_ready: [
    'The CLI exists, but the readiness probe still fails. Run a verification command first to inspect the exact error.',
  ],
  config_incomplete: [
    'This provider registration is incomplete. Check that the CLI command, model, and machine binding are all filled in.',
  ],
  l4_snapshot_missing: [
    'OpenASE does not yet have a usable fresh machine snapshot. Rerunning detection will fetch one again.',
  ],
  stale_l4_snapshot: [
    'OpenASE does not yet have a usable fresh machine snapshot. Rerunning detection will fetch one again.',
  ],
}

export function reasonSpecificHints(provider: AgentProvider | null): string[] {
  if (!provider) {
    return []
  }
  return reasonHints[provider.availability_reason ?? ''] ?? []
}
