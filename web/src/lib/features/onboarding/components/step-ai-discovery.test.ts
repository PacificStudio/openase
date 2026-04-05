import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import StepAiDiscovery from './step-ai-discovery.svelte'

const { goto } = vi.hoisted(() => ({
  goto: vi.fn(),
}))

vi.mock('$app/navigation', () => ({
  goto,
}))

describe('StepAiDiscovery', () => {
  beforeEach(() => {
    goto.mockResolvedValue(undefined)
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('completes onboarding when a Project AI action is clicked', async () => {
    const onOpenProjectAI = vi.fn()
    const onComplete = vi.fn()
    const { getByText } = render(StepAiDiscovery, {
      props: {
        orgId: 'org-1',
        projectId: 'project-1',
        hasWorkflow: true,
        onOpenProjectAI,
        onComplete,
      },
    })

    expect(getByText('On the final step, clicking any button will finish the tour.')).toBeTruthy()
    expect(
      getByText(
        'You can try Project AI, open the workflow editor, or click "Got it" to end the tour now.',
      ),
    ).toBeTruthy()

    await fireEvent.click(getByText('Break down 3 follow-up tickets'))

    expect(onOpenProjectAI).toHaveBeenCalledWith(
      'Based on the current project and existing tickets, break down 3 follow-up tickets for me.',
    )
    expect(onComplete).toHaveBeenCalledTimes(1)
    expect(goto).not.toHaveBeenCalled()
  })

  it('completes onboarding when the workflow editor button is clicked', async () => {
    const onComplete = vi.fn()
    const { getByText } = render(StepAiDiscovery, {
      props: {
        orgId: 'org-1',
        projectId: 'project-1',
        hasWorkflow: true,
        onOpenProjectAI: vi.fn(),
        onComplete,
      },
    })

    await fireEvent.click(getByText('Open workflow editor'))

    await waitFor(() => {
      expect(goto).toHaveBeenCalled()
      expect(onComplete).toHaveBeenCalledTimes(1)
    })
  })

  it('completes onboarding when the acknowledge button is clicked', async () => {
    const onComplete = vi.fn()
    const { getByText } = render(StepAiDiscovery, {
      props: {
        orgId: 'org-1',
        projectId: 'project-1',
        hasWorkflow: false,
        onOpenProjectAI: vi.fn(),
        onComplete,
      },
    })

    await fireEvent.click(getByText('Got it'))

    expect(onComplete).toHaveBeenCalledTimes(1)
    expect(goto).not.toHaveBeenCalled()
  })
})
