import { describe, expect, it } from 'vitest'

import {
  buildTicketDetailContext,
  buildTicketDetailLiveContext,
  buildTicketDetailProjectReferenceData,
  selectTicketDetailReferenceData,
} from './context'
import {
  detailPayloadFixture,
  repoPayloadFixture,
  statusPayloadFixture,
  ticketPayloadFixture,
} from './context.test-fixtures'

describe('buildTicketDetailContext', () => {
  const referenceData = buildTicketDetailProjectReferenceData(
    statusPayloadFixture,
    repoPayloadFixture,
    ticketPayloadFixture,
  )

  it('maps assigned agent details from the explicit ticket detail payload', () => {
    const detail = buildTicketDetailContext(detailPayloadFixture, referenceData, 'ticket-1')

    expect(detail.ticket.assignedAgent).toEqual({
      id: 'agent-1',
      name: 'todo-app-coding-01',
      provider: 'codex-cloud',
      runtimeControlState: 'active',
      runtimePhase: 'executing',
    })
    expect(detail.ticket.costTokensInput).toBe(1444743)
    expect(detail.ticket.costTokensOutput).toBe(23322)
  })

  it('parses the unified timeline payload into description, comment, and activity items', () => {
    const detail = buildTicketDetailContext(detailPayloadFixture, referenceData, 'ticket-1')

    expect(detail.timeline).toEqual([
      {
        id: 'description:ticket-1',
        ticketId: 'ticket-1',
        kind: 'description',
        actor: { name: 'test', type: 'user' },
        title: 'Implement ticket detail agent binding',
        bodyMarkdown: '',
        createdAt: '2026-03-27T12:00:00Z',
        updatedAt: '2026-03-27T12:00:00Z',
        editedAt: undefined,
        isCollapsible: false,
        isDeleted: false,
        identifier: 'ASE-1',
      },
      {
        id: 'comment:comment-1',
        ticketId: 'ticket-1',
        kind: 'comment',
        commentId: 'comment-1',
        actor: { name: 'reviewer', type: 'user' },
        bodyMarkdown: 'LGTM',
        createdAt: '2026-03-27T12:05:00Z',
        updatedAt: '2026-03-27T12:10:00Z',
        editedAt: '2026-03-27T12:10:00Z',
        isCollapsible: true,
        isDeleted: false,
        editCount: 1,
        revisionCount: 2,
        lastEditedBy: 'user:reviewer',
      },
      {
        id: 'activity:event-1',
        ticketId: 'ticket-1',
        kind: 'activity',
        actor: { name: 'dispatcher', type: 'system' },
        eventType: 'pr.opened',
        title: 'pr.opened',
        bodyText: 'Opened frontend PR #9',
        createdAt: '2026-03-27T12:06:00Z',
        updatedAt: '2026-03-27T12:06:00Z',
        editedAt: undefined,
        isCollapsible: true,
        isDeleted: false,
        metadata: {
          event_type: 'pr.opened',
          pull_request_url: 'https://github.com/acme/frontend/pull/9',
        },
      },
    ])
  })

  it('maps dependency relationships from the current ticket perspective', () => {
    const detail = buildTicketDetailContext(detailPayloadFixture, referenceData, 'ticket-1')

    expect(detail.ticket.dependencies).toEqual([
      {
        id: 'dep-1',
        targetId: 'ticket-2',
        identifier: 'ASE-2',
        title: 'Backend migration',
        relation: 'blocked_by',
        stage: 'started',
      },
      {
        id: 'dep-2',
        targetId: 'ticket-3',
        identifier: 'ASE-3',
        title: 'Frontend polish',
        relation: 'blocks',
        stage: 'completed',
      },
    ])
  })

  it('builds project reference data once and filters dependency candidates per ticket', () => {
    const selected = selectTicketDetailReferenceData(referenceData, 'ticket-1')

    expect(referenceData.statusLookup).toEqual([
      { id: 'status-1', stage: 'unstarted', color: '#2563eb' },
      { id: 'status-2', stage: 'started', color: '#f59e0b' },
      { id: 'status-3', stage: 'completed', color: '#10b981' },
    ])
    expect(selected.statuses).toEqual([
      { id: 'status-1', name: 'Todo', color: '#2563eb' },
      { id: 'status-2', name: 'In Progress', color: '#f59e0b' },
      { id: 'status-3', name: 'Done', color: '#10b981' },
    ])
    expect(selected.dependencyCandidates).toEqual([
      { id: 'ticket-2', identifier: 'ASE-2', title: 'Backend migration' },
    ])
  })

  it('builds live ticket context from cached status lookup without reloading project references', () => {
    const liveContext = buildTicketDetailLiveContext(
      detailPayloadFixture,
      referenceData.statusLookup,
    )

    expect(liveContext.ticket.status).toEqual({
      id: 'status-1',
      name: 'Todo',
      color: '#2563eb',
    })
    expect(liveContext.ticket.dependencies).toEqual([
      {
        id: 'dep-1',
        targetId: 'ticket-2',
        identifier: 'ASE-2',
        title: 'Backend migration',
        relation: 'blocked_by',
        stage: 'started',
      },
      {
        id: 'dep-2',
        targetId: 'ticket-3',
        identifier: 'ASE-3',
        title: 'Frontend polish',
        relation: 'blocks',
        stage: 'completed',
      },
    ])
  })
})
