import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TicketRun, TicketRunTranscriptBlock } from '../types'
import TicketRunTranscriptPanel from './ticket-run-transcript-panel.svelte'

const liveRun: TicketRun = {
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
  currentStepStatus: 'running_command',
  currentStepSummary: 'Running checks.',
  createdAt: '2026-04-01T10:05:00Z',
  runtimeStartedAt: '2026-04-01T10:05:30Z',
  lastHeartbeatAt: '2026-04-01T10:07:00Z',
}

const failedRun: TicketRun = {
  id: 'run-3',
  attemptNumber: 3,
  agentId: 'agent-1',
  agentName: 'Ticket Runner',
  provider: 'Codex',
  adapterType: 'codex-app-server',
  modelName: 'gpt-5.4',
  usage: {
    total: 220,
    input: 150,
    output: 70,
    cachedInput: 0,
    cacheCreation: 0,
    reasoning: 0,
    prompt: 110,
    candidate: 50,
    tool: 10,
  },
  status: 'failed',
  currentStepStatus: 'launch_failed',
  currentStepSummary: 'Workspace preparation failed.',
  createdAt: '2026-04-01T10:08:00Z',
  terminalAt: '2026-04-01T10:08:09Z',
  lastError: 'prepare repo openase: resolve existing head: reference not found',
}

describe('TicketRunTranscriptPanel', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('renders interrupt blocks and expandable terminal output', async () => {
    const longOutput = Array.from({ length: 18 }, (_, index) => `line ${index + 1}`).join('\n')
    const blocks: TicketRunTranscriptBlock[] = [
      {
        kind: 'interrupt',
        id: 'interrupt:approval-1',
        interruptKind: 'command_execution_approval',
        title: 'Command approval required',
        summary: 'Waiting for command approval.',
        at: '2026-04-01T10:06:00Z',
        payload: { command: 'make check' },
        options: [{ id: 'approve_once', label: 'Approve once' }],
      },
      {
        kind: 'terminal_output',
        id: 'terminal_output:command-1',
        stream: 'command',
        command: 'make check',
        text: longOutput,
        streaming: true,
      },
    ]

    const { container, getAllByText, getByRole } = render(TicketRunTranscriptPanel, {
      props: {
        run: liveRun,
        blocks,
        latestRunId: liveRun.id,
        streamState: 'live',
      },
    })

    expect(getAllByText('Command approval required').length).toBeGreaterThan(0)
    expect(getAllByText('make check').length).toBeGreaterThanOrEqual(2)
    expect(getAllByText('Approve once').length).toBeGreaterThan(0)

    await fireEvent.click(getByRole('button', { name: /make check/ }))
    expect(container.textContent).toContain('... +8 lines hidden — click to expand')
    expect(container.textContent).not.toContain('line 10')

    await fireEvent.click(getByRole('button', { name: /\+8 lines hidden/ }))
    expect(container.textContent).toContain('line 18')
    expect(getByRole('button', { name: 'Collapse output' })).toBeTruthy()
  })

  it('shows jump-to-live when the user scrolls away from the live bottom', async () => {
    const blocks: TicketRunTranscriptBlock[] = [
      {
        kind: 'assistant_message',
        id: 'assistant_message:1',
        text: 'First chunk',
        streaming: true,
      },
    ]

    const { component, container, getByText, queryByText, rerender } = render(
      TicketRunTranscriptPanel,
      {
        props: {
          run: liveRun,
          blocks,
          latestRunId: liveRun.id,
          streamState: 'live',
        },
      },
    )

    await rerender({
      run: liveRun,
      blocks: [
        ...blocks,
        {
          kind: 'assistant_message',
          id: 'assistant_message:2',
          text: 'Second chunk',
          streaming: true,
        },
      ],
      latestRunId: liveRun.id,
      streamState: 'live',
    })

    const viewport = container.querySelector('.overflow-y-auto') as HTMLDivElement
    Object.defineProperty(viewport, 'clientHeight', { configurable: true, value: 200 })
    Object.defineProperty(viewport, 'scrollHeight', { configurable: true, value: 1000 })
    Object.defineProperty(viewport, 'scrollTop', { configurable: true, writable: true, value: 450 })

    await fireEvent.scroll(viewport)

    expect(getByText('Jump to live')).toBeTruthy()

    viewport.scrollTop = 0
    await fireEvent.click(getByText('Jump to live'))

    expect(viewport.scrollTop).toBe(1000)
    expect(queryByText('Jump to live')).toBeNull()
    void component
  })

  it('renders a clear error details section for failed runs', () => {
    const { getByText } = render(TicketRunTranscriptPanel, {
      props: {
        run: failedRun,
        blocks: [],
        latestRunId: liveRun.id,
        streamState: 'idle',
      },
    })

    expect(getByText('Error details')).toBeTruthy()
    expect(
      getByText('prepare repo openase: resolve existing head: reference not found'),
    ).toBeTruthy()
  })
})
