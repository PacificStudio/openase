import { cleanup, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ActivityEntry } from '../types'
import ActivityTimeline from './activity-timeline.svelte'

const entries: ActivityEntry[] = [
  {
    id: 'event-1',
    eventType: 'project.created',
    message: 'Created project OpenASE',
    timestamp: '2026-04-01T11:50:00Z',
  },
  {
    id: 'event-2',
    eventType: 'ticket_status.reordered',
    message: 'Reordered ticket status Ready for QA',
    timestamp: '2026-04-01T11:51:00Z',
  },
  {
    id: 'event-3',
    eventType: 'workflow.harness_updated',
    message: 'Updated workflow harness for Coding Workflow',
    timestamp: '2026-04-01T11:52:00Z',
  },
  {
    id: 'event-4',
    eventType: 'provider.availability_changed',
    message: 'Provider Codex availability changed to unavailable',
    timestamp: '2026-04-01T11:53:00Z',
  },
]

describe('ActivityTimeline', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-01T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
    cleanup()
  })

  it('renders canonical labels for representative project activity events', () => {
    const { getByText } = render(ActivityTimeline, {
      props: { entries },
    })

    expect(getByText('Project created')).toBeTruthy()
    expect(getByText('Ticket status reordered')).toBeTruthy()
    expect(getByText('Workflow harness updated')).toBeTruthy()
    expect(getByText('Provider availability changed')).toBeTruthy()
  })
})
