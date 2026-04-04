<script lang="ts">
  import { cn } from '$lib/utils'
  import { ChevronRight, Terminal, FileText, FilePenLine, Search, Keyboard } from '@lucide/svelte'
  import type { ProjectConversationToolCallEntry } from './project-conversation-transcript-types'
  import {
    summarizeToolCall,
    isExploringToolCall,
  } from './project-conversation-transcript-grouping'

  let {
    entry,
    standalone = false,
  }: { entry: ProjectConversationToolCallEntry; standalone?: boolean } = $props()

  let expanded = $state(false)

  const summary = $derived(summarizeToolCall(entry))
  const isExploring = $derived(isExploringToolCall(entry))

  function toolIcon(tool: string) {
    if (tool === 'functions.exec_command') return Terminal
    if (tool === 'functions.apply_patch') return FilePenLine
    if (tool === 'functions.write_stdin') return Keyboard
    if (tool.includes('search') || tool.includes('grep')) return Search
    return FileText
  }

  function readString(...keys: string[]) {
    const record =
      entry.arguments && typeof entry.arguments === 'object' && !Array.isArray(entry.arguments)
        ? (entry.arguments as Record<string, unknown>)
        : null
    for (const key of keys) {
      const value = record?.[key]
      if (typeof value === 'string' && value.trim()) return value
    }
    return ''
  }

  const command = $derived(readString('cmd', 'command'))
  const target = $derived(readString('path', 'file', 'target', 'workdir'))
  const patch = $derived(readString('patch'))

  const Icon = $derived(toolIcon(entry.tool))
</script>

<div class={cn('group', standalone && 'border-border/60 bg-muted/20 rounded-lg border')}>
  <button
    type="button"
    class="hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
    onclick={() => (expanded = !expanded)}
  >
    <ChevronRight
      class={cn(
        'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
        expanded && 'rotate-90',
      )}
    />
    <Icon class={cn('size-3.5 shrink-0', isExploring ? 'text-sky-500' : 'text-violet-500')} />
    <span class="text-foreground min-w-0 flex-1 truncate">
      {summary}
    </span>
    <span class="text-muted-foreground/60 shrink-0 font-mono text-[10px]">
      {entry.tool.replace('functions.', '')}
    </span>
  </button>

  {#if expanded}
    <div class="border-border/40 ml-5 space-y-2 border-l pt-1 pb-2 pl-3">
      {#if command}
        <div>
          <div
            class="text-muted-foreground mb-0.5 text-[10px] font-medium tracking-wider uppercase"
          >
            command
          </div>
          <pre
            class="bg-muted/60 overflow-x-auto rounded-md px-2.5 py-1.5 font-mono text-xs leading-5 whitespace-pre-wrap">{command}</pre>
        </div>
      {/if}

      {#if target}
        <div>
          <div
            class="text-muted-foreground mb-0.5 text-[10px] font-medium tracking-wider uppercase"
          >
            target
          </div>
          <div class="text-foreground/80 font-mono text-xs">{target}</div>
        </div>
      {/if}

      {#if patch}
        <div>
          <div
            class="text-muted-foreground mb-0.5 text-[10px] font-medium tracking-wider uppercase"
          >
            patch
          </div>
          <pre
            class="bg-muted/60 max-h-60 overflow-auto rounded-md px-2.5 py-1.5 font-mono text-xs leading-5 whitespace-pre-wrap">{patch}</pre>
        </div>
      {/if}

      <details class="text-xs">
        <summary class="text-muted-foreground hover:text-foreground cursor-pointer">
          Raw arguments
        </summary>
        <pre
          class="bg-muted/60 mt-1 max-h-48 overflow-auto rounded-md px-2.5 py-1.5 font-mono text-[11px] leading-5 whitespace-pre-wrap">{JSON.stringify(
            entry.arguments,
            null,
            2,
          )}</pre>
      </details>
    </div>
  {/if}
</div>
