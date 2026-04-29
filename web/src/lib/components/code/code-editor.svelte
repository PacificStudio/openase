<script lang="ts">
  import { onMount } from 'svelte'
  import { i18nStore } from '$lib/i18n/store.svelte'
  import { cn } from '$lib/utils'
  import {
    EditorView,
    GutterMarker,
    gutter,
    keymap,
    lineNumbers,
    highlightActiveLine,
    placeholder as cmPlaceholder,
    type KeyBinding,
  } from '@codemirror/view'
  import {
    Compartment,
    EditorState,
    RangeSet,
    RangeSetBuilder,
    StateEffect,
    StateField,
    type Extension,
  } from '@codemirror/state'
  import { defaultKeymap, history, historyKeymap, indentWithTab } from '@codemirror/commands'
  import { HighlightStyle, syntaxHighlighting, bracketMatching } from '@codemirror/language'
  import { searchKeymap, highlightSelectionMatches } from '@codemirror/search'
  import { tags as t } from '@lezer/highlight'
  import { appStore } from '$lib/stores/app.svelte'
  import { detectLanguage } from './lang'
  import type { EditorWrapMode } from './wrap-mode'

  type EditorTheme = 'light' | 'dark'

  /** Per-line diff markers passed in by the parent (1-based line numbers). */
  export type CodeEditorDiffMarkers = {
    added: number[]
    modified: number[]
    deletionAbove: number[]
    deletionAtEnd: boolean
  }

  const EMPTY_DIFF_MARKERS: CodeEditorDiffMarkers = {
    added: [],
    modified: [],
    deletionAbove: [],
    deletionAtEnd: false,
  }

  let {
    value = '',
    filePath = '',
    language = '',
    readonly = false,
    wrapMode = 'wrap',
    placeholder = '',
    class: className = '',
    diffMarkers = null,
    onchange,
    onselectionchange,
    onFormatDocument,
    onFormatSelection,
    onSave,
    onRevert,
    onExplainSelection,
    onRewriteSelection,
  }: {
    /** Current document content */
    value?: string
    /** Used to auto-detect language when `language` is not set */
    filePath?: string
    /** Explicit language id override */
    language?: string
    /** Read-only mode */
    readonly?: boolean
    /** Visual line wrapping mode */
    wrapMode?: EditorWrapMode
    /** Placeholder text when empty */
    placeholder?: string
    class?: string
    /** Per-line diff markers rendered in a left gutter. Pass null to hide. */
    diffMarkers?: CodeEditorDiffMarkers | null
    /** Fires on every edit */
    onchange?: (value: string) => void
    onselectionchange?: (selection: { from: number; to: number } | null) => void
    /** Format the whole document. Shown in the context menu and bound to Shift+Alt+F when no selection. */
    onFormatDocument?: () => void
    /** Format the current selection. Shown in the context menu and bound to Shift+Alt+F when a selection exists. */
    onFormatSelection?: () => void
    /** Save the current draft. Bound to Mod+S. */
    onSave?: () => void
    /** Revert the current draft. Shown in the context menu; pass undefined to hide. */
    onRevert?: () => void
    /** Ask the project assistant to explain the current selection. */
    onExplainSelection?: () => void
    /** Ask the project assistant to rewrite the current selection. */
    onRewriteSelection?: () => void
  } = $props()

  let container: HTMLDivElement
  let view: EditorView | undefined
  let suppressExternalUpdate = false

  type ContextMenuState = { x: number; y: number; hasSelection: boolean }
  let contextMenu = $state<ContextMenuState | null>(null)
  let contextMenuEl = $state<HTMLDivElement | null>(null)

  const lang = $derived(language || detectLanguage(filePath))

  function currentHasSelection(): boolean {
    if (!view) return false
    const sel = view.state.selection.main
    return sel.from !== sel.to
  }

  function closeContextMenu() {
    contextMenu = null
  }

  function handleContextMenu(event: MouseEvent) {
    const hasAnyAction =
      !!onFormatDocument ||
      !!onFormatSelection ||
      !!onRevert ||
      !!onExplainSelection ||
      !!onRewriteSelection
    if (!hasAnyAction) return
    event.preventDefault()
    // Rough clamp against the viewport so the menu doesn't spill off-screen.
    // A post-render effect could measure the real element, but a fixed
    // estimate is simpler and close enough for a 6-item menu.
    const estimatedWidth = 220
    const estimatedHeight = 240
    const x = Math.min(event.clientX, window.innerWidth - estimatedWidth - 8)
    const y = Math.min(event.clientY, window.innerHeight - estimatedHeight - 8)
    contextMenu = {
      x: Math.max(8, x),
      y: Math.max(8, y),
      hasSelection: currentHasSelection(),
    }
  }

  function runMenuAction(action?: () => void) {
    closeContextMenu()
    action?.()
  }

  $effect(() => {
    if (!contextMenu) return
    const onPointerDown = (event: MouseEvent) => {
      if (contextMenuEl && event.target instanceof Node && contextMenuEl.contains(event.target)) {
        return
      }
      closeContextMenu()
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        closeContextMenu()
      }
    }
    window.addEventListener('mousedown', onPointerDown)
    window.addEventListener('keydown', onKeyDown)
    return () => {
      window.removeEventListener('mousedown', onPointerDown)
      window.removeEventListener('keydown', onKeyDown)
    }
  })

  const customKeymap: KeyBinding[] = [
    {
      key: 'Shift-Alt-f',
      preventDefault: true,
      run: (target) => {
        const sel = target.state.selection.main
        if (sel.from !== sel.to && onFormatSelection) {
          onFormatSelection()
          return true
        }
        if (onFormatDocument) {
          onFormatDocument()
          return true
        }
        return false
      },
    },
    {
      key: 'Mod-s',
      preventDefault: true,
      run: () => {
        if (!onSave) return false
        onSave()
        return true
      },
    },
  ]

  // Theme shell (background, gutters, cursor, selection). Mirrors the harness
  // look used elsewhere, parameterized for light vs. dark.
  function editorThemeFor(mode: EditorTheme): Extension {
    const isDark = mode === 'dark'
    return EditorView.theme(
      {
        '&': {
          backgroundColor: 'transparent',
          color: isDark ? '#e6edf3' : '#1f2328',
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
        '.cm-cursor': { borderLeftColor: isDark ? '#e6edf3' : '#1f2328' },
        '.cm-selectionBackground': {
          backgroundColor: isDark
            ? 'rgba(56, 139, 253, 0.25) !important'
            : 'rgba(9, 105, 218, 0.18) !important',
        },
        '.cm-matchingBracket': {
          backgroundColor: isDark ? 'rgba(56, 139, 253, 0.15)' : 'rgba(9, 105, 218, 0.12)',
          outline: isDark
            ? '1px solid rgba(56, 139, 253, 0.4)'
            : '1px solid rgba(9, 105, 218, 0.35)',
        },
        '.cm-placeholder': {
          color: 'color-mix(in srgb, currentColor 30%, transparent)',
        },
      },
      { dark: isDark },
    )
  }

  // Syntax highlight palette matching shiki `github-dark-default`, so the
  // edit view stays visually consistent with the preview/CodeViewer.
  const githubDarkHighlight = HighlightStyle.define([
    { tag: t.keyword, color: '#ff7b72' },
    { tag: [t.name, t.deleted, t.character, t.propertyName, t.macroName], color: '#d2a8ff' },
    { tag: [t.function(t.variableName), t.labelName], color: '#d2a8ff' },
    { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: '#79c0ff' },
    { tag: [t.definition(t.name), t.separator], color: '#ffa657' },
    {
      tag: [
        t.typeName,
        t.className,
        t.number,
        t.changed,
        t.annotation,
        t.modifier,
        t.self,
        t.namespace,
      ],
      color: '#ffa657',
    },
    {
      tag: [t.operator, t.operatorKeyword, t.url, t.escape, t.regexp, t.special(t.string)],
      color: '#ff7b72',
    },
    { tag: [t.meta, t.comment], color: '#8b949e', fontStyle: 'italic' },
    { tag: t.strong, fontWeight: 'bold' },
    { tag: t.emphasis, fontStyle: 'italic' },
    { tag: t.strikethrough, textDecoration: 'line-through' },
    { tag: t.link, color: '#a5d6ff', textDecoration: 'underline' },
    { tag: t.heading, fontWeight: 'bold', color: '#79c0ff' },
    { tag: [t.atom, t.bool, t.special(t.variableName)], color: '#79c0ff' },
    { tag: [t.processingInstruction, t.string, t.inserted], color: '#a5d6ff' },
    { tag: t.invalid, color: '#f85149' },
  ])

  // Syntax highlight palette matching shiki `github-light-default`.
  const githubLightHighlight = HighlightStyle.define([
    { tag: t.keyword, color: '#cf222e' },
    { tag: [t.name, t.deleted, t.character, t.propertyName, t.macroName], color: '#8250df' },
    { tag: [t.function(t.variableName), t.labelName], color: '#8250df' },
    { tag: [t.color, t.constant(t.name), t.standard(t.name)], color: '#0550ae' },
    { tag: [t.definition(t.name), t.separator], color: '#953800' },
    {
      tag: [
        t.typeName,
        t.className,
        t.number,
        t.changed,
        t.annotation,
        t.modifier,
        t.self,
        t.namespace,
      ],
      color: '#953800',
    },
    {
      tag: [t.operator, t.operatorKeyword, t.url, t.escape, t.regexp, t.special(t.string)],
      color: '#cf222e',
    },
    { tag: [t.meta, t.comment], color: '#6e7781', fontStyle: 'italic' },
    { tag: t.strong, fontWeight: 'bold' },
    { tag: t.emphasis, fontStyle: 'italic' },
    { tag: t.strikethrough, textDecoration: 'line-through' },
    { tag: t.link, color: '#0a3069', textDecoration: 'underline' },
    { tag: t.heading, fontWeight: 'bold', color: '#0550ae' },
    { tag: [t.atom, t.bool, t.special(t.variableName)], color: '#0550ae' },
    { tag: [t.processingInstruction, t.string, t.inserted], color: '#0a3069' },
    { tag: t.invalid, color: '#82071e' },
  ])

  const themeCompartment = new Compartment()
  const wrapCompartment = new Compartment()

  function buildThemeExtensions(mode: EditorTheme): Extension[] {
    return [
      editorThemeFor(mode),
      syntaxHighlighting(mode === 'dark' ? githubDarkHighlight : githubLightHighlight, {
        fallback: true,
      }),
    ]
  }

  function buildWrapModeExtension(mode: EditorWrapMode): Extension {
    return mode === 'wrap' ? EditorView.lineWrapping : []
  }

  // ──────────────────────────── diff gutter ────────────────────────────
  // A 4 px-wide column placed to the left of the line numbers. Each line that
  // changed against the saved version gets a colored bar (green = added,
  // amber = modified). Lines below a deletion gap get a small red triangle at
  // the top edge; deletions past the end of the document put the triangle on
  // the bottom edge of the last line.

  type DiffMarkerInfo = {
    kind?: 'added' | 'modified'
    deletionAbove?: boolean
    deletionBelow?: boolean
  }

  class DiffGutterMarker extends GutterMarker {
    info: DiffMarkerInfo
    constructor(info: DiffMarkerInfo) {
      super()
      this.info = info
    }
    eq(other: GutterMarker): boolean {
      if (!(other instanceof DiffGutterMarker)) return false
      return (
        other.info.kind === this.info.kind &&
        Boolean(other.info.deletionAbove) === Boolean(this.info.deletionAbove) &&
        Boolean(other.info.deletionBelow) === Boolean(this.info.deletionBelow)
      )
    }
    toDOM() {
      const el = document.createElement('div')
      el.className = 'cm-diff-marker'
      if (this.info.kind) {
        el.dataset.kind = this.info.kind
      }
      if (this.info.deletionAbove) {
        el.dataset.deletionAbove = 'true'
      }
      if (this.info.deletionBelow) {
        el.dataset.deletionBelow = 'true'
      }
      if (this.info.kind) {
        const indicator = document.createElement('span')
        indicator.className = 'cm-diff-marker-indicator'
        indicator.setAttribute('aria-hidden', 'true')
        el.appendChild(indicator)
      }
      return el
    }
  }

  const setDiffMarkersEffect = StateEffect.define<CodeEditorDiffMarkers>()

  function buildDiffMarkerSet(
    state: EditorState,
    markers: CodeEditorDiffMarkers,
  ): RangeSet<DiffGutterMarker> {
    const totalLines = state.doc.lines
    const byLine = new Map<number, DiffMarkerInfo>()
    const stamp = (line: number, patch: DiffMarkerInfo) => {
      if (line < 1 || line > totalLines) return
      const existing = byLine.get(line) ?? {}
      byLine.set(line, { ...existing, ...patch })
    }
    for (const line of markers.added) stamp(line, { kind: 'added' })
    for (const line of markers.modified) stamp(line, { kind: 'modified' })
    for (const line of markers.deletionAbove) stamp(line, { deletionAbove: true })
    if (markers.deletionAtEnd && totalLines > 0) {
      stamp(totalLines, { deletionBelow: true })
    }

    const builder = new RangeSetBuilder<DiffGutterMarker>()
    const sorted = [...byLine.keys()].sort((a, b) => a - b)
    for (const lineNum of sorted) {
      const lineObj = state.doc.line(lineNum)
      const info = byLine.get(lineNum)
      if (!info) continue
      builder.add(lineObj.from, lineObj.from, new DiffGutterMarker(info))
    }
    return builder.finish()
  }

  const diffMarkersField = StateField.define<RangeSet<DiffGutterMarker>>({
    create() {
      // Real markers are seeded by the parent via setDiffMarkersEffect right
      // after the view is mounted, so an empty initial value is fine.
      return RangeSet.empty
    },
    update(value, tr) {
      let next = value.map(tr.changes)
      for (const effect of tr.effects) {
        if (effect.is(setDiffMarkersEffect)) {
          next = buildDiffMarkerSet(tr.state, effect.value)
        }
      }
      return next
    },
  })

  const diffGutterExtension: Extension = [
    diffMarkersField,
    gutter({
      class: 'cm-diff-gutter',
      markers: (v) => v.state.field(diffMarkersField),
      initialSpacer: () => new DiffGutterMarker({}),
    }),
  ]

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
      diffGutterExtension,
      lineNumbers(),
      history(),
      bracketMatching(),
      highlightActiveLine(),
      highlightSelectionMatches(),
      themeCompartment.of(buildThemeExtensions(appStore.theme)),
      wrapCompartment.of(buildWrapModeExtension(wrapMode)),
      // Custom bindings come first so Shift+Alt+F and Mod+S beat any default
      // binding that might otherwise swallow them.
      keymap.of([
        ...customKeymap,
        ...defaultKeymap,
        ...historyKeymap,
        ...searchKeymap,
        indentWithTab,
      ]),
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
          if (update.docChanged || update.selectionSet) {
            const selection = update.state.selection.main
            onselectionchange?.(
              selection.from === selection.to ? null : { from: selection.from, to: selection.to },
            )
          }
        }),
      )
    }

    return exts
  }

  async function createEditor() {
    const currentTheme = appStore.theme
    const currentDiffMarkers = diffMarkers ?? EMPTY_DIFF_MARKERS
    const langExts = await loadLanguageExtension(lang)
    view = new EditorView({
      state: EditorState.create({
        doc: value,
        extensions: [...buildBaseExtensions(), ...langExts],
      }),
      parent: container,
    })
    lastTheme = currentTheme
    lastWrapMode = wrapMode
    // Seed the diff gutter with whatever the parent passed at mount time. We
    // dispatch instead of using StateField.init() so the same code path runs
    // for both initial creation and later prop changes.
    view.dispatch({ effects: setDiffMarkersEffect.of(currentDiffMarkers) })
    lastDiffMarkers = currentDiffMarkers
  }

  let lastTheme: EditorTheme | '' = ''
  let lastWrapMode: EditorWrapMode = 'wrap'
  let lastDiffMarkers: CodeEditorDiffMarkers | null = null

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

  // Swap theme + highlight extensions when the app theme toggles, without
  // destroying the editor state.
  $effect(() => {
    const nextTheme = appStore.theme
    if (!view || nextTheme === lastTheme) return
    lastTheme = nextTheme
    view.dispatch({
      effects: themeCompartment.reconfigure(buildThemeExtensions(nextTheme)),
    })
  })

  $effect(() => {
    const nextWrapMode = wrapMode
    if (!view || nextWrapMode === lastWrapMode) return
    lastWrapMode = nextWrapMode
    view.dispatch({
      effects: wrapCompartment.reconfigure(buildWrapModeExtension(nextWrapMode)),
    })
  })

  // Push new diff marker payloads into the editor as they come from the parent.
  // We compare by reference: callers should produce a fresh object on every
  // change so this effect re-runs.
  $effect(() => {
    const nextMarkers = diffMarkers ?? EMPTY_DIFF_MARKERS
    if (!view || nextMarkers === lastDiffMarkers) return
    lastDiffMarkers = nextMarkers
    view.dispatch({ effects: setDiffMarkersEffect.of(nextMarkers) })
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
  oncontextmenu={handleContextMenu}
  role="presentation"
></div>

{#if contextMenu}
  <div
    bind:this={contextMenuEl}
    class="border-border bg-popover text-popover-foreground fixed z-50 min-w-[13rem] rounded-md border p-1 shadow-md"
    style="left: {contextMenu.x}px; top: {contextMenu.y}px"
    role="menu"
    data-testid="code-editor-context-menu"
  >
    {#if contextMenu.hasSelection && onFormatSelection}
      <button
        type="button"
        role="menuitem"
        class="hover:bg-accent hover:text-accent-foreground flex w-full items-center justify-between gap-4 rounded-sm px-2 py-1.5 text-left text-[12px]"
        onclick={() => runMenuAction(onFormatSelection)}
      >
        <span>{i18nStore.t('codeEditor.contextMenu.formatSelection')}</span>
        <span class="text-muted-foreground text-[10px]"
          >{i18nStore.t('codeEditor.contextMenu.formatShortcut')}</span
        >
      </button>
    {:else if onFormatDocument}
      <button
        type="button"
        role="menuitem"
        class="hover:bg-accent hover:text-accent-foreground flex w-full items-center justify-between gap-4 rounded-sm px-2 py-1.5 text-left text-[12px]"
        onclick={() => runMenuAction(onFormatDocument)}
      >
        <span>{i18nStore.t('codeEditor.contextMenu.formatDocument')}</span>
        <span class="text-muted-foreground text-[10px]"
          >{i18nStore.t('codeEditor.contextMenu.formatShortcut')}</span
        >
      </button>
    {/if}

    {#if onRevert}
      {#if (contextMenu.hasSelection && onFormatSelection) || onFormatDocument}
        <div class="bg-border my-1 h-px"></div>
      {/if}
      <button
        type="button"
        role="menuitem"
        class="hover:bg-accent hover:text-accent-foreground flex w-full items-center justify-between gap-4 rounded-sm px-2 py-1.5 text-left text-[12px]"
        onclick={() => runMenuAction(onRevert)}
      >
        <span>{i18nStore.t('codeEditor.contextMenu.revertFile')}</span>
      </button>
    {/if}

    {#if contextMenu.hasSelection && (onExplainSelection || onRewriteSelection)}
      <div class="bg-border my-1 h-px"></div>
      {#if onExplainSelection}
        <button
          type="button"
          role="menuitem"
          class="hover:bg-accent hover:text-accent-foreground flex w-full items-center justify-between gap-4 rounded-sm px-2 py-1.5 text-left text-[12px]"
          onclick={() => runMenuAction(onExplainSelection)}
        >
          <span>{i18nStore.t('codeEditor.contextMenu.explainSelection')}</span>
        </button>
      {/if}
      {#if onRewriteSelection}
        <button
          type="button"
          role="menuitem"
          class="hover:bg-accent hover:text-accent-foreground flex w-full items-center justify-between gap-4 rounded-sm px-2 py-1.5 text-left text-[12px]"
          onclick={() => runMenuAction(onRewriteSelection)}
        >
          <span>{i18nStore.t('codeEditor.contextMenu.rewriteSelection')}</span>
        </button>
      {/if}
    {/if}
  </div>
{/if}

<style>
  .code-editor :global(.cm-editor) {
    height: 100%;
  }

  /* ── diff gutter ─────────────────────────────────────────────── */
  .code-editor :global(.cm-gutter.cm-diff-gutter) {
    width: 12px;
    min-width: 12px;
    padding: 0;
    background: transparent;
    border-right: none;
    overflow: visible;
  }
  .code-editor :global(.cm-diff-gutter .cm-gutterElement) {
    padding: 0;
    overflow: visible;
  }
  .code-editor :global(.cm-diff-marker) {
    position: relative;
    width: 12px;
    height: 100%;
  }
  .code-editor :global(.cm-diff-marker[data-kind='added'])::before,
  .code-editor :global(.cm-diff-marker[data-kind='modified'])::before {
    content: '';
    position: absolute;
    top: 0;
    bottom: 0;
    left: 1px;
    width: 3px;
    border-radius: 999px;
  }
  .code-editor :global(.cm-diff-marker[data-kind='added'])::before {
    background-color: #2da44e;
  }
  .code-editor :global(.cm-diff-marker[data-kind='modified']) {
    background-color: transparent;
  }
  .code-editor :global(.cm-diff-marker[data-kind='modified'])::before {
    background-color: #d4a72c;
  }
  .code-editor :global(.cm-diff-marker-indicator) {
    display: none;
  }
  /* Red right-pointing triangle straddling the top edge of the line. */
  .code-editor :global(.cm-diff-marker[data-deletion-above='true'])::before {
    content: '';
    position: absolute;
    top: -3px;
    left: 1px;
    width: 0;
    height: 0;
    border-top: 4px solid transparent;
    border-bottom: 4px solid transparent;
    border-left: 9px solid #cf222e;
    pointer-events: none;
  }
  /* Same triangle on the bottom edge for end-of-file deletions. */
  .code-editor :global(.cm-diff-marker[data-deletion-below='true'])::after {
    content: '';
    position: absolute;
    bottom: -3px;
    left: 1px;
    width: 0;
    height: 0;
    border-top: 4px solid transparent;
    border-bottom: 4px solid transparent;
    border-left: 9px solid #cf222e;
    pointer-events: none;
  }
</style>
