import { describe, expect, it } from 'vitest'

import {
  capabilityCatalog,
  getSettingsSectionCapability,
  type CapabilityKey,
  type CapabilityState,
} from '$lib/features/capabilities'
import type { SettingsSection } from '$lib/features/settings/types'
import agentSettingsSource from './settings/components/agent-settings.svelte?raw'
import connectorsSettingsSource from './settings/components/connectors-settings.svelte?raw'
import generalSettingsSource from './settings/components/general-settings.svelte?raw'
import notificationSettingsSource from './settings/components/notification-settings.svelte?raw'
import repositoriesSettingsSource from './settings/components/repositories-settings.svelte?raw'
import securitySettingsSource from './settings/components/security-settings.svelte?raw'
import settingsPageSource from './settings/components/settings-page.svelte?raw'
import statusSettingsSource from './settings/components/status-settings.svelte?raw'
import workflowSettingsSource from './settings/components/workflow-settings.svelte?raw'
import workflowManagementSource from './workflows/workflow-management.ts?raw'

type SourceEvidence = {
  file: string
  snippets: string[]
}

type SettingsAuditCase = {
  section: SettingsSection
  capability: CapabilityKey
  expectedState: CapabilityState
  summarySnippets: string[]
  sources: SourceEvidence[]
}

const sourceByFile: Record<string, string> = {
  './settings/components/agent-settings.svelte': agentSettingsSource,
  './settings/components/connectors-settings.svelte': connectorsSettingsSource,
  './settings/components/general-settings.svelte': generalSettingsSource,
  './settings/components/notification-settings.svelte': notificationSettingsSource,
  './settings/components/repositories-settings.svelte': repositoriesSettingsSource,
  './settings/components/security-settings.svelte': securitySettingsSource,
  './settings/components/settings-page.svelte': settingsPageSource,
  './settings/components/status-settings.svelte': statusSettingsSource,
  './settings/components/workflow-settings.svelte': workflowSettingsSource,
  './workflows/workflow-management.ts': workflowManagementSource,
}

function expectCapabilitySummary(summary: string, snippets: string[]) {
  for (const snippet of snippets) {
    expect(summary).toContain(snippet)
  }
}

function expectSourceEvidence(sources: SourceEvidence[]) {
  for (const source of sources) {
    const contents = sourceByFile[source.file]
    for (const snippet of source.snippets) {
      expect(contents).toContain(snippet)
    }
  }
}

const settingsAuditCases: SettingsAuditCase[] = [
  {
    section: 'general',
    capability: 'generalSettings',
    expectedState: 'available',
    summarySnippets: ['PATCH /api/v1/projects/{projectId}'],
    sources: [
      {
        file: './settings/components/general-settings.svelte',
        snippets: ['listWorkflows(projectId)', 'updateProject(projectId, {'],
      },
    ],
  },
  {
    section: 'repositories',
    capability: 'repositoriesSettings',
    expectedState: 'available',
    summarySnippets: ['project repo list/create/update/delete', 'primary repo management'],
    sources: [
      {
        file: './settings/components/repositories-settings.svelte',
        snippets: [
          'createProjectRepo(projectId, parsed.value)',
          'updateProjectRepo(projectId, selectedRepo.id, parsed.value)',
          'deleteProjectRepo(projectId, selectedRepo.id)',
        ],
      },
    ],
  },
  {
    section: 'statuses',
    capability: 'statusesSettings',
    expectedState: 'available',
    summarySnippets: ['created, edited, deleted, reset, and reordered'],
    sources: [
      {
        file: './settings/components/status-settings.svelte',
        snippets: [
          'createStatus(projectId, {',
          'updateStatus(statusId, body)',
          'deleteStatus(status.id)',
          'resetStatuses(projectId)',
        ],
      },
    ],
  },
  {
    section: 'workflows',
    capability: 'workflowsSettings',
    expectedState: 'available',
    summarySnippets: [
      'lifecycle management',
      'renaming, scheduling policy, activation, and deletion',
    ],
    sources: [
      {
        file: './settings/components/workflow-settings.svelte',
        snippets: ['WorkflowLifecycleSidebar', 'loadWorkflowCatalog(projectId)'],
      },
      {
        file: './workflows/workflow-management.ts',
        snippets: ['updateWorkflow(workflowId, payload)', 'deleteWorkflow(workflowId)'],
      },
    ],
  },
  {
    section: 'agents',
    capability: 'agentsSettings',
    expectedState: 'available',
    summarySnippets: ['default provider selection', 'registered agent inventory'],
    sources: [
      {
        file: './settings/components/agent-settings.svelte',
        snippets: ['listProviders(orgId)', 'listAgents(projectId)', 'updateProject(projectId, {'],
      },
    ],
  },
  {
    section: 'connectors',
    capability: 'connectorsSettings',
    expectedState: 'unwired',
    summarySnippets: ['connector runtime surface', 'connector CRUD', 'dedicated management APIs'],
    sources: [
      {
        file: './settings/components/connectors-settings.svelte',
        snippets: [
          'Current exported surface',
          'Deferred management scope',
          'POST /api/v1/webhooks/:connector/:provider',
        ],
      },
    ],
  },
  {
    section: 'notifications',
    capability: 'notificationsSettings',
    expectedState: 'available',
    summarySnippets: ['org-level channel CRUD', 'project rule CRUD', 'test send'],
    sources: [
      {
        file: './settings/components/notification-settings.svelte',
        snippets: [
          'createNotificationChannel(orgId, input)',
          'updateNotificationChannel(channelId, input)',
          'deleteNotificationChannel(channelId)',
          'testNotificationChannel(channelId)',
          'createNotificationRule(projectId, input)',
          'updateNotificationRule(ruleId, input)',
          'deleteNotificationRule(ruleId)',
        ],
      },
    ],
  },
  {
    section: 'security',
    capability: 'securitySettings',
    expectedState: 'available',
    summarySnippets: ['GET /api/v1/projects/{projectId}/security', 'explicitly deferred'],
    sources: [
      {
        file: './settings/components/settings-page.svelte',
        snippets: ['import SecuritySettings from', '<SecuritySettings />'],
      },
      {
        file: './settings/components/security-settings.svelte',
        snippets: ['const payload = await getSecuritySettings(projectId)', 'Explicitly deferred'],
      },
    ],
  },
]

describe('capability catalog source audit', () => {
  it.each(settingsAuditCases)(
    'keeps the $section settings capability aligned with the shipped surface',
    ({ section, capability, expectedState, summarySnippets, sources }) => {
      const descriptor = capabilityCatalog[capability]

      expect(getSettingsSectionCapability(section)).toBe(descriptor)
      expect(descriptor.state).toBe(expectedState)
      expectCapabilitySummary(descriptor.summary, summarySnippets)
      expectSourceEvidence(sources)
    },
  )
})
