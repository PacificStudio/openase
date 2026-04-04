import { describe, expect, it, vi } from 'vitest'

import {
  ticketRunStatusClass,
  ticketRunStatusLabel,
  ticketRunSummaryLine,
} from './ticket-run-history-panel-view-model'
import type { TicketRun } from '../types'

describe('ticket run history panel view model', () => {
  it('labels ended runs without reusing completed wording', () => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-03T12:10:00Z'))

    const run: TicketRun = {
      id: 'run-1',
      attemptNumber: 1,
      agentId: 'agent-1',
      agentName: 'Runner',
      provider: 'Codex',
      status: 'ended',
      createdAt: '2026-04-03T12:00:00Z',
      terminalAt: '2026-04-03T12:02:00Z',
    }

    expect(ticketRunStatusLabel(run)).toBe('Ended')
    expect(ticketRunStatusClass(run)).toContain('slate')
    expect(ticketRunSummaryLine(run)).toBe('Ended 8m ago')

    vi.useRealTimers()
  })
})
