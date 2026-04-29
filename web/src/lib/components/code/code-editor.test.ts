import { fireEvent, render, waitFor, within } from '@testing-library/svelte'
import { EditorView } from '@codemirror/view'
import { describe, expect, it, vi } from 'vitest'

import CodeEditor from './code-editor.svelte'

async function getEditorView(container: HTMLElement): Promise<EditorView> {
  await waitFor(() => expect(container.querySelector('.cm-editor')).not.toBeNull())

  const editorDom = container.querySelector('.cm-editor')
  expect(editorDom).not.toBeNull()

  const editorView = EditorView.findFromDOM(editorDom as HTMLElement)
  expect(editorView).not.toBeNull()
  return editorView as EditorView
}

async function pressEditorShortcut(editorView: EditorView, init: KeyboardEventInit) {
  editorView.focus()
  await fireEvent.keyDown(editorView.contentDOM, {
    bubbles: true,
    cancelable: true,
    ...init,
  })
}

describe('CodeEditor', () => {
  it('renders visible diff gutter indicators next to line numbers', async () => {
    const view = render(CodeEditor, {
      props: {
        value: 'alpha\nbeta\ngamma\n',
        filePath: 'src/example.ts',
        diffMarkers: {
          added: [1],
          modified: [2],
          deletionAbove: [3],
          deletionAtEnd: true,
        },
      },
    })

    await waitFor(() => expect(view.container.querySelector('.cm-editor')).not.toBeNull())

    expect(view.container.querySelector(".cm-diff-marker[data-kind='added']")).not.toBeNull()
    expect(
      view.container.querySelector(".cm-diff-marker[data-kind='added'] .cm-diff-marker-indicator"),
    ).not.toBeNull()
    expect(view.container.querySelector(".cm-diff-marker[data-kind='modified']")).not.toBeNull()
    expect(
      view.container.querySelector(
        ".cm-diff-marker[data-kind='modified'] .cm-diff-marker-indicator",
      ),
    ).not.toBeNull()
    expect(
      view.container.querySelector(".cm-diff-marker[data-deletion-above='true']"),
    ).not.toBeNull()
    expect(
      view.container.querySelector(".cm-diff-marker[data-deletion-below='true']"),
    ).not.toBeNull()
  })

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

  it('routes keyboard shortcuts and context menu actions according to the current selection', async () => {
    const onFormatDocument = vi.fn()
    const onFormatSelection = vi.fn()
    const onSave = vi.fn()
    const onRevert = vi.fn()
    const onExplainSelection = vi.fn()
    const onRewriteSelection = vi.fn()

    const view = render(CodeEditor, {
      props: {
        value: 'const alpha = 1;\nconst beta = 2;\n',
        filePath: 'src/example.ts',
        onFormatDocument,
        onFormatSelection,
        onSave,
        onRevert,
        onExplainSelection,
        onRewriteSelection,
      },
    })

    const editorView = await getEditorView(view.container)
    const editorShell = view.container.querySelector('.code-editor') as HTMLElement

    await pressEditorShortcut(editorView, { key: 's', code: 'KeyS', ctrlKey: true })
    expect(onSave).toHaveBeenCalledTimes(1)

    await pressEditorShortcut(editorView, { key: 'f', code: 'KeyF', altKey: true, shiftKey: true })
    expect(onFormatDocument).toHaveBeenCalledTimes(1)
    expect(onFormatSelection).not.toHaveBeenCalled()

    editorView.dispatch({ selection: { anchor: 0, head: 5 } })

    await pressEditorShortcut(editorView, { key: 'f', code: 'KeyF', altKey: true, shiftKey: true })
    expect(onFormatSelection).toHaveBeenCalledTimes(1)

    await fireEvent.contextMenu(editorShell, { clientX: 96, clientY: 128 })

    const selectionMenu = await view.findByTestId('code-editor-context-menu')
    expect(within(selectionMenu).queryByRole('menuitem', { name: 'Format Document' })).toBeNull()
    expect(within(selectionMenu).getByRole('menuitem', { name: /^Format Selection/ })).toBeTruthy()
    expect(within(selectionMenu).getByRole('menuitem', { name: 'Revert File' })).toBeTruthy()
    expect(within(selectionMenu).getByRole('menuitem', { name: 'Explain Selection' })).toBeTruthy()
    expect(within(selectionMenu).getByRole('menuitem', { name: 'Rewrite Selection' })).toBeTruthy()
    expect(within(selectionMenu).queryByRole('menuitem', { name: 'Save' })).toBeNull()

    await fireEvent.click(
      within(selectionMenu).getByRole('menuitem', { name: 'Explain Selection' }),
    )
    expect(onExplainSelection).toHaveBeenCalledTimes(1)
    await waitFor(() => expect(view.queryByTestId('code-editor-context-menu')).toBeNull())

    editorView.dispatch({ selection: { anchor: 0, head: 0 } })

    await fireEvent.contextMenu(editorShell, { clientX: 112, clientY: 144 })

    const documentMenu = await view.findByTestId('code-editor-context-menu')
    expect(within(documentMenu).getByRole('menuitem', { name: /^Format Document/ })).toBeTruthy()
    expect(within(documentMenu).queryByRole('menuitem', { name: 'Format Selection' })).toBeNull()
    expect(within(documentMenu).queryByRole('menuitem', { name: 'Explain Selection' })).toBeNull()
    expect(within(documentMenu).queryByRole('menuitem', { name: 'Rewrite Selection' })).toBeNull()

    await fireEvent.click(within(documentMenu).getByRole('menuitem', { name: 'Revert File' }))
    expect(onRevert).toHaveBeenCalledTimes(1)

    editorView.dispatch({ selection: { anchor: 0, head: 5 } })
    await fireEvent.contextMenu(editorShell, { clientX: 136, clientY: 168 })
    const actionMenu = await view.findByTestId('code-editor-context-menu')
    await fireEvent.click(within(actionMenu).getByRole('menuitem', { name: 'Rewrite Selection' }))
    expect(onRewriteSelection).toHaveBeenCalledTimes(1)
  })
})
