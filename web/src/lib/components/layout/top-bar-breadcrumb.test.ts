import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest'

import type { Organization, Project } from '$lib/api/contracts'

const { goto, preloadCode, preloadData } = vi.hoisted(() => ({
  goto: vi.fn(),
  preloadCode: vi.fn(),
  preloadData: vi.fn(),
}))

vi.mock('$app/navigation', () => ({
  goto,
  preloadCode,
  preloadData,
}))

import TopBarBreadcrumb from './top-bar-breadcrumb.svelte'

function makeOrganizations(count: number): Organization[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `org-${index + 1}`,
    name: `Organization ${index + 1}`,
    slug: `organization-${index + 1}`,
    status: 'active',
    default_agent_provider_id: null,
  }))
}

function makeProjects(count: number, organizationId = 'org-1'): Project[] {
  return Array.from({ length: count }, (_, index) => ({
    id: `project-${index + 1}`,
    organization_id: organizationId,
    name: `Project ${index + 1}`,
    slug: `project-${index + 1}`,
    description: '',
    status: 'active',
    default_agent_provider_id: null,
    accessible_machine_ids: [],
    max_concurrent_agents: 4,
  }))
}

async function openDropdown(name: string) {
  const trigger = screen.getByRole('button', { name })
  await fireEvent.click(trigger)
  await waitFor(() => {
    expect(screen.getByRole('menu')).toBeTruthy()
  })
  return screen.getByRole('menu')
}

describe('TopBarBreadcrumb', () => {
  beforeAll(() => {
    HTMLElement.prototype.scrollIntoView ??= vi.fn()
    HTMLElement.prototype.hasPointerCapture ??= vi.fn(() => false)
    HTMLElement.prototype.releasePointerCapture ??= vi.fn()
    globalThis.ResizeObserver ??= class {
      observe() {}
      unobserve() {}
      disconnect() {}
    }
  })

  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('caps the organization menu height so long org lists stay scrollable', async () => {
    const organizations = makeOrganizations(40)

    render(TopBarBreadcrumb, {
      props: {
        organizations,
        currentOrgId: 'org-1',
        orgName: 'Organization 1',
      },
    })

    const menu = await openDropdown('Organization 1')

    expect(menu.className).toContain(
      'max-h-[min(18rem,var(--bits-dropdown-menu-content-available-height))]',
    )
    expect(menu.className).toContain('overflow-y-auto')
    expect(menu.className).toContain('overscroll-contain')
    expect(await screen.findByRole('menuitem', { name: 'Organization 40' })).toBeTruthy()
  })

  it('keeps the project menu scrollable for long project lists too', async () => {
    const organizations = makeOrganizations(1)
    const projects = makeProjects(32)

    render(TopBarBreadcrumb, {
      props: {
        organizations,
        projects,
        currentOrgId: 'org-1',
        currentProjectId: 'project-1',
        orgName: 'Organization 1',
        projectName: 'Project 1',
      },
    })

    const menu = await openDropdown('Project 1')

    expect(menu.className).toContain(
      'max-h-[min(18rem,var(--bits-dropdown-menu-content-available-height))]',
    )
    expect(menu.className).toContain('overflow-y-auto')
    expect(menu.className).toContain('overscroll-contain')
    expect(await screen.findByRole('menuitem', { name: 'Project 32' })).toBeTruthy()
  })
})
