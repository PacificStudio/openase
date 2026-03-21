import type { SettingsSection } from '$lib/features/settings/types'

export type CapabilityState = 'available' | 'unwired' | 'backend_missing'

export type CapabilityKey =
  | 'generalSettings'
  | 'search'
  | 'newTicket'
  | 'agentRegistration'
  | 'providerConfigure'
  | 'agentOutput'
  | 'agentPause'
  | 'agentResume'
  | 'repositoriesSettings'
  | 'statusesSettings'
  | 'workflowsSettings'
  | 'agentsSettings'
  | 'connectorsSettings'
  | 'notificationsSettings'
  | 'securitySettings'

export type CapabilityDescriptor = {
  state: CapabilityState
  summary: string
}

export const capabilityCatalog: Record<CapabilityKey, CapabilityDescriptor> = {
  generalSettings: {
    state: 'available',
    summary: 'General project settings are already wired to PATCH /api/v1/projects/{projectId}.',
  },
  search: {
    state: 'backend_missing',
    summary: 'Search stays disabled because no search endpoint is exported in the current API.',
  },
  newTicket: {
    state: 'available',
    summary: 'Ticket creation is wired to POST /api/v1/projects/{projectId}/tickets.',
  },
  agentRegistration: {
    state: 'available',
    summary:
      'Agent registration is available from the Agents page and submits to POST /api/v1/projects/{projectId}/agents.',
  },
  providerConfigure: {
    state: 'available',
    summary:
      'Providers can be updated from the Agents page via PATCH /api/v1/providers/{providerId}.',
  },
  agentOutput: {
    state: 'backend_missing',
    summary: 'Agent output stays disabled because no agent log/output endpoint is exported yet.',
  },
  agentPause: {
    state: 'backend_missing',
    summary: 'Agent pause stays disabled because no pause endpoint is exported yet.',
  },
  agentResume: {
    state: 'backend_missing',
    summary: 'Agent resume stays disabled because no resume endpoint is exported yet.',
  },
  repositoriesSettings: {
    state: 'available',
    summary:
      'Repository settings now wire project repo list/create/update/delete flows and primary repo management to the existing catalog API.',
  },
  statusesSettings: {
    state: 'available',
    summary:
      'Statuses can now be created, edited, deleted, reset, and reordered directly from Settings.',
  },
  workflowsSettings: {
    state: 'available',
    summary:
      'Workflow settings now expose lifecycle management for renaming, scheduling policy, activation, and deletion from the shipped Settings surface.',
  },
  agentsSettings: {
    state: 'available',
    summary:
      'Agent governance settings now surface default provider selection, registered agent inventory, and ownership boundaries while runtime controls remain on the Agents page.',
  },
  connectorsSettings: {
    state: 'unwired',
    summary:
      'Settings now documents the live connector runtime surface, while project-scoped connector CRUD and operator controls remain deferred until dedicated management APIs are exported.',
  },
  notificationsSettings: {
    state: 'available',
    summary:
      'Notifications settings are wired to org-level channel CRUD, project rule CRUD, test send, and enable/disable controls.',
  },
  securitySettings: {
    state: 'backend_missing',
    summary:
      'Security settings stay placeholder because no dedicated security settings API is exported yet.',
  },
}

export const settingsCapabilityBySection: Record<SettingsSection, CapabilityKey> = {
  general: 'generalSettings',
  repositories: 'repositoriesSettings',
  statuses: 'statusesSettings',
  workflows: 'workflowsSettings',
  agents: 'agentsSettings',
  connectors: 'connectorsSettings',
  notifications: 'notificationsSettings',
  security: 'securitySettings',
}

export function getSettingsSectionCapability(section: SettingsSection): CapabilityDescriptor {
  return capabilityCatalog[settingsCapabilityBySection[section]]
}

export function capabilityStateLabel(state: CapabilityState) {
  if (state === 'available') return 'Available'
  if (state === 'unwired') return 'Needs Wiring'
  return 'Backend Missing'
}

export function capabilityStateClasses(state: CapabilityState) {
  if (state === 'available') {
    return 'border-emerald-500/40 bg-emerald-500/10 text-emerald-700 dark:text-emerald-300'
  }
  if (state === 'unwired') {
    return 'border-amber-500/40 bg-amber-500/10 text-amber-700 dark:text-amber-300'
  }
  return 'border-slate-500/40 bg-slate-500/10 text-slate-700 dark:text-slate-300'
}
