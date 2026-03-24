import { describe, expect, it } from 'vitest'

import {
  capabilityCatalog,
  type CapabilityKey,
  type CapabilityState,
} from '$lib/features/capabilities'
import openaseSource from '$lib/api/openase.ts?raw'
import topBarSource from '../components/layout/top-bar.svelte?raw'
import agentListSource from './agents/components/agent-list.svelte?raw'
import agentOutputStateSource from './agents/components/agent-output-state.svelte.ts?raw'
import agentOutputStreamSource from './agents/components/agent-output-stream.svelte.ts?raw'
import agentsPageSource from './agents/components/agents-page.svelte?raw'
import machinePageActionsSource from './machines/components/machine-page-actions.svelte?raw'
import machinesPageSource from './machines/components/machines-page.svelte?raw'
import providerEditorStateSource from './agents/components/provider-editor-state.svelte.ts?raw'
import providerListSource from './agents/components/provider-list.svelte?raw'
import runtimeActionsSource from './agents/runtime-actions.ts?raw'
import projectShellSource from './app-shell/components/project-shell.svelte?raw'
import newTicketDialogSource from './tickets/components/new-ticket-dialog.svelte?raw'
import ticketsPageSource from './tickets/components/tickets-page.svelte?raw'
import orgDashboardSource from '../../routes/(app)/orgs/[orgId]/+page.svelte?raw'
import workspaceDashboardSource from '../../routes/(app)/+page.svelte?raw'

type SourceEvidence = {
  file: string
  snippets?: string[]
  absentSnippets?: string[]
}

type CapabilityAuditCase = {
  capability: CapabilityKey
  expectedState: CapabilityState
  summarySnippets: string[]
  sources: SourceEvidence[]
}

const sourceByFile: Record<string, string> = {
  '$lib/api/openase.ts': openaseSource,
  '../components/layout/top-bar.svelte': topBarSource,
  './agents/components/agent-list.svelte': agentListSource,
  './agents/components/agent-output-state.svelte.ts': agentOutputStateSource,
  './agents/components/agent-output-stream.svelte.ts': agentOutputStreamSource,
  './agents/components/agents-page.svelte': agentsPageSource,
  './machines/components/machine-page-actions.svelte': machinePageActionsSource,
  './machines/components/machines-page.svelte': machinesPageSource,
  './agents/components/provider-editor-state.svelte.ts': providerEditorStateSource,
  './agents/components/provider-list.svelte': providerListSource,
  './agents/runtime-actions.ts': runtimeActionsSource,
  './app-shell/components/project-shell.svelte': projectShellSource,
  './tickets/components/new-ticket-dialog.svelte': newTicketDialogSource,
  './tickets/components/tickets-page.svelte': ticketsPageSource,
  '../../routes/(app)/+page.svelte': workspaceDashboardSource,
  '../../routes/(app)/orgs/[orgId]/+page.svelte': orgDashboardSource,
}

function expectCapabilitySummary(summary: string, snippets: string[]) {
  for (const snippet of snippets) {
    expect(summary).toContain(snippet)
  }
}

function expectSourceEvidence(sources: SourceEvidence[]) {
  for (const source of sources) {
    const contents = sourceByFile[source.file]
    for (const snippet of source.snippets ?? []) {
      expect(contents).toContain(snippet)
    }
    for (const snippet of source.absentSnippets ?? []) {
      expect(contents).not.toContain(snippet)
    }
  }
}

const capabilityAuditCases: CapabilityAuditCase[] = [
  {
    capability: 'organizationCreation',
    expectedState: 'unwired',
    summarySnippets: ['POST /api/v1/orgs', 'does not expose a create-organization flow'],
    sources: [
      {
        file: '$lib/api/openase.ts',
        absentSnippets: ['export function createOrganization('],
      },
      {
        file: '../../routes/(app)/+page.svelte',
        snippets: ['Create or seed an organization to start using the OpenASE dashboard.'],
        absentSnippets: ['Create organization', 'New organization'],
      },
    ],
  },
  {
    capability: 'projectCreation',
    expectedState: 'unwired',
    summarySnippets: ['POST /api/v1/orgs/{orgId}/projects', 'lists and switches existing projects'],
    sources: [
      {
        file: '$lib/api/openase.ts',
        snippets: ['export function listProjects(orgId: string) {'],
        absentSnippets: ['export function createProject('],
      },
      {
        file: '../../routes/(app)/orgs/[orgId]/+page.svelte',
        snippets: ['Use direct links or the top-bar switcher to move between projects.'],
        absentSnippets: ['Create project', 'New project'],
      },
    ],
  },
  {
    capability: 'machineCreation',
    expectedState: 'available',
    summarySnippets: ['Machines page', 'POST /api/v1/orgs/{orgId}/machines'],
    sources: [
      {
        file: '$lib/api/openase.ts',
        snippets: ['export function createMachine(orgId: string, body: MachineMutationBody) {'],
      },
      {
        file: './machines/components/machines-page.svelte',
        snippets: ['const payload = await createMachine(routeOrgId, parsed.value)'],
      },
      {
        file: './machines/components/machine-page-actions.svelte',
        snippets: ['New machine'],
      },
    ],
  },
  {
    capability: 'providerCreation',
    expectedState: 'unwired',
    summarySnippets: [
      'POST /api/v1/orgs/{orgId}/providers',
      'only edits providers that already exist',
    ],
    sources: [
      {
        file: '$lib/api/openase.ts',
        snippets: ['export function listProviders(orgId: string) {'],
        absentSnippets: ['export function createProvider(', 'export function createAgentProvider('],
      },
      {
        file: './agents/components/agents-page.svelte',
        snippets: ['providerEditor = createProviderEditorState()', 'providerItems.length === 0'],
        absentSnippets: ['Create provider', 'New provider'],
      },
      {
        file: './agents/components/provider-editor-state.svelte.ts',
        snippets: ['const payload = await updateProvider(provider.id, parsed.value)'],
      },
    ],
  },
  {
    capability: 'search',
    expectedState: 'available',
    summarySnippets: ['Cmd+K', 'tickets, workflows, agents'],
    sources: [
      {
        file: './app-shell/components/project-shell.svelte',
        snippets: [
          'const searchCapability = capabilityCatalog.search',
          "searchEnabled={searchCapability.state === 'available' && data.organizations.length > 0}",
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
        file: './agents/runtime-actions.ts',
        snippets: ['await createAgent(input.projectId, {'],
      },
      {
        file: './agents/components/agents-page.svelte',
        snippets: ['const result = await registerAgentAndReload({'],
      },
    ],
  },
  {
    capability: 'providerConfigure',
    expectedState: 'available',
    summarySnippets: ['PATCH /api/v1/providers/{providerId}'],
    sources: [
      {
        file: './agents/components/provider-editor-state.svelte.ts',
        snippets: ['const payload = await updateProvider(provider.id, parsed.value)'],
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
    expectedState: 'available',
    summarySnippets: ['dedicated fetch and stream endpoints'],
    sources: [
      {
        file: './agents/components/agent-list.svelte',
        snippets: [
          'const agentOutputCapability = capabilityCatalog.agentOutput',
          'aria-label="View output"',
          'title={agentOutputCapability.summary}',
        ],
      },
      {
        file: './agents/components/agent-output-state.svelte.ts',
        snippets: ['await listAgentOutput(projectId, agentId, { limit: agentOutputLimit })'],
      },
      {
        file: './agents/components/agent-output-stream.svelte.ts',
        snippets: ['`/api/v1/projects/${projectId}/agents/${agentId}/output/stream`'],
      },
    ],
  },
  {
    capability: 'agentPause',
    expectedState: 'available',
    summarySnippets: ['POST /api/v1/agents/{agentId}/pause'],
    sources: [
      {
        file: './agents/components/agent-list.svelte',
        snippets: [
          'const agentPauseCapability = capabilityCatalog.agentPause',
          'aria-label="Pause agent"',
          'return agentPauseCapability.summary',
        ],
      },
      {
        file: './agents/runtime-actions.ts',
        snippets: [
          "input.action === 'pause' ? await pauseAgent(input.agentId) : await resumeAgent(input.agentId)",
        ],
      },
    ],
  },
  {
    capability: 'agentResume',
    expectedState: 'available',
    summarySnippets: ['POST /api/v1/agents/{agentId}/resume'],
    sources: [
      {
        file: './agents/components/agent-list.svelte',
        snippets: [
          'const agentResumeCapability = capabilityCatalog.agentResume',
          'aria-label="Resume agent"',
          'return agentResumeCapability.summary',
        ],
      },
      {
        file: './agents/runtime-actions.ts',
        snippets: [
          "input.action === 'pause' ? await pauseAgent(input.agentId) : await resumeAgent(input.agentId)",
        ],
      },
    ],
  },
]

describe('capability catalog boundary audit', () => {
  it.each(capabilityAuditCases)(
    'keeps the $capability catalog entry aligned with its live UI/API boundary',
    ({ capability, expectedState, summarySnippets, sources }) => {
      const descriptor = capabilityCatalog[capability]

      expect(descriptor.state).toBe(expectedState)
      expectCapabilitySummary(descriptor.summary, summarySnippets)
      expectSourceEvidence(sources)
    },
  )
})
