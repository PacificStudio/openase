import { describe, expect, it } from 'vitest'

import { buildProjectNav } from './sidebar-nav'

describe('buildProjectNav', () => {
  it('builds scoped project section hrefs', () => {
    const items = buildProjectNav({
      currentPath: '/orgs/org-1/projects/project-1/tickets',
      currentOrgId: 'org-1',
      currentProjectId: 'project-1',
      agentCount: 2,
    })

    expect(items.find((item) => item.label === 'Tickets')?.href).toBe(
      '/orgs/org-1/projects/project-1/tickets',
    )
    expect(items.find((item) => item.label === 'Settings')?.href).toBe(
      '/orgs/org-1/projects/project-1/settings',
    )
  })

  it('does not fall back to legacy top-level section paths when no project is selected', () => {
    const items = buildProjectNav({
      currentPath: '/orgs/org-1',
      currentOrgId: 'org-1',
      currentProjectId: null,
      agentCount: 0,
    })

    expect(items.every((item) => !item.href.startsWith('/tickets'))).toBe(true)
    expect(items.every((item) => !item.href.startsWith('/agents'))).toBe(true)
    expect(items.every((item) => item.href === '/orgs/org-1')).toBe(true)
  })
})
