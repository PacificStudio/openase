<script lang="ts">
  import { ChevronRight } from '@lucide/svelte'
  import { cn } from '$lib/utils'
  import { ChatMarkdownContent } from '$lib/features/chat'
  import type { TicketRunTranscriptBlock } from '../types'

  let {
    block,
    expanded = false,
    onToggle,
  }: {
    block: Extract<TicketRunTranscriptBlock, { kind: 'assistant_message' | 'terminal_output' }>
    expanded?: boolean
    onToggle?: () => Promise<void> | void
  } = $props()

  function truncateInline(text: string, max: number) {
    if (text.length <= max) return text
    return `${text.slice(0, max - 3)}...`
  }

  function lineCount(text: string) {
    return text.split('\n').length
  }

  function truncateOutput(text: string, headLines = 5, tailLines = 5) {
    const lines = text.split('\n')
    if (lines.length <= headLines + tailLines) return null
    const omitted = lines.length - headLines - tailLines
    return {
      head: lines.slice(0, headLines).join('\n'),
      omitted,
      tail: lines.slice(lines.length - tailLines).join('\n'),
    }
  }
</script>

{#if block.kind === 'assistant_message'}
  <div class="prose prose-sm prose-neutral max-w-none break-words">
    <ChatMarkdownContent source={block.text} />
  </div>
{:else}
  {@const isStderr = block.stream === 'stderr'}
  {@const title = block.command ? truncateInline(block.command, 72) : 'Output'}
  {@const lines = lineCount(block.text)}
  {@const truncated = truncateOutput(block.text, 5, 5)}

  <div class="group">
    <button
      type="button"
      class="hover:bg-muted/40 flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-xs transition-colors"
      onclick={() => void onToggle?.()}
    >
      <ChevronRight
        class={cn(
          'text-muted-foreground size-3 shrink-0 transition-transform duration-150',
          expanded && 'rotate-90',
        )}
      />
      <span class={cn('size-1.5 shrink-0 rounded-full', isStderr ? 'bg-red-400' : 'bg-emerald-400')}
      ></span>
      <span class={cn('text-foreground min-w-0 flex-1 truncate', block.command && 'font-mono')}>
        {title}
      </span>
      <span class="text-muted-foreground/60 flex shrink-0 items-center gap-1.5 text-[10px]">
        {#if block.stream}
          <span>{block.stream}</span>
          <span class="opacity-40">&middot;</span>
        {/if}
        <span>{lines} line{lines === 1 ? '' : 's'}</span>
        {#if block.streaming}
          <span class="opacity-40">&middot;</span>
          <span>streaming</span>
        {/if}
      </span>
    </button>

    {#if expanded}
      <div class="border-border/40 ml-5 border-l pt-1 pb-2 pl-3">
        {#if block.command}
          <div class="mb-2">
            <div
              class="text-muted-foreground mb-0.5 text-[10px] font-medium tracking-wider uppercase"
            >
              command
            </div>
            <pre
              class="bg-muted/60 overflow-x-auto rounded-md px-2.5 py-1.5 font-mono text-xs leading-5 whitespace-pre-wrap">{block.command}</pre>
          </div>
        {/if}
        <div
          class="overflow-x-auto rounded-md bg-slate-950 px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap text-slate-200"
        >
          {#if truncated && !expanded}
            {truncated.head}
          {:else}
            {block.text}
          {/if}
        </div>
      </div>
    {/if}
  </div>
{/if}
