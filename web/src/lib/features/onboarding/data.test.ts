import { describe, expect, it, vi, beforeEach } from 'vitest'

import { loadOnboardingData } from './data'

const {
  getProject,
  getSecuritySettings,
  listAgents,
  listGitHubNamespaces,
  listProjectRepos,
  listProviders,
  listStatuses,
  listTickets,
  listWorkflows,
} = vi.hoisted(() => ({
  getProject: vi.fn(),
  getSecuritySettings: vi.fn(),
  listAgents: vi.fn(),
  listGitHubNamespaces: vi.fn(),
  listProjectRepos: vi.fn(),
  listProviders: vi.fn(),
  listStatuses: vi.fn(),
  listTickets: vi.fn(),
  listWorkflows: vi.fn(),
}))

vi.mock('$lib/api/openase', () => ({
  getProject,
  getSecuritySettings,
  listAgents,
  listGitHubNamespaces,
  listProjectRepos,
  listProviders,
  listStatuses,
  listTickets,
  listWorkflows,
}))

describe('loadOnboardingData', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getProject.mockResolvedValue({
      project: {
        id: 'project-1',
        organization_id: 'org-1',
        name: 'OpenASE',
        slug: 'openase',
        description: '',
        status: 'Planned',
        default_agent_provider_id: null,
        accessible_machine_ids: [],
        max_concurrent_agents: 0,
      },
    })
    getSecuritySettings.mockResolvedValue({
      security: {
        github: {
          effective: {
            configured: true,
            probe: {
              state: 'valid',
              login: 'octocat',
            },
          },
        },
      },
    })
    listProjectRepos.mockResolvedValue({ repos: [] })
    listProviders.mockResolvedValue({ providers: [] })
    listAgents.mockResolvedValue({ agents: [] })
    listWorkflows.mockResolvedValue({ workflows: [] })
    listTickets.mockResolvedValue({ tickets: [] })
    listStatuses.mockResolvedValue({ statuses: [] })
    listGitHubNamespaces.mockResolvedValue({
      namespaces: [{ id: 'ns-1', name: 'octo-org', kind: 'organization' }],
    })
  })

  it('hydrates GitHub login and namespaces from the effective credential slot', async () => {
    const data = await loadOnboardingData('project-1', 'org-1')

    expect(data.github).toEqual({
      hasToken: true,
      probeStatus: 'valid',
      login: 'octocat',
      confirmed: true,
    })
    expect(data.repo.namespaces).toEqual([{ id: 'ns-1', name: 'octo-org', kind: 'organization' }])
    expect(listGitHubNamespaces).toHaveBeenCalledWith('project-1')
  })

  it('does not request namespaces when the GitHub credential is not valid', async () => {
    getSecuritySettings.mockResolvedValue({
      security: {
        github: {
          effective: {
            configured: true,
            probe: {
              state: 'invalid',
              login: 'octocat',
            },
          },
        },
      },
    })

    const data = await loadOnboardingData('project-1', 'org-1')

    expect(data.github.probeStatus).toBe('invalid')
    expect(data.repo.namespaces).toEqual([])
    expect(listGitHubNamespaces).not.toHaveBeenCalled()
  })
})
