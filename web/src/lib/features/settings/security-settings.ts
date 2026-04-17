import type { SecuritySettingsResponse } from '$lib/api/contracts'

type Security = SecuritySettingsResponse['security']
type GitHubSlot = Security['github']['organization']

export function normalizeSecuritySettings(
  security: SecuritySettingsResponse['security'],
): SecuritySettingsResponse['security'] {
  return {
    ...security,
    deferred: Array.isArray(security.deferred) ? security.deferred : [],
    github: {
      ...security.github,
      effective: normalizeGitHubSlot(security.github.effective),
      organization: normalizeGitHubSlot(security.github.organization),
      project_override: normalizeGitHubSlot(security.github.project_override),
    },
    secret_hygiene: {
      ...security.secret_hygiene,
      rollout_checklist: Array.isArray(security.secret_hygiene.rollout_checklist)
        ? security.secret_hygiene.rollout_checklist
        : [],
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
