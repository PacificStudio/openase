import { render, waitFor } from '@testing-library/svelte'
import { describe, expect, it } from 'vitest'

import CodeEditor from './code-editor.svelte'

describe('CodeEditor', () => {
  it('switches wrap mode without recreating the editor DOM or losing content', async () => {
    const value = Array.from({ length: 40 }, (_, index) => `line ${index} ${'x'.repeat(120)}`).join(
      '\n',
    )

    const view = render(CodeEditor, {
      props: {
        value,
        filePath: 'notes.md',
        wrapMode: 'wrap',
      },
    })

    await waitFor(() => expect(view.container.querySelector('.cm-editor')).not.toBeNull())

    const content = view.container.querySelector('.cm-content')
    const scroller = view.container.querySelector('.cm-scroller') as HTMLElement | null

    expect(content).not.toBeNull()
    expect(scroller).not.toBeNull()
    expect(view.container.querySelector('.cm-lineWrapping')).not.toBeNull()
    scroller!.scrollTop = 240

    await view.rerender({
      value,
      filePath: 'notes.md',
      wrapMode: 'nowrap',
    })

    await waitFor(() => expect(view.container.querySelector('.cm-lineWrapping')).toBeNull())
    expect(view.container.querySelector('.cm-content')).toBe(content)
    expect(view.container.querySelector('.cm-scroller')).toBe(scroller)
    expect(scroller!.scrollTop).toBe(240)
    expect(view.container.querySelector('.cm-content')?.textContent).toContain('line 0')

    await view.rerender({
      value,
      filePath: 'notes.md',
      wrapMode: 'wrap',
    })

    await waitFor(() => expect(view.container.querySelector('.cm-lineWrapping')).not.toBeNull())
    expect(view.container.querySelector('.cm-content')).toBe(content)
    expect(view.container.querySelector('.cm-scroller')).toBe(scroller)
    expect(scroller!.scrollTop).toBe(240)
    expect(view.container.querySelector('.cm-content')?.textContent).toContain('line 0')
  })
})
