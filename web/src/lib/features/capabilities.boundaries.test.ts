import { describe, expect, it } from 'vitest'

import {
  capabilityCatalog,
  type CapabilityKey,
  type CapabilityState,
} from '$lib/features/capabilities'
import topBarSource from '../components/layout/top-bar.svelte?raw'
import agentListSource from './agents/components/agent-list.svelte?raw'
import agentOutputStateSource from './agents/components/agent-output-state.svelte.ts?raw'
import agentOutputStreamSource from './agents/components/agent-output-stream.svelte.ts?raw'
import agentsPageSource from './agents/components/agents-page.svelte?raw'
import providerEditorStateSource from './agents/components/provider-editor-state.svelte.ts?raw'
import providerListSource from './agents/components/provider-list.svelte?raw'
import runtimeActionsSource from './agents/runtime-actions.ts?raw'
import projectShellSource from './app-shell/components/project-shell.svelte?raw'
import newTicketDialogSource from './tickets/components/new-ticket-dialog.svelte?raw'
import ticketsPageSource from './tickets/components/tickets-page.svelte?raw'

type SourceEvidence = {
  file: string
  snippets: string[]
}

type CapabilityAuditCase = {
  capability: CapabilityKey
  expectedState: CapabilityState
  summarySnippets: string[]
  sources: SourceEvidence[]
}

const sourceByFile: Record<string, string> = {
  '../components/layout/top-bar.svelte': topBarSource,
  './agents/components/agent-list.svelte': agentListSource,
  './agents/components/agent-output-state.svelte.ts': agentOutputStateSource,
  './agents/components/agent-output-stream.svelte.ts': agentOutputStreamSource,
  './agents/components/agents-page.svelte': agentsPageSource,
  './agents/components/provider-editor-state.svelte.ts': providerEditorStateSource,
  './agents/components/provider-list.svelte': providerListSource,
  './agents/runtime-actions.ts': runtimeActionsSource,
  './app-shell/components/project-shell.svelte': projectShellSource,
  './tickets/components/new-ticket-dialog.svelte': newTicketDialogSource,
  './tickets/components/tickets-page.svelte': ticketsPageSource,
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

const capabilityAuditCases: CapabilityAuditCase[] = [
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
        snippets: ["input.action === 'pause' ? await pauseAgent(input.agentId) : await resumeAgent(input.agentId)"],
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
        snippets: ["input.action === 'pause' ? await pauseAgent(input.agentId) : await resumeAgent(input.agentId)"],
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
