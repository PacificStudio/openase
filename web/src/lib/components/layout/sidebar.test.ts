import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

const { preloadCode, preloadData } = vi.hoisted(() => ({
  preloadCode: vi.fn(),
  preloadData: vi.fn(),
}))

vi.mock('$app/navigation', () => ({
  preloadCode,
  preloadData,
}))

import Sidebar from './sidebar.svelte'

describe('Sidebar', () => {
  afterEach(() => {
    cleanup()
    vi.clearAllMocks()
  })

  it('warms project routes without preloading page data on sidebar hover', async () => {
    const { getByText } = render(Sidebar, {
      props: {
        currentPath: '/orgs/org-1/projects/project-1',
        currentOrgId: 'org-1',
        currentProjectId: 'project-1',
        projectSelected: true,
      },
    })

    const ticketsLink = getByText('Tickets').closest('a')
    expect(ticketsLink).toBeTruthy()
    expect(ticketsLink?.getAttribute('data-sveltekit-preload-code')).toBe('hover')
    expect(ticketsLink?.hasAttribute('data-sveltekit-preload-data')).toBe(false)

    await fireEvent.pointerEnter(ticketsLink!)

    expect(preloadCode).toHaveBeenCalledWith('/orgs/org-1/projects/project-1/tickets')
    expect(preloadData).not.toHaveBeenCalled()
  })
})
