import type { SettingsSection } from '$lib/features/settings/types'

export type CapabilityState = 'available' | 'unwired' | 'backend_missing'

export type CapabilityKey =
  | 'generalSettings'
  | 'search'
  | 'newTicket'
  | 'statusMutation'
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
    state: 'unwired',
    summary:
      'Ticket creation backend support exists at POST /api/v1/projects/{projectId}/tickets, but this UI slice still lacks a create flow.',
  },
  statusMutation: {
    state: 'available',
    summary:
      'Status CRUD, default selection, reset, and ordering are wired in Settings, and dependent views refresh after changes.',
  },
  agentRegistration: {
    state: 'unwired',
    summary:
      'Agent registration backend support exists at POST /api/v1/projects/{projectId}/agents, but this page still lacks a registration form.',
  },
  providerConfigure: {
    state: 'unwired',
    summary:
      'Provider updates are supported by PATCH /api/v1/providers/{providerId}, but the provider configuration UI is not wired yet.',
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
    state: 'unwired',
    summary:
      'Repository CRUD routes are available, but repository settings screens have not been wired in this frontend slice yet.',
  },
  statusesSettings: {
    state: 'available',
    summary:
      'Statuses can now be created, edited, deleted, reset, and reordered directly from Settings.',
  },
  workflowsSettings: {
    state: 'unwired',
    summary:
      'Workflow update/delete APIs already exist, but this settings section still points to a placeholder instead of lifecycle management UI.',
  },
  agentsSettings: {
    state: 'unwired',
    summary:
      'Agent create/detail/delete APIs exist, but agent governance settings are still a placeholder in this slice.',
  },
  connectorsSettings: {
    state: 'backend_missing',
    summary:
      'Connector settings stay placeholder because no connector management API is exported yet.',
  },
  notificationsSettings: {
    state: 'unwired',
    summary:
      'Notification channel and rule APIs exist, but the notifications settings UI has not been connected yet.',
  },
  securitySettings: {
    state: 'backend_missing',
    summary:
      'Security settings stay placeholder because no dedicated security settings API is exported yet.',
  },
}

export const settingsCapabilityBySection: Partial<Record<SettingsSection, CapabilityKey>> = {
  general: 'generalSettings',
  repositories: 'repositoriesSettings',
  statuses: 'statusesSettings',
  workflows: 'workflowsSettings',
  agents: 'agentsSettings',
  connectors: 'connectorsSettings',
  notifications: 'notificationsSettings',
  security: 'securitySettings',
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
