<script lang="ts">
  import { cn } from '$lib/utils'
  import { ChevronRight } from '@lucide/svelte'
  import type { ProjectConversationCommandOutputEntry } from './project-conversation-transcript-types'
  import { countOutputLines, truncateInline, truncateOutput } from './command-output-truncation'

  let {
    entry,
    standalone = false,
  }: { entry: ProjectConversationCommandOutputEntry; standalone?: boolean } = $props()

  let expanded = $state(false)
  let showFullOutput = $state(false)

  const title = $derived(entry.command ? truncateInline(entry.command, 72) : 'Output')
  const truncated = $derived(truncateOutput(entry.content, 5, 5))
  const lineCount = $derived(countOutputLines(entry.content))
  const isStderr = $derived(entry.stream === 'stderr')

  const metaParts = $derived.by(() => {
    const parts: string[] = []
    if (entry.stream) parts.push(entry.stream)
    if (entry.phase) parts.push(entry.phase)
    parts.push(`${lineCount} line${lineCount === 1 ? '' : 's'}`)
    return parts
  })
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
    <span class={cn('size-1.5 shrink-0 rounded-full', isStderr ? 'bg-red-400' : 'bg-emerald-400')}
    ></span>
    <span class={cn('text-foreground min-w-0 flex-1 truncate', entry.command && 'font-mono')}>
      {title}
    </span>
    <span class="text-muted-foreground/60 flex shrink-0 items-center gap-1.5 text-[10px]">
      {#each metaParts as part, i}
        {#if i > 0}<span class="opacity-40">&middot;</span>{/if}
        <span>{part}</span>
      {/each}
    </span>
  </button>

  {#if expanded}
    <div class="border-border/40 ml-5 border-l pt-1 pb-2 pl-3">
      {#if entry.command}
        <div class="mb-2">
          <div
            class="text-muted-foreground mb-0.5 text-[10px] font-medium tracking-wider uppercase"
          >
            command
          </div>
          <pre
            class="bg-muted/60 overflow-x-auto rounded-md px-2.5 py-1.5 font-mono text-xs leading-5 whitespace-pre-wrap">{entry.command}</pre>
        </div>
      {/if}
      <div
        class="overflow-x-auto rounded-md bg-slate-950 px-3 py-2 font-mono text-xs leading-5 whitespace-pre-wrap text-slate-200"
      >
        {#if truncated && !showFullOutput}
          {truncated.head}
          <button
            type="button"
            class="my-1 block w-full rounded bg-slate-800 px-2 py-0.5 text-center text-[11px] text-slate-400 transition-colors hover:bg-slate-700 hover:text-slate-200"
            onclick={(e) => {
              e.stopPropagation()
              showFullOutput = true
            }}
          >
            ... +{truncated.omitted} lines hidden — click to expand
          </button>
          {truncated.tail}
        {:else}
          {entry.content}
        {/if}
      </div>
      {#if showFullOutput && truncated}
        <button
          type="button"
          class="text-muted-foreground hover:text-foreground mt-1 text-[11px]"
          onclick={() => (showFullOutput = false)}
        >
          Collapse output
        </button>
      {/if}
    </div>
  {/if}
</div>
