import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

const { listProviders, retainOrganizationEventBus, subscribeOrganizationProviderEvents } =
  vi.hoisted(() => ({
    listProviders: vi.fn(),
    retainOrganizationEventBus: vi.fn(),
    subscribeOrganizationProviderEvents: vi.fn(),
  }))

const { listProjectConversations } = vi.hoisted(() => ({
  listProjectConversations: vi.fn(),
}))

vi.mock('$lib/api/chat', () => ({
  listProjectConversations,
  watchProjectConversationMuxStream: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  listProviders,
}))

vi.mock('$lib/features/org-events', () => ({
  retainOrganizationEventBus,
  subscribeOrganizationProviderEvents,
}))

import ProjectConversationPanel from './project-conversation-panel.svelte'
import { providerFixtures } from './ephemeral-chat-session-controller.test-helpers'

async function openProviderMenu(trigger: HTMLElement) {
  await fireEvent.pointerDown(trigger)
  await fireEvent.keyDown(trigger, { key: 'ArrowDown' })
}

async function chooseNextProvider(trigger: HTMLElement) {
  await openProviderMenu(trigger)
  await fireEvent.keyDown(trigger, { key: 'Enter' })
}

describe('ProjectConversationPanel provider refresh', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
    window.localStorage.clear()
  })

  it('refetches providers on org lifecycle events so new options appear and selected labels refresh', async () => {
    let providerEventListener:
      | ((event: { topic: string; type: string; payload: unknown; publishedAt: string }) => void)
      | undefined

    listProjectConversations.mockResolvedValue({ conversations: [] })
    retainOrganizationEventBus.mockReturnValue(() => {})
    subscribeOrganizationProviderEvents.mockImplementation((_orgId, listener) => {
      providerEventListener = listener
      return () => {
        providerEventListener = undefined
      }
    })
    listProviders.mockResolvedValueOnce({ providers: providerFixtures }).mockResolvedValueOnce({
      providers: [
        providerFixtures[0],
        {
          ...providerFixtures[1],
          model_name: 'claude-opus-4',
        },
        {
          ...providerFixtures[0],
          id: 'provider-3',
          name: 'Gemini',
          adapter_type: 'gemini-cli',
          cli_command: 'gemini',
          model_name: 'gemini-2.5-pro',
        },
      ],
    })

    const { getByLabelText, getByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        organizationId: 'org-1',
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    const trigger = getByLabelText('Chat model')

    await waitFor(() => {
      expect(listProviders).toHaveBeenCalledTimes(1)
      expect(trigger.textContent ?? '').toContain('gpt-5.4')
    })

    await chooseNextProvider(trigger)
    await waitFor(() => {
      expect(trigger.textContent ?? '').toContain('claude-sonnet-4')
    })

    providerEventListener?.({
      topic: 'provider.events',
      type: 'provider.updated',
      payload: null,
      publishedAt: '2026-04-05T06:30:00Z',
    })

    await waitFor(() => {
      expect(listProviders).toHaveBeenCalledTimes(2)
      expect(trigger.textContent ?? '').toContain('claude-opus-4')
    })

    await openProviderMenu(trigger)
    expect(getByText('gemini-2.5-pro')).toBeTruthy()
  })

  it('falls back to a valid provider and disables invalidated options after availability refreshes', async () => {
    let providerEventListener:
      | ((event: { topic: string; type: string; payload: unknown; publishedAt: string }) => void)
      | undefined

    listProjectConversations.mockResolvedValue({ conversations: [] })
    retainOrganizationEventBus.mockReturnValue(() => {})
    subscribeOrganizationProviderEvents.mockImplementation((_orgId, listener) => {
      providerEventListener = listener
      return () => {
        providerEventListener = undefined
      }
    })
    listProviders.mockResolvedValueOnce({ providers: providerFixtures }).mockResolvedValueOnce({
      providers: [
        providerFixtures[0],
        {
          ...providerFixtures[1],
          availability_state: 'unavailable',
          available: false,
          capabilities: {
            ephemeral_chat: {
              state: 'unavailable',
              reason: 'machine_offline',
            },
          },
        },
      ],
    })

    const { getByLabelText, getByText } = render(ProjectConversationPanel, {
      props: {
        context: { projectId: 'project-1' },
        organizationId: 'org-1',
        defaultProviderId: 'provider-1',
        placeholder: 'Ask anything about this project…',
      },
    })

    const trigger = getByLabelText('Chat model')

    await waitFor(() => {
      expect(listProviders).toHaveBeenCalledTimes(1)
    })

    await chooseNextProvider(trigger)
    await waitFor(() => {
      expect(trigger.textContent ?? '').toContain('claude-sonnet-4')
    })

    providerEventListener?.({
      topic: 'provider.events',
      type: 'provider.unavailable',
      payload: null,
      publishedAt: '2026-04-05T06:31:00Z',
    })

    await waitFor(() => {
      expect(listProviders).toHaveBeenCalledTimes(2)
      expect(trigger.textContent ?? '').toContain('gpt-5.4')
    })

    await openProviderMenu(trigger)
    expect(getByText('Host machine is offline.')).toBeTruthy()
    const unavailableOption = getByText('Claude · claude-code-cli').closest(
      '[data-slot="select-item"]',
    )
    expect(unavailableOption?.hasAttribute('data-disabled')).toBe(true)
  })
})
