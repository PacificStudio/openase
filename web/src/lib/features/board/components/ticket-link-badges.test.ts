import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import TicketLinkBadges from './ticket-link-badges.svelte'
import type { BoardExternalLink } from '../types'

function buildLink(overrides: Partial<BoardExternalLink> = {}): BoardExternalLink {
  return {
    id: 'link-1',
    type: 'external',
    url: 'https://example.com/reference',
    externalId: 'EXT-1',
    title: 'External reference',
    relation: 'references',
    ...overrides,
  }
}

afterEach(() => {
  cleanup()
  vi.restoreAllMocks()
})

describe('TicketLinkBadges', () => {
  it('renders a single PR badge and opens the PR directly', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const prURL = 'https://github.com/openase/openase/pull/42'

    const { getByText, queryByRole } = render(TicketLinkBadges, {
      pullRequestURLs: [prURL],
    })

    expect(queryByRole('menu')).toBeNull()

    await fireEvent.click(getByText('#42'))

    expect(openSpy).toHaveBeenCalledWith(prURL, '_blank', 'noopener,noreferrer')
  })

  it('deduplicates PR badges across repo-scope URLs and external links', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const pr101 = 'https://github.com/openase/openase/pull/101'
    const pr102 = 'https://github.com/openase/openase/pull/102'
    const pr103 = 'https://github.com/openase/openase/pull/103'

    const { getByRole, findByRole, queryAllByText } = render(TicketLinkBadges, {
      pullRequestURLs: [pr101, pr102],
      links: [
        buildLink({ id: 'link-pr-102', url: pr102, externalId: 'PR-102' }),
        buildLink({ id: 'link-pr-103', url: pr103, externalId: 'PR-103' }),
      ],
    })

    await fireEvent.click(getByRole('button', { name: '3' }))

    expect(queryAllByText('#101')).toHaveLength(1)
    expect(queryAllByText('#102')).toHaveLength(1)
    expect(queryAllByText('#103')).toHaveLength(1)

    await fireEvent.click(await findByRole('menuitem', { name: /#103/ }))

    expect(openSpy).toHaveBeenCalledWith(pr103, '_blank', 'noopener,noreferrer')
  })

  it('keeps PR badges separate from GitHub issue badges', async () => {
    const openSpy = vi.spyOn(window, 'open').mockImplementation(() => null)
    const prURL = 'https://github.com/openase/openase/pull/42'
    const issueURL = 'https://github.com/openase/openase/issues/77'

    const { getByText } = render(TicketLinkBadges, {
      pullRequestURLs: [prURL],
      links: [
        buildLink({
          id: 'link-issue-77',
          type: 'issue',
          url: issueURL,
          externalId: '77',
          title: 'Fix the broken dropdown',
        }),
      ],
    })

    await fireEvent.click(getByText('#42'))
    await fireEvent.click(getByText('#77'))

    expect(openSpy).toHaveBeenNthCalledWith(1, prURL, '_blank', 'noopener,noreferrer')
    expect(openSpy).toHaveBeenNthCalledWith(2, issueURL, '_blank', 'noopener,noreferrer')
  })

  it('renders nothing when there are no PRs or external links', () => {
    const { container } = render(TicketLinkBadges, {
      pullRequestURLs: [],
      links: [],
    })

    expect(container.firstElementChild).toBeNull()
  })
})
