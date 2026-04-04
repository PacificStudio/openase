<script lang="ts">
  import { tick } from 'svelte'
  import { cn } from '$lib/utils'
  import { FileCode, Copy, Check } from '@lucide/svelte'
  import {
    filterSuggestions,
    findCompletionState,
    flattenSuggestions,
  } from './harness-editor-autocomplete'
  import type { CompletionState, Suggestion } from './harness-editor-autocomplete'
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
  let textareaElement = $state<HTMLTextAreaElement | null>(null)
  let lineNumberElement = $state<HTMLDivElement | null>(null)
  let completionState = $state<CompletionState | null>(null)
  let activeSuggestionIndex = $state(0)
  let lines = $derived(content.rawContent.split('\n'))
  let suggestions = $derived(flattenSuggestions(variableGroups))
  let filteredSuggestions = $derived(filterSuggestions(suggestions, completionState))

  $effect(() => {
    if (activeSuggestionIndex >= filteredSuggestions.length) activeSuggestionIndex = 0
  })

  $effect(() => {
    if (filePath !== undefined) {
      completionState = null
      activeSuggestionIndex = 0
    }
  })

  function handleInput(e: Event) {
    const target = e.target as HTMLTextAreaElement
    onchange?.(target.value)
    refreshCompletion(target)
  }

  function handleCursorActivity(e: Event) {
    refreshCompletion(e.target as HTMLTextAreaElement)
  }

  async function handleKeydown(e: KeyboardEvent) {
    if (!completionState || filteredSuggestions.length === 0) {
      return
    }

    if (e.key === 'ArrowDown') {
      e.preventDefault()
      activeSuggestionIndex = (activeSuggestionIndex + 1) % filteredSuggestions.length
      return
    }

    if (e.key === 'ArrowUp') {
      e.preventDefault()
      activeSuggestionIndex =
        (activeSuggestionIndex + filteredSuggestions.length - 1) % filteredSuggestions.length
      return
    }

    if (e.key === 'Escape') {
      completionState = null
      activeSuggestionIndex = 0
      return
    }

    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      await applySuggestion(filteredSuggestions[activeSuggestionIndex])
    }
  }

  async function copyContent() {
    await navigator.clipboard.writeText(content.rawContent)
    copied = true
    setTimeout(() => (copied = false), 1500)
  }

  function handleScroll() {
    if (lineNumberElement && textareaElement) {
      lineNumberElement.scrollTop = textareaElement.scrollTop
    }
  }

  function refreshCompletion(target: HTMLTextAreaElement) {
    const nextState = findCompletionState(target.value, target.selectionStart)
    completionState = nextState
    activeSuggestionIndex = 0
  }

  async function applySuggestion(suggestion: Suggestion | undefined) {
    if (!suggestion || !completionState || !textareaElement) return

    const cursor = textareaElement.selectionStart
    const nextValue =
      textareaElement.value.slice(0, completionState.tokenStart) +
      suggestion.insertText +
      textareaElement.value.slice(cursor)
    const nextCursor = completionState.tokenStart + suggestion.insertText.length

    onchange?.(nextValue)
    completionState = null
    await tick()

    textareaElement?.focus()
    textareaElement?.setSelectionRange(nextCursor, nextCursor)
    if (textareaElement) refreshCompletion(textareaElement)
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

  <div class="relative flex flex-1 overflow-hidden bg-[#0d1117]">
    <div
      bind:this={lineNumberElement}
      class="shrink-0 overflow-hidden border-r border-neutral-800 bg-[#0d1117] px-3 py-3 text-right font-mono text-xs leading-6 text-neutral-600 select-none"
      aria-hidden="true"
    >
      {#each lines as _, i}
        <div>{i + 1}</div>
      {/each}
    </div>

    <textarea
      bind:this={textareaElement}
      class="h-full flex-1 resize-none bg-transparent p-3 font-mono text-xs leading-6 text-neutral-200 outline-none placeholder:text-neutral-600"
      spellcheck="false"
      value={content.rawContent}
      oninput={handleInput}
      onclick={handleCursorActivity}
      onkeyup={handleCursorActivity}
      onkeydown={handleKeydown}
      onscroll={handleScroll}
    ></textarea>

    {#if completionState && filteredSuggestions.length > 0}
      <div
        class="absolute right-4 bottom-4 z-10 w-[26rem] max-w-[calc(100%-2rem)] overflow-hidden rounded-lg border border-neutral-800 bg-[#111827]/95 shadow-2xl backdrop-blur"
      >
        <div
          class="flex items-center justify-between border-b border-neutral-800 px-3 py-2 text-[11px] tracking-[0.12em] text-neutral-400 uppercase"
        >
          <span>{completionState.mode === 'filter' ? 'Filters' : 'Variables'}</span>
          <span>{completionState.query || 'browse'}</span>
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
              onclick={() => applySuggestion(suggestion)}
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
          Enter / Tab inserts the highlighted item.
        </div>
      </div>
    {/if}
  </div>
</div>
