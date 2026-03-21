import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

import {
  capabilityCatalog,
  getSettingsSectionCapability,
  type CapabilityKey,
  type CapabilityState,
} from '$lib/features/capabilities'
import type { SettingsSection } from '$lib/features/settings/types'

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

type CapabilityAuditCase = {
  capability: CapabilityKey
  expectedState: CapabilityState
  summarySnippets: string[]
  sources: SourceEvidence[]
}

const featuresDir = dirname(fileURLToPath(import.meta.url))

function readFeatureSource(relativePath: string) {
  return readFileSync(resolve(featuresDir, relativePath), 'utf8')
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
    expectedState: 'backend_missing',
    summarySnippets: ['no dedicated security settings API'],
    sources: [
      {
        file: './settings/components/settings-page.svelte',
        snippets: ['SettingsPlaceholder', 'section="security"', 'title="Security"'],
      },
    ],
  },
]

const capabilityAuditCases: CapabilityAuditCase[] = [
  {
    capability: 'search',
    expectedState: 'backend_missing',
    summarySnippets: ['backend search contract'],
    sources: [
      {
        file: './app-shell/components/project-shell.svelte',
        snippets: [
          'const searchCapability = capabilityCatalog.search',
          "searchEnabled={searchCapability.state === 'available'}",
        ],
      },
      {
        file: '../components/layout/top-bar.svelte',
        snippets: ['searchEnabled = false', '{#if searchEnabled}'],
      },
    ],
  },
  {
    capability: 'newTicket',
    expectedState: 'available',
    summarySnippets: ['POST /api/v1/projects/{projectId}/tickets'],
    sources: [
      {
        file: './tickets/components/new-ticket-dialog.svelte',
        snippets: ['createTicket(projectId, parsedDraft.payload)'],
      },
      {
        file: './tickets/components/tickets-page.svelte',
        snippets: [
          'const newTicketCapability = capabilityCatalog.newTicket',
          "disabled={newTicketCapability.state !== 'available' || !appStore.currentProject?.id}",
        ],
      },
    ],
  },
  {
    capability: 'agentRegistration',
    expectedState: 'available',
    summarySnippets: ['POST /api/v1/projects/{projectId}/agents'],
    sources: [
      {
        file: './agents/components/agents-page.svelte',
        snippets: ['await createAgent(projectId, {'],
      },
    ],
  },
  {
    capability: 'providerConfigure',
    expectedState: 'available',
    summarySnippets: ['PATCH /api/v1/providers/{providerId}'],
    sources: [
      {
        file: './agents/components/agents-page.svelte',
        snippets: ['const payload = await updateProvider(selectedProvider.id, parsed.value)'],
      },
      {
        file: './agents/components/provider-list.svelte',
        snippets: [
          'const providerConfigureCapability = capabilityCatalog.providerConfigure',
          'title={providerConfigureCapability.summary}',
        ],
      },
    ],
  },
  {
    capability: 'agentOutput',
    expectedState: 'backend_missing',
    summarySnippets: ['no agent log/output endpoint'],
    sources: [
      {
        file: './agents/components/agent-list.svelte',
        snippets: [
          'const agentOutputCapability = capabilityCatalog.agentOutput',
          'aria-label="View output"',
          'title={agentOutputCapability.summary}',
        ],
      },
    ],
  },
  {
    capability: 'agentPause',
    expectedState: 'backend_missing',
    summarySnippets: ['no pause endpoint'],
    sources: [
      {
        file: './agents/components/agent-list.svelte',
        snippets: [
          'const agentPauseCapability = capabilityCatalog.agentPause',
          'aria-label="Pause agent"',
          'title={agentPauseCapability.summary}',
        ],
      },
    ],
  },
  {
    capability: 'agentResume',
    expectedState: 'backend_missing',
    summarySnippets: ['no resume endpoint'],
    sources: [
      {
        file: './agents/components/agent-list.svelte',
        snippets: [
          'const agentResumeCapability = capabilityCatalog.agentResume',
          'aria-label="Resume agent"',
          'title={agentResumeCapability.summary}',
        ],
      },
    ],
  },
]

describe('capability catalog source audit', () => {
  it.each(settingsAuditCases)(
    'keeps the $section settings capability aligned with the shipped surface',
    ({ section, capability, expectedState, summarySnippets, sources }) => {
      expect(getSettingsSectionCapability(section)).toBe(capabilityCatalog[capability])
      expect(capabilityCatalog[capability].state).toBe(expectedState)

      for (const snippet of summarySnippets) {
        expect(capabilityCatalog[capability].summary).toContain(snippet)
      }

      for (const source of sources) {
        const contents = readFeatureSource(source.file)
        for (const snippet of source.snippets) {
          expect(contents).toContain(snippet)
        }
      }
    },
  )

  it.each(capabilityAuditCases)(
    'keeps the $capability catalog entry aligned with its live UI/API boundary',
    ({ capability, expectedState, summarySnippets, sources }) => {
      expect(capabilityCatalog[capability].state).toBe(expectedState)

      for (const snippet of summarySnippets) {
        expect(capabilityCatalog[capability].summary).toContain(snippet)
      }

      for (const source of sources) {
        const contents = readFeatureSource(source.file)
        for (const snippet of source.snippets) {
          expect(contents).toContain(snippet)
        }
      }
    },
  )
})
