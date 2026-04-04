<script lang="ts">
  import { ChevronRight, FileDiff } from '@lucide/svelte'
  import type { ChatDiffPayload } from '$lib/api/chat'
  import { cn } from '$lib/utils'

  let { diff }: { diff: ChatDiffPayload } = $props()

  let expanded = $state(false)

  const adds = $derived(
    diff.hunks.reduce((n, h) => n + h.lines.filter((line) => line.op === 'add').length, 0),
  )
  const removes = $derived(
    diff.hunks.reduce((n, h) => n + h.lines.filter((line) => line.op === 'remove').length, 0),
  )

  function shortenPath(path: string) {
    const parts = path.split('/')
    return parts.length <= 3 ? path : `.../${parts.slice(-2).join('/')}`
  }

  function linePrefix(op: 'context' | 'add' | 'remove') {
    if (op === 'add') return '+'
    if (op === 'remove') return '-'
    return ' '
  }

  function hunkHeader(oldStart: number, oldLines: number, newStart: number, newLines: number) {
    return `@@ -${oldStart},${oldLines} +${newStart},${newLines} @@`
  }
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
    <FileDiff class="size-3.5 shrink-0 text-sky-500" />
    <span class="text-foreground min-w-0 flex-1 truncate font-mono">{shortenPath(diff.file)}</span>
    <span class="text-muted-foreground/60 flex shrink-0 items-center gap-1.5 text-[10px]">
      {#if adds}<span class="text-emerald-600">+{adds}</span>{/if}
      {#if removes}<span class="text-rose-600">-{removes}</span>{/if}
      <span>{diff.hunks.length} hunk{diff.hunks.length === 1 ? '' : 's'}</span>
    </span>
  </button>

  {#if expanded}
    <div class="border-border/40 ml-5 border-l pt-1 pb-2 pl-3">
      <div class="overflow-hidden rounded-md bg-slate-950">
        {#each diff.hunks as hunk, i}
          <div
            class="px-3 py-0.5 font-mono text-[10px] text-slate-400"
            class:border-t={i > 0}
            class:border-slate-800={i > 0}
          >
            {hunkHeader(hunk.oldStart, hunk.oldLines, hunk.newStart, hunk.newLines)}
          </div>
          {#each hunk.lines as line}
            <div
              class={cn(
                'px-3 font-mono text-[11px] leading-5 whitespace-pre-wrap',
                line.op === 'add' && 'bg-emerald-500/15 text-emerald-300',
                line.op === 'remove' && 'bg-rose-500/15 text-rose-300',
                line.op === 'context' && 'text-slate-400',
              )}
            >
              {linePrefix(line.op)}{line.text}
            </div>
          {/each}
        {/each}
      </div>
    </div>
  {/if}
</div>
