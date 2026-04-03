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
})
