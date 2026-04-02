<script lang="ts">
  import { ChevronRight, FilePenLine, FileText, Keyboard, Search, Terminal } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import type { TicketRunTranscriptBlock } from '../types'

  let { block }: { block: Extract<TicketRunTranscriptBlock, { kind: 'tool_call' }> } = $props()

  let expanded = $state(false)

  function toolIcon(tool: string) {
    if (tool === 'functions.exec_command' || tool === 'exec_command') return Terminal
    if (tool === 'functions.apply_patch' || tool === 'apply_patch') return FilePenLine
    if (tool === 'functions.write_stdin' || tool === 'write_stdin') return Keyboard
    if (tool.includes('search') || tool.includes('grep')) return Search
    return FileText
  }

  function isExploring(tool: string) {
    return (
      tool.includes('read') ||
      tool.includes('search') ||
      tool.includes('grep') ||
      tool.includes('list') ||
      tool.includes('glob') ||
      tool.includes('find')
    )
  }

  function readString(...keys: string[]) {
    const record =
      block.arguments && typeof block.arguments === 'object' && !Array.isArray(block.arguments)
        ? (block.arguments as Record<string, unknown>)
        : null
    for (const key of keys) {
      const value = record?.[key]
      if (typeof value === 'string' && value.trim()) return value
    }
    return ''
  }

  function truncateInline(text: string, max: number) {
    const line = text.replace(/\n/g, ' ').trim()
    return line.length <= max ? line : `${line.slice(0, max - 3)}...`
  }

  function shortenPath(path: string) {
    const parts = path.split('/')
    return parts.length <= 3 ? path : `.../${parts.slice(-2).join('/')}`
  }

  const Icon = $derived(toolIcon(block.toolName))
  const command = $derived(readString('cmd', 'command'))
  const target = $derived(readString('path', 'file', 'target', 'workdir'))
  const patch = $derived(readString('patch'))

  const summary = $derived.by(() => {
    if (command) return `Ran \`${truncateInline(command, 60)}\``
    if (target) return shortenPath(target)
    return block.toolName.replace('functions.', '')
  })
</script>

<div class="group">
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
    <Icon
      class={cn(
        'size-3.5 shrink-0',
        isExploring(block.toolName) ? 'text-sky-500' : 'text-violet-500',
      )}
    />
    <span class="text-foreground min-w-0 flex-1 truncate">
      {summary}
    </span>
    <span class="text-muted-foreground/60 shrink-0 font-mono text-[10px]">
      {block.toolName.replace('functions.', '')}
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
        <summary class="text-muted-foreground hover:text-foreground cursor-pointer"
          >Raw arguments</summary
        >
        <pre
          class="bg-muted/60 mt-1 max-h-48 overflow-auto rounded-md px-2.5 py-1.5 font-mono text-[11px] leading-5 whitespace-pre-wrap">{JSON.stringify(
            block.arguments ?? {},
            null,
            2,
          )}</pre>
      </details>
    </div>
  {/if}
</div>
