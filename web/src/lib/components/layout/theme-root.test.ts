import { cleanup, render } from '@testing-library/svelte'
import { tick } from 'svelte'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'

import { i18nStore } from '$lib/i18n/store.svelte'
import { appStore } from '$lib/stores/app.svelte'

import ThemeRoot from './theme-root.svelte'

describe('ThemeRoot', () => {
  beforeEach(() => {
    appStore.setTheme('dark')
    i18nStore.setLocale('en')
    document.documentElement.className = ''
    document.documentElement.lang = ''
    window.localStorage.clear()
  })

  afterEach(() => {
    cleanup()
    appStore.setTheme('dark')
    i18nStore.setLocale('en')
    document.documentElement.className = ''
    document.documentElement.lang = ''
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

  it('syncs the document language with the active locale', async () => {
    render(ThemeRoot)

    await tick()
    expect(document.documentElement.lang).toBe('en')

    i18nStore.setLocale('zh')
    await tick()

    expect(document.documentElement.lang).toBe('zh')
  })
})
