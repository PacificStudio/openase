import { describe, expect, it } from 'vitest'

import type { ProjectConversation } from '$lib/api/chat'
import { formatProjectConversationLabel } from './project-conversation-panel-labels'

describe('formatProjectConversationLabel', () => {
  it('prefers the stable title over rolling summary', () => {
    const conversations: ProjectConversation[] = [
      {
        id: 'conversation-1',
        projectId: 'project-1',
        userId: 'user-1',
        source: 'project_sidebar',
        providerId: 'provider-1',
        title: 'Use the first user sentence',
        status: 'active',
        rollingSummary: 'Latest thread summary',
        lastActivityAt: '2026-04-06T12:00:00Z',
        createdAt: '2026-04-06T11:00:00Z',
        updatedAt: '2026-04-06T12:00:00Z',
      },
    ]

    const label = formatProjectConversationLabel(
      {
        conversationId: 'conversation-1',
        entries: [],
        draft: '',
      },
      conversations,
    )

    expect(label).toBe('Use the first user sentence')
  })

  it('falls back to the first user message instead of the most recent one', () => {
    const label = formatProjectConversationLabel(
      {
        conversationId: 'conversation-1',
        entries: [
          {
            id: 'entry-1',
            kind: 'text',
            role: 'user',
            content: 'First user prompt should win',
            streaming: false,
          },
          {
            id: 'entry-2',
            kind: 'text',
            role: 'assistant',
            content: 'Assistant reply',
            streaming: false,
          },
          {
            id: 'entry-3',
            kind: 'text',
            role: 'user',
            content: 'Later user follow-up should not rename it',
            streaming: false,
          },
        ],
        draft: '',
      },
      [
        {
          id: 'conversation-1',
          projectId: 'project-1',
          userId: 'user-1',
          source: 'project_sidebar',
          providerId: 'provider-1',
          title: '',
          status: 'active',
          rollingSummary: '',
          lastActivityAt: '2026-04-06T12:00:00Z',
          createdAt: '2026-04-06T11:00:00Z',
          updatedAt: '2026-04-06T12:00:00Z',
        },
      ],
    )

    expect(label).toBe('First user prompt should win')
  })

  it('does not use rolling summary as the primary label when title is missing', () => {
    const label = formatProjectConversationLabel(
      {
        conversationId: 'conversation-1',
        entries: [],
        draft: '',
      },
      [
        {
          id: 'conversation-1',
          projectId: 'project-1',
          userId: 'user-1',
          source: 'project_sidebar',
          providerId: 'provider-1',
          title: '',
          status: 'active',
          rollingSummary: 'Recovery-only summary',
          lastActivityAt: '2026-04-06T12:00:00Z',
          createdAt: '2026-04-06T11:00:00Z',
          updatedAt: '2026-04-06T12:00:00Z',
        },
      ],
    )

    expect(label).toMatch(/^Conversation ·/)
  })
})
