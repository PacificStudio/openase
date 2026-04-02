import { describe, expect, it } from 'vitest'

import { buildOnboardingSteps, currentActiveStep, getBootstrapPreset } from './model'
import type { OnboardingData } from './types'

function emptyData(): OnboardingData {
  return {
    github: { hasToken: false, probeStatus: 'unknown', login: '', confirmed: false },
    repo: { repos: [], namespaces: [] },
    provider: { providers: [], selectedProviderId: '' },
    agentWorkflow: { agents: [], workflows: [], statuses: [] },
    firstTicket: { ticketCount: 0 },
    aiDiscovery: { completed: false },
    projectStatus: 'Planned',
  }
}

describe('onboarding model', () => {
  it('treats AI discovery as the final remaining step until it is completed', () => {
    const data: OnboardingData = {
      ...emptyData(),
      github: { hasToken: true, probeStatus: 'valid', login: 'octocat', confirmed: true },
      repo: { repos: [{ id: 'repo-1', name: 'app' } as never], namespaces: [] },
      provider: {
        providers: [{ id: 'provider-1', availability_state: 'available' } as never],
        selectedProviderId: 'provider-1',
      },
      agentWorkflow: {
        agents: [{ id: 'agent-1', name: 'coder-01' } as never],
        workflows: [{ id: 'workflow-1', name: 'Coder Workflow' } as never],
        statuses: [],
      },
      firstTicket: { ticketCount: 1 },
      aiDiscovery: { completed: false },
    }

    expect(currentActiveStep(data)).toBe('ai_discovery')
    expect(buildOnboardingSteps(data).at(-1)?.status).toBe('active')

    const completed = { ...data, aiDiscovery: { completed: true } }
    expect(currentActiveStep(completed)).toBeNull()
    expect(buildOnboardingSteps(completed).every((step) => step.status === 'completed')).toBe(true)
  })

  it('uses the role presets expected by project status', () => {
    expect(getBootstrapPreset('Planned').roleSlug).toBe('product-manager')
    expect(getBootstrapPreset('Planned').pickupStatusName).toBe('Backlog')
    expect(getBootstrapPreset('Planned').finishStatusName).toBe('Done')
    expect(getBootstrapPreset('In Progress').roleSlug).toBe('fullstack-developer')
    expect(getBootstrapPreset('In Progress').pickupStatusName).toBe('Backlog')
    expect(getBootstrapPreset('In Progress').finishStatusName).toBe('Done')
  })
})
