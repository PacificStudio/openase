import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'
import ProjectConversationInterruptCard from './project-conversation-interrupt-card.svelte'

describe('ProjectConversationInterruptCard', () => {
  afterEach(() => {
    cleanup()
  })

  it('submits codex approval decisions with the selected option id', async () => {
    const onRespondInterrupt = vi.fn()
    const { getByRole } = render(ProjectConversationInterruptCard, {
      props: {
        entry: {
          id: 'entry-1',
          kind: 'interrupt',
          role: 'system',
          interruptId: 'interrupt-1',
          provider: 'codex',
          interruptKind: 'command_execution_approval',
          payload: {
            command: 'git status',
          },
          options: [{ id: 'approve_once', label: 'Approve once' }],
          status: 'pending',
        },
        onRespondInterrupt,
      },
    })

    await fireEvent.click(getByRole('button', { name: 'Approve once' }))

    expect(onRespondInterrupt).toHaveBeenCalledWith({
      interruptId: 'interrupt-1',
      decision: 'approve_once',
    })
  })

  it('submits structured answers for user-input interrupts', async () => {
    const onRespondInterrupt = vi.fn()
    const { getByRole } = render(ProjectConversationInterruptCard, {
      props: {
        entry: {
          id: 'entry-2',
          kind: 'interrupt',
          role: 'system',
          interruptId: 'interrupt-2',
          provider: 'codex',
          interruptKind: 'user_input',
          payload: {
            questions: [
              {
                id: 'approval',
                question: 'Approve the next step?',
                options: [{ label: 'Yes' }, { label: 'No' }],
              },
            ],
          },
          options: [],
          status: 'pending',
        },
        onRespondInterrupt,
      },
    })

    await fireEvent.click(getByRole('button', { name: 'Yes' }))

    expect(onRespondInterrupt).toHaveBeenCalledWith({
      interruptId: 'interrupt-2',
      answer: {
        approval: {
          answers: ['Yes'],
        },
      },
    })
  })
})
