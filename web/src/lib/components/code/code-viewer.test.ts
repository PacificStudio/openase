import { render, waitFor } from '@testing-library/svelte'
import { tick } from 'svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { appStore } from '$lib/stores/app.svelte'

import CodeViewer from './code-viewer.svelte'

const { codeToHtml } = vi.hoisted(() => ({
  codeToHtml: vi.fn(async (_code: string, options: { theme: string }) => {
    if (options.theme === 'github-light-default') {
      return '<pre class="shiki light"><code><span class="line">light</span></code></pre>'
    }
    throw new Error('unsupported language')
  }),
}))

vi.mock('shiki', () => ({ codeToHtml }))

describe('CodeViewer', () => {
  beforeEach(() => {
    appStore.setTheme('dark')
    codeToHtml.mockClear()
  })

  afterEach(() => {
    appStore.setTheme('dark')
  })

  it('keeps wide code inside a single bounded scrollport', () => {
    const view = render(CodeViewer, {
      props: {
        code: 'const veryLongLine = "' + 'x'.repeat(400) + '"\n',
        language: 'definitely-not-a-real-language',
      },
    })

    expect(view.getByTestId('code-viewer-scrollport').className).toContain('overflow-auto')
    expect(view.getByTestId('code-viewer-content').className).toContain('w-max')
    expect(view.getByTestId('code-viewer-content').className).toContain('min-w-full')
  })

  it('re-highlights with the matching shiki theme when app theme changes', async () => {
    const view = render(CodeViewer, {
      props: {
        code: 'export const themed = true\n',
        language: 'ts',
      },
    })

    await waitFor(() =>
      expect(codeToHtml).toHaveBeenCalledWith(
        'export const themed = true\n',
        expect.objectContaining({ theme: 'github-dark-default' }),
      ),
    )
    expect(view.container.querySelector('.shiki.light')).toBeNull()

    appStore.setTheme('light')
    await tick()

    await waitFor(() =>
      expect(codeToHtml).toHaveBeenLastCalledWith(
        'export const themed = true\n',
        expect.objectContaining({ theme: 'github-light-default' }),
      ),
    )
    await waitFor(() => expect(view.container.querySelector('.shiki.light')).not.toBeNull())
  })
})
