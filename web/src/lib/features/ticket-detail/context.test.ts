import { describe, expect, it } from 'vitest'

import { buildTicketDetailContext } from './context'
import {
  detailPayloadFixture,
  repoPayloadFixture,
  statusPayloadFixture,
  ticketPayloadFixture,
  workflowPayloadFixture,
} from './context.test-fixtures'

describe('buildTicketDetailContext', () => {
  it('maps assigned agent details from the explicit ticket detail payload', () => {
    const detail = buildTicketDetailContext(
      detailPayloadFixture,
      statusPayloadFixture,
      workflowPayloadFixture,
      repoPayloadFixture,
      ticketPayloadFixture,
      'ticket-1',
    )

    expect(detail.ticket.assignedAgent).toEqual({
      id: 'agent-1',
      name: 'todo-app-coding-01',
      provider: 'codex-cloud',
      runtimeControlState: 'active',
      runtimePhase: 'executing',
    })
  })

  it('parses the unified timeline payload into description, comment, and activity items', () => {
    const detail = buildTicketDetailContext(
      detailPayloadFixture,
      statusPayloadFixture,
      workflowPayloadFixture,
      repoPayloadFixture,
      ticketPayloadFixture,
      'ticket-1',
    )

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
    const detail = buildTicketDetailContext(
      detailPayloadFixture,
      statusPayloadFixture,
      workflowPayloadFixture,
      repoPayloadFixture,
      ticketPayloadFixture,
      'ticket-1',
    )

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
})
