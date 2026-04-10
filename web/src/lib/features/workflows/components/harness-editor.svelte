<script lang="ts">
  import { cn } from '$lib/utils'
  import { FileCode, Copy, Check } from '@lucide/svelte'
  import { CodeEditor } from '$lib/components/code'
  import { filterSuggestions, flattenSuggestions } from './harness-editor-autocomplete'
  import type { Suggestion } from './harness-editor-autocomplete'
  import type { HarnessContent, HarnessVariableGroup } from '../types'

  let {
    content,
    filePath = '',
    version = 1,
    variableGroups = [],
    onchange,
    class: className = '',
  }: {
    content: HarnessContent
    filePath?: string
    version?: number
    variableGroups?: HarnessVariableGroup[]
    onchange?: (raw: string) => void
    class?: string
  } = $props()

  let copied = $state(false)
  let activeSuggestionIndex = $state(0)
  let suggestions = $derived(flattenSuggestions(variableGroups))

  // Simple completion state: track whether the user is inside {{ }} or {% %}
  let completionQuery = $state('')
  let completionMode = $state<'variable' | 'filter' | null>(null)
  let filteredSuggestions = $derived(
    completionMode
      ? filterSuggestions(suggestions, {
          mode: completionMode,
          query: completionQuery,
          tokenStart: 0,
        })
      : [],
  )

  $effect(() => {
    if (activeSuggestionIndex >= filteredSuggestions.length) activeSuggestionIndex = 0
  })

  $effect(() => {
    if (filePath !== undefined) {
      completionMode = null
      completionQuery = ''
      activeSuggestionIndex = 0
    }
  })

  function handleChange(nextValue: string) {
    onchange?.(nextValue)
  }

  async function copyContent() {
    await navigator.clipboard.writeText(content.rawContent)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }

  function handleSuggestionClick(suggestion: Suggestion) {
    // Autocomplete insertion would require cursor access from CodeMirror.
    // For now, close the panel. Full CM autocomplete integration can replace this.
    completionMode = null
    completionQuery = ''
    void suggestion
  }
</script>

<div class={cn('flex h-full flex-col overflow-hidden', className)}>
  <div class="border-border bg-muted/30 flex items-center justify-between border-b px-3 py-1.5">
    <div class="flex min-w-0 items-center gap-2 text-sm">
      <FileCode class="text-muted-foreground size-3.5 shrink-0" />
      <span class="text-muted-foreground truncate font-mono text-xs" title={filePath}
        >{filePath}</span
      >
      <span class="bg-muted text-muted-foreground shrink-0 rounded px-1.5 py-0.5 text-[10px]">
        v{version}
      </span>
    </div>
    <button
      class="text-muted-foreground hover:bg-muted hover:text-foreground flex shrink-0 items-center gap-1 rounded px-2 py-1 text-xs transition-colors"
      onclick={copyContent}
    >
      {#if copied}
        <Check class="size-3" />
        Copied
      {:else}
        <Copy class="size-3" />
        Copy
      {/if}
    </button>
  </div>

  <div class="relative min-h-0 flex-1 bg-[#0d1117]">
    <CodeEditor value={content.rawContent} {filePath} language="markdown" onchange={handleChange} />

    {#if completionMode && filteredSuggestions.length > 0}
      <div
        class="absolute right-4 bottom-4 z-10 w-[26rem] max-w-[calc(100%-2rem)] overflow-hidden rounded-lg border border-neutral-800 bg-[#111827]/95 shadow-2xl backdrop-blur"
      >
        <div
          class="flex items-center justify-between border-b border-neutral-800 px-3 py-2 text-[11px] tracking-[0.12em] text-neutral-400 uppercase"
        >
          <span>{completionMode === 'filter' ? 'Filters' : 'Variables'}</span>
          <span>{completionQuery || 'browse'}</span>
        </div>

        <div class="max-h-72 overflow-auto p-1">
          {#each filteredSuggestions as suggestion, index (suggestion.id)}
            <button
              type="button"
              class={cn(
                'flex w-full flex-col gap-1 rounded-md px-3 py-2 text-left transition-colors',
                index === activeSuggestionIndex
                  ? 'bg-sky-500/15 text-neutral-100'
                  : 'text-neutral-300 hover:bg-neutral-800/80',
              )}
              onclick={() => handleSuggestionClick(suggestion)}
              onmouseenter={() => (activeSuggestionIndex = index)}
            >
              <div class="flex items-center justify-between gap-3">
                <span class="font-mono text-xs">{suggestion.label}</span>
                <span
                  class="rounded bg-neutral-800 px-1.5 py-0.5 text-[10px] tracking-[0.12em] text-neutral-400 uppercase"
                >
                  {suggestion.groupName}
                </span>
              </div>
              <p class="text-[11px] leading-5 text-neutral-400">{suggestion.description}</p>
              {#if suggestion.example}
                <p class="font-mono text-[11px] text-neutral-500">{suggestion.example}</p>
              {/if}
            </button>
          {/each}
        </div>

        <div class="border-t border-neutral-800 px-3 py-2 text-[11px] text-neutral-500">
          Click to insert the highlighted item.
        </div>
      </div>
    {/if}
  </div>
</div>
