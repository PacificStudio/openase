import type { SecuritySettingsResponse } from '$lib/api/contracts'
import type { TranslationKey, TranslationParams } from '$lib/i18n'

type Security = SecuritySettingsResponse['security']

export type DisplayText =
  | {
      kind: 'translation'
      key: TranslationKey
      params?: TranslationParams
    }
  | {
      kind: 'raw'
      value: string
    }

type DeferredCapabilityDisplay = {
  title: DisplayText
  summary: DisplayText
}

const AUTH_MODE_LABELS: Record<string, TranslationKey> = {
  disabled: 'adminAuth.overview.modeLabel.disabled',
  oidc: 'adminAuth.overview.modeLabel.oidc',
}

const GITHUB_PROBE_STATE_LABELS: Record<string, TranslationKey> = {
  configured: 'settings.security.github.state.configured',
  error: 'settings.security.github.state.error',
  insufficient_permissions: 'settings.security.github.state.insufficientPermissions',
  probing: 'settings.security.github.state.probing',
  revoked: 'settings.security.github.state.revoked',
  valid: 'settings.security.github.state.valid',
}

const GITHUB_SOURCE_LABELS: Record<string, TranslationKey> = {
  device_flow: 'settings.security.github.source.deviceFlow',
  gh_cli_import: 'settings.security.github.source.ghCliImport',
  manual_paste: 'settings.security.github.source.manualPaste',
}

const GITHUB_REPO_ACCESS_LABELS: Record<string, TranslationKey> = {
  denied: 'settings.security.github.repoAccess.denied',
  granted: 'settings.security.github.repoAccess.granted',
  not_checked: 'settings.security.github.repoAccess.notChecked',
}

const DEFERRED_CAPABILITY_LABELS: Record<string, DeferredCapabilityDisplay> = {
  'github-device-flow': {
    title: {
      kind: 'translation',
      key: 'settings.security.github.outbound.deviceFlowLabel',
    },
    summary: {
      kind: 'translation',
      key: 'settings.security.github.outbound.deviceFlowSummary',
    },
  },
  'provider-secret-rotation': {
    title: {
      kind: 'translation',
      key: 'settings.security.deferred.providerSecretRotation.title',
    },
    summary: {
      kind: 'translation',
      key: 'settings.security.deferred.providerSecretRotation.summary',
    },
  },
}

export function formatDisplayText(
  value: DisplayText,
  translate: (key: TranslationKey, params?: TranslationParams) => string,
) {
  return value.kind === 'translation' ? translate(value.key, value.params) : value.value
}

export function translateDisplayText(
  value: DisplayText,
  translate: (key: TranslationKey, params?: TranslationParams) => string,
) {
  return formatDisplayText(value, translate)
}

export function parseSecurityAuthMode(mode: string | null | undefined): DisplayText {
  const normalized = mode?.trim().toLowerCase() ?? ''
  const labelKey = AUTH_MODE_LABELS[normalized]
  if (labelKey) {
    return {
      kind: 'translation',
      key: labelKey,
    }
  }
  return {
    kind: 'raw',
    value: mode?.trim() ?? '',
  }
}

export function parseGitHubProbeState(
  slot: Security['github']['effective'] | Security['github']['organization'] | Security['github']['project_override'],
): DisplayText {
  if (!slot.configured) {
    return {
      kind: 'translation',
      key: 'settings.security.github.status.notConfigured',
    }
  }
  const labelKey = GITHUB_PROBE_STATE_LABELS[slot.probe.state]
  if (labelKey) {
    return {
      kind: 'translation',
      key: labelKey,
    }
  }
  return {
    kind: 'raw',
    value: humanizeIdentifier(slot.probe.state),
  }
}

export function parseGitHubCredentialSource(source: string | null | undefined): DisplayText {
  const normalized = source?.trim() ?? ''
  if (normalized === '') {
    return {
      kind: 'raw',
      value: '—',
    }
  }
  const labelKey = GITHUB_SOURCE_LABELS[normalized]
  if (labelKey) {
    return {
      kind: 'translation',
      key: labelKey,
    }
  }
  return {
    kind: 'raw',
    value: humanizeIdentifier(normalized),
  }
}

export function parseGitHubRepoAccess(repoAccess: string | null | undefined): DisplayText {
  const normalized = repoAccess?.trim() ?? ''
  const labelKey = GITHUB_REPO_ACCESS_LABELS[normalized]
  if (labelKey) {
    return {
      kind: 'translation',
      key: labelKey,
    }
  }
  if (normalized === '') {
    return {
      kind: 'translation',
      key: 'settings.security.github.repoAccess.notChecked',
    }
  }
  return {
    kind: 'raw',
    value: humanizeIdentifier(normalized),
  }
}

export function parseDeferredCapability(
  deferred: Security['deferred'] | null | undefined,
  key: string,
): DeferredCapabilityDisplay | null {
  if (!Array.isArray(deferred)) {
    return null
  }
  const matched = deferred.find((item) => item.key === key)
  if (!matched) {
    return null
  }
  return DEFERRED_CAPABILITY_LABELS[key] ?? null
}

function humanizeIdentifier(value: string) {
  return value
    .trim()
    .replaceAll(/[_-]+/g, ' ')
    .replaceAll(/\s+/g, ' ')
}
