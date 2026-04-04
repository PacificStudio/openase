import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { TicketActivityTimelineItem } from '../types'
import TicketTimelineActivityItem from './ticket-timeline-activity-item.svelte'

const item: TicketActivityTimelineItem = {
  id: 'activity:event-1',
  ticketId: 'ticket-1',
  kind: 'activity',
  actor: { name: 'workflow-seed', type: 'agent' },
  eventType: 'agent.ready',
  title: 'agent.ready',
  bodyText: 'Codex session started and is ready to execute work.',
  createdAt: '2026-04-01T11:48:00Z',
  updatedAt: '2026-04-01T11:48:00Z',
  isCollapsible: true,
  isDeleted: false,
  metadata: {
    runtime_control_state: 'active',
    runtime_phase: 'ready',
    status: 'running',
    current_run_id: '6f26e8b7-9f18-4c4e-a9c1-d24d2f0ee8fb',
    target_machine_name: 'local',
  },
}

describe('TicketTimelineActivityItem', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('renders the event summary without anonymous runtime chips', () => {
    const { getByText, queryByText } = render(TicketTimelineActivityItem, {
      props: { item },
    })

    expect(getByText('Agent ready')).toBeTruthy()
    expect(getByText('Codex session started and is ready to execute work.')).toBeTruthy()
    expect(getByText('Agent workflow-seed')).toBeTruthy()
    expect(getByText('Machine local')).toBeTruthy()
    expect(getByText('Run 6f26e8b7')).toBeTruthy()
    expect(queryByText(/^active$/i)).toBeNull()
    expect(queryByText(/^running$/i)).toBeNull()
  })
})
