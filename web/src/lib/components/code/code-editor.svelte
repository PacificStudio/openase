<script lang="ts">
  import { onMount } from 'svelte'
  import { cn } from '$lib/utils'
  import {
    EditorView,
    keymap,
    lineNumbers,
    highlightActiveLine,
    placeholder as cmPlaceholder,
  } from '@codemirror/view'
  import { EditorState, type Extension } from '@codemirror/state'
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
  import { syntaxHighlighting, defaultHighlightStyle, bracketMatching } from '@codemirror/language'
  import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
  import { detectLanguage } from './lang'

  let {
    value = '',
    filePath = '',
    language = '',
    readonly = false,
    placeholder = '',
    class: className = '',
    onchange,
  }: {
    /** Current document content */
    value?: string
    /** Used to auto-detect language when `language` is not set */
    filePath?: string
    /** Explicit language id override */
    language?: string
    /** Read-only mode */
    readonly?: boolean
    /** Placeholder text when empty */
    placeholder?: string
    class?: string
    /** Fires on every edit */
    onchange?: (value: string) => void
  } = $props()

  let container: HTMLDivElement
  let view: EditorView | undefined
  let suppressExternalUpdate = false

  const lang = $derived(language || detectLanguage(filePath))

  // Build a dark theme that matches the existing harness-editor look
  const darkTheme = EditorView.theme(
    {
      '&': {
        backgroundColor: 'transparent',
        color: '#e6edf3',
        fontSize: '13px',
        height: '100%',
      },
      '&.cm-focused': { outline: 'none' },
      '.cm-scroller': { fontFamily: 'inherit', lineHeight: '1.5rem', overflow: 'auto' },
      '.cm-gutters': {
        backgroundColor: 'transparent',
        borderRight: '1px solid color-mix(in srgb, currentColor 12%, transparent)',
        color: 'color-mix(in srgb, currentColor 35%, transparent)',
        minWidth: '3rem',
      },
      '.cm-activeLineGutter': {
        backgroundColor: 'transparent',
        color: 'color-mix(in srgb, currentColor 60%, transparent)',
      },
      '.cm-activeLine': {
        backgroundColor: 'color-mix(in srgb, currentColor 4%, transparent)',
      },
      '.cm-cursor': { borderLeftColor: '#e6edf3' },
      '.cm-selectionBackground': {
        backgroundColor: 'rgba(56, 139, 253, 0.25) !important',
      },
      '.cm-matchingBracket': {
        backgroundColor: 'rgba(56, 139, 253, 0.15)',
        outline: '1px solid rgba(56, 139, 253, 0.4)',
      },
      '.cm-placeholder': {
        color: 'color-mix(in srgb, currentColor 30%, transparent)',
      },
    },
    { dark: true },
  )

  async function loadLanguageExtension(langId: string): Promise<Extension[]> {
    try {
      switch (langId) {
        case 'yaml': {
          const { yaml } = await import('@codemirror/lang-yaml')
          return [yaml()]
        }
        case 'markdown':
        case 'mdx': {
          const { markdown } = await import('@codemirror/lang-markdown')
          return [markdown()]
        }
        case 'javascript':
        case 'jsx': {
          const { javascript } = await import('@codemirror/lang-javascript')
          return [javascript({ jsx: langId === 'jsx' })]
        }
        case 'typescript':
        case 'tsx': {
          const { javascript } = await import('@codemirror/lang-javascript')
          return [javascript({ jsx: langId === 'tsx', typescript: true })]
        }
        case 'json': {
          const { json } = await import('@codemirror/lang-json')
          return [json()]
        }
        case 'html':
        case 'svelte':
        case 'vue': {
          const { html } = await import('@codemirror/lang-html')
          return [html()]
        }
        case 'css':
        case 'scss':
        case 'less': {
          const { css } = await import('@codemirror/lang-css')
          return [css()]
        }
        case 'python': {
          const { python } = await import('@codemirror/lang-python')
          return [python()]
        }
        default:
          return []
      }
    } catch {
      return []
    }
  }

  function buildBaseExtensions(): Extension[] {
    const exts: Extension[] = [
      lineNumbers(),
      history(),
      bracketMatching(),
      highlightActiveLine(),
      highlightSelectionMatches(),
      syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
      darkTheme,
      keymap.of([...defaultKeymap, ...historyKeymap, ...searchKeymap, indentWithTab]),
      EditorView.lineWrapping,
    ]

    if (placeholder) {
      exts.push(cmPlaceholder(placeholder))
    }

    if (readonly) {
      exts.push(EditorState.readOnly.of(true), EditorView.editable.of(false))
    } else {
      exts.push(
        EditorView.updateListener.of((update) => {
          if (update.docChanged) {
            suppressExternalUpdate = true
            onchange?.(update.state.doc.toString())
            suppressExternalUpdate = false
          }
        }),
      )
    }

    return exts
  }

  async function createEditor() {
    const langExts = await loadLanguageExtension(lang)
    view = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [...buildBaseExtensions(), ...langExts],
      }),
      parent: container,
    })
  }

  onMount(() => {
    void createEditor()
    return () => {
      view?.destroy()
      view = undefined
    }
  })

  // Sync external value changes into the editor
  $effect(() => {
    const nextValue = value
    if (!view || suppressExternalUpdate) return
    const currentValue = view.state.doc.toString()
    if (nextValue === currentValue) return

    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: nextValue },
    })
  })

  // Recreate editor when language changes
  let lastLang = ''
  $effect(() => {
    const nextLang = lang
    if (nextLang === lastLang || !view) return
    lastLang = nextLang

    const currentDoc = view.state.doc.toString()
    view.destroy()
    value = currentDoc
    void createEditor()
  })
</script>

<div
  bind:this={container}
  class={cn('code-editor h-full min-h-0 overflow-hidden font-mono', className)}
></div>

<style>
  .code-editor :global(.cm-editor) {
    height: 100%;
  }
</style>
