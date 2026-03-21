import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it, vi } from 'vitest'

import TopBar from './top-bar.svelte'

describe('TopBar', () => {
  afterEach(() => {
    cleanup()
  })

  it('hides the global search affordance while search is unavailable', () => {
    const { queryByText } = render(TopBar, {
      props: {
        searchEnabled: false,
      },
    })

    expect(queryByText('Search...')).toBeNull()
  })

  it('renders the global search affordance once search is available', () => {
    const { getByText } = render(TopBar, {
      props: {
        searchEnabled: true,
      },
    })

    expect(getByText('Search...')).toBeTruthy()
  })

  it('does not fail when search is enabled without a click handler', async () => {
    const { getByText } = render(TopBar, {
      props: {
        searchEnabled: true,
      },
    })

    await expect(fireEvent.click(getByText('Search...'))).resolves.toBe(true)
  })

  it('invokes the search handler when one is provided', async () => {
    const onOpenSearch = vi.fn()
    const { getByText } = render(TopBar, {
      props: {
        searchEnabled: true,
        onOpenSearch,
      },
    })

    await fireEvent.click(getByText('Search...'))

    expect(onOpenSearch).toHaveBeenCalledTimes(1)
  })
})
