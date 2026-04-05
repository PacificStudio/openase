import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import type { TicketDetail, TicketRun } from '../types'
import TicketTokenUsageSummary from './ticket-token-usage-summary.svelte'

const ticket: Pick<TicketDetail, 'id' | 'costTokensTotal'> = {
  id: 'ticket-1',
  costTokensTotal: 1540,
}

const runs: TicketRun[] = [
  {
    id: 'run-2',
    attemptNumber: 2,
    agentId: 'agent-1',
    agentName: 'Ticket Runner',
    provider: 'Codex',
    adapterType: 'codex-app-server',
    modelName: 'gpt-5.4',
    usage: {
      total: 1540,
      input: 1200,
      output: 340,
      cachedInput: 120,
      cacheCreation: 45,
      reasoning: 80,
      prompt: 920,
      candidate: 260,
      tool: 30,
    },
    status: 'executing',
    createdAt: '2026-04-01T10:05:00Z',
  },
]

describe('TicketTokenUsageSummary', () => {
  afterEach(() => {
    cleanup()
  })

  it('shows total tokens and defers run loading until breakdown is opened', async () => {
    const onLoadRuns = vi.fn()
    const { getByText } = render(TicketTokenUsageSummary, {
      props: {
        ticket,
        runs: [],
        runsLoaded: false,
        loadingRuns: false,
        runsError: '',
        onLoadRuns,
      },
    })

    expect(getByText('Total Tokens')).toBeTruthy()
    expect(getByText('1,540')).toBeTruthy()
    expect(onLoadRuns).not.toHaveBeenCalled()

    await fireEvent.click(getByText('Breakdown'))
    expect(onLoadRuns).toHaveBeenCalledTimes(1)
  })

  it('renders loading, empty, and error states for the lazy-loaded breakdown', async () => {
    const onLoadRuns = vi.fn()
    const { getByText, rerender } = render(TicketTokenUsageSummary, {
      props: {
        ticket,
        runs: [],
        runsLoaded: false,
        loadingRuns: false,
        runsError: '',
        onLoadRuns,
      },
    })

    await fireEvent.click(getByText('Breakdown'))
    await rerender({
      ticket,
      runs: [],
      runsLoaded: false,
      loadingRuns: true,
      runsError: '',
      onLoadRuns,
    })
    expect(getByText('Loading run usage breakdown…')).toBeTruthy()

    await rerender({
      ticket,
      runs: [],
      runsLoaded: true,
      loadingRuns: false,
      runsError: '',
      onLoadRuns,
    })
    expect(getByText('No run usage yet.')).toBeTruthy()

    await rerender({
      ticket,
      runs: [],
      runsLoaded: false,
      loadingRuns: false,
      runsError: 'Failed to load ticket runs.',
      onLoadRuns,
    })
    expect(getByText('Failed to load ticket runs.')).toBeTruthy()
    expect(getByText('Retry')).toBeTruthy()
  })

  it('renders run usage details with adapter and model metadata after loading', async () => {
    const { getByText } = render(TicketTokenUsageSummary, {
      props: {
        ticket,
        runs,
        runsLoaded: true,
        loadingRuns: false,
        runsError: '',
      },
    })

    await fireEvent.click(getByText('Breakdown'))

    expect(getByText('codex-app-server')).toBeTruthy()
    expect(getByText('gpt-5.4')).toBeTruthy()
    expect(getByText('Attempt #2 · Codex')).toBeTruthy()
    expect(getByText('Cached Input')).toBeTruthy()
    expect(getByText('Cache Creation')).toBeTruthy()
    expect(getByText('Reasoning')).toBeTruthy()
    expect(getByText('Prompt')).toBeTruthy()
    expect(getByText('Candidate')).toBeTruthy()
    expect(getByText('Tool')).toBeTruthy()
    expect(getByText('920')).toBeTruthy()
  })
})
