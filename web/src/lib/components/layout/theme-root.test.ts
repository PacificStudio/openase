import { cleanup, render } from '@testing-library/svelte'
import { tick } from 'svelte'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'

import ThemeRoot from './theme-root.svelte'

describe('ThemeRoot', () => {
  beforeEach(() => {
    appStore.setTheme('dark')
    document.documentElement.className = ''
  })

  afterEach(() => {
    cleanup()
    appStore.setTheme('dark')
    document.documentElement.className = ''
  })

  it('syncs the root dark class when the theme toggles', async () => {
    const { container } = render(ThemeRoot)

    await tick()

    const root = container.firstElementChild
    expect(root?.classList.contains('dark')).toBe(true)
    expect(document.documentElement.classList.contains('dark')).toBe(true)

    appStore.toggleTheme()
    await tick()

    expect(root?.classList.contains('dark')).toBe(false)
    expect(document.documentElement.classList.contains('dark')).toBe(false)
  })
})
