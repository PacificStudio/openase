import { beforeEach, describe, expect, it, vi } from 'vitest'

import { loadAppContext, loadOrganizations } from './app-context'

describe('app context api', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads organizations from /api/v1/orgs', async () => {
    const fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        organizations: [
          {
            id: 'org-1',
            name: 'Acme',
            slug: 'acme',
            default_agent_provider_id: null,
            status: 'active',
          },
          {
            id: 'org-2',
            name: 'Beta',
            slug: 'beta',
            default_agent_provider_id: null,
            status: 'active',
          },
        ],
      }),
    })

    await expect(loadOrganizations(fetch as typeof globalThis.fetch)).resolves.toEqual([
      {
        id: 'org-1',
        name: 'Acme',
        slug: 'acme',
        default_agent_provider_id: null,
        status: 'active',
      },
      {
        id: 'org-2',
        name: 'Beta',
        slug: 'beta',
        default_agent_provider_id: null,
        status: 'active',
      },
    ])
    expect(fetch).toHaveBeenCalledWith('/api/v1/orgs')
  })

  it('loads workspace app context for the /orgs route without org or project scope', async () => {
    const fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        organizations: [
          {
            id: 'org-1',
            name: 'Acme',
            slug: 'acme',
            default_agent_provider_id: null,
            status: 'active',
          },
        ],
        projects: [],
        providers: [],
        agent_count: 0,
      }),
    })

    await expect(loadAppContext(fetch as typeof globalThis.fetch)).resolves.toEqual({
      organizations: [
        {
          id: 'org-1',
          name: 'Acme',
          slug: 'acme',
          default_agent_provider_id: null,
          status: 'active',
        },
      ],
      projects: [],
      providers: [],
      agentCount: 0,
    })
    expect(fetch).toHaveBeenCalledWith('/api/v1/app-context')
  })

  it('loads org-scoped app context for the organization detail route', async () => {
    const fetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({
        organizations: [
          {
            id: 'org-1',
            name: 'Acme',
            slug: 'acme',
            default_agent_provider_id: 'provider-1',
            status: 'active',
          },
        ],
        projects: [
          {
            id: 'project-1',
            organization_id: 'org-1',
            name: 'Todo App',
            slug: 'todo-app',
            description: 'Track work',
            status: 'active',
            default_agent_provider_id: 'provider-1',
            max_concurrent_agents: 2,
            accessible_machine_ids: [],
          },
        ],
        providers: [],
        agent_count: 2,
      }),
    })

    await expect(
      loadAppContext(fetch as typeof globalThis.fetch, {
        orgId: 'org-1',
        projectId: null,
      }),
    ).resolves.toEqual({
      organizations: [
        {
          id: 'org-1',
          name: 'Acme',
          slug: 'acme',
          default_agent_provider_id: 'provider-1',
          status: 'active',
        },
      ],
      projects: [
        {
          id: 'project-1',
          organization_id: 'org-1',
          name: 'Todo App',
          slug: 'todo-app',
          description: 'Track work',
          status: 'active',
          default_agent_provider_id: 'provider-1',
          max_concurrent_agents: 2,
          accessible_machine_ids: [],
        },
      ],
      providers: [],
      agentCount: 2,
    })
    expect(fetch).toHaveBeenCalledWith('/api/v1/app-context?org_id=org-1')
  })
})
