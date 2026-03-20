<script lang="ts">
  import { cn } from '$lib/utils'
  import { ScrollArea } from '$ui/scroll-area'
  import type { DiffPreview } from '../assistant'

  let {
    preview,
  }: {
    preview: DiffPreview
  } = $props()
</script>

<div class="border-border overflow-hidden rounded-lg border">
  <div class="bg-muted/40 border-border flex items-center justify-between border-b px-3 py-2">
    <div class="text-foreground text-xs font-medium">Suggested Diff</div>
    <div class="flex items-center gap-3 text-[11px] font-medium">
      <span class="text-emerald-400">+{preview.addedCount}</span>
      <span class="text-rose-400">-{preview.removedCount}</span>
    </div>
  </div>

  <ScrollArea class="h-56 bg-[#0f1720]">
    <div class="font-mono text-[11px] leading-5 text-slate-200">
      {#each preview.lines as line, index (index)}
        <div
          class={cn(
            'grid grid-cols-[3.5rem_3.5rem_1.25rem_minmax(0,1fr)] gap-2 px-3 py-0.5',
            line.kind === 'context' && 'text-slate-500',
            line.kind === 'add' && 'bg-emerald-500/10 text-emerald-300',
            line.kind === 'remove' && 'bg-rose-500/10 text-rose-300',
          )}
        >
          <span class="text-right tabular-nums">
            {line.beforeLineNumber ?? ''}
          </span>
          <span class="text-right tabular-nums">
            {line.afterLineNumber ?? ''}
          </span>
          <span>{line.kind === 'add' ? '+' : line.kind === 'remove' ? '-' : ' '}</span>
          <span class="break-words whitespace-pre-wrap">{line.content || ' '}</span>
        </div>
      {/each}
    </div>
  </ScrollArea>
</div>
