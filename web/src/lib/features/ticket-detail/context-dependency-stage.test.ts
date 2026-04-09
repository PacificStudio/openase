import { expect, it } from 'vitest'

import { buildTicketDetailLiveContext, buildTicketDetailProjectReferenceData } from './context'
import {
  detailPayloadFixture,
  repoPayloadFixture,
  statusPayloadFixture,
  ticketPayloadFixture,
} from './context.test-fixtures'

const archivedStatusPayload = {
  statuses: [
    ...statusPayloadFixture.statuses,
    {
      id: 'status-4',
      project_id: 'project-1',
      name: 'Archived',
      stage: 'canceled',
      color: '#4b5563',
      icon: '',
      is_default: false,
      description: '',
      position: 4,
      active_runs: 0,
      max_active_runs: null,
    },
  ],
}

const archivedTicketPayload = {
  tickets: [
    ...ticketPayloadFixture.tickets,
    {
      id: 'ticket-4',
      project_id: 'project-1',
      identifier: 'ASE-4',
      title: 'Retire legacy flow',
      description: '',
      status_id: 'status-4',
      status_name: 'Archived',
      priority: 'medium',
      type: 'feature',
      archived: false,
      workflow_id: null,
      current_run_id: null,
      target_machine_id: null,
      created_by: 'user:test',
      parent: null,
      children: [],
      dependencies: [],
      external_links: [],
      pull_request_urls: [],
      external_ref: '',
      budget_usd: 0,
      cost_tokens_input: 0,
      cost_tokens_output: 0,
      cost_tokens_total: 0,
      cost_amount: 0,
      attempt_count: 0,
      consecutive_errors: 0,
      started_at: null,
      completed_at: null,
      next_retry_at: null,
      retry_paused: false,
      pause_reason: '',
      created_at: '2026-03-27T12:00:00Z',
    },
  ],
}

const archivedReferenceData = buildTicketDetailProjectReferenceData(
  archivedStatusPayload,
  repoPayloadFixture,
  archivedTicketPayload,
)

it('maps archived dependency statuses onto the canceled terminal stage', () => {
  const liveContext = buildTicketDetailLiveContext(
    {
      ...detailPayloadFixture,
      ticket: {
        ...detailPayloadFixture.ticket,
        dependencies: [
          {
            id: 'dep-archived',
            type: 'blocks',
            target: {
              id: 'ticket-4',
              identifier: 'ASE-4',
              title: 'Retire legacy flow',
              status_id: 'status-4',
              status_name: 'Archived',
            },
          },
        ],
      },
    },
    archivedReferenceData.statusLookup,
  )

  expect(liveContext.ticket.dependencies).toEqual([
    {
      id: 'dep-archived',
      targetId: 'ticket-4',
      identifier: 'ASE-4',
      title: 'Retire legacy flow',
      relation: 'blocks',
      stage: 'canceled',
    },
  ])
})
