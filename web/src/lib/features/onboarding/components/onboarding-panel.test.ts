import { cleanup, fireEvent, render, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import OnboardingPanel from './onboarding-panel.svelte'

const { loadOnboardingData } = vi.hoisted(() => ({
  loadOnboardingData: vi.fn(),
}))

vi.mock('../data', () => ({
  loadOnboardingData,
}))

describe('OnboardingPanel', () => {
  beforeEach(() => {
    loadOnboardingData.mockResolvedValue({
      github: { hasToken: false, probeStatus: 'unknown', login: '', confirmed: false },
      repo: { repos: [], namespaces: [] },
      provider: { providers: [], selectedProviderId: '' },
      agentWorkflow: { agents: [], workflows: [], statuses: [] },
      firstTicket: { ticketCount: 0 },
      aiDiscovery: { completed: false },
      projectStatus: 'Planned',
    })
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('lets users skip onboarding from the current step and finish immediately', async () => {
    const onOnboardingComplete = vi.fn()
    const { findByText, getByText } = render(OnboardingPanel, {
      props: {
        projectId: 'project-1',
        orgId: 'org-1',
        projectName: 'OpenASE',
        projectStatus: 'Planned',
        onOnboardingComplete,
      },
    })

    expect(await findByText('跳过导览')).toBeTruthy()
    expect(getByText('不想继续配置的话，可以直接跳过导览并结束。')).toBeTruthy()

    await fireEvent.click(getByText('跳过导览'))

    await waitFor(() => {
      expect(onOnboardingComplete).toHaveBeenCalledTimes(1)
    })
  })
})
