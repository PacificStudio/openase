import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ProjectConversation } from '$lib/api/chat'
import ProjectConversationHistoryPopover from './project-conversation-history-popover.svelte'

const conversationFixtures: ProjectConversation[] = [
  {
    id: 'conversation-1',
    projectId: 'project-1',
    userId: 'user-1',
    source: 'project_sidebar',
    providerId: 'provider-1',
    title: 'Keep the first question stable',
    status: 'active',
    rollingSummary: 'Latest recovery summary',
    lastActivityAt: '2026-04-06T11:55:00Z',
    createdAt: '2026-04-06T11:50:00Z',
    updatedAt: '2026-04-06T11:55:00Z',
  },
  {
    id: 'conversation-2',
    projectId: 'project-1',
    userId: 'user-1',
    source: 'project_sidebar',
    providerId: 'provider-1',
    title: '',
    status: 'idle',
    rollingSummary: '',
    lastActivityAt: '2026-04-06T10:00:00Z',
    createdAt: '2026-04-06T10:00:00Z',
    updatedAt: '2026-04-06T10:05:00Z',
  },
]

describe('ProjectConversationHistoryPopover', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-06T12:00:00Z'))
  })

  afterEach(() => {
    cleanup()
    vi.useRealTimers()
  })

  it('keeps the history list in a bounded internal scroll container', () => {
    const { getByTestId, getByText } = render(ProjectConversationHistoryPopover, {
      props: {
        conversations: conversationFixtures,
        openConversationIds: ['conversation-1'],
      },
    })

    const scrollContainer = getByTestId('conversation-history-scroll')

    expect(scrollContainer.className).toContain('max-h-80')
    expect(scrollContainer.className).toContain('overflow-y-auto')
    expect(getByText('Keep the first question stable')).toBeTruthy()
    expect(getByText('Latest recovery summary')).toBeTruthy()
    expect(getByText('New conversation')).toBeTruthy()
    expect(getByText('open')).toBeTruthy()
  })

  it('selects a conversation when an item is clicked', async () => {
    const onSelect = vi.fn()
    const { getByRole } = render(ProjectConversationHistoryPopover, {
      props: {
        conversations: conversationFixtures,
        onSelect,
      },
    })

    await fireEvent.click(getByRole('button', { name: /Keep the first question stable/i }))

    expect(onSelect).toHaveBeenCalledWith('conversation-1')
  })
})
