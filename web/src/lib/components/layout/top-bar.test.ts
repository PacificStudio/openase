import { cleanup, render } from '@testing-library/svelte'
import { afterEach, describe, expect, it } from 'vitest'

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
})
