import type { SecuritySettingsResponse } from '$lib/api/contracts'

type Security = SecuritySettingsResponse['security']
type GitHubSlot = Security['github']['organization']

export function normalizeSecuritySettings(
  security: SecuritySettingsResponse['security'],
): SecuritySettingsResponse['security'] {
  return {
    ...security,
    github: {
      ...security.github,
      effective: normalizeGitHubSlot(security.github.effective),
      organization: normalizeGitHubSlot(security.github.organization),
      project_override: normalizeGitHubSlot(security.github.project_override),
    },
  }
}

function normalizeGitHubSlot(slot: GitHubSlot): GitHubSlot {
  return {
    ...slot,
    probe: {
      ...slot.probe,
      permissions: Array.isArray(slot.probe.permissions) ? slot.probe.permissions : [],
    },
  }
}
