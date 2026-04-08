import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import StepAiDiscovery from './step-ai-discovery.svelte'

describe('StepAiDiscovery', () => {
  beforeEach(() => {})

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('completes onboarding when a Project AI action is clicked', async () => {
    const onOpenProjectAI = vi.fn()
    const onComplete = vi.fn()
    const { getByText } = render(StepAiDiscovery, {
      props: {
        onOpenProjectAI,
        onComplete,
      },
    })

    await fireEvent.click(getByText('Break down 3 follow-up tickets'))

    expect(onOpenProjectAI).toHaveBeenCalledWith(
      'Based on the current project and existing tickets, break down 3 follow-up tickets for me.',
    )
    expect(onComplete).toHaveBeenCalledTimes(1)
  })

  it('completes onboarding when the acknowledge button is clicked', async () => {
    const onComplete = vi.fn()
    const { getByText } = render(StepAiDiscovery, {
      props: {
        onOpenProjectAI: vi.fn(),
        onComplete,
      },
    })

    await fireEvent.click(getByText('Got it'))

    expect(onComplete).toHaveBeenCalledTimes(1)
  })
})
