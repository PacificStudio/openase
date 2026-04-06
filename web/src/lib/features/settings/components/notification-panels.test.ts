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

describe('Notification channel panel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('renders channel cards with switch toggles', async () => {
    const { getByText, getAllByRole } = render(NotificationChannelPanel, {
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
          {
            id: 'channel-2',
            organization_id: 'org-1',
            name: 'Dev Slack',
            type: 'slack',
            config: { webhook_url: 'https://hooks.slack.com/***' },
            is_enabled: false,
            created_at: '2026-03-21T00:00:00Z',
          },
        ],
        onCreate: vi.fn(),
        onUpdate: vi.fn(),
        onDelete: vi.fn(),
        onToggle: vi.fn().mockResolvedValue({
          id: 'channel-1',
          is_enabled: false,
          name: 'Ops Webhook',
          type: 'webhook',
          config: {},
          organization_id: 'org-1',
          created_at: '2026-03-20T00:00:00Z',
        }),
        onTest: vi.fn(),
      },
    })

    expect(getByText('Ops Webhook')).toBeTruthy()
    expect(getByText('Dev Slack')).toBeTruthy()
    expect(getByText('Webhook')).toBeTruthy()
    expect(getByText('Slack')).toBeTruthy()

    const switches = getAllByRole('switch')
    expect(switches.length).toBe(2)
  })

  it('surfaces channel test failures via toast', async () => {
    const { getAllByText } = render(NotificationChannelPanel, {
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

    await fireEvent.click(getAllByText('Send test')[0])

    await waitFor(() => {
      expect(toastStore.error).toHaveBeenCalledWith('Webhook offline')
    })
  })

  it('shows empty state when no channels exist', () => {
    const { getByText } = render(NotificationChannelPanel, {
      props: {
        channels: [],
        onCreate: vi.fn(),
        onUpdate: vi.fn(),
        onDelete: vi.fn(),
        onToggle: vi.fn(),
        onTest: vi.fn(),
      },
    })

    expect(getByText('No channels configured.')).toBeTruthy()
  })
})

describe('Notification rule panel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.spyOn(window, 'confirm').mockReturnValue(true)
  })

  afterEach(() => {
    cleanup()
    vi.restoreAllMocks()
  })

  it('renders event groups with severity indicators', async () => {
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
            group: 'Ticket lifecycle',
            level: 'info',
            default_template: 'Ticket created: {{ ticket.identifier }}',
          },
          {
            event_type: 'ticket.budget_exhausted',
            label: 'Ticket Budget Exhausted',
            group: 'Ticket health',
            level: 'critical',
            default_template: 'Budget exhausted',
          },
          {
            event_type: 'agent.failed',
            label: 'Agent Failed',
            group: 'Agents',
            level: 'critical',
            default_template: 'Agent failed',
          },
        ],
        rules: [],
        onCreate: vi.fn(),
        onUpdate: vi.fn(),
        onDelete: vi.fn(),
      },
    })

    expect(getByText('Ticket lifecycle')).toBeTruthy()
    expect(getByText('Ticket health')).toBeTruthy()
    expect(getByText('Agents')).toBeTruthy()
  })

  it('surfaces rule toggle failures via toast', async () => {
    const { getByText, getAllByRole } = render(NotificationRulePanel, {
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
            group: 'Ticket lifecycle',
            level: 'info',
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
        onDelete: vi.fn().mockRejectedValue(new Error('Rule toggle failed')),
        onUpdate: vi.fn(),
      },
    })

    await fireEvent.click(getByText('Ticket lifecycle'))

    const switches = getAllByRole('switch')
    await fireEvent.click(switches[0])

    await waitFor(() => {
      expect(toastStore.error).toHaveBeenCalledWith('Rule toggle failed')
    })
  })

  it('shows prerequisite message when no channels exist', () => {
    const { getByText } = render(NotificationRulePanel, {
      props: {
        channels: [],
        eventTypes: [
          {
            event_type: 'ticket.created',
            label: 'Ticket Created',
            group: 'Ticket lifecycle',
            level: 'info',
            default_template: 'Ticket created',
          },
        ],
        rules: [],
        onCreate: vi.fn(),
        onUpdate: vi.fn(),
        onDelete: vi.fn(),
      },
    })

    expect(getByText('Add a channel first to create notification rules.')).toBeTruthy()
  })
})
