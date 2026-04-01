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
  status: 'executing',
  currentStepStatus: 'running_command',
  currentStepSummary: 'Running checks.',
  createdAt: '2026-04-01T10:05:00Z',
  runtimeStartedAt: '2026-04-01T10:05:30Z',
  lastHeartbeatAt: '2026-04-01T10:07:00Z',
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
        text: longOutput,
        streaming: true,
      },
    ]

    const { getByText, queryByText } = render(TicketRunTranscriptPanel, {
      props: {
        run: liveRun,
        blocks,
        latestRunId: liveRun.id,
        streamState: 'live',
      },
    })

    expect(getByText('Command approval required')).toBeTruthy()
    expect(getByText('make check')).toBeTruthy()
    expect(getByText('Approve once')).toBeTruthy()
    expect(getByText('Expand output')).toBeTruthy()

    await fireEvent.click(getByText('Expand output'))
    expect(queryByText('Collapse output')).toBeTruthy()
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
})
