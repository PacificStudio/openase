<script lang="ts">
  import { cn } from '$lib/utils'
  import type { EphemeralChatDiffEntry } from './transcript'

  let { entry }: { entry: EphemeralChatDiffEntry } = $props()

  const lineCount = $derived(entry.diff.hunks.reduce((count, hunk) => count + hunk.lines.length, 0))

  function hunkHeader(oldStart: number, oldLines: number, newStart: number, newLines: number) {
    return `@@ -${oldStart},${oldLines} +${newStart},${newLines} @@`
  }

  function linePrefix(op: 'context' | 'add' | 'remove') {
    if (op === 'add') {
      return '+'
    }
    if (op === 'remove') {
      return '-'
    }
    return ' '
  }
</script>

<div class="space-y-3 rounded-2xl border border-sky-500/30 bg-sky-500/8 p-3 text-sm">
  <div class="text-[10px] font-semibold tracking-[0.16em] uppercase opacity-70">assistant</div>
  <div class="font-medium">Structured Diff</div>
  <p class="text-muted-foreground text-xs leading-5">
    Target: <span class="font-mono">{entry.diff.file}</span>. {entry.diff.hunks.length} hunk{entry
      .diff.hunks.length === 1
      ? ''
      : 's'} / {lineCount} line operation{lineCount === 1 ? '' : 's'}.
  </p>

  {#each entry.diff.hunks as hunk}
    <div class="overflow-hidden rounded-xl border border-sky-500/20 bg-white/80">
      <div class="border-b border-sky-500/20 px-3 py-1.5 font-mono text-xs text-sky-900">
        {hunkHeader(hunk.oldStart, hunk.oldLines, hunk.newStart, hunk.newLines)}
      </div>
      <div>
        {#each hunk.lines as line}
          <div
            class={cn(
              'px-3 py-1 font-mono text-xs leading-5 whitespace-pre-wrap',
              line.op === 'add' && 'bg-emerald-500/10 text-emerald-950',
              line.op === 'remove' && 'bg-rose-500/10 text-rose-950',
              line.op === 'context' && 'text-slate-700',
            )}
          >
            {linePrefix(line.op)}{line.text}
          </div>
        {/each}
      </div>
    </div>
  {/each}
</div>
