import type { SettingsSection } from '$lib/features/settings/types'

export type CapabilityState = 'available' | 'unwired' | 'backend_missing'

export type CapabilityKey =
  | 'organizationCreation'
  | 'projectCreation'
  | 'machineCreation'
  | 'providerCreation'
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
  | 'skillsSettings'
  | 'workflowsSettings'
  | 'agentsSettings'
  | 'notificationsSettings'
  | 'securitySettings'

export type CapabilityDescriptor = {
  state: CapabilityState
  summary: string
}

// Keep this inventory aligned with the shipped UI/API boundary. Source-backed audit tests catch
// drift when product surface changes.
export const capabilityCatalog: Record<CapabilityKey, CapabilityDescriptor> = {
  organizationCreation: {
    state: 'available',
    summary:
      'Organization creation is available from the workspace empty state and submits to POST /api/v1/orgs.',
  },
  projectCreation: {
    state: 'available',
    summary:
      'Project creation is available from the organization dashboard and submits to POST /api/v1/orgs/{orgId}/projects.',
  },
  machineCreation: {
    state: 'available',
    summary:
      'Machine creation is available from the Machines page and submits to POST /api/v1/orgs/{orgId}/machines.',
  },
  providerCreation: {
    state: 'available',
    summary:
      'Provider creation is available from the organization dashboard and submits to POST /api/v1/orgs/{orgId}/providers.',
  },
  generalSettings: {
    state: 'available',
    summary:
      'General project settings are wired to PATCH /api/v1/projects/{projectId}, and project archive is available via DELETE /api/v1/projects/{projectId}.',
  },
  search: {
    state: 'available',
    summary:
      'Global search is available from the top bar and Cmd+K, aggregating navigation, project context, tickets, workflows, agents, and commands from existing APIs.',
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
    state: 'available',
    summary:
      'Agent output is available from /agents via dedicated fetch and stream endpoints for runtime logs.',
  },
  agentPause: {
    state: 'available',
    summary:
      'Agent pause is wired to POST /api/v1/agents/{agentId}/pause and reconciles through the orchestrator runtime.',
  },
  agentResume: {
    state: 'available',
    summary:
      'Agent resume is wired to POST /api/v1/agents/{agentId}/resume once a paused agent is ready to relaunch.',
  },
  repositoriesSettings: {
    state: 'available',
    summary:
      'Repository settings now wire project repo list/create/update/delete flows to the existing catalog API.',
  },
  skillsSettings: {
    state: 'available',
    summary:
      'Skills settings now expose project skill list/create flows, enable/disable state, and workflow binding management from the shipped Settings surface.',
  },
  statusesSettings: {
    state: 'available',
    summary:
      'Statuses can now be created, edited, deleted, reset, and reordered directly from Settings.',
  },
  workflowsSettings: {
    state: 'available',
    summary:
      'Workflow lifecycle management for explicit agent binding, renaming, scheduling policy, activation, and deletion is accessible from the Workflows page.',
  },
  agentsSettings: {
    state: 'available',
    summary:
      'Agent governance settings now surface default provider selection, registered agent inventory, inline deletion for inactive agent definitions, and ownership boundaries while live runtime controls stay on the Agents page.',
  },
  notificationsSettings: {
    state: 'available',
    summary:
      'Notifications settings are wired to org-level channel CRUD, project rule CRUD, test send, and enable/disable controls.',
  },
  securitySettings: {
    state: 'available',
    summary:
      'Security settings are available via GET /api/v1/projects/{projectId}/security-settings, surfacing shipped human auth, RBAC, and outbound GitHub credential diagnostics while approval policy expansion and GitHub Device Flow remain explicitly deferred.',
  },
}

export const settingsCapabilityBySection: Record<SettingsSection, CapabilityKey> = {
  general: 'generalSettings',
  archived: 'generalSettings',
  repositories: 'repositoriesSettings',
  statuses: 'statusesSettings',
  agents: 'agentsSettings',
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
