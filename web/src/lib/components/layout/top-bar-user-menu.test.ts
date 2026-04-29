import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { i18nStore } from '$lib/i18n/store.svelte'

import TopBarUserMenu from './top-bar-user-menu.svelte'

describe('TopBarUserMenu', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    i18nStore.setLocale('en')
  })

  afterEach(async () => {
    cleanup()
    await vi.runOnlyPendingTimersAsync()
    vi.useRealTimers()
    i18nStore.setLocale('en')
  })

  it('switches the active locale from the user menu', async () => {
    const { getByRole, getByText } = render(TopBarUserMenu, {
      props: {
        userDisplayName: 'Test User',
        userInitials: 'TU',
      },
    })

    await fireEvent.click(getByRole('button'))
    await fireEvent.click(getByText('Chinese'))

    expect(i18nStore.locale).toBe('zh')
    expect(getByText('语言')).toBeTruthy()
  })
})
