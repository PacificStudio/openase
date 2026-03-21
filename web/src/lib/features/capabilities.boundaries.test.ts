import { describe, expect, it } from 'vitest'

import {
  capabilityCatalog,
  type CapabilityKey,
  type CapabilityState,
} from '$lib/features/capabilities'
import topBarSource from '../components/layout/top-bar.svelte?raw'
import agentListSource from './agents/components/agent-list.svelte?raw'
import agentsPageSource from './agents/components/agents-page.svelte?raw'
import providerListSource from './agents/components/provider-list.svelte?raw'
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
  './agents/components/agents-page.svelte': agentsPageSource,
  './agents/components/provider-list.svelte': providerListSource,
  './app-shell/components/project-shell.svelte': projectShellSource,
  './tickets/components/new-ticket-dialog.svelte': newTicketDialogSource,
  './tickets/components/tickets-page.svelte': ticketsPageSource,
}

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

describe('capability catalog boundary audit', () => {
  it.each(capabilityAuditCases)(
    'keeps the $capability catalog entry aligned with its live UI/API boundary',
    ({ capability, expectedState, summarySnippets, sources }) => {
      expect(capabilityCatalog[capability].state).toBe(expectedState)

      for (const snippet of summarySnippets) {
        expect(capabilityCatalog[capability].summary).toContain(snippet)
      }

      for (const source of sources) {
        const contents = sourceByFile[source.file]
        for (const snippet of source.snippets) {
          expect(contents).toContain(snippet)
        }
      }
    },
  )
})
