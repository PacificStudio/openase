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

    expect(getByText('最后一步里，点击任意一个按钮都可以结束导览。')).toBeTruthy()
    expect(
      getByText('你可以直接体验 Project AI、前往 Workflow 编辑器，或者点“我知道了”先结束导览。'),
    ).toBeTruthy()

    await fireEvent.click(getByText('帮我拆 3 个后续工单'))

    expect(onOpenProjectAI).toHaveBeenCalledWith('基于当前项目和已有 Ticket，再帮我拆 3 个后续工单')
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

    await fireEvent.click(getByText('前往 Workflow 编辑器'))

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

    await fireEvent.click(getByText('我知道了'))

    expect(onComplete).toHaveBeenCalledTimes(1)
    expect(goto).not.toHaveBeenCalled()
  })
})
