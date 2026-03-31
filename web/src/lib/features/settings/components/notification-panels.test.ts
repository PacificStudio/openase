import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import NotificationChannelPanel from './notification-channel-panel.svelte'
import NotificationRulePanel from './notification-rule-panel.svelte'

const { toastStore } = vi.hoisted(() => ({
  toastStore: {
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    warning: vi.fn(),
  },
}))

vi.mock('$lib/stores/toast.svelte', () => ({
  toastStore,
}))

describe('Notification settings panels', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('surfaces channel action failures inline', async () => {
    const { getByText } = render(NotificationChannelPanel, {
      props: {
        channels: [
          {
            id: 'channel-1',
            organization_id: 'org-1',
            name: 'Ops Webhook',
            type: 'webhook',
            config: { url: 'https://hooks.example.com/***' },
            is_enabled: true,
            created_at: '2026-03-20T00:00:00Z',
          },
        ],
        onCreate: vi.fn(),
        onUpdate: vi.fn(),
        onDelete: vi.fn(),
        onToggle: vi.fn(),
        onTest: vi.fn().mockRejectedValue(new Error('Webhook offline')),
      },
    })

    await fireEvent.click(getByText('Ops Webhook'))
    await fireEvent.click(getByText('Send test'))

    await waitFor(() => {
      expect(toastStore.error).toHaveBeenCalledWith('Webhook offline')
    })
  })

  it('surfaces rule action failures inline', async () => {
    const { getByText } = render(NotificationRulePanel, {
      props: {
        channels: [
          {
            id: 'channel-1',
            organization_id: 'org-1',
            name: 'Ops Webhook',
            type: 'webhook',
            config: { url: 'https://hooks.example.com/***' },
            is_enabled: true,
            created_at: '2026-03-20T00:00:00Z',
          },
        ],
        eventTypes: [
          {
            event_type: 'ticket.created',
            label: 'Ticket Created',
            default_template: 'Ticket created: {{ ticket.identifier }}',
          },
        ],
        rules: [
          {
            id: 'rule-1',
            project_id: 'project-1',
            channel_id: 'channel-1',
            name: 'Created alerts',
            event_type: 'ticket.created',
            filter: {},
            template: 'Ticket created: {{ ticket.identifier }}',
            is_enabled: true,
            created_at: '2026-03-20T00:00:00Z',
            channel: {
              id: 'channel-1',
              organization_id: 'org-1',
              name: 'Ops Webhook',
              type: 'webhook',
              config: { url: 'https://hooks.example.com/***' },
              is_enabled: true,
              created_at: '2026-03-20T00:00:00Z',
            },
          },
        ],
        onCreate: vi.fn(),
        onUpdate: vi.fn(),
        onDelete: vi.fn(),
        onToggle: vi.fn().mockRejectedValue(new Error('Rule toggle failed')),
      },
    })

    await fireEvent.click(getByText('Created alerts'))
    await fireEvent.click(getByText('Disable'))

    await waitFor(() => {
      expect(toastStore.error).toHaveBeenCalledWith('Rule toggle failed')
    })
  })
})
