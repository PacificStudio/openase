import { describe, expect, it } from 'vitest'

import {
  applyChannelTypeTemplate,
  buildUpdateChannelInput,
  createChannelDraft,
} from './notification-channels'
import {
  buildNotificationToggleRuleInput,
  groupNotificationEventTypes,
} from './notification-event-toggles'
import { formatJSONObject } from './notification-support'
import { applyRuleEventType } from './notification-rules'

describe('settings notifications helpers', () => {
  it('keeps masked channel config out of update payloads when config is untouched', () => {
    const current = {
      id: 'channel-1',
      organization_id: 'org-1',
      name: 'Ops Webhook',
      type: 'webhook',
      config: { url: 'https://hooks.example.com/***', secret: '******1234' },
      is_enabled: true,
      created_at: '2026-03-20T00:00:00Z',
    }

    const draft = {
      id: current.id,
      name: 'Ops Notifications',
      type: current.type,
      configText: formatJSONObject(current.config),
      isEnabled: current.is_enabled,
    }

    const result = buildUpdateChannelInput(draft, current)
    expect(result.ok).toBe(true)
    if (!result.ok) {
      return
    }

    expect(result.value.changed).toBe(true)
    expect(result.value.value).toEqual({ name: 'Ops Notifications' })
  })

  it('swaps channel config templates when the previous template was untouched', () => {
    const nextDraft = applyChannelTypeTemplate(createChannelDraft('webhook'), 'slack')

    expect(nextDraft.type).toBe('slack')
    expect(nextDraft.configText).toContain('webhook_url')
  })

  it('updates rule templates alongside event changes when the default was still in use', () => {
    const nextDraft = applyRuleEventType(
      {
        id: null,
        name: '',
        eventType: 'ticket.created',
        filterText: '{}',
        channelId: 'channel-1',
        template: 'Ticket created: {{ ticket.identifier }}\n{{ ticket.title }}',
        isEnabled: true,
      },
      'ticket.status_changed',
      [
        {
          event_type: 'ticket.created',
          label: 'Ticket Created',
          group: 'Ticket lifecycle',
          level: 'info',
          default_template: 'Ticket created: {{ ticket.identifier }}\n{{ ticket.title }}',
        },
        {
          event_type: 'ticket.status_changed',
          label: 'Ticket Status Changed',
          group: 'Ticket lifecycle',
          level: 'info',
          default_template: 'Ticket status changed: {{ ticket.identifier }}\n{{ new_status }}',
        },
      ],
    )

    expect(nextDraft.eventType).toBe('ticket.status_changed')
    expect(nextDraft.template).toContain('Ticket status changed')
  })

  it('groups notification event types by the API-provided group label', () => {
    const grouped = groupNotificationEventTypes([
      {
        event_type: 'ticket.created',
        label: 'Ticket Created',
        group: 'Ticket lifecycle',
        level: 'info',
        default_template: 'Ticket created',
      },
      {
        event_type: 'agent.failed',
        label: 'Agent Failed',
        group: 'Agent / execution',
        level: 'critical',
        default_template: 'Agent failed',
      },
      {
        event_type: 'ticket.status_changed',
        label: 'Ticket Status Changed',
        group: 'Ticket lifecycle',
        level: 'info',
        default_template: 'Ticket status changed',
      },
    ])

    expect(grouped).toEqual([
      {
        group: 'Ticket lifecycle',
        events: [
          {
            event_type: 'ticket.created',
            label: 'Ticket Created',
            group: 'Ticket lifecycle',
            level: 'info',
            default_template: 'Ticket created',
          },
          {
            event_type: 'ticket.status_changed',
            label: 'Ticket Status Changed',
            group: 'Ticket lifecycle',
            level: 'info',
            default_template: 'Ticket status changed',
          },
        ],
      },
      {
        group: 'Agent / execution',
        events: [
          {
            event_type: 'agent.failed',
            label: 'Agent Failed',
            group: 'Agent / execution',
            level: 'critical',
            default_template: 'Agent failed',
          },
        ],
      },
    ])
  })

  it('builds a default toggle rule payload without a custom template', () => {
    expect(
      buildNotificationToggleRuleInput(
        {
          event_type: 'ticket.created',
          label: 'Ticket Created',
          group: 'Ticket lifecycle',
          level: 'info',
          default_template: 'Ticket created',
        },
        {
          id: 'channel-1',
          organization_id: 'org-1',
          name: 'Ops Webhook',
          type: 'webhook',
          config: { url: 'https://hooks.example.com/***' },
          is_enabled: true,
          created_at: '2026-03-20T00:00:00Z',
        },
      ),
    ).toEqual({
      name: 'Ticket Created via Ops Webhook',
      event_type: 'ticket.created',
      filter: {},
      channel_id: 'channel-1',
      template: '',
      is_enabled: true,
    })
  })
})
